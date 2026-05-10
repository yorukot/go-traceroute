package probe

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	udpHeaderLen           = 8
	internalUDPDefaultPort = 33434
)

var newUDPProber = openUDPProber

type udpProber struct {
	sendConn *net.UDPConn
	recvConn *icmp.PacketConn
	dst      net.IP

	ipv6          bool
	packetSize    int
	firstHop      int
	queriesPerHop int
	udpBasePort   int
	sourcePort    int

	mu sync.Mutex
}

type udpReply struct {
	headerToken uint32
	kind        ReplyKind
	icmpType    int
	icmpCode    int
	annotation  string
}

func openUDPProber(ctx context.Context, dst netip.Addr, opts Options) (Prober, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	opts = normalizeUDPOptions(opts)
	if opts.UDPBasePort < 1 || opts.UDPBasePort > 65535 {
		return nil, fmt.Errorf("traceroute: UDPBasePort must be between 1 and 65535")
	}

	udpNetwork := "udp4"
	udpAddress := net.IPv4zero
	icmpNetwork := "ip4:icmp"
	icmpAddress := "0.0.0.0"
	ipv6Trace := dst.Is6()
	if ipv6Trace {
		udpNetwork = "udp6"
		udpAddress = net.IPv6zero
		icmpNetwork = "ip6:ipv6-icmp"
		icmpAddress = "::"
	}

	sendConn, err := net.ListenUDP(udpNetwork, &net.UDPAddr{IP: udpAddress})
	if err != nil {
		return nil, socketError(err)
	}

	recvConn, err := listenICMPPacket(icmpNetwork, icmpAddress)
	if err != nil {
		_ = sendConn.Close()
		return nil, socketError(err)
	}

	packetSize := opts.PacketSize
	if packetSize < udpHeaderLen {
		packetSize = udpHeaderLen
	}

	return &udpProber{
		sendConn:      sendConn,
		recvConn:      recvConn,
		dst:           netIPFromAddr(dst),
		ipv6:          ipv6Trace,
		packetSize:    packetSize,
		firstHop:      opts.FirstHop,
		queriesPerHop: opts.QueriesPerHop,
		udpBasePort:   opts.UDPBasePort,
		sourcePort:    sendConn.LocalAddr().(*net.UDPAddr).Port,
	}, nil
}

func normalizeUDPOptions(opts Options) Options {
	if opts.FirstHop <= 0 {
		opts.FirstHop = 1
	}
	if opts.QueriesPerHop <= 0 {
		opts.QueriesPerHop = 1
	}
	if opts.UDPBasePort == 0 {
		opts.UDPBasePort = internalUDPDefaultPort
	}
	return opts
}

