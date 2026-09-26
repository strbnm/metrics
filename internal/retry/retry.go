package retry

import (
	"context"
	"time"
)

var DefaultIntervals = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

// IsRetriable определяет, стоит ли повторять попытку.
type IsRetriable func(err error) bool

// Do выполняет fn с повторами. ctx отменяет ожидание между попытками.
func Do(ctx context.Context, fn func() error, intervals []time.Duration, isRetriable IsRetriable) error {
	var lastErr error

	for attempt := 0; attempt <= len(intervals); attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !isRetriable(lastErr) {
			return lastErr
		}
		if attempt < len(intervals) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(intervals[attempt]):
			}
		}
	}
	return lastErr
}
