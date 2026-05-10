package clock

import (
	"context"
	"time"
)

// Clock lets engine tests avoid depending on wall-clock time.
type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}
