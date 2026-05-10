package traceroute

// Hooks provides optional callbacks for callers that want progress updates
// without using TraceStream.
type Hooks struct {
	OnProbeSent     func(HopProbe)
	OnProbeReceived func(Probe)
	OnHopComplete   func(Hop)
}

// HopProbe describes a probe as it is sent.
type HopProbe struct {
	TTL     int
	Attempt int
}
