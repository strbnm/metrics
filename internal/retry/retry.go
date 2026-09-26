package retry

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

var DefaultIntervals = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

// IsRetriable определяет, стоит ли повторять попытку.
type IsRetriable func(err error) bool

// Do выполняет fn с повторами. ctx отменяет ожидание между попытками.
func Do(ctx context.Context, log *zap.SugaredLogger, fn func() error, intervals []time.Duration, isRetriable IsRetriable) error {
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
			if log != nil {
				log.Warnw("retrying after error",
					"attempt", attempt+1,
					"max_retries", len(intervals),
					"error", lastErr.Error(),
					"wait", intervals[attempt].String(),
				)
			} else {
				msg := fmt.Sprintf("retrying after error: attempt=%d/%d error=%q wait=%s",
					attempt+1, len(intervals), lastErr.Error(), intervals[attempt].String())
				fmt.Fprintln(os.Stderr, msg)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(intervals[attempt]):
			}
		}
	}
	return lastErr
}
