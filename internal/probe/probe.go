package probe

import (
	"context"
	"errors"
	"net/netip"
	"time"
)

var (
	ErrPermission  = errors.New("traceroute: permission denied")
	ErrUnsupported = errors.New("traceroute: unsupported platform or method")
	ErrTimeout     = errors.New("traceroute: timeout")
)

// Options is the internal prober configuration copied from the public API.
type Options struct {
	Method    string
	IPVersion int

	Timeout time.Duration

	PacketSize int

	SourceAddress netip.Addr
	Interface     string

	BasePort        uint16
	DestinationPort uint16
	SourcePort      uint16

	TOS          int
	TrafficClass int
	DontFragment bool
}

// ID correlates a sent probe with a reply.
type ID struct {
	Token   uint64
	TTL     int
	Attempt int
}

// Sent describes a probe after it has been sent.
type Sent struct {
	ID      ID
	TTL     int
	Attempt int
	SentAt  time.Time
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
	ID         ID
	From       netip.Addr
	ReceivedAt time.Time
	RTT        time.Duration

	Kind       ReplyKind
	ICMPType   int
	ICMPCode   int
	Annotation string
	MTU        int
}

// Prober hides method-specific send/receive details from the engine.
type Prober interface {
	Send(ctx context.Context, ttl int, attempt int) (Sent, error)
	Receive(ctx context.Context, sent Sent, timeout time.Duration) (Reply, error)
	Close() error
}

// Factory creates a method-specific prober for a resolved destination.
type Factory interface {
	New(ctx context.Context, dst netip.Addr, opts Options) (Prober, error)
}
