package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInitialize(t *testing.T) {
	t.Cleanup(func() {
		Log = zap.NewNop().Sugar()
	})

	err := Initialize("info")

	require.NoError(t, err)
	require.NotNil(t, Log)
}

func TestInitialize_InvalidLevel(t *testing.T) {
	t.Cleanup(func() {
		Log = zap.NewNop().Sugar()
	})

	err := Initialize("invalid-level")

	require.Error(t, err)
}

func TestSetupLogger(t *testing.T) {
	t.Cleanup(func() {
		Log = zap.NewNop().Sugar()
	})

	cleanup := SetupLogger("info")

	require.NotNil(t, cleanup)
	require.NotNil(t, Log)

	cleanup()
}

func TestSetupLogger_InvalidLevel(t *testing.T) {
	require.Panics(t, func() {
		SetupLogger("invalid-level")
	})
}
