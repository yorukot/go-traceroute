package traceroute

import (
	"context"
	"errors"
	"fmt"
	"net/netip"

	"github.com/yorukot/go-traceroute/internal/engine"
	"github.com/yorukot/go-traceroute/internal/probe"
)

// Tracer is a reusable tracer configured with Options.
type Tracer struct {
	opts Options
}

// New creates a tracer after normalizing and validating options.
func New(opts Options) (*Tracer, error) {
	opts = opts.Normalize()
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	return &Tracer{opts: opts}, nil
}

// TraceRoute performs a single blocking trace.
func TraceRoute(ctx context.Context, target string, opts Options) (*Trace, error) {
	t, err := New(opts)
	if err != nil {
		return nil, err
	}

	return t.Trace(ctx, target)
}

// Trace performs a blocking trace and returns structured results.
func (t *Tracer) Trace(ctx context.Context, target string) (*Trace, error) {
	eng := t.newEngine(nil)
	trace, err := eng.Trace(ctx, target)
	if err != nil {
		return convertTrace(trace), t.translateError(err)
	}
	return convertTrace(trace), nil
}

// TraceStream starts a trace and emits progress events until the trace finishes
// or the context is canceled.
func (t *Tracer) TraceStream(ctx context.Context, target string) (<-chan Event, error) {
	events := make(chan Event)
	eng := t.newEngine(events)

	go func() {
		defer close(events)

		if _, err := eng.Trace(ctx, target); err != nil {
			events <- Event{Kind: EventError, Error: t.translateError(err)}
		}
	}()

	return events, nil
}

func (t *Tracer) newEngine(events chan<- Event) *engine.Engine {
	return engine.New(
		toEngineOptions(t.opts),
		resolverAdapter{},
		probe.NewFactory(),
		traceSink{
			events: events,
			hooks:  t.opts.Hooks,
		},
	)
}

func (t *Tracer) translateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, probe.ErrPermission) {
		return &PermissionError{
			Operation: "socket",
			Cause:     err,
		}
	}
	if errors.Is(err, probe.ErrTimeout) {
		return fmt.Errorf("%w: %w", ErrTimeout, err)
	}
	if errors.Is(err, engine.ErrNoAddress) {
		return fmt.Errorf("%w: %w", ErrNoAddress, err)
	}
	return err
}

func toEngineOptions(opts Options) engine.Options {
	return engine.Options{
		IPVersion:     engine.IPVersion(opts.IPVersion),
		FirstHop:      opts.FirstHop,
		MaxHops:       opts.MaxHops,
		QueriesPerHop: opts.QueriesPerHop,
		Timeout:       opts.Timeout,
		PacketSize:    opts.PacketSize,
		ResolveNames:  opts.ResolveNames,
	}
}

type resolverAdapter struct {
}

func (r resolverAdapter) LookupIP(ctx context.Context, host string, version engine.IPVersion) ([]netip.Addr, error) {
	addrs, err := (defaultResolver{}).LookupIP(ctx, host, IPVersion(version))
	if errors.Is(err, ErrNoAddress) {
		return nil, engine.ErrNoAddress
	}
	return addrs, err
}

func (r resolverAdapter) LookupAddr(ctx context.Context, addr netip.Addr) ([]string, error) {
	return (defaultResolver{}).LookupAddr(ctx, addr)
}

type traceSink struct {
	events chan<- Event
	hooks  Hooks
}

func (s traceSink) Emit(event engine.Event) {
	switch event.Kind {
	case engine.EventProbeSent:
		if s.hooks.OnProbeSent != nil && event.HopProbe != nil {
			s.hooks.OnProbeSent(HopProbe{
				TTL:     event.HopProbe.TTL,
				Attempt: event.HopProbe.Attempt,
			})
		}
	case engine.EventProbe:
		if event.Probe == nil {
			return
		}
		probe := convertProbe(*event.Probe)
		if s.hooks.OnProbeReceived != nil && probe.Status != StatusTimeout && probe.Status != StatusError {
			s.hooks.OnProbeReceived(probe)
		}
		if s.events != nil {
			s.events <- Event{Kind: EventProbe, Probe: &probe}
		}
	case engine.EventHop:
		if event.Hop == nil {
			return
		}
		hop := convertHop(*event.Hop)
		if s.hooks.OnHopComplete != nil {
			s.hooks.OnHopComplete(hop)
		}
		if s.events != nil {
			s.events <- Event{Kind: EventHop, Hop: &hop}
		}
	case engine.EventDone:
		if event.Trace == nil {
			return
		}
		trace := convertTrace(event.Trace)
		if s.events != nil {
			s.events <- Event{Kind: EventDone, Trace: trace}
		}
	}
}

func convertTrace(trace *engine.Trace) *Trace {
	if trace == nil {
		return nil
	}

	converted := &Trace{
		Target:      trace.Target,
		Destination: trace.Destination,
		IPVersion:   IPVersion(trace.IPVersion),
		StartedAt:   trace.StartedAt,
		FinishedAt:  trace.FinishedAt,
		Hops:        make([]Hop, 0, len(trace.Hops)),
	}

	for _, hop := range trace.Hops {
		converted.Hops = append(converted.Hops, convertHop(hop))
	}

	return converted
}

func convertHop(hop engine.Hop) Hop {
	converted := Hop{
		TTL:    hop.TTL,
		Probes: make([]Probe, 0, len(hop.Probes)),
	}

	for _, probe := range hop.Probes {
		converted.Probes = append(converted.Probes, convertProbe(probe))
	}

	return converted
}

func convertProbe(probe engine.Probe) Probe {
	return Probe{
		Attempt:    probe.Attempt,
		Addr:       probe.Addr,
		Hostname:   probe.Hostname,
		SentAt:     probe.SentAt,
		ReceivedAt: probe.ReceivedAt,
		RTT:        probe.RTT,
		Status:     Status(probe.Status),
		Annotation: probe.Annotation,
		ICMPType:   probe.ICMPType,
		ICMPCode:   probe.ICMPCode,
		Error:      probe.Error,
	}
}
