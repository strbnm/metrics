package repository

import (
	"errors"

	models "github.com/strbnm/metrics/internal/model"
)

type MemStorage struct {
	metrics map[string]models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}

func (r *MemStorage) Save(metric models.Metrics) error {
	key := buildKey(metric.ID, metric.MType)
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return errors.New("gauge metric must have a value")
		}
		r.metrics[key] = metric
	case models.Counter:
		if metric.Delta == nil {
			return errors.New("counter metric must have a delta")
		}
		if existing, ok := r.metrics[key]; ok {
			existingDelta := *existing.Delta
			newDelta := *metric.Delta
			r.metrics[key] = models.Metrics{
				ID:    metric.ID,
				MType: metric.MType,
				Delta: func(v int64) *int64 { return &v }(existingDelta + newDelta),
			}
		} else {
			r.metrics[key] = metric
		}
	default:
		return errors.New("unknown metric type")
	}

	return nil
}

func (r *MemStorage) Get(id string, mType string) (models.Metrics, error) {
	key := buildKey(id, mType)
	metric, ok := r.metrics[key]
	if !ok {
		return models.Metrics{}, errors.New("metric not found")
	}
	return metric, nil
}

func (r *MemStorage) List() ([]models.Metrics, error) {
	metrics := make([]models.Metrics, 0, len(r.metrics))
	for _, metric := range r.metrics {
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

func buildKey(id string, mType string) string {
	return id + mType
}
