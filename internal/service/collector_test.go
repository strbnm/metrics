package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/strbnm/metrics/internal/model"
)

func TestNewCollector(t *testing.T) {
	pollInterval := 1 * time.Second
	reportInterval := 5 * time.Second
	rateLimit := 3

	collector := NewCollector(pollInterval, reportInterval, rateLimit)

	require.NotNil(t, collector, "NewCollector returned nil")
	assert.Equal(t, pollInterval, collector.pollInterval, "pollInterval mismatch")
	assert.Equal(t, reportInterval, collector.reportInterval, "reportInterval mismatch")
	assert.Equal(t, rateLimit, collector.rateLimit, "rateLimit mismatch")
	require.NotNil(t, collector.sendQueue, "sendQueue should be initialized")
	assert.Equal(t, 0, len(collector.sendQueue), "sendQueue should be empty on creation")
}

func TestPtr(t *testing.T) {
	v := 42
	p := ptr(v)
	require.NotNil(t, p, "ptr returned nil")
	assert.Equal(t, 42, *p, "ptr should store the value")

	assert.Equal(t, 3.14, *ptr(3.14), "ptr should work for float64")
	assert.Equal(t, "test", *ptr("test"), "ptr should work for string")
}

func findMetric(t *testing.T, metrics []models.Metrics, id string) models.Metrics {
	t.Helper()
	for _, m := range metrics {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("metric %q not found", id)
	return models.Metrics{}
}

func TestCollectRuntimeMetrics(t *testing.T) {
	t.Run("первый опрос включает PollCount = 1", func(t *testing.T) {
		metrics := collectRuntimeMetrics(1, 0)
		require.NotEmpty(t, metrics, "collectRuntimeMetrics produced no metrics")

		pollCountMetric := findMetric(t, metrics, "PollCount")
		require.NotNil(t, pollCountMetric.Delta, "PollCount Delta should not be nil")
		assert.Equal(t, models.Counter, pollCountMetric.MType, "PollCount should be a counter")
		assert.Equal(t, int64(1), *pollCountMetric.Delta, "PollCount metric should have value 1")

		randomValueMetric := findMetric(t, metrics, "RandomValue")
		require.NotNil(t, randomValueMetric.Value, "RandomValue Value should not be nil")
		value := *randomValueMetric.Value
		assert.GreaterOrEqual(t, value, 0.0, "RandomValue should be >= 0")
		assert.Less(t, value, 1.0, "RandomValue should be < 1")

		// Ключевые gauge-метрики из runtime.MemStats
		for _, id := range []string{"Alloc", "HeapAlloc", "Sys", "TotalAlloc"} {
			m := findMetric(t, metrics, id)
			assert.Equal(t, models.Gauge, m.MType, "%s should be a gauge", id)
			require.NotNil(t, m.Value, "%s Value should not be nil", id)
			assert.GreaterOrEqual(t, *m.Value, 0.0, "%s should be >= 0", id)
		}
	})

	t.Run("дельта PollCount считается как разность с lastSentPollCount", func(t *testing.T) {
		metrics := collectRuntimeMetrics(3, 1)

		pollCountMetric := findMetric(t, metrics, "PollCount")
		require.NotNil(t, pollCountMetric.Delta)
		assert.Equal(t, int64(2), *pollCountMetric.Delta,
			"дельта PollCount должна быть pollCount - lastSentPollCount")
	})

	t.Run("при нулевой дельте PollCount не добавляется", func(t *testing.T) {
		metrics := collectRuntimeMetrics(2, 2)
		require.NotEmpty(t, metrics)

		for _, m := range metrics {
			assert.NotEqual(t, "PollCount", m.ID,
				"PollCount не должен попадать в снапшот при нулевой дельте")
		}
	})
}

func TestCollectSystemMetrics(t *testing.T) {
	metrics := collectSystemMetrics()
	require.NotEmpty(t, metrics, "collectSystemMetrics should return memory/CPU metrics")

	ids := make(map[string]bool)
	for _, m := range metrics {
		assert.Equal(t, models.Gauge, m.MType, "system metric %s should be a gauge", m.ID)
		require.NotNil(t, m.Value, "system metric %s should have a value", m.ID)
		assert.GreaterOrEqual(t, *m.Value, 0.0, "system metric %s should be >= 0", m.ID)
		ids[m.ID] = true
	}

	assert.True(t, ids["TotalMemory"], "TotalMemory metric not found")
	assert.True(t, ids["FreeMemory"], "FreeMemory metric not found")

	hasCPU := false
	for id := range ids {
		if len(id) >= len("CPUUtilization") && id[:len("CPUUtilization")] == "CPUUtilization" {
			hasCPU = true
			break
		}
	}
	assert.True(t, hasCPU, "CPU utilization metrics not found")
}

func TestCollectorRuntimeCollector(t *testing.T) {
	collector := NewCollector(time.Second, time.Second, 1)

	pollTicker := make(chan time.Time)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		collector.runtimeCollector(ctx, pollTicker)
	}()

	// Два тика опроса
	pollTicker <- time.Now()
	pollTicker <- time.Now()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runtimeCollector did not stop after context cancellation")
	}

	collector.mu.RLock()
	defer collector.mu.RUnlock()
	require.NotEmpty(t, collector.lastRuntime, "runtimeCollector should store collected metrics")

	pollCountMetric := findMetric(t, collector.lastRuntime, "PollCount")
	require.NotNil(t, pollCountMetric.Delta)
	assert.Equal(t, int64(2), *pollCountMetric.Delta,
		"после двух опросов дельта PollCount должна быть 2")
}

