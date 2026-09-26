package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	t.Parallel()
	calls := 0
	err := Do(context.Background(), nil, func() error {
		calls++
		return nil
	}, DefaultIntervals, func(err error) bool { return true })
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	t.Parallel()
	calls := 0
	err := Do(context.Background(), nil, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	}, DefaultIntervals, func(err error) bool {
		return err.Error() == "transient"
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDo_NonRetriableError(t *testing.T) {
	t.Parallel()
	calls := 0
	err := Do(context.Background(), nil, func() error {
		calls++
		return errors.New("permanent")
	}, DefaultIntervals, func(err error) bool {
		return false
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_ExhaustsRetries(t *testing.T) {
	t.Parallel()
	calls := 0
	err := Do(context.Background(), nil, func() error {
		calls++
		return errors.New("always fails")
	}, DefaultIntervals, func(err error) bool { return true })
	require.Error(t, err)
	// 1 initial + 3 retries = 4
	assert.Equal(t, 4, calls)
}

func TestDo_RespectsContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	calls := 0
	err := Do(ctx, nil, func() error {
		calls++
		return errors.New("transient")
	}, DefaultIntervals, func(err error) bool { return true })
	require.Error(t, err)
	assert.LessOrEqual(t, calls, 2) // не успеет дойти до 3-й попытки
}
