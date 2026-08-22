package service

import (
	"math/rand/v2"
	"runtime"
	"time"

	models "github.com/strbnm/metrics/internal/model"
)

type Collector struct {
	pollInterval   time.Duration
	reportInterval time.Duration
	pollCount      int64
}

func NewCollector(pollInterval, reportInterval time.Duration) *Collector {
	return &Collector{
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		pollCount:      0,
	}
}

func uint64ToFloat64(value uint64) *float64 {
	v := float64(value)
	return &v
}

func uint32ToFloat64(value uint32) *float64 {
	v := float64(value)
	return &v
}

func (c *Collector) CollectRuntimeMetrics() []models.Metrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.pollCount++

	metrics := []models.Metrics{
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
		{ID: "PollCount", MType: models.Counter, Delta: &c.pollCount},
		{ID: "RandomValue", MType: models.Gauge, Value: func() *float64 {
			v := rand.Float64()
			return &v
		}()},
	}

	return metrics
}

func (c *Collector) Start(sendFunc func([]models.Metrics) error) {
	ticker := time.NewTicker(c.reportInterval)
	defer ticker.Stop()

	for range ticker.C {
		metrics := c.CollectRuntimeMetrics()
		if err := sendFunc(metrics); err != nil {
			continue
		}
	}
}
