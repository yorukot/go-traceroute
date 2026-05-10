package traceroute

import "fmt"

// Method identifies the probe strategy used for a trace.
type Method string

const (
	MethodICMP Method = "icmp"
	MethodUDP  Method = "udp"
	MethodTCP  Method = "tcp"

	MethodUDPParis  Method = "udp-paris"
	MethodICMPParis Method = "icmp-paris"
)

func (m Method) String() string {
	return string(m)
}

// ParseMethod parses a probe method name.
func ParseMethod(s string) (Method, error) {
	switch Method(s) {
	case MethodICMP, MethodUDP, MethodTCP, MethodUDPParis, MethodICMPParis:
		return Method(s), nil
	default:
		return "", fmt.Errorf("traceroute: unknown method %q", s)
	}
}
