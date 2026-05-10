package traceroute

import (
	"fmt"
	"time"
)

// IPVersion selects which destination address family a trace should use.
type IPVersion int

const (
	IPAny IPVersion = 0
	IPv4  IPVersion = 4
	IPv6  IPVersion = 6
)

// Options controls ICMP trace behavior.
type Options struct {
	IPVersion IPVersion

	FirstHop      int
	MaxHops       int
	QueriesPerHop int

	Timeout time.Duration

	PacketSize int

	ResolveNames bool
}

// DefaultOptions returns the package defaults.
func DefaultOptions() Options {
	return Options{
		IPVersion:     IPAny,
		FirstHop:      1,
		MaxHops:       64,
		QueriesPerHop: 3,
		Timeout:       3 * time.Second,
		PacketSize:    48,
		ResolveNames:  false,
	}
}

// Normalize fills zero-valued options with defaults.
func (o Options) Normalize() Options {
	d := DefaultOptions()

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
	if o.PacketSize < 1 {
		return fmt.Errorf("traceroute: PacketSize must be >= 1")
	}

	switch o.IPVersion {
	case IPAny, IPv4, IPv6:
		return nil
	default:
		return fmt.Errorf("traceroute: unsupported IPVersion %d", o.IPVersion)
	}
}
