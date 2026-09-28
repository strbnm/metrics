package repository

import (
	"context"
	"testing"

	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Базовый набор тест-кейсов для интерфейса Repository
func testRepositoryContract(ctx context.Context, t *testing.T, newRepo func() service.Repository) {
	t.Run("save and get gauge metric", func(t *testing.T) {
		repo := newRepo()

		metric := models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(1.0),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		require.NoError(t, repo.Save(ctx, metric))

		got, err := repo.Get(ctx, metric.ID, metric.MType)
		require.NoError(t, err)
		assert.Equal(t, metric, got)
	})

	t.Run("save and get counter metric", func(t *testing.T) {
		repo := newRepo()

		metric := models.Metrics{
			ID:    "PollCount",
			MType: models.Counter,
			Delta: func(v int64) *int64 { return &v }(100),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		require.NoError(t, repo.Save(ctx, metric))

		got, err := repo.Get(ctx, metric.ID, metric.MType)
		require.NoError(t, err)
		assert.Equal(t, metric, got)
	})

	t.Run("get non-existent metric returns error", func(t *testing.T) {
		repo := newRepo()

		_, err := repo.Get(ctx, "NonExistent", models.Gauge)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMetricNotFound)
	})

	t.Run("list empty storage", func(t *testing.T) {
		repo := newRepo()

		metrics, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, metrics)
	})

	t.Run("list with multiple metrics", func(t *testing.T) {
		repo := newRepo()

		gaugeMetric := models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(2.5),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}
		counterMetric := models.Metrics{
			ID:    "Requests",
			MType: models.Counter,
			Delta: func(v int64) *int64 { return &v }(42),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		require.NoError(t, repo.Save(ctx, gaugeMetric))
		require.NoError(t, repo.Save(ctx, counterMetric))

		metrics, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Len(t, metrics, 2)

		// Проверяем, что обе метрики присутствуют в списке
		var foundGauge, foundCounter bool
		for _, m := range metrics {
			if m.ID == "Alloc" && m.MType == models.Gauge {
				foundGauge = true
			}
			if m.ID == "Requests" && m.MType == models.Counter {
				foundCounter = true
			}
		}
		assert.True(t, foundGauge)
		assert.True(t, foundCounter)
	})

	t.Run("gauge without value returns error", func(t *testing.T) {
		repo := newRepo()

		metric := models.Metrics{
			ID:    "InvalidGauge",
			MType: models.Gauge,
			Value: nil,
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		err := repo.Save(ctx, metric)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrGaugeMetricIsEmptyValue)
	})

	t.Run("counter without delta returns error", func(t *testing.T) {
		repo := newRepo()

		metric := models.Metrics{
			ID:    "InvalidCounter",
			MType: models.Counter,
			Delta: nil,
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		err := repo.Save(ctx, metric)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCounterMetricIsEmptyValue)
	})

	t.Run("update existing gauge_value returns new value", func(t *testing.T) {
		repo := newRepo()

		oldMetric := models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(2.5),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		newMetric := models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(25.0),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		err := repo.Save(ctx, oldMetric)
		require.NoError(t, err)

		err = repo.Save(ctx, newMetric)
		require.NoError(t, err)

		got, err := repo.Get(ctx, newMetric.ID, models.Gauge)
		require.NoError(t, err)
		assert.Equal(t, 25.0, *got.Value)
	})

	t.Run("update existing counter returns sum_old_and_new_delta", func(t *testing.T) {
		repo := newRepo()

		oldMetric := models.Metrics{
			ID:    "PullCount",
			MType: models.Counter,
			Delta: func(v int64) *int64 { return &v }(50),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		newMetric := models.Metrics{
			ID:    "PullCount",
			MType: models.Counter,
			Delta: func(v int64) *int64 { return &v }(100),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		err := repo.Save(ctx, oldMetric)
		require.NoError(t, err)

		err = repo.Save(ctx, newMetric)
		require.NoError(t, err)

		got, err := repo.Get(ctx, newMetric.ID, models.Counter)
		require.NoError(t, err)
		assert.Equal(t, int64(150), *got.Delta)
	})

	t.Run("unknown metric type returns error", func(t *testing.T) {
		repo := newRepo()

		metric := models.Metrics{
			ID:    "Unknown",
			MType: "unknown_type",
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		err := repo.Save(ctx, metric)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMetricType)
	})

	t.Run("save with canceled context returns error", func(t *testing.T) {
		repo := newRepo()

		childCtx, cancel := context.WithCancel(ctx)
		cancel()

		metric := models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(1.0),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}

		err := repo.Save(childCtx, metric)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("save batch with mixed metrics", func(t *testing.T) {
		repo := newRepo()

		batch := []models.Metrics{
			{
				ID:    "Alloc",
				MType: models.Gauge,
				Value: func(v float64) *float64 { return &v }(1.5),
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
			{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: func(v int64) *int64 { return &v }(10),
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
			{
				ID:    "HeapInuse",
				MType: models.Gauge,
				Value: func(v float64) *float64 { return &v }(42.0),
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		}

		require.NoError(t, repo.SaveBatch(ctx, batch))

		// проверяем, что все три метрики сохранились
		got, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Len(t, got, 3)
	})

	t.Run("save empty batch does nothing", func(t *testing.T) {
		repo := newRepo()

		err := repo.SaveBatch(ctx, []models.Metrics{})
		require.NoError(t, err)

		metrics, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, metrics)
	})

	t.Run("save batch with counter accumulates delta", func(t *testing.T) {
		repo := newRepo()

		// сначала сохраняем начальное значение
		require.NoError(t, repo.Save(ctx, models.Metrics{
			ID:    "PollCount",
			MType: models.Counter,
			Delta: func(v int64) *int64 { return &v }(50),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}))

		// батчем добавляем ещё
		batch := []models.Metrics{
			{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: func(v int64) *int64 { return &v }(30),
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
			{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: func(v int64) *int64 { return &v }(20),
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		}
		require.NoError(t, repo.SaveBatch(ctx, batch))

		got, err := repo.Get(ctx, "PollCount", models.Counter)
		require.NoError(t, err)
		assert.Equal(t, int64(100), *got.Delta)
	})

	t.Run("save batch with gauge overwrites value", func(t *testing.T) {
		repo := newRepo()

		require.NoError(t, repo.Save(ctx, models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(10.0),
			Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}))

		batch := []models.Metrics{
			{
				ID:    "Alloc",
				MType: models.Gauge,
				Value: func(v float64) *float64 { return &v }(99.9),
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		}
		require.NoError(t, repo.SaveBatch(ctx, batch))

		got, err := repo.Get(ctx, "Alloc", models.Gauge)
		require.NoError(t, err)
		assert.Equal(t, 99.9, *got.Value)
	})

	t.Run("save batch with invalid metric rolls back", func(t *testing.T) {
		repo := newRepo()

		batch := []models.Metrics{
			{
				ID:    "ValidGauge",
				MType: models.Gauge,
				Value: func(v float64) *float64 { return &v }(1.0),
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
			{
				ID:    "InvalidGauge",
				MType: models.Gauge,
				Value: nil,
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		}

		err := repo.SaveBatch(ctx, batch)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrGaugeMetricIsEmptyValue)

		// валидная метрика не должна сохраниться — транзакция откатилась
		metrics, listErr := repo.List(ctx)
		require.NoError(t, listErr)
		assert.Empty(t, metrics)
	})

	t.Run("save batch with unknown metric type rolls back", func(t *testing.T) {
		repo := newRepo()

		batch := []models.Metrics{
			{
				ID:    "Alloc",
				MType: models.Gauge,
				Value: func(v float64) *float64 { return &v }(1.0),
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
			{
				ID:    "Unknown",
				MType: "unknown_type",
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		}

		err := repo.SaveBatch(ctx, batch)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMetricType)

		metrics, listErr := repo.List(ctx)
		require.NoError(t, listErr)
		assert.Empty(t, metrics)
	})

	t.Run("save batch with canceled context returns error", func(t *testing.T) {
		repo := newRepo()

		childCtx, cancel := context.WithCancel(ctx)
		cancel()

		batch := []models.Metrics{
			{
				ID:    "Alloc",
				MType: models.Gauge,
				Value: func(v float64) *float64 { return &v }(1.0),
				Hash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		}

		err := repo.SaveBatch(childCtx, batch)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}
