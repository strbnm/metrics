package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/strbnm/metrics/internal/compress"
	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/retry"

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

func (s *Sender) Send(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	return retry.Do(ctx, nil, func() error {
		return s.sendMetrics(&metrics)
	}, nil, func(err error) bool {
		// ретрай только на сетевые/транспортные ошибки
		var netErr net.Error
		return errors.As(err, &netErr)
	})
}

func (s *Sender) sendMetric(metric *models.Metrics) error {
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

func (s *Sender) sendMetrics(metrics *[]models.Metrics) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	compressed, err := compress.Compress(data)
	if err != nil {
		return fmt.Errorf("failed to compress metrics: %w", err)
	}

	resp, err := s.client.R().
		SetBody(compressed).
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("Content-Encoding", "gzip").
		Post("/updates")
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode())
	}

	return nil
}
