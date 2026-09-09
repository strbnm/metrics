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
	originalEnv, originalEnvSet := os.LookupEnv("ADDRESS")

	// Восстановление исходного состояния после всех тестов
	defer func() {
		os.Args = originalArgs
		if originalEnvSet {
			os.Setenv("ADDRESS", originalEnv)
		} else {
			os.Unsetenv("ADDRESS")
		}
	}()

	tests := []struct {
		name         string
		args         []string
		envValue     string
		envSet       bool
		wantErr      bool
		expectedAddr string
	}{
		{
			name:         "Default values (no ENV, no flag)",
			args:         []string{"cmd"},
			envSet:       false,
			wantErr:      false,
			expectedAddr: "localhost:8080",
		},
		{
			name:         "Flag overrides default (no ENV)",
			args:         []string{"cmd", "-a", "custom:9090"},
			envSet:       false,
			wantErr:      false,
			expectedAddr: "custom:9090",
		},
		{
			name:         "ENV overrides flag",
			args:         []string{"cmd", "-a", "ignored:8000"},
			envValue:     "env-priority:9999",
			envSet:       true,
			wantErr:      false,
			expectedAddr: "env-priority:9999",
		},
		{
			name:         "Only ENV (no flag)",
			args:         []string{"cmd"},
			envValue:     "only-env:5000",
			envSet:       true,
			wantErr:      false,
			expectedAddr: "only-env:5000",
		},
		{
			name:         "Empty ENV string (ignored, flag used)",
			args:         []string{"cmd", "-a", "flag-value:7000"},
			envValue:     "",
			envSet:       true, // ADDRESS="" — установлена, но пустая
			wantErr:      false,
			expectedAddr: "flag-value:7000",
		},
		{
			name:         "Empty ENV string (ignored, default used)",
			args:         []string{"cmd"},
			envValue:     "",
			envSet:       true,
			wantErr:      false,
			expectedAddr: "localhost:8080",
		},
		{
			name:         "Invalid ENV format (validation fails)",
			args:         []string{"cmd"},
			envValue:     "invalid-no-colon",
			envSet:       true,
			wantErr:      true,
			expectedAddr: "",
		},
		{
			name:         "Invalid flag format, but valid ENV",
			args:         []string{"cmd", "-a", "invalid"},
			envValue:     "valid-env:8888",
			envSet:       true,
			wantErr:      false,
			expectedAddr: "valid-env:8888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сброс флагов перед каждым тестом
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Установка переменной окружения
			if tt.envSet {
				os.Setenv("ADDRESS", tt.envValue)
			} else {
				os.Unsetenv("ADDRESS")
			}

			os.Args = tt.args

			cfg, err := ParseFlags()

			if tt.wantErr {
				require.Error(t, err, "ParseFlags() должна возвращать ошибку")
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err, "ParseFlags() не должна возвращать ошибку")
				require.NotNil(t, cfg)
				assert.Equal(t, tt.expectedAddr, cfg.RunAddr,
					"RunAddr должно совпадать с ожидаемым")
			}
		})
	}
}
