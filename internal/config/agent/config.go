package agent

import (
	"flag"
	"fmt"
	"time"
)

type CollectorConfig struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
}

type ServerConfig struct {
	Schema     string
	ServerAddr string
}

func (cfg *ServerConfig) ServerURL() string {
	return cfg.Schema + "://" + cfg.ServerAddr
}

type Config struct {
	ServerConfig    ServerConfig
	CollectorConfig CollectorConfig
}

// NewAgentConfig создаёт конфигурацию со значениями по умолчанию
func NewAgentConfig() *Config {
	return &Config{
		ServerConfig: ServerConfig{
			Schema:     "http",
			ServerAddr: "localhost:8080",
		},
		CollectorConfig: CollectorConfig{
			PollInterval:   2 * time.Second,
			ReportInterval: 10 * time.Second,
		},
	}
}

// ParseFlags обрабатывает аргументы командной строки и возвращает конфигурацию
func ParseFlags() (*Config, error) {
	cfg := NewAgentConfig()

	// Регистрируем флаги
	flag.StringVar(&cfg.ServerConfig.ServerAddr, "a", cfg.ServerConfig.ServerAddr, "address and port to run server")

	pollInterval := flag.Int64("p", int64(cfg.CollectorConfig.PollInterval/time.Second),
		"time interval between polls in seconds")
	reportInterval := flag.Int64("r", int64(cfg.CollectorConfig.ReportInterval/time.Second),
		"time interval between reports in seconds")

	flag.Parse()

	// Применяем и валидируем значения
	if err := applyAndValidateIntervals(cfg, *pollInterval, *reportInterval); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyAndValidateIntervals применяет значения интервалов и проверяет их корректность
func applyAndValidateIntervals(cfg *Config, pollSec, reportSec int64) error {
	if pollSec < 0 {
		return fmt.Errorf("poll interval cannot be negative")
	}
	if reportSec < 0 {
		return fmt.Errorf("report interval cannot be negative")
	}

	cfg.CollectorConfig.PollInterval = time.Duration(pollSec) * time.Second
	cfg.CollectorConfig.ReportInterval = time.Duration(reportSec) * time.Second

	return nil
}
