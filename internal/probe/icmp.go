package probe

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	protocolICMPv4 = 1
	protocolICMPv6 = 58
	icmpHeaderLen  = 8

	readPollInterval = 100 * time.Millisecond
)

var listenICMPPacket = icmp.ListenPacket

var icmpHeaderCounter atomic.Uint32

type icmpProber struct {
	conn *icmp.PacketConn
	dst  net.Addr

	ipv6       bool
	echoType   icmp.Type
	packetSize int

	mu sync.Mutex
}

type icmpReply struct {
	headerToken uint32
	kind        ReplyKind
	icmpType    int
	icmpCode    int
	annotation  string
}

func init() {
	var seed [8]byte
	if _, err := rand.Read(seed[:]); err != nil {
		binary.BigEndian.PutUint64(seed[:], uint64(time.Now().UnixNano())^uint64(os.Getpid())<<32)
	}

	value := binary.BigEndian.Uint64(seed[:])
	icmpHeaderCounter.Store(uint32(value))
}

func newICMPProber(ctx context.Context, dst netip.Addr, opts Options) (Prober, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	network := "ip4:icmp"
	address := "0.0.0.0"
	echoType := icmp.Type(ipv4.ICMPTypeEcho)
	ipv6Trace := dst.Is6()
	if ipv6Trace {
		network = "ip6:ipv6-icmp"
		address = "::"
		echoType = icmp.Type(ipv6.ICMPTypeEchoRequest)
	}

	conn, err := listenICMPPacket(network, address)
	if err != nil {
		return nil, socketError(err)
	}

	packetSize := opts.PacketSize
	if packetSize < icmpHeaderLen {
		packetSize = icmpHeaderLen
	}

	return &icmpProber{
		conn:       conn,
		dst:        &net.IPAddr{IP: netIPFromAddr(dst)},
		ipv6:       ipv6Trace,
		echoType:   echoType,
		packetSize: packetSize,
	}, nil
}

func (p *icmpProber) Send(ctx context.Context, ttl int, attempt int) (Sent, error) {
	if err := ctx.Err(); err != nil {
		return Sent{}, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	identifier, seq, headerToken := nextICMPID()

	if err := p.setHopLimit(ttl); err != nil {
		return Sent{}, socketError(err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = p.conn.SetWriteDeadline(deadline)
	} else {
		_ = p.conn.SetWriteDeadline(time.Time{})
	}

	msg := icmp.Message{
		Type: p.echoType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   identifier,
			Seq:  seq,
			Data: paddingData(p.packetSize - icmpHeaderLen),
		},
	}
	packet, err := msg.Marshal(nil)
	if err != nil {
		return Sent{}, err
	}

	sentAt := time.Now()
	if _, err := p.conn.WriteTo(packet, p.dst); err != nil {
		return Sent{}, socketError(err)
	}

	return Sent{
		HeaderToken: headerToken,
		TTL:         ttl,
		Attempt:     attempt,
		SentAt:      sentAt,
	}, nil
}

func (p *icmpProber) Receive(ctx context.Context, sent Sent, timeout time.Duration) (Reply, error) {
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	buf := make([]byte, 1500)
	for {
		if err := ctx.Err(); err != nil {
			return Reply{}, err
		}

		readDeadline := deadline
		if pollDeadline := time.Now().Add(readPollInterval); pollDeadline.Before(readDeadline) {
			readDeadline = pollDeadline
		}
		if err := p.conn.SetReadDeadline(readDeadline); err != nil {
			return Reply{}, err
		}

		n, peer, err := p.conn.ReadFrom(buf)
		receivedAt := time.Now()
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return Reply{}, ctxErr
			}
			if isTimeout(err) {
				if time.Now().Before(deadline) {
					continue
				}
				return Reply{}, ErrTimeout
			}
			return Reply{}, socketError(err)
		}

		parsed, ok := parseICMPReply(p.ipv6, buf[:n])
		if !ok || parsed.headerToken != sent.HeaderToken {
			continue
		}

		return Reply{
			From:       addrFromNetAddr(peer),
			ReceivedAt: receivedAt,
			RTT:        receivedAt.Sub(sent.SentAt),
			Kind:       parsed.kind,
			ICMPType:   parsed.icmpType,
			ICMPCode:   parsed.icmpCode,
			Annotation: parsed.annotation,
		}, nil
	}
}

func (p *icmpProber) Close() error {
	return p.conn.Close()
}

