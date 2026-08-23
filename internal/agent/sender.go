package agent

import (
	"fmt"
	"net/http"
	"strconv"

	models "github.com/strbnm/metrics/internal/model"
)

type Sender struct {
	serverURL string
	client    *http.Client
}

func NewSender(serverURL string) (*Sender, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("serverURL cannot be empty")
	}

	return &Sender{
		serverURL: serverURL,
		client:    &http.Client{},
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

	url := fmt.Sprintf("%s/update/%s/%s/%s",
		s.serverURL, metric.MType, metric.ID, valueStr)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}
