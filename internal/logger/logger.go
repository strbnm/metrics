package logger

import (
	"os"

	"go.uber.org/zap"
)

// Initialize инициализирует логгер с необходимым уровнем логирования.
func Initialize(level string) (*zap.SugaredLogger, error) {

	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	log := zl.Sugar()
	return log, nil
}

func SetupLogger(level string) (*zap.SugaredLogger, func(), error) {
	logger, err := Initialize(level)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = logger.Sync()
	}

	return logger, cleanup, nil
}
