package probe

import (
	"context"
	"fmt"
	"net/netip"
)

type unsupportedFactory struct{}

// NewFactory returns the default backend factory.
//
// Real socket backends are intentionally not implemented in this scaffold yet;
// callers receive ErrUnsupported instead of leaking incomplete socket behavior.
func NewFactory() Factory {
	return unsupportedFactory{}
}

func (unsupportedFactory) New(ctx context.Context, dst netip.Addr, opts Options) (Prober, error) {
	_ = ctx
	_ = dst
	return nil, fmt.Errorf("%w: method %s", ErrUnsupported, opts.Method)
}
