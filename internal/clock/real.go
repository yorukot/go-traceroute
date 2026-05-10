package clock

import (
	"context"
	"time"
)

// RealClock uses the process wall clock.
type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

func (RealClock) Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
