package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/strbnm/metrics/internal/compress"
	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/retry"
	"github.com/strbnm/metrics/internal/signer"

	"github.com/go-resty/resty/v2"
)

const (
	senderTimeout = 10 * time.Second
	HashHeader    = "Hash-Sha256"
)

type Sender struct {
	client *resty.Client
	key    string
}

func NewSender(serverURL, key string) (*Sender, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("serverURL cannot be empty")
	}

	client := resty.New().
		SetBaseURL(serverURL).
		SetTimeout(senderTimeout).
		SetRetryCount(0) // повторы выполняет retry.Do

	return &Sender{
		client: client,
		key:    key,
	}, nil
}

func (s *Sender) Send(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	return retry.Do(ctx, nil, func() error {
		return s.post(ctx, "/updates", metrics)
	}, nil, retry.IsNetError)
}

func (s *Sender) post(ctx context.Context, path string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	compressed, err := compress.Compress(data)
	if err != nil {
		return fmt.Errorf("failed to compress payload: %w", err)
	}

	req := s.client.R().
		SetContext(ctx).
		SetBody(compressed).
		SetHeader("Content-Type", "application/json; charset=utf-8").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("Content-Encoding", "gzip")

	if s.key != "" {
		hash := signer.Sign(compressed, s.key)
		req.SetHeader(HashHeader, hash)
	}

	resp, err := req.Post(path)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode())
	}

	return nil
}
