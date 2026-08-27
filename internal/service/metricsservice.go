package service

import (
	"fmt"
	"strconv"

	models "github.com/strbnm/metrics/internal/model"
)

type MetricsService struct {
	repo Repository
}

func (s *MetricsService) UpdateMetric(metricType, metricName, valueStr string) error {
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

	return s.repo.Save(metric)
}

func (s *MetricsService) ListAllMetrics() ([]models.Metrics, error) {
	return s.repo.List()
}

func (s *MetricsService) GetMetricValue(metricName, metricType string) (string, error) {
	metric, err := s.repo.Get(metricName, metricType)
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

func NewMetricsService(repo Repository) *MetricsService {
	return &MetricsService{repo: repo}
}
