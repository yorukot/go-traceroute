package traceroute

// EventKind identifies streaming trace events.
type EventKind string

const (
	EventProbe EventKind = "probe"
	EventHop   EventKind = "hop"
	EventDone  EventKind = "done"
	EventError EventKind = "error"
)

// Event is emitted by TraceStream as probes and hops complete.
type Event struct {
	Kind  EventKind
	Probe *Probe
	Hop   *Hop
	Trace *Trace
	Error error
}
