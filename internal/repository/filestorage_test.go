package repository

import (
	"os"
	"testing"

	"github.com/strbnm/metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewFileStorage(t *testing.T) {
	t.Run("positive - create new memory storage", func(t *testing.T) {
		storage := NewMemStorage()

		require.NotNil(t, storage)
		require.NotNil(t, storage.metrics)
		assert.Empty(t, storage.metrics, "new storage should have empty metrics map")
	})
}

// TestFileStorage_Contract — тесты контракта Repository для FileStorage
func TestFileStorage_Contract(t *testing.T) {
	ctx := t.Context()
	testLogger := zap.NewNop().Sugar()

	setup := func(t *testing.T) string {
		t.Helper()
		f, err := os.CreateTemp("", "test_metrics_*.json")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		return f.Name()
	}

	cleanup := func(t *testing.T, path string) {
		t.Helper()
		_ = os.Remove(path)
	}

	testRepositoryContract(ctx, t, func() service.Repository {
		path := setup(t)
		t.Cleanup(func() { cleanup(t, path) })
		return NewFileStorage(path, true, true, testLogger)
	})
}
