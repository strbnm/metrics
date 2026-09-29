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
	}, nil, func(err error) bool { return true })
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
	}, nil, func(err error) bool {
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
	}, nil, func(err error) bool {
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
	}, nil, func(err error) bool { return true })
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
	}, nil, func(err error) bool { return true })
	require.Error(t, err)
	assert.LessOrEqual(t, calls, 2) // не успеет дойти до 3-й попытки
}

func TestDo_CustomIntervals(t *testing.T) {
	t.Parallel()

	// Кастомные интервалы: 10 мс, 20 мс, 30 мс
	customIntervals := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}

	calls := 0
	startTime := time.Now()

	err := Do(context.Background(), nil, func() error {
		calls++
		// Имитируем ошибку на первых 3 попытках, успех на 4-й
		if calls < 4 {
			return errors.New("transient")
		}
		return nil
	}, customIntervals, func(err error) bool {
		return err.Error() == "transient"
	})

	require.NoError(t, err)
	assert.Equal(t, 4, calls) // 1 попытка + 3 ретрая

	elapsed := time.Since(startTime)

	// Минимальное время: сумма всех интервалов (10+20+30 = 60 мс)
	minExpected := customIntervals[0] + customIntervals[1] + customIntervals[2]

	tolerance := 5 * time.Millisecond

	assert.GreaterOrEqual(t, elapsed, minExpected)
	assert.LessOrEqual(t, elapsed, minExpected+tolerance)
}

func TestDo_NilIntervalsUsesDefault(t *testing.T) {
	t.Parallel()

	defaultIntervals := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}

	calls := 0
	startTime := time.Now()

	err := Do(context.Background(), nil, func() error {
		calls++
		if calls < 4 {
			return errors.New("transient")
		}
		return nil
	}, nil, func(err error) bool {
		return err.Error() == "transient"
	})

	require.NoError(t, err)
	assert.Equal(t, 4, calls)

	elapsed := time.Since(startTime)
	minExpected := defaultIntervals[0] + defaultIntervals[1] + defaultIntervals[2]
	tolerance := 200 * time.Millisecond // допуск больше, потому что секунды

	assert.GreaterOrEqual(t, elapsed, minExpected)
	assert.LessOrEqual(t, elapsed, minExpected+tolerance)
}
