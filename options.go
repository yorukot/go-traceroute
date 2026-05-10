package traceroute

import (
	"fmt"
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

// Protocol selects which probe packets are sent during a trace.
type Protocol int

const (
	ProtocolICMP Protocol = iota
	ProtocolUDP
)

const defaultUDPBasePort = 33434

// Options controls trace behavior.
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

// DefaultOptions returns the package defaults.
func DefaultOptions() Options {
	return Options{
		Protocol:      ProtocolICMP,
		IPVersion:     IPAny,
		FirstHop:      1,
		MaxHops:       64,
		QueriesPerHop: 3,
		Timeout:       3 * time.Second,
		PacketSize:    48,
		UDPBasePort:   defaultUDPBasePort,
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
	if o.UDPBasePort == 0 {
		o.UDPBasePort = d.UDPBasePort
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
	if o.UDPBasePort < 1 || o.UDPBasePort > 65535 {
		return fmt.Errorf("traceroute: UDPBasePort must be between 1 and 65535")
	}

	switch o.IPVersion {
	case IPAny, IPv4, IPv6:
	default:
		return fmt.Errorf("traceroute: unsupported IPVersion %d", o.IPVersion)
	}

	switch o.Protocol {
	case ProtocolICMP:
		return nil
	case ProtocolUDP:
		if o.udpPortRangeOverflows() {
			return fmt.Errorf("traceroute: UDPBasePort range exceeds 65535")
		}
		return nil
	default:
		return fmt.Errorf("traceroute: unsupported Protocol %d", o.Protocol)
	}
}

func (o Options) udpPortRangeOverflows() bool {
	ttlCount := int64(o.MaxHops - o.FirstHop + 1)
	queryCount := int64(o.QueriesPerHop)
	available := int64(65535 - o.UDPBasePort + 1)

	return ttlCount > available/queryCount
}

func addressMatchesVersion(addr netip.Addr, version IPVersion) bool {
	switch version {
	case IPAny:
		return true
	case IPv4:
		return addr.Is4()
	case IPv6:
		return addr.Is6()
	default:
		return false
	}
}
