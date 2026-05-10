package probe

import (
	"encoding/binary"
	"net"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestClassifyUDPIPv4TimeExceeded(t *testing.T) {
	embedded := ipv4UDPPacket(t, 40000, 33434)
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeTimeExceeded,
		Code: 0,
		Body: &icmp.TimeExceeded{Data: embedded},
	}

	reply, ok := classifyUDPICMPMessage(false, msg)
	if !ok {
		t.Fatal("classifyUDPICMPMessage returned false")
	}
	if reply.headerToken != makeUDPHeaderToken(40000, 33434) {
		t.Fatalf("headerToken = %d, want %d", reply.headerToken, makeUDPHeaderToken(40000, 33434))
	}
	if reply.kind != ReplyTimeExceeded {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyTimeExceeded)
	}
}

func TestClassifyUDPIPv4PortUnreachableIsDestination(t *testing.T) {
	embedded := ipv4UDPPacket(t, 40000, 33435)
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeDestinationUnreachable,
		Code: 3,
		Body: &icmp.DstUnreach{Data: embedded},
	}

	reply, ok := classifyUDPICMPMessage(false, msg)
	if !ok {
		t.Fatal("classifyUDPICMPMessage returned false")
	}
	if reply.kind != ReplyDestination {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyDestination)
	}
	if reply.annotation != "port unreachable" {
		t.Fatalf("annotation = %q, want port unreachable", reply.annotation)
	}
}

func TestClassifyUDPIPv4Filtered(t *testing.T) {
	embedded := ipv4UDPPacket(t, 40000, 33436)
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeDestinationUnreachable,
		Code: 13,
		Body: &icmp.DstUnreach{Data: embedded},
	}

	reply, ok := classifyUDPICMPMessage(false, msg)
	if !ok {
		t.Fatal("classifyUDPICMPMessage returned false")
	}
	if reply.kind != ReplyFiltered {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyFiltered)
	}
}

func TestClassifyUDPIPv6TimeExceeded(t *testing.T) {
	embedded := ipv6UDPPacket(40000, 33434)
	msg := &icmp.Message{
		Type: ipv6.ICMPTypeTimeExceeded,
		Code: 0,
		Body: &icmp.TimeExceeded{Data: embedded},
	}

	reply, ok := classifyUDPICMPMessage(true, msg)
	if !ok {
		t.Fatal("classifyUDPICMPMessage returned false")
	}
	if reply.headerToken != makeUDPHeaderToken(40000, 33434) {
		t.Fatalf("headerToken = %d, want %d", reply.headerToken, makeUDPHeaderToken(40000, 33434))
	}
	if reply.kind != ReplyTimeExceeded {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyTimeExceeded)
	}
}

func TestClassifyUDPIPv6PortUnreachableIsDestination(t *testing.T) {
	embedded := ipv6UDPPacket(40000, 33435)
	msg := &icmp.Message{
		Type: ipv6.ICMPTypeDestinationUnreachable,
		Code: 4,
		Body: &icmp.DstUnreach{Data: embedded},
	}

	reply, ok := classifyUDPICMPMessage(true, msg)
	if !ok {
		t.Fatal("classifyUDPICMPMessage returned false")
	}
	if reply.kind != ReplyDestination {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyDestination)
	}
}

func TestClassifyUDPIPv6PacketTooBig(t *testing.T) {
	embedded := ipv6UDPPacket(40000, 33436)
	msg := &icmp.Message{
		Type: ipv6.ICMPTypePacketTooBig,
		Code: 0,
		Body: &icmp.PacketTooBig{Data: embedded},
	}

	reply, ok := classifyUDPICMPMessage(true, msg)
	if !ok {
		t.Fatal("classifyUDPICMPMessage returned false")
	}
	if reply.kind != ReplyPacketTooBig {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyPacketTooBig)
	}
}

func TestHeaderTokenFromEmbeddedUDPRejectsMalformedPackets(t *testing.T) {
	if _, ok := headerTokenFromEmbeddedUDPPacket(false, []byte{1, 2, 3}); ok {
		t.Fatal("short packet should not match")
	}
	if _, ok := headerTokenFromEmbeddedUDPPacket(false, ipv4EchoPacket(t, 1234, 1)); ok {
		t.Fatal("embedded ICMP packet should not match UDP")
	}
}

func TestUDPProberDestinationPortUsesTTLAndAttemptOffset(t *testing.T) {
	prober := udpProber{
		firstHop:      2,
		queriesPerHop: 3,
		udpBasePort:   33434,
	}

	tests := []struct {
		ttl     int
		attempt int
		want    int
	}{
		{ttl: 2, attempt: 1, want: 33434},
		{ttl: 2, attempt: 3, want: 33436},
		{ttl: 3, attempt: 1, want: 33437},
		{ttl: 4, attempt: 2, want: 33441},
	}

	for _, tt := range tests {
		if got := prober.destinationPort(tt.ttl, tt.attempt); got != tt.want {
			t.Fatalf("destinationPort(%d, %d) = %d, want %d", tt.ttl, tt.attempt, got, tt.want)
		}
	}
}

func ipv4UDPPacket(t *testing.T, sourcePort int, destinationPort int) []byte {
	t.Helper()

	udp := udpHeader(sourcePort, destinationPort)
	header, err := (&ipv4.Header{
		Version:  4,
		Len:      ipv4.HeaderLen,
		TotalLen: ipv4.HeaderLen + len(udp),
		TTL:      1,
		Protocol: protocolUDP,
		Src:      net.IPv4(192, 0, 2, 1),
		Dst:      net.IPv4(198, 51, 100, 1),
	}).Marshal()
	if err != nil {
		t.Fatal(err)
	}

	return append(header, udp...)
}

func ipv6UDPPacket(sourcePort int, destinationPort int) []byte {
	udp := udpHeader(sourcePort, destinationPort)
	header := []byte{
		0x60, 0, 0, 0,
		byte(len(udp) >> 8), byte(len(udp)),
		protocolUDP,
		1,
	}
	header = append(header, net.ParseIP("2001:db8::1").To16()...)
	header = append(header, net.ParseIP("2001:db8::2").To16()...)

	return append(header, udp...)
}

func udpHeader(sourcePort int, destinationPort int) []byte {
	header := make([]byte, udpHeaderLen)
	binary.BigEndian.PutUint16(header[0:2], uint16(sourcePort))
	binary.BigEndian.PutUint16(header[2:4], uint16(destinationPort))
	binary.BigEndian.PutUint16(header[4:6], uint16(udpHeaderLen))
	return header
}
