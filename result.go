package traceroute

import (
	"net/netip"
	"time"
)

// Trace is the structured result of a network path trace.
type Trace struct {
	Target      string     `json:"target"`
	Destination netip.Addr `json:"destination"`
	Method      Method     `json:"method"`
	IPVersion   IPVersion  `json:"ip_version"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  time.Time  `json:"finished_at"`
	Hops        []Hop      `json:"hops"`
}

// Hop contains all probes sent with the same TTL or HopLimit.
type Hop struct {
	TTL    int     `json:"ttl"`
	Probes []Probe `json:"probes"`
}

// Probe describes one sent probe and its observed response.
type Probe struct {
	Attempt int `json:"attempt"`

	Addr     netip.Addr `json:"addr,omitempty"`
	Hostname string     `json:"hostname,omitempty"`

	SentAt     time.Time     `json:"sent_at"`
	ReceivedAt time.Time     `json:"received_at,omitempty"`
	RTT        time.Duration `json:"rtt,omitempty"`

	Status     Status `json:"status"`
	Annotation string `json:"annotation,omitempty"`

	ICMPType int `json:"icmp_type,omitempty"`
	ICMPCode int `json:"icmp_code,omitempty"`

	ASN  *ASN        `json:"asn,omitempty"`
	MPLS []MPLSLabel `json:"mpls,omitempty"`

	Error string `json:"error,omitempty"`
}

// Status classifies the outcome of a probe.
type Status string

const (
	StatusOK           Status = "ok"
	StatusTimeout      Status = "timeout"
	StatusDestination  Status = "destination"
	StatusUnreachable  Status = "unreachable"
	StatusFiltered     Status = "filtered"
	StatusPacketTooBig Status = "packet-too-big"
	StatusError        Status = "error"
)

// ASN contains optional autonomous system metadata for a responding hop.
type ASN struct {
	Number int    `json:"number"`
	Name   string `json:"name,omitempty"`
}

// MPLSLabel contains optional MPLS label metadata reported by a hop.
type MPLSLabel struct {
	Label int  `json:"label"`
	Exp   int  `json:"exp"`
	S     bool `json:"s"`
	TTL   int  `json:"ttl"`
}
