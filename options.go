package traceroute

import (
	"fmt"
	"log/slog"
	"net/netip"
	"time"
)

// IPVersion selects which destination address family a trace should use.
type IPVersion int

const (
	IPAny IPVersion = 0
	IPv4  IPVersion = 4
	IPv6  IPVersion = 6
)

// Options controls trace behavior.
type Options struct {
	Method    Method
	IPVersion IPVersion

	FirstHop      int
	MaxHops       int
	QueriesPerHop int

	Timeout time.Duration
	Wait    time.Duration

	PacketSize int

	SourceAddress netip.Addr
	Interface     string

	BasePort        uint16
	DestinationPort uint16
	SourcePort      uint16

	TOS          int
	TrafficClass int
	DontFragment bool

	ResolveNames bool
	LookupASN    bool
	IncludeMPLS  bool

	Parallelism int

	Resolver Resolver
	Hooks    Hooks
	Logger   *slog.Logger
}

// DefaultOptions returns the package defaults.
func DefaultOptions() Options {
	return Options{
		Method:        MethodUDP,
		IPVersion:     IPAny,
		FirstHop:      1,
		MaxHops:       30,
		QueriesPerHop: 3,
		Timeout:       3 * time.Second,
		Wait:          0,
		PacketSize:    60,
		BasePort:      33434,
		ResolveNames:  false,
		Parallelism:   1,
	}
}

// Normalize fills zero-valued options with defaults.
func (o Options) Normalize() Options {
	d := DefaultOptions()

	if o.Method == "" {
		o.Method = d.Method
	}
	if o.FirstHop == 0 {
		o.FirstHop = d.FirstHop
	}
	if o.MaxHops == 0 {
		o.MaxHops = d.MaxHops
	}
	if o.QueriesPerHop == 0 {
		o.QueriesPerHop = d.QueriesPerHop
	}
	if o.Timeout == 0 {
		o.Timeout = d.Timeout
	}
	if o.PacketSize == 0 {
		o.PacketSize = d.PacketSize
	}
	if o.BasePort == 0 {
		o.BasePort = d.BasePort
	}
	if o.Parallelism == 0 {
		o.Parallelism = d.Parallelism
	}

	return o
}

// Validate checks whether options are internally consistent.
func (o Options) Validate() error {
	if o.FirstHop < 1 {
		return fmt.Errorf("traceroute: FirstHop must be >= 1")
	}
	if o.MaxHops < o.FirstHop {
		return fmt.Errorf("traceroute: MaxHops must be >= FirstHop")
	}
	if o.QueriesPerHop < 1 {
		return fmt.Errorf("traceroute: QueriesPerHop must be >= 1")
	}
	if o.Timeout <= 0 {
		return fmt.Errorf("traceroute: Timeout must be positive")
	}
	if o.Wait < 0 {
		return fmt.Errorf("traceroute: Wait must be >= 0")
	}
	if o.PacketSize < 1 {
		return fmt.Errorf("traceroute: PacketSize must be >= 1")
	}
	if o.TOS < 0 || o.TOS > 255 {
		return fmt.Errorf("traceroute: TOS must be between 0 and 255")
	}
	if o.TrafficClass < 0 || o.TrafficClass > 255 {
		return fmt.Errorf("traceroute: TrafficClass must be between 0 and 255")
	}
	if o.Parallelism < 1 {
		return fmt.Errorf("traceroute: Parallelism must be >= 1")
	}

	switch o.IPVersion {
	case IPAny, IPv4, IPv6:
	default:
		return fmt.Errorf("traceroute: unsupported IPVersion %d", o.IPVersion)
	}

	switch o.Method {
	case MethodICMP, MethodUDP, MethodTCP, MethodUDPParis, MethodICMPParis:
		return nil
	default:
		return fmt.Errorf("traceroute: unsupported method %q", o.Method)
	}
}
