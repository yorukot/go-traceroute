package traceroute

import (
	"bytes"
	"encoding/json"
	"net/netip"
	"testing"
)

func TestTraceJSON(t *testing.T) {
	trace := Trace{
		Target:      "example.com",
		Destination: netip.MustParseAddr("93.184.216.34"),
		Method:      MethodICMP,
		IPVersion:   IPv4,
		Hops: []Hop{
			{
				TTL: 1,
				Probes: []Probe{
					{
						Attempt: 1,
						Addr:    netip.MustParseAddr("192.168.1.1"),
						Status:  StatusOK,
					},
				},
			},
		},
	}

	data, err := json.Marshal(trace)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(data, []byte(`"target":"example.com"`)) {
		t.Fatalf("missing target in JSON: %s", data)
	}
	if !bytes.Contains(data, []byte(`"destination":"93.184.216.34"`)) {
		t.Fatalf("missing destination in JSON: %s", data)
	}
}
