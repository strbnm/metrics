package service

import (
	"context"
	"fmt"
	"strconv"

	models "github.com/strbnm/metrics/internal/model"
	"go.uber.org/zap"
)

type MetricsService struct {
	repo   Repository
	logger *zap.SugaredLogger
}

func (s *MetricsService) UpdateMetric(ctx context.Context, metricType, metricName, valueStr string) error {
	if metricType != models.Gauge && metricType != models.Counter {
		return ErrInvalidMetricType
	}
	if metricName == "" {
		return ErrEmptyMetricName
	}

	if valueStr == "" {
		return ErrEmptyMetricValue
	}

	var metric models.Metrics

	switch metricType {
	case models.Gauge:
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return ErrInvalidMetricValue
		}
		metric = models.Metrics{
			ID:    metricName,
			MType: models.Gauge,
			Value: &value,
		}
	case models.Counter:
		delta, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return ErrInvalidMetricValue
		}
		metric = models.Metrics{
			ID:    metricName,
			MType: models.Counter,
			Delta: &delta,
		}
	}

	return s.repo.Save(ctx, metric)
}

func (s *MetricsService) ListAllMetrics(ctx context.Context) ([]models.Metrics, error) {
	return s.repo.List(ctx)
}

func (s *MetricsService) GetMetricValue(ctx context.Context, metricName, metricType string) (string, error) {
	metric, err := s.repo.Get(ctx, metricName, metricType)
	if err != nil {
		return "", err
	}
	switch metric.MType {
	case models.Gauge:
		return fmt.Sprint(*metric.Value), nil
	case models.Counter:
		return fmt.Sprint(*metric.Delta), nil
	default:
		return "unknown", nil
	}
}

func (s *MetricsService) UpdateMetricFromModel(ctx context.Context, m models.Metrics) error {
	switch m.MType {
	case models.Counter:
		if m.Delta == nil {
			s.logger.Debugw("nil metric counter value with MType=counter", "metric", m)
			return ErrEmptyMetricValue
		}
		m.Value = nil
	case models.Gauge:
		if m.Value == nil {
			s.logger.Debugw("nil metric gauge value with MType=gauge", "metric", m)
			return ErrEmptyMetricValue
		}
		m.Delta = nil
	}
	return s.repo.Save(ctx, m)
}

func (s *MetricsService) GetMetric(ctx context.Context, m *models.Metrics) error {
	metric, err := s.repo.Get(ctx, m.ID, m.MType)
	if err != nil {
		return err
	}
	switch metric.MType {
	case models.Gauge:
		if metric.Value != nil {
			m.Value = metric.Value
		}
	case models.Counter:
		if metric.Delta != nil {
			m.Delta = metric.Delta
		}
	}
	return nil
}

func (s *MetricsService) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	for _, m := range metrics {
		if m.MType != models.Gauge && m.MType != models.Counter {
			return ErrInvalidMetricType
		}
		if m.ID == "" {
			return ErrEmptyMetricName
		}
	}

	return s.repo.SaveBatch(ctx, metrics)
}

func NewMetricsService(repo Repository, logger *zap.SugaredLogger) *MetricsService {
	return &MetricsService{
		repo:   repo,
		logger: logger,
	}
}
