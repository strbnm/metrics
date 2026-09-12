package agent

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/strbnm/metrics/internal/compress"
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
		err := s.sendMetric(&metric)
		if err != nil {
			return fmt.Errorf("failed to send metric %s: %w", metric.ID, err)
		}
	}
	return nil
}

func (s *Sender) sendMetric(metric *models.Metrics) error {
	// 1. Маршалинг структуры в JSON байты
	data, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	compressed, err := compress.Compress(data)
	if err != nil {
		return fmt.Errorf("failed to compress metric: %w", err)
	}

	resp, err := s.client.R().
		SetBody(compressed).
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("Content-Encoding", "gzip").
		Post("/update")
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode())
	}

	return nil
}
