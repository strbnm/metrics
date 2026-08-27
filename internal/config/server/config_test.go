package server

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerConfig(t *testing.T) {
	cfg := NewServerConfig()

	assert.Equal(t, "localhost:8080", cfg.RunAddr, "RunAddr должен быть 'localhost:8080' по умолчанию")
}

func TestValidateRunAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "Valid localhost with port",
			addr:    "localhost:8080",
			wantErr: false,
		},
		{
			name:    "Valid IP with port",
			addr:    "127.0.0.1:9000",
			wantErr: false,
		},
		{
			name:    "Valid domain with port",
			addr:    "example.com:443",
			wantErr: false,
		},
		{
			name:    "Empty address",
			addr:    "",
			wantErr: true,
		},
		{
			name:    "Address without port",
			addr:    "localhost",
			wantErr: true,
		},
		{
			name:    "Missing port",
			addr:    "localhost:",
			wantErr: true,
		},
		{
			name:    "Invalid format (no colon)",
			addr:    "localhost8080",
			wantErr: true,
		},
		{
			name:    "Invalid port",
			addr:    "localhost:invalid",
			wantErr: true,
		},
		{
			name:    "Only port",
			addr:    ":8080",
			wantErr: false, // net.SplitHostPort допускает такой формат
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRunAddr(tt.addr)

			if tt.wantErr {
				require.Error(t, err, "Должна быть ошибка валидации для '%s'", tt.addr)
			} else {
				require.NoError(t, err, "Не должно быть ошибок валидации для '%s'", tt.addr)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }() // Восстанавливаем аргументы после теста

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		expected *Config
	}{
		{
			name:    "Default values",
			args:    []string{"cmd"},
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
			},
		},
		{
			name:    "Custom address",
			args:    []string{"cmd", "-a", "test:9090"},
			wantErr: false,
			expected: &Config{
				RunAddr: "test:9090",
			},
		},
		{
			name:    "IP address",
			args:    []string{"cmd", "-a", "192.168.1.100:8080"},
			wantErr: false,
			expected: &Config{
				RunAddr: "192.168.1.100:8080",
			},
		},
		{
			name:    "Empty address (invalid)",
			args:    []string{"cmd", "-a", ""},
			wantErr: true,
		},
		{
			name:    "Address without port (invalid)",
			args:    []string{"cmd", "-a", "localhost"},
			wantErr: true,
		},
		{
			name:    "Invalid format (no colon)",
			args:    []string{"cmd", "-a", "localhost8080"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ОЧИЩАЕМ ФЛАГИ ПЕРЕД КАЖДЫМ ТЕСТОМ
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			os.Args = tt.args
			cfg, err := ParseFlags()

			if tt.wantErr {
				require.Error(t, err, "ParseFlags() должна возвращать ошибку")
			} else {
				require.NoError(t, err, "ParseFlags() не должна возвращать ошибку")
				assert.Equal(t, tt.expected.RunAddr, cfg.RunAddr,
					"RunAddr должно совпадать с ожидаемым значением")
			}
		})
	}
}
