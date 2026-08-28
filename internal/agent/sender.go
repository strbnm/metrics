package agent

import (
	"fmt"
	"net/http"
	"strconv"

	models "github.com/strbnm/metrics/internal/model"

	"github.com/go-resty/resty/v2"
)

type Sender struct {
	client *resty.Client
}

func NewSender(serverURL string) (*Sender, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("serverURL cannot be empty")
	}

	return &Sender{
		client: resty.New().SetBaseURL(serverURL),
	}, nil
}

func (s *Sender) Send(metrics []models.Metrics) error {
	for _, metric := range metrics {
		err := s.sendMetric(metric)
		if err != nil {
			return fmt.Errorf("failed to send metric %s: %w", metric.ID, err)
		}
	}
	return nil
}

func (s *Sender) sendMetric(metric models.Metrics) error {
	var valueStr string

	switch metric.MType {
	case models.Gauge:
		valueStr = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
	case models.Counter:
		valueStr = strconv.FormatInt(*metric.Delta, 10)
	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}

	resp, err := s.client.R().
		SetPathParams(map[string]string{
			"metricType":  metric.MType,
			"metricName":  metric.ID,
			"metricValue": valueStr,
		}).
		SetHeader("Content-Type", "text/plain; charset=utf-8").
		Post("/update/{metricType}/{metricName}/{metricValue}")
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode())
	}

	return nil
}
