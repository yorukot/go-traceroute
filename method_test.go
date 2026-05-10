package traceroute

import "testing"

func TestParseMethod(t *testing.T) {
	tests := []struct {
		input string
		want  Method
	}{
		{input: "icmp", want: MethodICMP},
		{input: "udp", want: MethodUDP},
		{input: "tcp", want: MethodTCP},
		{input: "udp-paris", want: MethodUDPParis},
		{input: "icmp-paris", want: MethodICMPParis},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseMethod(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("ParseMethod(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseMethodRejectsUnknown(t *testing.T) {
	if _, err := ParseMethod("gre"); err == nil {
		t.Fatal("expected error")
	}
}
