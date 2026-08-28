package service

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"time"

	models "github.com/strbnm/metrics/internal/model"
)

type Collector struct {
	pollInterval      time.Duration
	reportInterval    time.Duration
	pollCount         int64
	lastSentPollCount int64
	lastMetrics       []models.Metrics
	lastSendTime      time.Time
}

func NewCollector(pollInterval, reportInterval time.Duration) *Collector {
	return &Collector{
		pollInterval:      pollInterval,
		reportInterval:    reportInterval,
		pollCount:         0,
		lastSentPollCount: 0,
		lastSendTime:      time.Now(),
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

func (c *Collector) collectOnce() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.pollCount++

	delta := c.pollCount - c.lastSentPollCount

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
		{ID: "PollCount", MType: models.Counter, Delta: &delta},
		{ID: "RandomValue", MType: models.Gauge, Value: func() *float64 {
			v := rand.Float64()
			return &v
		}()},
	}

	c.lastMetrics = metrics
}

func (c *Collector) GetLastMetrics() []models.Metrics {
	return c.lastMetrics
}

func (c *Collector) Start(sendFunc func([]models.Metrics) error) {
	c.collectOnce()

	for {
		time.Sleep(c.pollInterval)
		c.collectOnce()

		if time.Since(c.lastSendTime) >= c.reportInterval {
			metrics := c.GetLastMetrics()
			if err := sendFunc(metrics); err != nil {

				fmt.Printf("Failed to send metrics: %v\n", err)
			} else {
				c.lastSentPollCount = c.pollCount
			}
			c.lastSendTime = time.Now()
		}
	}
}
