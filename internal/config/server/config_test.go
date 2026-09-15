package server

import (
	"flag"
	"os"
	"testing"
	"time"

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
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name     string
		args     []string
		env      map[string]string
		wantErr  bool
		expected *Config
	}{
		{
			name:    "Default values (no args, no env)",
			args:    []string{"cmd"},
			env:     nil,
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   300 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name:    "Custom server address via flag (env not set)",
			args:    []string{"cmd", "-a", "test:9090"},
			env:     nil,
			wantErr: false,
			expected: &Config{
				RunAddr: "test:9090",
				Store: StoreConfig{
					StoreInterval:   300 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name:    "Custom store interval via flags (env not set)",
			args:    []string{"cmd", "-i", "5"},
			env:     nil,
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   5 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name:    "Custom file path via flags (env not set)",
			args:    []string{"cmd", "-f", "test_metrics.json"},
			env:     nil,
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   300 * time.Second,
					FileStoragePath: "test_metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name:    "Custom restore flag (true) via flags (env not set)",
			args:    []string{"cmd", "-r=true"},
			env:     nil,
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   300 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name:    "Custom restore flag (false) via flags (env not set)",
			args:    []string{"cmd", "-r=false"},
			env:     nil,
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   300 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         false,
				},
			},
		},
		{
			name: "Env overrides flag for server address",
			args: []string{"cmd", "-a", "flag-addr:8000"},
			env: map[string]string{
				"ADDRESS": "env-addr:9000",
			},
			wantErr: false,
			expected: &Config{
				RunAddr: "env-addr:9000",
				Store: StoreConfig{
					StoreInterval:   300 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name: "Env overrides flags for store interval",
			args: []string{"cmd", "-i", "10"},
			env: map[string]string{
				"STORE_INTERVAL": "30",
			},
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   30 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name: "Env overrides flags for file storage path",
			args: []string{"cmd", "-f", "flag_metrics.json"},
			env: map[string]string{
				"FILE_STORAGE_PATH": "env_metrics.json",
			},
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   300 * time.Second,
					FileStoragePath: "env_metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name: "Env overrides flags for restore",
			args: []string{"cmd", "-r=true"},
			env: map[string]string{
				"RESTORE": "false",
			},
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   300 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         false,
				},
			},
		},
		{
			name: "Env overrides flags for restore",
			args: []string{"cmd", "-r=false"},
			env: map[string]string{
				"RESTORE": "true",
			},
			wantErr: false,
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   300 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name: "Invalid env value for STORE_INTERVAL (fallback to flag)",
			args: []string{"cmd", "-i", "10"},
			env: map[string]string{
				"STORE_INTERVAL": "not-a-number",
			},
			wantErr: false, // ошибка в env не должна ломать парсинг — используется флаг
			expected: &Config{
				RunAddr: "localhost:8080",
				Store: StoreConfig{
					StoreInterval:   10 * time.Second,
					FileStoragePath: "metrics.json",
					Restore:         true,
				},
			},
		},
		{
			name: "Invalid env value for STORE_INTERVAL (fallback to default)",
			args: []string{"cmd"}, // нет флага -i
			env: map[string]string{
				"STORE_INTERVAL": "-5", // отрицательное значение, но сначала проверим парсинг
			},
			wantErr: true, // applyAndValidateIntervals должен вернуть ошибку из-за отрицательного значения
		},
		{
			name:    "Negative interval via flag (validation fails)",
			args:    []string{"cmd", "-i", "-1"},
			env:     nil,
			wantErr: true,
		},
		{
			name: "Empty env value ignored (fallback to flag)",
			args: []string{"cmd", "-a", "flags:9090", "-i", "15", "-f", "test_metrics.json", "-r=false"},
			env: map[string]string{
				"ADDRESS":           "", // пустая строка — игнорируется
				"STORE_INTERVAL":    "", // пустая строка — игнорируется
				"RESTORE":           "", // пустая строка — игнорируется
				"FILE_STORAGE_PATH": "", // пустая строка — игнорируется
			},
			wantErr: false,
			expected: &Config{
				RunAddr: "flags:9090",
				Store: StoreConfig{
					StoreInterval:   15 * time.Second,
					FileStoragePath: "test_metrics.json",
					Restore:         false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalEnv := make(map[string]*string)
			for _, key := range []string{"ADDRESS", "STORE_INTERVAL", "FILE_STORAGE_PATH", "RESTORE"} {
				if val, ok := os.LookupEnv(key); ok {
					originalEnv[key] = &val
				} else {
					originalEnv[key] = nil
				}
			}

			defer func() {
				for key, valPtr := range originalEnv {
					if valPtr == nil {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, *valPtr)
					}
				}
			}()

			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			if tt.env != nil {
				for k, v := range tt.env {
					os.Setenv(k, v)
				}
			}

			os.Args = tt.args

			cfg, err := ParseFlags()

			if tt.wantErr {
				require.Error(t, err, "ParseFlags() должна возвращать ошибку")
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err, "ParseFlags() не должна возвращать ошибку")
				require.NotNil(t, cfg)
				assert.Equal(t, tt.expected.RunAddr, cfg.RunAddr,
					"RunAddr должно совпадать с ожидаемым")
				assert.Equal(t, tt.expected.Store.StoreInterval, cfg.Store.StoreInterval,
					"Store.StoreInterval должно совпадать с ожидаемым")
				assert.Equal(t, tt.expected.Store.FileStoragePath, cfg.Store.FileStoragePath,
					"Store.FileStoragePath должно совпадать с ожидаемым")
				assert.Equal(t, tt.expected.Store.Restore, cfg.Store.Restore,
					"Store.Restore должен совпадать с ожидаемым")
			}
		})
	}
}
