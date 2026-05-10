package probe

import (
	"context"
	"net/netip"
	"testing"
	"time"
)

func TestDefaultFactorySelectsICMP(t *testing.T) {
	originalICMP := newICMPProber
	originalUDP := newUDPProber
	defer func() {
		newICMPProber = originalICMP
		newUDPProber = originalUDP
	}()

	var icmpCalled bool
	newICMPProber = func(ctx context.Context, dst netip.Addr, opts Options) (Prober, error) {
		_ = ctx
		_ = dst
		if opts.Protocol != ProtocolICMP {
			t.Fatalf("Protocol = %d, want %d", opts.Protocol, ProtocolICMP)
		}
		icmpCalled = true
		return noopProber{}, nil
	}
	newUDPProber = func(ctx context.Context, dst netip.Addr, opts Options) (Prober, error) {
		t.Fatal("UDP prober should not be selected")
		return nil, nil
	}

	prober, err := defaultFactory{}.New(context.Background(), netip.MustParseAddr("192.0.2.1"), Options{
		Protocol: ProtocolICMP,
	})
	if err != nil {
		t.Fatal(err)
	}
	if prober == nil || !icmpCalled {
		t.Fatal("ICMP prober was not returned")
	}
}

func TestDefaultFactorySelectsUDP(t *testing.T) {
	originalICMP := newICMPProber
	originalUDP := newUDPProber
	defer func() {
		newICMPProber = originalICMP
		newUDPProber = originalUDP
	}()

	var udpCalled bool
	newICMPProber = func(ctx context.Context, dst netip.Addr, opts Options) (Prober, error) {
		t.Fatal("ICMP prober should not be selected")
		return nil, nil
	}
	newUDPProber = func(ctx context.Context, dst netip.Addr, opts Options) (Prober, error) {
		_ = ctx
		_ = dst
		if opts.Protocol != ProtocolUDP {
			t.Fatalf("Protocol = %d, want %d", opts.Protocol, ProtocolUDP)
		}
		udpCalled = true
		return noopProber{}, nil
	}

	prober, err := defaultFactory{}.New(context.Background(), netip.MustParseAddr("192.0.2.1"), Options{
		Protocol: ProtocolUDP,
	})
	if err != nil {
		t.Fatal(err)
	}
	if prober == nil || !udpCalled {
		t.Fatal("UDP prober was not returned")
	}
}

type noopProber struct{}

func (noopProber) Send(ctx context.Context, ttl int, attempt int) (Sent, error) {
	_ = ctx
	_ = ttl
	_ = attempt
	return Sent{}, nil
}

func (noopProber) Receive(ctx context.Context, sent Sent, timeout time.Duration) (Reply, error) {
	_ = ctx
	_ = sent
	_ = timeout
	return Reply{}, ErrTimeout
}

func (noopProber) Close() error {
	return nil
}
