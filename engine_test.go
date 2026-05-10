package traceroute

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/yorukot/go-traceroute/internal/probe"
	"github.com/yorukot/go-traceroute/internal/testutil"
)

type fakeResolver struct {
	addrs []netip.Addr
	names map[netip.Addr][]string
	err   error
}

func (r fakeResolver) LookupIP(ctx context.Context, host string, version IPVersion) ([]netip.Addr, error) {
	_ = ctx
	_ = host
	_ = version
	if r.err != nil {
		return nil, r.err
	}
	return r.addrs, nil
}

func (r fakeResolver) LookupAddr(ctx context.Context, addr netip.Addr) ([]string, error) {
	_ = ctx
	if r.names == nil {
		return nil, errors.New("not found")
	}
	return r.names[addr], nil
}

func testOptions() Options {
	return Options{
		IPVersion:     IPAny,
		FirstHop:      1,
		MaxHops:       64,
		QueriesPerHop: 1,
		Timeout:       time.Second,
		PacketSize:    48,
		UDPBasePort:   defaultUDPBasePort,
	}
}

func TestEngineStopsWhenDestinationReached(t *testing.T) {
	dst := netip.MustParseAddr("93.184.216.34")
	prober := testutil.NewFakeProber([]testutil.FakeHop{
		{
			TTL: 1,
			Replies: []probe.Reply{
				{
					From: netip.MustParseAddr("192.168.1.1"),
					Kind: probe.ReplyTimeExceeded,
				},
			},
		},
		{
			TTL: 2,
			Replies: []probe.Reply{
				{
					From: dst,
					Kind: probe.ReplyDestination,
				},
			},
		},
	})

	eng := newTraceEngine(
		testOptions(),
		fakeResolver{addrs: []netip.Addr{dst}},
		testutil.FakeFactory{Prober: prober},
		nil,
	)

	trace, err := eng.Trace(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if len(trace.Hops) != 2 {
		t.Fatalf("len(Hops) = %d, want 2", len(trace.Hops))
	}
	if got := trace.Hops[1].Probes[0].Status; got != StatusDestination {
		t.Fatalf("destination status = %q, want %q", got, StatusDestination)
	}
	if !prober.Closed() {
		t.Fatal("prober was not closed")
	}
}

func TestEngineContinuesAfterTimeout(t *testing.T) {
	dst := netip.MustParseAddr("93.184.216.34")
	prober := testutil.NewFakeProber([]testutil.FakeHop{
		{
			TTL: 2,
			Replies: []probe.Reply{
				{
					From: dst,
					Kind: probe.ReplyDestination,
				},
			},
		},
	})

	eng := newTraceEngine(
		testOptions(),
		fakeResolver{addrs: []netip.Addr{dst}},
		testutil.FakeFactory{Prober: prober},
		nil,
	)

	trace, err := eng.Trace(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if got := trace.Hops[0].Probes[0].Status; got != StatusTimeout {
		t.Fatalf("first probe status = %q, want %q", got, StatusTimeout)
	}
	if len(trace.Hops) != 2 {
		t.Fatalf("len(Hops) = %d, want 2", len(trace.Hops))
	}
}

func TestEngineMaxHopsIsPartialTraceNotFatal(t *testing.T) {
	dst := netip.MustParseAddr("93.184.216.34")
	opts := testOptions()
	opts.MaxHops = 2

	prober := testutil.NewFakeProber([]testutil.FakeHop{
		{
			TTL: 1,
			Replies: []probe.Reply{
				{
					From: netip.MustParseAddr("192.168.1.1"),
					Kind: probe.ReplyTimeExceeded,
				},
			},
		},
		{
			TTL: 2,
			Replies: []probe.Reply{
				{
					From: netip.MustParseAddr("192.168.1.2"),
					Kind: probe.ReplyTimeExceeded,
				},
			},
		},
	})

	eng := newTraceEngine(
		opts,
		fakeResolver{addrs: []netip.Addr{dst}},
		testutil.FakeFactory{Prober: prober},
		nil,
	)

	trace, err := eng.Trace(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Hops) != 2 {
		t.Fatalf("len(Hops) = %d, want 2", len(trace.Hops))
	}
}

func TestEngineResolveNamesDoesNotFailTrace(t *testing.T) {
	dst := netip.MustParseAddr("93.184.216.34")
	hopAddr := netip.MustParseAddr("192.168.1.1")
	opts := testOptions()
	opts.ResolveNames = true

	prober := testutil.NewFakeProber([]testutil.FakeHop{
		{
			TTL: 1,
			Replies: []probe.Reply{
				{
					From: hopAddr,
					Kind: probe.ReplyTimeExceeded,
				},
			},
		},
	})

	eng := newTraceEngine(
		opts,
		fakeResolver{
			addrs: []netip.Addr{dst},
			names: map[netip.Addr][]string{hopAddr: {"router.local"}},
		},
		testutil.FakeFactory{Prober: prober},
		nil,
	)

	trace, err := eng.Trace(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got := trace.Hops[0].Probes[0].Hostname; got != "router.local" {
		t.Fatalf("Hostname = %q, want router.local", got)
	}
}

func TestEngineContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dst := netip.MustParseAddr("93.184.216.34")
	prober := testutil.NewFakeProber(nil)
	eng := newTraceEngine(
		testOptions(),
		fakeResolver{addrs: []netip.Addr{dst}},
		testutil.FakeFactory{Prober: prober},
		nil,
	)

	_, err := eng.Trace(ctx, "example.com")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Trace() error = %v, want context.Canceled", err)
	}
}

func TestEngineEmitsStreamEventsInOrder(t *testing.T) {
	dst := netip.MustParseAddr("93.184.216.34")
	prober := testutil.NewFakeProber([]testutil.FakeHop{
		{
			TTL: 1,
			Replies: []probe.Reply{
				{
					From: dst,
					Kind: probe.ReplyDestination,
				},
			},
		},
	})
	var sink collectSink

	eng := newTraceEngine(
		testOptions(),
		fakeResolver{addrs: []netip.Addr{dst}},
		testutil.FakeFactory{Prober: prober},
		&sink,
	)

	if _, err := eng.Trace(context.Background(), "example.com"); err != nil {
		t.Fatal(err)
	}

	kinds := sink.kinds()
	want := []EventKind{EventProbe, EventHop, EventDone}
	if len(kinds) != len(want) {
		t.Fatalf("event count = %d, want %d: %v", len(kinds), len(want), kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("event %d = %q, want %q", i, kinds[i], want[i])
		}
	}
}

type collectSink []Event

func (s *collectSink) Emit(event Event) {
	*s = append(*s, event)
}

func (s collectSink) kinds() []EventKind {
	kinds := make([]EventKind, len(s))
	for i, event := range s {
		kinds[i] = event.Kind
	}
	return kinds
}
