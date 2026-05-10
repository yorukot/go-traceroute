package traceroute

import (
	"context"
	"errors"
	"net/netip"
	"testing"
)

func TestDefaultResolverAcceptsLiteralMatchingVersion(t *testing.T) {
	addrs, err := (defaultResolver{}).LookupIP(context.Background(), "127.0.0.1", IPv4)
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != netip.MustParseAddr("127.0.0.1") {
		t.Fatalf("LookupIP returned %v, want 127.0.0.1", addrs)
	}
}

func TestDefaultResolverRejectsLiteralWrongVersion(t *testing.T) {
	_, err := (defaultResolver{}).LookupIP(context.Background(), "127.0.0.1", IPv6)
	if !errors.Is(err, ErrNoAddress) {
		t.Fatalf("LookupIP error = %v, want ErrNoAddress", err)
	}
}
