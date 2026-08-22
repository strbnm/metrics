package repository

import models "github.com/strbnm/metrics/internal/model"

type Repository interface {
	Save(metric models.Metrics) error
	Get(id string, mType string) (models.Metrics, error)
	List() ([]models.Metrics, error)
}
