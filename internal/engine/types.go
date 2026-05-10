package engine

import (
	"context"
	"errors"
	"net/netip"
	"time"
)

var (
	ErrNoAddress = errors.New("traceroute: no usable destination address")
)

type IPVersion int

const (
	IPAny IPVersion = 0
	IPv4  IPVersion = 4
	IPv6  IPVersion = 6
)

type Protocol int

const (
	ProtocolICMP Protocol = iota
	ProtocolUDP
)

type Options struct {
	Protocol  Protocol
	IPVersion IPVersion

	FirstHop      int
	MaxHops       int
	QueriesPerHop int

	Timeout time.Duration

	PacketSize  int
	UDPBasePort int

	ResolveNames bool
}

type Resolver interface {
	LookupIP(ctx context.Context, host string, version IPVersion) ([]netip.Addr, error)
	LookupAddr(ctx context.Context, addr netip.Addr) ([]string, error)
}

type Trace struct {
	Target      string
	Destination netip.Addr
	IPVersion   IPVersion
	StartedAt   time.Time
	FinishedAt  time.Time
	Hops        []Hop
}

type Hop struct {
	TTL    int
	Probes []Probe
}

type Probe struct {
	Attempt int

	Addr     netip.Addr
	Hostname string

	SentAt     time.Time
	ReceivedAt time.Time
	RTT        time.Duration

	Status     Status
	Annotation string

	ICMPType int
	ICMPCode int

	Error string
}

type Status string

const (
	StatusOK           Status = "ok"
	StatusTimeout      Status = "timeout"
	StatusDestination  Status = "destination"
	StatusUnreachable  Status = "unreachable"
	StatusFiltered     Status = "filtered"
	StatusPacketTooBig Status = "packet-too-big"
	StatusError        Status = "error"
)

type EventKind string

const (
	EventProbe EventKind = "probe"
	EventHop   EventKind = "hop"
	EventDone  EventKind = "done"
)

type Event struct {
	Kind  EventKind
	Probe *Probe
	Hop   *Hop
	Trace *Trace
}

type Sink interface {
	Emit(Event)
}
