package server

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type StoreConfig struct {
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
}

type Config struct {
	RunAddr string
	Store   StoreConfig
}

// NewServerConfig создаёт конфигурацию со значениями по умолчанию
func NewServerConfig() *Config {
	return &Config{
		RunAddr: "localhost:8080",
		Store: StoreConfig{
			StoreInterval:   300 * time.Second,
			FileStoragePath: "metrics.json",
			Restore:         true,
		},
	}
}

// ParseFlags обрабатывает аргументы командной строки и возвращает конфигурацию
func ParseFlags() (*Config, error) {
	cfg := NewServerConfig()

	// Регистрируем флаги
	flag.StringVar(&cfg.RunAddr, "a", cfg.RunAddr, "address and port to run server with format host:port")
	flag.StringVar(&cfg.Store.FileStoragePath, "f", cfg.Store.FileStoragePath, "file path to store metrics")
	flag.BoolVar(&cfg.Store.Restore, "r", cfg.Store.Restore, "restore metrics from disk storage")
	storeInterval := flag.Int64("i", int64(cfg.Store.StoreInterval/time.Second),
		"time interval between write data on disk in seconds")

	flag.Parse()

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok && envAddr != "" {
		cfg.RunAddr = envAddr
	}
	if envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok && envStoreInterval != "" {
		value, err := strconv.ParseInt(envStoreInterval, 10, 64)
		if err != nil {
			fmt.Printf("invalid value for system environment STORE_INTERVAL: %s. Will use default value or value of cmd argument -i if present.\n", envStoreInterval)
		} else {
			storeInterval = &value
		}
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok && envFileStoragePath != "" {
		cfg.Store.FileStoragePath = envFileStoragePath
	}

	if envRestore, ok := os.LookupEnv("RESTORE"); ok && envRestore != "" {
		envRestore = strings.TrimSpace(envRestore)
		if envRestore == "true" {
			cfg.Store.Restore = true
		}
	}

	// Валидируем RunAddr
	if err := validateRunAddr(cfg.RunAddr); err != nil {
		return nil, err
	}

	// Валидируем Store.StoreInterval
	if err := validateAndApplyInterval(cfg, *storeInterval); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateRunAddr(addr string) error {
	if addr == "" {
		return fmt.Errorf("host:port cannot be empty")
	}

	if !strings.Contains(addr, ":") {
		return fmt.Errorf("invalid format: expected host:port, got '%s'", addr)
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid host:port '%s': %w", addr, err)
	}

	if port == "" {
		return fmt.Errorf("port is missing in '%s'", addr)
	}
	if _, err2 := strconv.Atoi(port); err2 != nil {
		return fmt.Errorf("port must be integer number, got '%s'", addr)
	}

	return nil
}

// applyAndValidateIntervals применяет значения интервалов и проверяет их корректность
func validateAndApplyInterval(cfg *Config, intervalSec int64) error {
	if intervalSec < 0 {
		return fmt.Errorf("time interval cannot be negative")
	}

	cfg.Store.StoreInterval = time.Duration(intervalSec) * time.Second

	return nil
}
