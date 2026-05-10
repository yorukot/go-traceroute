package probe

import (
	"context"
	"fmt"
	"net/netip"
)

type defaultFactory struct{}

// NewFactory returns the default backend factory.
func NewFactory() Factory {
	return defaultFactory{}
}

func (defaultFactory) New(ctx context.Context, dst netip.Addr, opts Options) (Prober, error) {
	switch opts.Protocol {
	case ProtocolICMP:
		return newICMPProber(ctx, dst, opts)
	case ProtocolUDP:
		return newUDPProber(ctx, dst, opts)
	default:
		return nil, fmt.Errorf("traceroute: unsupported probe protocol %d", opts.Protocol)
	}
}
