package service

import (
	"fmt"
	"strconv"

	"github.com/strbnm/metrics/internal/logger"
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

func (s *MetricsService) UpdateMetricFromModel(m models.Metrics) error {
	switch m.MType {
	case models.Counter:
		if m.Delta == nil {
			logger.Log.Debugw("nil metric counter value with MType=counter", "metric", m)
			return ErrEmptyMetricValue
		}
		m.Value = nil
	case models.Gauge:
		if m.Value == nil {
			logger.Log.Debugw("nil metric gauge value with MType=gauge", "metric", m)
			return ErrEmptyMetricValue
		}
		m.Delta = nil
	}
	return s.repo.Save(m)
}

func (s *MetricsService) GetMetric(m *models.Metrics) error {
	metric, err := s.repo.Get(m.ID, m.MType)
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

func NewMetricsService(repo Repository) *MetricsService {
	return &MetricsService{repo: repo}
}
