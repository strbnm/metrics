package agent

import (
	"fmt"
	"net/http"

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

	resp, err := s.client.R().
		SetBody(metric).
		SetHeader("Content-Type", "application/json; charset=utf-8").
		Post("/update")
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode())
	}

	return nil
}
