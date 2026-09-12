package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

var Log *zap.SugaredLogger = zap.NewNop().Sugar()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {

	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	var cfg zap.Config
	if os.Getenv("APP_ENV") == "development" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zl.Sugar()
	return nil
}

func SetupLogger(level string) func() {
	if err := Initialize(level); err != nil {
		panic(fmt.Sprintf("Initializing logger failed: %v", err))
	}

	return func() {
		_ = Log.Sync()
	}
}
