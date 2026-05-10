package probe

import (
	"net"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestClassifyIPv4EchoReply(t *testing.T) {
	token := makeICMPHeaderToken(1234, 7)
	payloadToken := uint64(0x0102030405060708)
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEchoReply,
		Code: 0,
		Body: &icmp.Echo{ID: 1234, Seq: 7, Data: echoData(payloadToken, icmpTokenLen)},
	}

	reply, ok := classifyICMPMessage(false, msg)
	if !ok {
		t.Fatal("classifyICMPMessage returned false")
	}
	if reply.headerToken != token {
		t.Fatalf("headerToken = %d, want %d", reply.headerToken, token)
	}
	if !reply.hasPayloadToken {
		t.Fatal("missing payload token")
	}
	if reply.payloadToken != payloadToken {
		t.Fatalf("payloadToken = %d, want %d", reply.payloadToken, payloadToken)
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
	if reply.headerToken != makeICMPHeaderToken(1234, 8) {
		t.Fatalf("headerToken = %d, want %d", reply.headerToken, makeICMPHeaderToken(1234, 8))
	}
	if reply.kind != ReplyTimeExceeded {
		t.Fatalf("kind = %q, want %q", reply.kind, ReplyTimeExceeded)
	}
}

func TestICMPReplyPrefersPayloadTokenForMatching(t *testing.T) {
	reply := icmpReply{
		payloadToken:    100,
		hasPayloadToken: true,
		headerToken:     200,
	}

	if !reply.matches(ID{Token: 100, HeaderToken: 999}) {
		t.Fatal("payload token should match even when header token differs")
	}
	if reply.matches(ID{Token: 999, HeaderToken: 200}) {
		t.Fatal("payload token should reject even when header token matches")
	}
}

func TestICMPReplyFallsBackToHeaderTokenForMatching(t *testing.T) {
	reply := icmpReply{
		headerToken: 200,
	}

	if !reply.matches(ID{Token: 999, HeaderToken: 200}) {
		t.Fatal("header token should match when payload token is unavailable")
	}
	if reply.matches(ID{Token: 999, HeaderToken: 201}) {
		t.Fatal("header token should reject a different probe")
	}
}

func TestTokenFromEchoUsesPayloadWhenAvailable(t *testing.T) {
	payloadToken := uint64(0x1122334455667788)
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEchoReply,
		Code: 0,
		Body: &icmp.Echo{ID: 1234, Seq: 7, Data: echoData(payloadToken, icmpTokenLen)},
	}

	token, ok := tokenFromEcho(msg)
	if !ok {
		t.Fatal("tokenFromEcho returned false")
	}
	if token.header != makeICMPHeaderToken(1234, 7) {
		t.Fatalf("header = %d, want %d", token.header, makeICMPHeaderToken(1234, 7))
	}
	if !token.hasPayload {
		t.Fatal("missing payload token")
	}
	if token.payload != payloadToken {
		t.Fatalf("payload = %d, want %d", token.payload, payloadToken)
	}
}

func TestTokenFromEchoFallsBackToHeaderWithoutPayload(t *testing.T) {
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEchoReply,
		Code: 0,
		Body: &icmp.Echo{ID: 1234, Seq: 7},
	}

	token, ok := tokenFromEcho(msg)
	if !ok {
		t.Fatal("tokenFromEcho returned false")
	}
	if token.header != makeICMPHeaderToken(1234, 7) {
		t.Fatalf("header = %d, want %d", token.header, makeICMPHeaderToken(1234, 7))
	}
	if token.hasPayload {
		t.Fatal("unexpected payload token")
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
