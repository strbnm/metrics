package service

import (
	"context"
	"log"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"

	models "github.com/strbnm/metrics/internal/model"
)

type Collector struct {
	pollInterval   time.Duration
	reportInterval time.Duration

	mu                sync.RWMutex
	pollCount         int64
	lastSentPollCount int64
	lastMetrics       []models.Metrics
}

func NewCollector(pollInterval, reportInterval time.Duration) *Collector {
	return &Collector{
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
	}
}

// ptr возвращает указатель на значение.
func ptr[T any](v T) *T {
	return &v
}

func uint64ToFloat64(value uint64) *float64 {
	return ptr(float64(value))
}

func uint32ToFloat64(value uint32) *float64 {
	return ptr(float64(value))
}

// collectRuntimeMetrics собирает gauge-метрики из runtime.MemStats
// и случайное значение RandomValue. Не зависит от состояния коллектора.
func collectRuntimeMetrics() []models.Metrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: uint64ToFloat64(m.Alloc)},
		{ID: "BuckHashSys", MType: models.Gauge, Value: uint64ToFloat64(m.BuckHashSys)},
		{ID: "Frees", MType: models.Gauge, Value: uint64ToFloat64(m.Frees)},
		{ID: "GCCPUFraction", MType: models.Gauge, Value: &m.GCCPUFraction},
		{ID: "GCSys", MType: models.Gauge, Value: uint64ToFloat64(m.GCSys)},
		{ID: "HeapAlloc", MType: models.Gauge, Value: uint64ToFloat64(m.HeapAlloc)},
		{ID: "HeapIdle", MType: models.Gauge, Value: uint64ToFloat64(m.HeapIdle)},
		{ID: "HeapInuse", MType: models.Gauge, Value: uint64ToFloat64(m.HeapInuse)},
		{ID: "HeapObjects", MType: models.Gauge, Value: uint64ToFloat64(m.HeapObjects)},
		{ID: "HeapReleased", MType: models.Gauge, Value: uint64ToFloat64(m.HeapReleased)},
		{ID: "HeapSys", MType: models.Gauge, Value: uint64ToFloat64(m.HeapSys)},
		{ID: "LastGC", MType: models.Gauge, Value: uint64ToFloat64(m.LastGC)},
		{ID: "Lookups", MType: models.Gauge, Value: uint64ToFloat64(m.Lookups)},
		{ID: "MCacheInuse", MType: models.Gauge, Value: uint64ToFloat64(m.MCacheInuse)},
		{ID: "MCacheSys", MType: models.Gauge, Value: uint64ToFloat64(m.MCacheSys)},
		{ID: "MSpanInuse", MType: models.Gauge, Value: uint64ToFloat64(m.MSpanInuse)},
		{ID: "MSpanSys", MType: models.Gauge, Value: uint64ToFloat64(m.MSpanSys)},
		{ID: "Mallocs", MType: models.Gauge, Value: uint64ToFloat64(m.Mallocs)},
		{ID: "NextGC", MType: models.Gauge, Value: uint64ToFloat64(m.NextGC)},
		{ID: "NumForcedGC", MType: models.Gauge, Value: uint32ToFloat64(m.NumForcedGC)},
		{ID: "NumGC", MType: models.Gauge, Value: uint32ToFloat64(m.NumGC)},
		{ID: "OtherSys", MType: models.Gauge, Value: uint64ToFloat64(m.OtherSys)},
		{ID: "PauseTotalNs", MType: models.Gauge, Value: uint64ToFloat64(m.PauseTotalNs)},
		{ID: "StackInuse", MType: models.Gauge, Value: uint64ToFloat64(m.StackInuse)},
		{ID: "StackSys", MType: models.Gauge, Value: uint64ToFloat64(m.StackSys)},
		{ID: "Sys", MType: models.Gauge, Value: uint64ToFloat64(m.Sys)},
		{ID: "TotalAlloc", MType: models.Gauge, Value: uint64ToFloat64(m.TotalAlloc)},
		// Дополнительные метрики
		{ID: "RandomValue", MType: models.Gauge, Value: ptr(rand.Float64())},
	}
}

func (c *Collector) collectOnce() {
	c.pollCount++

	metrics := collectRuntimeMetrics()

	delta := c.pollCount - c.lastSentPollCount
	metrics = append(metrics,
		models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &delta},
	)

	c.setMetrics(metrics)
}

func (c *Collector) setMetrics(metrics []models.Metrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastMetrics = metrics
}

func (c *Collector) GetLastMetrics() []models.Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastMetrics
}

func (c *Collector) Start(ctx context.Context, sendFunc func(ctx context.Context, metrics []models.Metrics) error) {
	c.collectOnce()

	pollTicker := time.NewTicker(c.pollInterval)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(c.reportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			c.collectOnce()
		case <-reportTicker.C:
			c.reportOnce(ctx, sendFunc)
		}
	}
}

// reportOnce отправляет последний снимок метрик; счётчик отправленных
// опросов увеличивается только после успешной отправки.
func (c *Collector) reportOnce(ctx context.Context, sendFunc func(ctx context.Context, metrics []models.Metrics) error) error {
	metrics := c.GetLastMetrics()
	if err := sendFunc(ctx, metrics); err != nil {
		log.Printf("Failed to send metrics: %v", err)
		return err
	}
	c.lastSentPollCount = c.pollCount
	return nil
}
