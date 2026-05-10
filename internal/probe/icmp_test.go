package probe

import (
	"net"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestClassifyIPv4EchoReply(t *testing.T) {
	token := makeICMPToken(1234, 7)
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEchoReply,
		Code: 0,
		Body: &icmp.Echo{ID: 1234, Seq: 7},
	}

	reply, ok := classifyICMPMessage(false, msg)
	if !ok {
		t.Fatal("classifyICMPMessage returned false")
	}
	if reply.token != token {
		t.Fatalf("token = %d, want %d", reply.token, token)
	}
	if reply.kind != ReplyDestination {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyDestination)
	}
}

func TestClassifyIPv4TimeExceeded(t *testing.T) {
	embedded := ipv4EchoPacket(t, 1234, 8)
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeTimeExceeded,
		Code: 0,
		Body: &icmp.TimeExceeded{Data: embedded},
	}

	reply, ok := classifyICMPMessage(false, msg)
	if !ok {
		t.Fatal("classifyICMPMessage returned false")
	}
	if reply.token != makeICMPToken(1234, 8) {
		t.Fatalf("token = %d, want %d", reply.token, makeICMPToken(1234, 8))
	}
	if reply.kind != ReplyTimeExceeded {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyTimeExceeded)
	}
}

func TestClassifyIPv4DestinationFiltered(t *testing.T) {
	embedded := ipv4EchoPacket(t, 1234, 9)
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeDestinationUnreachable,
		Code: 13,
		Body: &icmp.DstUnreach{Data: embedded},
	}

	reply, ok := classifyICMPMessage(false, msg)
	if !ok {
		t.Fatal("classifyICMPMessage returned false")
	}
	if reply.kind != ReplyFiltered {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyFiltered)
	}
}

func TestClassifyIPv6PacketTooBig(t *testing.T) {
	embedded := ipv6EchoPacket(t, 1234, 10)
	msg := &icmp.Message{
		Type: ipv6.ICMPTypePacketTooBig,
		Code: 0,
		Body: &icmp.PacketTooBig{MTU: 1280, Data: embedded},
	}

	reply, ok := classifyICMPMessage(true, msg)
	if !ok {
		t.Fatal("classifyICMPMessage returned false")
	}
	if reply.kind != ReplyPacketTooBig {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyPacketTooBig)
	}
	if reply.mtu != 1280 {
		t.Fatalf("mtu = %d, want 1280", reply.mtu)
	}
}

func ipv4EchoPacket(t *testing.T, id int, seq int) []byte {
	t.Helper()

	echo, err := (&icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: id, Seq: seq},
	}).Marshal(nil)
	if err != nil {
		t.Fatal(err)
	}

	header, err := (&ipv4.Header{
		Version:  4,
		Len:      ipv4.HeaderLen,
		TotalLen: ipv4.HeaderLen + len(echo),
		TTL:      1,
		Protocol: protocolICMPv4,
		Src:      net.IPv4(192, 0, 2, 1),
		Dst:      net.IPv4(198, 51, 100, 1),
	}).Marshal()
	if err != nil {
		t.Fatal(err)
	}

	return append(header, echo...)
}

func ipv6EchoPacket(t *testing.T, id int, seq int) []byte {
	t.Helper()

	echo, err := (&icmp.Message{
		Type: ipv6.ICMPTypeEchoRequest,
		Code: 0,
		Body: &icmp.Echo{ID: id, Seq: seq},
	}).Marshal(nil)
	if err != nil {
		t.Fatal(err)
	}

	header := []byte{
		0x60, 0, 0, 0,
		byte(len(echo) >> 8), byte(len(echo)),
		protocolICMPv6,
		1,
	}
	header = append(header, net.ParseIP("2001:db8::1").To16()...)
	header = append(header, net.ParseIP("2001:db8::2").To16()...)

	return append(header, echo...)
}
