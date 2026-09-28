package repository

import (
	"context"
	"sync"

	models "github.com/strbnm/metrics/internal/model"
)

type MemStorage struct {
	metrics map[string]models.Metrics
	mu      sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}

func (r *MemStorage) Save(ctx context.Context, metric models.Metrics) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := validate(metric); err != nil {
		return err
	}

	key := buildKey(metric.ID, metric.MType)
	switch metric.MType {
	case models.Gauge:
		r.metrics[key] = metric
	case models.Counter:
		if existing, ok := r.metrics[key]; ok {
			newDelta := *existing.Delta + *metric.Delta
			r.metrics[key] = models.Metrics{
				ID:    metric.ID,
				MType: metric.MType,
				Delta: &newDelta,
				Hash:  metric.Hash,
			}
		} else {
			r.metrics[key] = metric
		}
	}

	return nil
}

func (r *MemStorage) Get(ctx context.Context, id string, mType string) (models.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return models.Metrics{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	key := buildKey(id, mType)
	metric, ok := r.metrics[key]
	if !ok {
		return models.Metrics{}, ErrMetricNotFound
	}
	return metric, nil
}

func (r *MemStorage) List(ctx context.Context) ([]models.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make([]models.Metrics, 0, len(r.metrics))
	for _, metric := range r.metrics {
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

func (r *MemStorage) SaveBatch(ctx context.Context, metrics []models.Metrics) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	tmp := make(map[string]models.Metrics, len(r.metrics))
	for k, v := range r.metrics {
		tmp[k] = v
	}

	for _, m := range metrics {
		if err := validate(m); err != nil {
			return err
		}
		key := buildKey(m.ID, m.MType)

		switch m.MType {
		case models.Gauge:
			tmp[key] = models.Metrics{
				ID:    m.ID,
				MType: m.MType,
				Value: m.Value,
				Hash:  m.Hash,
			}
		case models.Counter:
			existing, ok := tmp[key]
			if !ok {
				tmp[key] = models.Metrics{
					ID:    m.ID,
					MType: m.MType,
					Delta: m.Delta,
					Hash:  m.Hash,
				}
			} else {
				newDelta := *existing.Delta + *m.Delta
				tmp[key] = models.Metrics{
					ID:    m.ID,
					MType: m.MType,
					Delta: &newDelta,
					Hash:  m.Hash,
				}
			}
		}
	}

	r.metrics = tmp
	return nil
}

func buildKey(id string, mType string) string {
	return id + mType
}

func validate(m models.Metrics) error {
	switch m.MType {
	case models.Gauge:
		if m.Value == nil {
			return ErrGaugeMetricIsEmptyValue
		}
	case models.Counter:
		if m.Delta == nil {
			return ErrCounterMetricIsEmptyValue
		}
	default:
		return ErrInvalidMetricType
	}
	return nil
}
