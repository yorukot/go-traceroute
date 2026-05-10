package engine

import (
	"context"
	"errors"
	"fmt"
	"net/netip"

	"github.com/yorukot/go-traceroute/internal/clock"
	"github.com/yorukot/go-traceroute/internal/probe"
)

// Engine orchestrates a trace without knowing how probes are implemented.
type Engine struct {
	opts    Options
	res     Resolver
	factory probe.Factory
	clock   clock.Clock
	sink    Sink
}

func New(opts Options, res Resolver, factory probe.Factory, clk clock.Clock, sink Sink) *Engine {
	if clk == nil {
		clk = clock.RealClock{}
	}
	return &Engine{
		opts:    opts,
		res:     res,
		factory: factory,
		clock:   clk,
		sink:    sink,
	}
}

func (e *Engine) Trace(ctx context.Context, target string) (*Trace, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if target == "" {
		return nil, fmt.Errorf("traceroute: target is empty")
	}
	if e.res == nil {
		return nil, fmt.Errorf("traceroute: resolver is required")
	}
	if e.factory == nil {
		return nil, fmt.Errorf("traceroute: prober factory is required")
	}

	dest, err := e.resolve(ctx, target)
	if err != nil {
		return nil, err
	}

	prober, err := e.factory.New(ctx, dest, e.probeOptions())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = prober.Close()
	}()

	trace := &Trace{
		Target:      target,
		Destination: dest,
		Method:      e.opts.Method,
		IPVersion:   versionOf(dest),
		StartedAt:   e.clock.Now(),
	}

	for ttl := e.opts.FirstHop; ttl <= e.opts.MaxHops; ttl++ {
		if err := ctx.Err(); err != nil {
			return trace, err
		}

		hop, err := e.traceHop(ctx, prober, ttl)
		if err != nil {
			return trace, err
		}

		trace.Hops = append(trace.Hops, hop)
		e.emit(Event{Kind: EventHop, Hop: &hop})

		if reachedDestination(hop) {
			break
		}

		if e.opts.Wait > 0 {
			if err := e.clock.Sleep(ctx, e.opts.Wait); err != nil {
				return trace, err
			}
		}
	}

	trace.FinishedAt = e.clock.Now()
	e.emit(Event{Kind: EventDone, Trace: trace})
	return trace, nil
}

func (e *Engine) resolve(ctx context.Context, target string) (netip.Addr, error) {
	if addr, err := netip.ParseAddr(target); err == nil {
		if addressMatchesVersion(addr, e.opts.IPVersion) {
			return addr, nil
		}
		return netip.Addr{}, ErrNoAddress
	}

	addrs, err := e.res.LookupIP(ctx, target, e.opts.IPVersion)
	if err != nil {
		if errors.Is(err, ErrNoAddress) {
			return netip.Addr{}, ErrNoAddress
		}
		return netip.Addr{}, err
	}

	for _, addr := range addrs {
		if addressMatchesVersion(addr, e.opts.IPVersion) {
			return addr, nil
		}
	}

	return netip.Addr{}, ErrNoAddress
}

func (e *Engine) probeOptions() probe.Options {
	return probe.Options{
		Method:          e.opts.Method,
		IPVersion:       int(e.opts.IPVersion),
		Timeout:         e.opts.Timeout,
		PacketSize:      e.opts.PacketSize,
		SourceAddress:   e.opts.SourceAddress,
		Interface:       e.opts.Interface,
		BasePort:        e.opts.BasePort,
		DestinationPort: e.opts.DestinationPort,
		SourcePort:      e.opts.SourcePort,
		TOS:             e.opts.TOS,
		TrafficClass:    e.opts.TrafficClass,
		DontFragment:    e.opts.DontFragment,
	}
}

func (e *Engine) emit(event Event) {
	if e.sink != nil {
		e.sink.Emit(event)
	}
}

func versionOf(addr netip.Addr) IPVersion {
	if addr.Is4() {
		return IPv4
	}
	if addr.Is6() {
		return IPv6
	}
	return IPAny
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
