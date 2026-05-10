package testutil

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/yorukot/go-traceroute/internal/probe"
)

// FakeHop configures replies returned for a TTL.
type FakeHop struct {
	TTL     int
	Replies []probe.Reply
	Errors  []error
}

// FakeFactory returns a fixed fake prober.
type FakeFactory struct {
	Prober probe.Prober
	Err    error
}

func (f FakeFactory) New(ctx context.Context, dst netip.Addr, opts probe.Options) (probe.Prober, error) {
	_ = ctx
	_ = dst
	_ = opts
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Prober, nil
}

// FakeProber is a deterministic prober for engine tests.
type FakeProber struct {
	mu      sync.Mutex
	hops    map[int]FakeHop
	sent    []probe.Sent
	closed  bool
	base    time.Time
	counter int
}

func NewFakeProber(hops []FakeHop) *FakeProber {
	byTTL := make(map[int]FakeHop, len(hops))
	for _, hop := range hops {
		byTTL[hop.TTL] = hop
	}

	return &FakeProber{
		hops: byTTL,
		base: time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC),
	}
}

func (p *FakeProber) Send(ctx context.Context, ttl int, attempt int) (probe.Sent, error) {
	if err := ctx.Err(); err != nil {
		return probe.Sent{}, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.counter++
	sent := probe.Sent{
		ID: probe.ID{
			Token:   uint64(p.counter),
			TTL:     ttl,
			Attempt: attempt,
		},
		TTL:     ttl,
		Attempt: attempt,
		SentAt:  p.base.Add(time.Duration(p.counter) * time.Millisecond),
	}
	p.sent = append(p.sent, sent)
	return sent, nil
}

func (p *FakeProber) Receive(ctx context.Context, sent probe.Sent, timeout time.Duration) (probe.Reply, error) {
	_ = timeout
	if err := ctx.Err(); err != nil {
		return probe.Reply{}, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	hop, ok := p.hops[sent.TTL]
	if !ok {
		return probe.Reply{}, probe.ErrTimeout
	}

	index := sent.Attempt - 1
	if index >= 0 && index < len(hop.Errors) && hop.Errors[index] != nil {
		return probe.Reply{}, hop.Errors[index]
	}
	if index < 0 || index >= len(hop.Replies) {
		return probe.Reply{}, probe.ErrTimeout
	}

	reply := hop.Replies[index]
	reply.ID = sent.ID
	if reply.ReceivedAt.IsZero() {
		reply.ReceivedAt = sent.SentAt.Add(10 * time.Millisecond)
	}
	if reply.RTT == 0 {
		reply.RTT = reply.ReceivedAt.Sub(sent.SentAt)
	}
	return reply, nil
}

func (p *FakeProber) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

func (p *FakeProber) Sent() []probe.Sent {
	p.mu.Lock()
	defer p.mu.Unlock()

	sent := make([]probe.Sent, len(p.sent))
	copy(sent, p.sent)
	return sent
}

func (p *FakeProber) Closed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}