func (p *udpProber) Send(ctx context.Context, ttl int, attempt int) (Sent, error) {
	if err := ctx.Err(); err != nil {
		return Sent{}, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	port := p.destinationPort(ttl, attempt)
	if port < 1 || port > 65535 {
		return Sent{}, fmt.Errorf("traceroute: UDP destination port %d is out of range", port)
	}

	if err := p.setHopLimit(ttl); err != nil {
		return Sent{}, socketError(err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = p.sendConn.SetWriteDeadline(deadline)
	} else {
		_ = p.sendConn.SetWriteDeadline(time.Time{})
	}

	sentAt := time.Now()
	dst := &net.UDPAddr{IP: p.dst, Port: port}
	if _, err := p.sendConn.WriteToUDP(paddingData(p.packetSize-udpHeaderLen), dst); err != nil {
		return Sent{}, socketError(err)
	}

	return Sent{
		HeaderToken: makeUDPHeaderToken(p.sourcePort, port),
		TTL:         ttl,
		Attempt:     attempt,
		SentAt:      sentAt,
	}, nil
}

func (p *udpProber) Receive(ctx context.Context, sent Sent, timeout time.Duration) (Reply, error) {
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
		if err := p.recvConn.SetReadDeadline(readDeadline); err != nil {
			return Reply{}, err
		}

		n, peer, err := p.recvConn.ReadFrom(buf)
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

		parsed, ok := parseUDPICMPReply(p.ipv6, buf[:n])
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

func (p *udpProber) Close() error {
	return errors.Join(p.sendConn.Close(), p.recvConn.Close())
}

func (p *udpProber) setHopLimit(ttl int) error {
	if p.ipv6 {
		return ipv6.NewPacketConn(p.sendConn).SetHopLimit(ttl)
	}
	return ipv4.NewPacketConn(p.sendConn).SetTTL(ttl)
}

func (p *udpProber) destinationPort(ttl int, attempt int) int {
	return p.udpBasePort + ((ttl-p.firstHop)*p.queriesPerHop + (attempt - 1))
}

func parseUDPICMPReply(ipv6Trace bool, packet []byte) (udpReply, bool) {
	protocol := protocolICMPv4
	if ipv6Trace {
		protocol = protocolICMPv6
		packet = stripIPv6Packet(packet)
	} else {
		packet = stripIPv4Packet(packet)
	}

	msg, err := icmp.ParseMessage(protocol, packet)
	if err != nil {
		return udpReply{}, false
	}
	return classifyUDPICMPMessage(ipv6Trace, msg)
}

func classifyUDPICMPMessage(ipv6Trace bool, msg *icmp.Message) (udpReply, bool) {
	reply := udpReply{
		icmpType: icmpTypeNumber(msg.Type),
		icmpCode: msg.Code,
	}

	if ipv6Trace {
		return classifyUDPICMPv6Message(reply, msg)
	}
	return classifyUDPICMPv4Message(reply, msg)
}

func classifyUDPICMPv4Message(reply udpReply, msg *icmp.Message) (udpReply, bool) {
	switch msg.Type {
	case ipv4.ICMPTypeTimeExceeded:
		body, ok := msg.Body.(*icmp.TimeExceeded)
		if !ok {
			return udpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedUDPPacket(false, body.Data)
		reply.headerToken = headerToken
		reply.kind = ReplyTimeExceeded
		reply.annotation = ipv4TimeExceededAnnotation(msg.Code)
		return reply, ok
	case ipv4.ICMPTypeDestinationUnreachable:
		body, ok := msg.Body.(*icmp.DstUnreach)
		if !ok {
			return udpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedUDPPacket(false, body.Data)
		reply.headerToken = headerToken
		reply.kind = ipv4UDPDestinationUnreachableKind(msg.Code)
		reply.annotation = ipv4DestinationUnreachableAnnotation(msg.Code)
		return reply, ok
	default:
		return udpReply{}, false
	}
}

func classifyUDPICMPv6Message(reply udpReply, msg *icmp.Message) (udpReply, bool) {
	switch msg.Type {
	case ipv6.ICMPTypeTimeExceeded:
		body, ok := msg.Body.(*icmp.TimeExceeded)
		if !ok {
			return udpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedUDPPacket(true, body.Data)
		reply.headerToken = headerToken
		reply.kind = ReplyTimeExceeded
		reply.annotation = ipv6TimeExceededAnnotation(msg.Code)
		return reply, ok
	case ipv6.ICMPTypeDestinationUnreachable:
		body, ok := msg.Body.(*icmp.DstUnreach)
		if !ok {
			return udpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedUDPPacket(true, body.Data)
		reply.headerToken = headerToken
		reply.kind = ipv6UDPDestinationUnreachableKind(msg.Code)
		reply.annotation = ipv6DestinationUnreachableAnnotation(msg.Code)
		return reply, ok
	case ipv6.ICMPTypePacketTooBig:
		body, ok := msg.Body.(*icmp.PacketTooBig)
		if !ok {
			return udpReply{}, false
		}
		headerToken, ok := headerTokenFromEmbeddedUDPPacket(true, body.Data)
		reply.headerToken = headerToken
		reply.kind = ReplyPacketTooBig
		reply.annotation = "packet too big"
		return reply, ok
	default:
		return udpReply{}, false
	}
}

func headerTokenFromEmbeddedUDPPacket(ipv6Trace bool, packet []byte) (uint32, bool) {
	var ok bool
	if packet, ok = udpPayloadFromEmbeddedPacket(ipv6Trace, packet); !ok {
		return 0, false
	}

	if len(packet) < udpHeaderLen {
		return 0, false
	}

	sourcePort := int(binary.BigEndian.Uint16(packet[0:2]))
	destinationPort := int(binary.BigEndian.Uint16(packet[2:4]))
	return makeUDPHeaderToken(sourcePort, destinationPort), true
}

func udpPayloadFromEmbeddedPacket(ipv6Trace bool, packet []byte) ([]byte, bool) {
	if len(packet) == 0 {
		return packet, true
	}
	if ipv6Trace {
		if packet[0]>>4 != 6 {
			return packet, true
		}
		header, err := ipv6.ParseHeader(packet)
		if err != nil || header.NextHeader != protocolUDP || len(packet) < ipv6.HeaderLen {
			return nil, false
		}
		return packet[ipv6.HeaderLen:], true
	}

	if packet[0]>>4 != 4 {
		return packet, true
	}
	header, err := ipv4.ParseHeader(packet)
	if err != nil || header.Protocol != protocolUDP || len(packet) < header.Len {
		return nil, false
	}
	return packet[header.Len:], true
}

func makeUDPHeaderToken(sourcePort int, destinationPort int) uint32 {
	return uint32(uint16(sourcePort))<<16 | uint32(uint16(destinationPort))
}

func ipv4UDPDestinationUnreachableKind(code int) ReplyKind {
	if code == 3 {
		return ReplyDestination
	}
	return ipv4DestinationUnreachableKind(code)
}

func ipv6UDPDestinationUnreachableKind(code int) ReplyKind {
	if code == 4 {
		return ReplyDestination
	}
	return ipv6DestinationUnreachableKind(code)
}
