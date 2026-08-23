package repository

import (
	"testing"

	models "github.com/strbnm/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Базовый набор тест-кейсов для интерфейса Repository
func testRepositoryContract(t *testing.T, newRepo func() Repository) {
	t.Run("save and get gauge metric", func(t *testing.T) {
		repo := newRepo()

		metric := models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(1.0),
		}

		require.NoError(t, repo.Save(metric))

		got, err := repo.Get(metric.ID, metric.MType)
		require.NoError(t, err)
		assert.Equal(t, metric, got)
	})

	t.Run("save and get counter metric", func(t *testing.T) {
		repo := newRepo()

		metric := models.Metrics{
			ID:    "PollCount",
			MType: models.Counter,
			Delta: func(v int64) *int64 { return &v }(100),
		}

		require.NoError(t, repo.Save(metric))

		got, err := repo.Get(metric.ID, metric.MType)
		require.NoError(t, err)
		assert.Equal(t, metric, got)
	})

	t.Run("get non-existent metric returns error", func(t *testing.T) {
		repo := newRepo()

		_, err := repo.Get("NonExistent", models.Gauge)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metric not found")
	})

	t.Run("list empty storage", func(t *testing.T) {
		repo := newRepo()

		metrics, err := repo.List()
		require.NoError(t, err)
		assert.Empty(t, metrics)
	})

	t.Run("list with multiple metrics", func(t *testing.T) {
		repo := newRepo()

		gaugeMetric := models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(2.5),
		}
		counterMetric := models.Metrics{
			ID:    "Requests",
			MType: models.Counter,
			Delta: func(v int64) *int64 { return &v }(42),
		}

		require.NoError(t, repo.Save(gaugeMetric))
		require.NoError(t, repo.Save(counterMetric))

		metrics, err := repo.List()
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
		}

		err := repo.Save(metric)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gauge metric must have a value")
	})

	t.Run("counter without delta returns error", func(t *testing.T) {
		repo := newRepo()

		metric := models.Metrics{
			ID:    "InvalidCounter",
			MType: models.Counter,
			Delta: nil,
		}

		err := repo.Save(metric)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "counter metric must have a delta")
	})

	t.Run("update existing gauge_value returns new value", func(t *testing.T) {
		repo := newRepo()

		old_metric := models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(2.5),
		}

		new_metric := models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: func(v float64) *float64 { return &v }(25.0),
		}

		err := repo.Save(old_metric)
		require.NoError(t, err)

		err = repo.Save(new_metric)
		require.NoError(t, err)

		got, err := repo.Get(new_metric.ID, models.Gauge)
		require.NoError(t, err)
		assert.Equal(t, 25.0, *got.Value)
	})

	t.Run("update existing counter returns sum_old_and_new_delta", func(t *testing.T) {
		repo := newRepo()

		old_metric := models.Metrics{
			ID:    "PullCount",
			MType: models.Counter,
			Delta: func(v int64) *int64 { return &v }(50),
		}

		new_metric := models.Metrics{
			ID:    "PullCount",
			MType: models.Counter,
			Delta: func(v int64) *int64 { return &v }(100),
		}

		err := repo.Save(old_metric)
		require.NoError(t, err)

		err = repo.Save(new_metric)
		require.NoError(t, err)

		got, err := repo.Get(new_metric.ID, models.Counter)
		require.NoError(t, err)
		assert.Equal(t, int64(150), *got.Delta)
	})

	t.Run("unknown metric type returns error", func(t *testing.T) {
		repo := newRepo()

		metric := models.Metrics{
			ID:    "Unknown",
			MType: "unknown_type",
		}

		err := repo.Save(metric)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown metric type")
	})
}
