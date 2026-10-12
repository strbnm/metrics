package service

import (
	"math/rand/v2"
	"runtime"

	models "github.com/strbnm/metrics/internal/model"
)

func collectRuntimeMetrics(pollCount, lastSentPollCount int64) []models.Metrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	delta := pollCount - lastSentPollCount

	metrics := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: ptr(float64(m.Alloc))},
		{ID: "BuckHashSys", MType: models.Gauge, Value: ptr(float64(m.BuckHashSys))},
		{ID: "Frees", MType: models.Gauge, Value: ptr(float64(m.Frees))},
		{ID: "GCCPUFraction", MType: models.Gauge, Value: &m.GCCPUFraction},
		{ID: "GCSys", MType: models.Gauge, Value: ptr(float64(m.GCSys))},
		{ID: "HeapAlloc", MType: models.Gauge, Value: ptr(float64(m.HeapAlloc))},
		{ID: "HeapIdle", MType: models.Gauge, Value: ptr(float64(m.HeapIdle))},
		{ID: "HeapInuse", MType: models.Gauge, Value: ptr(float64(m.HeapInuse))},
		{ID: "HeapObjects", MType: models.Gauge, Value: ptr(float64(m.HeapObjects))},
		{ID: "HeapReleased", MType: models.Gauge, Value: ptr(float64(m.HeapReleased))},
		{ID: "HeapSys", MType: models.Gauge, Value: ptr(float64(m.HeapSys))},
		{ID: "LastGC", MType: models.Gauge, Value: ptr(float64(m.LastGC))},
		{ID: "Lookups", MType: models.Gauge, Value: ptr(float64(m.Lookups))},
		{ID: "MCacheInuse", MType: models.Gauge, Value: ptr(float64(m.MCacheInuse))},
		{ID: "MCacheSys", MType: models.Gauge, Value: ptr(float64(m.MCacheSys))},
		{ID: "MSpanInuse", MType: models.Gauge, Value: ptr(float64(m.MSpanInuse))},
		{ID: "MSpanSys", MType: models.Gauge, Value: ptr(float64(m.MSpanSys))},
		{ID: "Mallocs", MType: models.Gauge, Value: ptr(float64(m.Mallocs))},
		{ID: "NextGC", MType: models.Gauge, Value: ptr(float64(m.NextGC))},
		{ID: "NumForcedGC", MType: models.Gauge, Value: ptr(float64(m.NumForcedGC))},
		{ID: "NumGC", MType: models.Gauge, Value: ptr(float64(m.NumGC))},
		{ID: "OtherSys", MType: models.Gauge, Value: ptr(float64(m.OtherSys))},
		{ID: "PauseTotalNs", MType: models.Gauge, Value: ptr(float64(m.PauseTotalNs))},
		{ID: "StackInuse", MType: models.Gauge, Value: ptr(float64(m.StackInuse))},
		{ID: "StackSys", MType: models.Gauge, Value: ptr(float64(m.StackSys))},
		{ID: "Sys", MType: models.Gauge, Value: ptr(float64(m.Sys))},
		{ID: "TotalAlloc", MType: models.Gauge, Value: ptr(float64(m.TotalAlloc))},
		// Дополнительные метрики
		{ID: "RandomValue", MType: models.Gauge, Value: ptr(rand.Float64())},
	}

	if delta > 0 {
		metrics = append(metrics, models.Metrics{
			ID: "PollCount", MType: models.Counter, Delta: ptr(delta),
		})
	}

	return metrics
}

func ptr[T any](v T) *T {
	return &v
}
