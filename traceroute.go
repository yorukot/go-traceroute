package traceroute

import (
	"context"
	"errors"
	"fmt"

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
		return trace, t.translateError(err)
	}
	return trace, nil
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

func (t *Tracer) newEngine(events chan<- Event) *traceEngine {
	var sink eventSink
	if events != nil {
		sink = channelSink{events: events}
	}

	return newTraceEngine(t.opts, defaultResolver{}, probe.NewFactory(), sink)
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
	if errors.Is(err, ErrNoAddress) {
		return fmt.Errorf("%w: %w", ErrNoAddress, err)
	}
	return err
}

type channelSink struct {
	events chan<- Event
}

func (s channelSink) Emit(event Event) {
	if s.events != nil {
		s.events <- event
	}
}