func (p *icmpProber) setHopLimit(ttl int) error {
	if p.ipv6 {
		return p.conn.IPv6PacketConn().SetHopLimit(ttl)
	}
	return p.conn.IPv4PacketConn().SetTTL(ttl)
}

func parseICMPReply(ipv6Trace bool, packet []byte) (icmpReply, bool) {
	protocol := protocolICMPv4
	if ipv6Trace {
		protocol = protocolICMPv6
		packet = stripIPv6Packet(packet)
	} else {
		packet = stripIPv4Packet(packet)
	}

	msg, err := icmp.ParseMessage(protocol, packet)
	if err != nil {
		return icmpReply{}, false
	}
	return classifyICMPMessage(ipv6Trace, msg)
}

func classifyICMPMessage(ipv6Trace bool, msg *icmp.Message) (icmpReply, bool) {
	reply := icmpReply{
		icmpType: icmpTypeNumber(msg.Type),
		icmpCode: msg.Code,
	}

	if ipv6Trace {
		return classifyICMPv6Message(reply, msg)
	}
	return classifyICMPv4Message(reply, msg)
}

func classifyICMPv4Message(reply icmpReply, msg *icmp.Message) (icmpReply, bool) {
	switch msg.Type {
	case ipv4.ICMPTypeEchoReply:
		headerToken, ok := headerTokenFromEcho(msg)
		reply.headerToken = headerToken
		reply.kind = ReplyDestination
		return reply, ok
	case ipv4.ICMPTypeTimeExceeded:
		body, ok := msg.Body.(*icmp.TimeExceeded)
		if !ok {
			return icmpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedPacket(false, body.Data)
		reply.headerToken = headerToken
		reply.kind = ReplyTimeExceeded
		reply.annotation = ipv4TimeExceededAnnotation(msg.Code)
		return reply, ok
	case ipv4.ICMPTypeDestinationUnreachable:
		body, ok := msg.Body.(*icmp.DstUnreach)
		if !ok {
			return icmpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedPacket(false, body.Data)
		reply.headerToken = headerToken
		reply.kind = ipv4DestinationUnreachableKind(msg.Code)
		reply.annotation = ipv4DestinationUnreachableAnnotation(msg.Code)
		return reply, ok
	default:
		return icmpReply{}, false
	}
}

func classifyICMPv6Message(reply icmpReply, msg *icmp.Message) (icmpReply, bool) {
	switch msg.Type {
	case ipv6.ICMPTypeEchoReply:
		headerToken, ok := headerTokenFromEcho(msg)
		reply.headerToken = headerToken
		reply.kind = ReplyDestination
		return reply, ok
	case ipv6.ICMPTypeTimeExceeded:
		body, ok := msg.Body.(*icmp.TimeExceeded)
		if !ok {
			return icmpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedPacket(true, body.Data)
		reply.headerToken = headerToken
		reply.kind = ReplyTimeExceeded
		reply.annotation = ipv6TimeExceededAnnotation(msg.Code)
		return reply, ok
	case ipv6.ICMPTypeDestinationUnreachable:
		body, ok := msg.Body.(*icmp.DstUnreach)
		if !ok {
			return icmpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedPacket(true, body.Data)
		reply.headerToken = headerToken
		reply.kind = ipv6DestinationUnreachableKind(msg.Code)
		reply.annotation = ipv6DestinationUnreachableAnnotation(msg.Code)
		return reply, ok
	case ipv6.ICMPTypePacketTooBig:
		body, ok := msg.Body.(*icmp.PacketTooBig)
		if !ok {
			return icmpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedPacket(true, body.Data)
		reply.headerToken = headerToken
		reply.kind = ReplyPacketTooBig
		reply.annotation = "packet too big"
		return reply, ok
	default:
		return icmpReply{}, false
	}
}

func headerTokenFromEcho(msg *icmp.Message) (uint32, bool) {
	echo, ok := msg.Body.(*icmp.Echo)
	if !ok {
		return 0, false
	}
	return makeICMPHeaderToken(echo.ID, echo.Seq), true
}

func headerTokenFromEmbeddedPacket(ipv6Trace bool, packet []byte) (uint32, bool) {
	protocol := protocolICMPv4
	if ipv6Trace {
		protocol = protocolICMPv6
		packet = stripIPv6Packet(packet)
	} else {
		packet = stripIPv4Packet(packet)
	}

	msg, err := icmp.ParseMessage(protocol, packet)
	if err != nil {
		return 0, false
	}

	if ipv6Trace && msg.Type != ipv6.ICMPTypeEchoRequest {
		return 0, false
	}
	if !ipv6Trace && msg.Type != ipv4.ICMPTypeEcho {
		return 0, false
	}
	return headerTokenFromEcho(msg)
}

func stripIPv4Packet(packet []byte) []byte {
	if len(packet) == 0 || packet[0]>>4 != 4 {
		return packet
	}
	header, err := ipv4.ParseHeader(packet)
	if err != nil || header.Protocol != protocolICMPv4 || len(packet) < header.Len {
		return packet
	}
	return packet[header.Len:]
}

func stripIPv6Packet(packet []byte) []byte {
	if len(packet) == 0 || packet[0]>>4 != 6 {
		return packet
	}
	header, err := ipv6.ParseHeader(packet)
	if err != nil || header.NextHeader != protocolICMPv6 || len(packet) < ipv6.HeaderLen {
		return packet
	}
	return packet[ipv6.HeaderLen:]
}

func nextICMPID() (identifier int, sequence int, headerToken uint32) {
	headerToken = icmpHeaderCounter.Add(1)
	identifier = int(headerToken >> 16)
	sequence = int(headerToken & 0xffff)

	return identifier, sequence, headerToken
}

func makeICMPHeaderToken(identifier int, sequence int) uint32 {
	return uint32(uint16(identifier))<<16 | uint32(uint16(sequence))
}

func paddingData(size int) []byte {
	if size <= 0 {
		return nil
	}
	return make([]byte, size)
}

func icmpTypeNumber(typ icmp.Type) int {
	switch v := typ.(type) {
	case ipv4.ICMPType:
		return int(v)
	case ipv6.ICMPType:
		return int(v)
	default:
		return 0
	}
}

func ipv4DestinationUnreachableKind(code int) ReplyKind {
	switch code {
	case 4:
		return ReplyPacketTooBig
	case 9, 10, 13:
		return ReplyFiltered
	default:
		return ReplyUnreachable
	}
}

func ipv6DestinationUnreachableKind(code int) ReplyKind {
	switch code {
	case 1, 5, 6:
		return ReplyFiltered
	default:
		return ReplyUnreachable
	}
}

func ipv4TimeExceededAnnotation(code int) string {
	switch code {
	case 0:
		return "time to live exceeded"
	case 1:
		return "fragment reassembly time exceeded"
	default:
		return "time exceeded"
	}
}

func ipv6TimeExceededAnnotation(code int) string {
	switch code {
	case 0:
		return "hop limit exceeded"
	case 1:
		return "fragment reassembly time exceeded"
	default:
		return "time exceeded"
	}
}

func ipv4DestinationUnreachableAnnotation(code int) string {
	switch code {
	case 0:
		return "network unreachable"
	case 1:
		return "host unreachable"
	case 2:
		return "protocol unreachable"
	case 3:
		return "port unreachable"
	case 4:
		return "fragmentation needed"
	case 9:
		return "network administratively prohibited"
	case 10:
		return "host administratively prohibited"
	case 13:
		return "communication administratively prohibited"
	default:
		return "destination unreachable"
	}
}

func ipv6DestinationUnreachableAnnotation(code int) string {
	switch code {
	case 0:
		return "no route to destination"
	case 1:
		return "communication administratively prohibited"
	case 3:
		return "address unreachable"
	case 4:
		return "port unreachable"
	case 5:
		return "source address failed policy"
	case 6:
		return "reject route to destination"
	default:
		return "destination unreachable"
	}
}

func socketError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		return errors.Join(ErrPermission, err)
	}
	return err
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func netIPFromAddr(addr netip.Addr) net.IP {
	if addr.Is4() {
		ip := addr.As4()
		return net.IPv4(ip[0], ip[1], ip[2], ip[3])
	}
	ip := addr.As16()
	return net.IP(append([]byte(nil), ip[:]...))
}

func addrFromNetAddr(addr net.Addr) netip.Addr {
	switch a := addr.(type) {
	case *net.IPAddr:
		return addrFromIP(a.IP)
	case *net.UDPAddr:
		return addrFromIP(a.IP)
	default:
		return netip.Addr{}
	}
}

func addrFromIP(ip net.IP) netip.Addr {
	if ip4 := ip.To4(); ip4 != nil {
		addr, _ := netip.AddrFromSlice(ip4)
		return addr
	}
	if ip16 := ip.To16(); ip16 != nil {
		addr, _ := netip.AddrFromSlice(ip16)
		return addr
	}
	return netip.Addr{}
}
