package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	models "github.com/strbnm/metrics/internal/model"
)

type DBStorage struct {
	db *sql.DB
}

const (
	upsertGaugeQuery = `
INSERT INTO metrics (id, type, delta, value, hash)
VALUES ($1, $2, NULL, $3, $4)
ON CONFLICT (id, type) DO UPDATE SET
    value = EXCLUDED.value,
    hash  = EXCLUDED.hash;`

	upsertCounterQuery = `
INSERT INTO metrics (id, type, delta, value, hash)
VALUES ($1, $2, $3, NULL, $4)
ON CONFLICT (id, type) DO UPDATE SET
    delta = metrics.delta + EXCLUDED.delta,
    hash  = EXCLUDED.hash;`

	findByIDAndTypeQuery = `
SELECT id, type, delta, value, hash
FROM metrics
WHERE id = $1 AND type = $2;`

	findAllQuery = `
SELECT id, type, delta, value, hash
FROM metrics;`
)

func (s *DBStorage) Save(ctx context.Context, metric models.Metrics) error {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return ErrGaugeMetricIsEmptyValue
		}
		_, err := s.db.ExecContext(ctx, upsertGaugeQuery, metric.ID, metric.MType, *metric.Value, metric.Hash)
		return err

	case models.Counter:
		if metric.Delta == nil {
			return ErrCounterMetricIsEmptyValue
		}
		_, err := s.db.ExecContext(ctx, upsertCounterQuery, metric.ID, metric.MType, *metric.Delta, metric.Hash)
		return err

	default:
		return ErrInvalidMetricType
	}
}

func (s *DBStorage) Get(ctx context.Context, id string, mType string) (models.Metrics, error) {
	var m models.Metrics
	err := s.db.QueryRowContext(ctx, findByIDAndTypeQuery, id, mType).Scan(&m.ID, &m.MType, &m.Delta, &m.Value, &m.Hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Metrics{}, fmt.Errorf("%w: %w", ErrMetricNotFound, err)
		}
		return models.Metrics{}, err
	}
	return m, err
}

func (s *DBStorage) List(ctx context.Context) ([]models.Metrics, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, findAllQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Metrics
	for rows.Next() {
		var m models.Metrics
		if err := rows.Scan(&m.ID, &m.MType, &m.Delta, &m.Value, &m.Hash); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func NewDBStorage(db *sql.DB) *DBStorage {
	return &DBStorage{db: db}
}

func (s *DBStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
