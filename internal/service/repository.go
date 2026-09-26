package service

import (
	"context"

	models "github.com/strbnm/metrics/internal/model"
)

type Repository interface {
	Save(ctx context.Context, metric models.Metrics) error
	Get(ctx context.Context, id string, mType string) (models.Metrics, error)
	List(ctx context.Context) ([]models.Metrics, error)
	SaveBatch(ctx context.Context, metrics []models.Metrics) error
}
