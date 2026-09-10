package server

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	RunAddr string
}

// NewServerConfig создаёт конфигурацию со значениями по умолчанию
func NewServerConfig() *Config {
	return &Config{
		RunAddr: "localhost:8080",
	}
}

// ParseFlags обрабатывает аргументы командной строки и возвращает конфигурацию
func ParseFlags() (*Config, error) {
	cfg := NewServerConfig()

	// Регистрируем флаги
	flag.StringVar(&cfg.RunAddr, "a", cfg.RunAddr, "address and port to run server with format host:port")

	flag.Parse()

	if envAddr, ok := os.LookupEnv("ADDRESS"); ok && envAddr != "" {
		cfg.RunAddr = envAddr
	}

	// Валидируем RunAddr
	if err := validateRunAddr(cfg.RunAddr); err != nil {
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
