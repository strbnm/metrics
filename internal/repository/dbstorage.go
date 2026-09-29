package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/repository/postgres"
	"github.com/strbnm/metrics/internal/retry"
	"go.uber.org/zap"
)

type DBStorage struct {
	db         *sql.DB
	classifier *postgres.PostgresErrorClassifier
	logger     *zap.SugaredLogger
}

func (s *DBStorage) isRetriable(err error) bool {
	return s.classifier.Classify(err) == postgres.Retriable || retry.IsNetError(err)
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
	if err := validate(metric); err != nil {
		return err
	}
	return retry.Do(ctx, s.logger, func() error {
		switch metric.MType {
		case models.Gauge:
			_, err := s.db.ExecContext(ctx, upsertGaugeQuery, metric.ID, metric.MType, *metric.Value, metric.Hash)
			return err
		case models.Counter:
			_, err := s.db.ExecContext(ctx, upsertCounterQuery, metric.ID, metric.MType, *metric.Delta, metric.Hash)
			return err
		}
		return nil
	}, nil, s.isRetriable)
}

func (s *DBStorage) Get(ctx context.Context, id string, mType string) (models.Metrics, error) {
	var result models.Metrics

	err := retry.Do(ctx, s.logger, func() error {
		err := s.db.QueryRowContext(ctx, findByIDAndTypeQuery, id, mType).
			Scan(&result.ID, &result.MType, &result.Delta, &result.Value, &result.Hash)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: %w", ErrMetricNotFound, err)
			}
			return err
		}
		return nil
	}, nil, s.isRetriable)

	return result, err
}

func (s *DBStorage) List(ctx context.Context) ([]models.Metrics, error) {
	var result []models.Metrics

	err := retry.Do(ctx, s.logger, func() error {
		result = nil // сброс перед каждой попыткой

		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelRepeatableRead,
			ReadOnly:  true,
		})
		if err != nil {
			return err
		}
		defer tx.Rollback()

		rows, err := tx.QueryContext(ctx, findAllQuery)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var m models.Metrics
			if err := rows.Scan(&m.ID, &m.MType, &m.Delta, &m.Value, &m.Hash); err != nil {
				return err
			}
			result = append(result, m)
		}
		return rows.Err()
	}, nil, s.isRetriable)

	return result, err
}

func (s *DBStorage) SaveBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	return retry.Do(ctx, s.logger, func() error {
		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
		})
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		stmtUpsertGauge, err := tx.PrepareContext(ctx, upsertGaugeQuery)
		if err != nil {
			return fmt.Errorf("prepare upsert gauge: %w", err)
		}
		defer stmtUpsertGauge.Close()

		stmtUpsertCounter, err := tx.PrepareContext(ctx, upsertCounterQuery)
		if err != nil {
			return fmt.Errorf("prepare upsert counter: %w", err)
		}
		defer stmtUpsertCounter.Close()

		for _, m := range metrics {
			if err = validate(m); err != nil {
				return err
			}
			switch m.MType {
			case models.Gauge:
				if _, err := stmtUpsertGauge.ExecContext(ctx, m.ID, m.MType, *m.Value, m.Hash); err != nil {
					return fmt.Errorf("upsert gauge in batch: %w", err)
				}
			case models.Counter:
				if _, err := stmtUpsertCounter.ExecContext(ctx, m.ID, m.MType, *m.Delta, m.Hash); err != nil {
					return fmt.Errorf("upsert counter in batch: %w", err)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit batch: %w", err)
		}
		return nil
	}, nil, s.isRetriable)
}

func NewDBStorage(db *sql.DB, logger *zap.SugaredLogger) *DBStorage {
	return &DBStorage{
		db:         db,
		classifier: postgres.NewPostgresErrorClassifier(),
		logger:     logger,
	}
}

func (s *DBStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
