package traceroute

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/yorukot/go-traceroute/internal/probe"
)

// traceEngine orchestrates a trace without knowing how probes are implemented.
type traceEngine struct {
	opts    Options
	res     traceResolver
	factory probe.Factory
	sink    eventSink
}

type traceResolver interface {
	LookupIP(ctx context.Context, host string, version IPVersion) ([]netip.Addr, error)
	LookupAddr(ctx context.Context, addr netip.Addr) ([]string, error)
}

type eventSink interface {
	Emit(Event)
}

func newTraceEngine(opts Options, res traceResolver, factory probe.Factory, sink eventSink) *traceEngine {
	return &traceEngine{
		opts:    opts,
		res:     res,
		factory: factory,
		sink:    sink,
	}
}

func (e *traceEngine) Trace(ctx context.Context, target string) (*Trace, error) {
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
		IPVersion:   versionOf(dest),
		StartedAt:   time.Now(),
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
	}

	trace.FinishedAt = time.Now()
	e.emit(Event{Kind: EventDone, Trace: trace})
	return trace, nil
}

func (e *traceEngine) resolve(ctx context.Context, target string) (netip.Addr, error) {
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

func (e *traceEngine) probeOptions() probe.Options {
	return probe.Options{
		Protocol:      probe.Protocol(e.opts.Protocol),
		FirstHop:      e.opts.FirstHop,
		QueriesPerHop: e.opts.QueriesPerHop,
		PacketSize:    e.opts.PacketSize,
		UDPBasePort:   e.opts.UDPBasePort,
	}
}

func (e *traceEngine) emit(event Event) {
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
