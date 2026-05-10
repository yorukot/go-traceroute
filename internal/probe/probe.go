package probe

import (
	"context"
	"errors"
	"net/netip"
	"time"
)

var (
	ErrPermission = errors.New("traceroute: permission denied")
	ErrTimeout    = errors.New("traceroute: timeout")
)

// Protocol selects which probe packets are sent during a trace.
type Protocol int

const (
	ProtocolICMP Protocol = iota
	ProtocolUDP
)

// Options is the internal prober configuration copied from the public API.
type Options struct {
	Protocol      Protocol
	FirstHop      int
	QueriesPerHop int
	PacketSize    int
	UDPBasePort   int
}

// Sent describes a probe after it has been sent.
type Sent struct {
	HeaderToken uint32
	TTL         int
	Attempt     int
	SentAt      time.Time
}

// ReplyKind classifies an internal backend reply.
type ReplyKind string

const (
	ReplyTimeExceeded ReplyKind = "time-exceeded"
	ReplyDestination  ReplyKind = "destination"
	ReplyUnreachable  ReplyKind = "unreachable"
	ReplyPacketTooBig ReplyKind = "packet-too-big"
	ReplyFiltered     ReplyKind = "filtered"
)

// Reply describes a response matched to a sent probe.
type Reply struct {
	From       netip.Addr
	ReceivedAt time.Time
	RTT        time.Duration

	Kind       ReplyKind
	ICMPType   int
	ICMPCode   int
	Annotation string
}

// Prober sends probes and waits for matching replies.
type Prober interface {
	Send(ctx context.Context, ttl int, attempt int) (Sent, error)
	Receive(ctx context.Context, sent Sent, timeout time.Duration) (Reply, error)
	Close() error
}

// Factory creates a prober for a resolved destination.
type Factory interface {
	New(ctx context.Context, dst netip.Addr, opts Options) (Prober, error)
}
