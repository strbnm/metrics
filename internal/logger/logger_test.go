package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitialize_ValidLevels(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"}

	for _, lvl := range validLevels {
		t.Run(lvl, func(t *testing.T) {
			logger, err := Initialize(lvl)
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestInitialize_InvalidLevel(t *testing.T) {
	testLogger, err := Initialize("invalid-level")
	require.Error(t, err)
	require.Nil(t, testLogger)

	// Проверяем, что ошибка понятная (содержит имя уровня)
	require.Contains(t, err.Error(), "invalid-level")
}

// Тест для ветки Development (APP_ENV=development)
func TestInitialize_DevelopmentMode(t *testing.T) {
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)
	os.Setenv("APP_ENV", "development")

	logger, err := Initialize("info")
	require.NoError(t, err)
	require.NotNil(t, logger)
}

// Тест для ветки Production (APP_ENV != development)
func TestInitialize_ProductionMode(t *testing.T) {
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)
	os.Setenv("APP_ENV", "production")

	logger, err := Initialize("info")
	require.NoError(t, err)
	require.NotNil(t, logger)
}

func TestSetupLogger_Success(t *testing.T) {
	logger, cleanup, err := SetupLogger("info")

	require.NoError(t, err)
	require.NotNil(t, logger)
	require.NotNil(t, cleanup)

	cleanup()
}

func TestSetupLogger_ErrorPropagation(t *testing.T) {
	// Эмулируем ошибку из Initialize, передав неверный уровень
	_, cleanup, err := SetupLogger("not-a-real-level")

	require.Error(t, err)
	require.Nil(t, cleanup) // При ошибке cleanup не должен создаваться
}