func TestCollectorSystemCollector(t *testing.T) {
	collector := NewCollector(time.Second, time.Second, 1)

	pollTicker := make(chan time.Time)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		collector.systemCollector(ctx, pollTicker)
	}()

	pollTicker <- time.Now()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("systemCollector did not stop after context cancellation")
	}

	collector.mu.RLock()
	defer collector.mu.RUnlock()
	require.NotEmpty(t, collector.lastSystem, "systemCollector should store collected metrics")

	for _, m := range collector.lastSystem {
		assert.Equal(t, models.Gauge, m.MType, "system metric %s should be a gauge", m.ID)
	}
}

func TestCollectorWorker(t *testing.T) {
	t.Run("доставляет пакеты из очереди и завершается по её закрытию", func(t *testing.T) {
		collector := NewCollector(time.Second, time.Second, 1)

		var mu sync.Mutex
		received := make([][]models.Metrics, 0, 2)

		// worker вызывает wg.Done() при выходе: повторяем контракт Start
		collector.wg.Add(1)
		done := make(chan struct{})
		go func() {
			defer close(done)
			collector.worker(t.Context(), func(ctx context.Context, metrics []models.Metrics) error {
				mu.Lock()
				defer mu.Unlock()
				received = append(received, metrics)
				return nil
			})
		}()

		collector.sendQueue <- []models.Metrics{{ID: "Batch1", MType: models.Gauge, Value: ptr(1.0)}}
		collector.sendQueue <- []models.Metrics{{ID: "Batch2", MType: models.Gauge, Value: ptr(2.0)}}
		close(collector.sendQueue)

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("worker did not finish after queue close")
		}

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, received, 2, "worker should deliver both batches")
		assert.Equal(t, "Batch1", received[0][0].ID, "batches should be delivered in order")
		assert.Equal(t, "Batch2", received[1][0].ID)
	})

	t.Run("ошибка отправки не останавливает воркер", func(t *testing.T) {
		collector := NewCollector(time.Second, time.Second, 1)

		var calls, delivered atomic.Int32

		collector.wg.Add(1)
		done := make(chan struct{})
		go func() {
			defer close(done)
			collector.worker(t.Context(), func(ctx context.Context, metrics []models.Metrics) error {
				if calls.Add(1) == 1 {
					return errors.New("send failed")
				}
				delivered.Add(1)
				return nil
			})
		}()

		collector.sendQueue <- []models.Metrics{{ID: "Fail", MType: models.Gauge, Value: ptr(1.0)}}
		collector.sendQueue <- []models.Metrics{{ID: "Ok", MType: models.Gauge, Value: ptr(2.0)}}
		close(collector.sendQueue)

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("worker did not finish after queue close")
		}

		assert.Equal(t, int32(2), calls.Load(), "worker should process both batches")
		assert.Equal(t, int32(1), delivered.Load(),
			"ошибка первого пакета не должна мешать доставке следующего")
	})

	t.Run("останавливается по отмене контекста", func(t *testing.T) {
		collector := NewCollector(time.Second, time.Second, 1)

		ctx, cancel := context.WithCancel(t.Context())

		collector.wg.Add(1)
		done := make(chan struct{})
		go func() {
			defer close(done)
			collector.worker(ctx, func(ctx context.Context, metrics []models.Metrics) error {
				return nil
			})
		}()

		cancel()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("worker did not stop after context cancellation")
		}
	})
}

func TestCollector_Start(t *testing.T) {
	pollInterval := 10 * time.Millisecond
	reportInterval := 50 * time.Millisecond

	collector := NewCollector(pollInterval, reportInterval, 2)

	var mu sync.Mutex
	runtimeBatches := make([][]models.Metrics, 0)
	systemBatches := make([][]models.Metrics, 0)

	sendFunc := func(ctx context.Context, metrics []models.Metrics) error {
		mu.Lock()
		defer mu.Unlock()
		// runtime-пакет содержит counter PollCount, system-пакет — нет
		isRuntime := false
		for _, m := range metrics {
			if m.ID == "PollCount" {
				isRuntime = true
				break
			}
		}
		if isRuntime {
			runtimeBatches = append(runtimeBatches, metrics)
		} else {
			systemBatches = append(systemBatches, metrics)
		}
		return nil
	}

	ctx, cancel := context.WithCancel(t.Context())
	startDone := make(chan error, 1)
	go func() {
		startDone <- collector.Start(ctx, sendFunc)
	}()

	// Даём коллектору поработать несколько отчётных циклов
	time.Sleep(300 * time.Millisecond)
	cancel()

	select {
	case err := <-startDone:
		require.NoError(t, err, "Start should return nil after context cancellation")
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}

	mu.Lock()
	defer mu.Unlock()

	require.NotEmpty(t, runtimeBatches, "Start should send runtime metrics batches")
	require.NotEmpty(t, systemBatches, "Start should send system metrics batches")

	for i, batch := range runtimeBatches {
		pollCountMetric := findMetric(t, batch, "PollCount")
		require.NotNil(t, pollCountMetric.Delta, "PollCount Delta should not be nil")
		assert.Greater(t, *pollCountMetric.Delta, int64(0),
			"runtime batch %d: PollCount should be positive", i)
		findMetric(t, batch, "Alloc")
		findMetric(t, batch, "RandomValue")
	}

	for i, batch := range systemBatches {
		require.NotEmpty(t, batch, "system batch %d should not be empty", i)
		for _, m := range batch {
			assert.Equal(t, models.Gauge, m.MType, "system metric %s should be a gauge", m.ID)
		}
	}
}
