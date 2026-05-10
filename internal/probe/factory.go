package probe

import (
	"context"
	"net/netip"
)

type defaultFactory struct{}

// NewFactory returns the default backend factory.
func NewFactory() Factory {
	return defaultFactory{}
}

func (defaultFactory) New(ctx context.Context, dst netip.Addr, opts Options) (Prober, error) {
	return newICMPProber(ctx, dst, opts)
}
