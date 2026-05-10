package clock

import (
	"context"
	"sync"
	"time"
)

// FakeClock is a deterministic clock for tests.
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func NewFake(now time.Time) *FakeClock {
	return &FakeClock{now: now}
}

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) Sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
	return nil
}
