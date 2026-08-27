package repository

import (
	"testing"

	"github.com/strbnm/metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemStorage(t *testing.T) {
	t.Run("positive - create new memory storage", func(t *testing.T) {
		storage := NewMemStorage()

		require.NotNil(t, storage)
		require.NotNil(t, storage.metrics)
		assert.Empty(t, storage.metrics, "new storage should have empty metrics map")
	})
}

// TestMemStorage_Contract — запускаем тесты контракта Repository для MemStorage
func TestMemStorage_Contract(t *testing.T) {
	service.testRepositoryContract(t, func() service.Repository {
		return NewMemStorage()
	})
}
