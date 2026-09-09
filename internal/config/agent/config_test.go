package agent

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentConfig(t *testing.T) {
	cfg := NewAgentConfig()

	// Проверяем значения по умолчанию с помощью assert
	assert.Equal(t, "http", cfg.ServerConfig.Schema, "Schema должно быть 'http'")
	assert.Equal(t, "localhost:8080", cfg.ServerConfig.ServerAddr, "ServerAddr должно быть 'localhost:8080'")
	assert.Equal(t, 2*time.Second, cfg.CollectorConfig.PollInterval, "PollInterval должно быть 2s")
	assert.Equal(t, 10*time.Second, cfg.CollectorConfig.ReportInterval, "ReportInterval должно быть 10s")
}

func TestServerURL(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		addr   string
		want   string
	}{
		{
			name:   "HTTP with localhost",
			schema: "http",
			addr:   "localhost:8080",
			want:   "http://localhost:8080",
		},
		{
			name:   "HTTPS with domain",
			schema: "https",
			addr:   "example.com:443",
			want:   "https://example.com:443",
		},
		{
			name:   "Custom schema",
			schema: "custom",
			addr:   "service:9000",
			want:   "custom://service:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ServerConfig{
				Schema:     tt.schema,
				ServerAddr: tt.addr,
			}
			got := cfg.ServerURL()
			assert.Equal(t, tt.want, got, "ServerURL() должно возвращать ожидаемый URL")
		})
	}
}

func TestApplyAndValidateIntervals(t *testing.T) {
	baseCfg := NewAgentConfig()

	tests := []struct {
		name           string
		pollSec        int64
		reportSec      int64
		wantErr        bool
		expectedPoll   time.Duration
		expectedReport time.Duration
	}{
		{
			name:           "Valid positive values",
			pollSec:        5,
			reportSec:      15,
			wantErr:        false,
			expectedPoll:   5 * time.Second,
			expectedReport: 15 * time.Second,
		},
		{
			name:           "Zero values",
			pollSec:        0,
			reportSec:      0,
			wantErr:        false,
			expectedPoll:   0 * time.Second,
			expectedReport: 0 * time.Second,
		},
		{
			name:      "Negative poll interval",
			pollSec:   -1,
			reportSec: 10,
			wantErr:   true,
		},
		{
			name:      "Negative report interval",
			pollSec:   5,
			reportSec: -5,
			wantErr:   true,
		},
		{
			name:      "Both negative",
			pollSec:   -2,
			reportSec: -3,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseCfg
			err := applyAndValidateIntervals(cfg, tt.pollSec, tt.reportSec)

			if tt.wantErr {
				require.Error(t, err, "Должна быть ошибка при отрицательных интервалах")
			} else {
				require.NoError(t, err, "Не должно быть ошибок при корректных интервалах")
				assert.Equal(t, tt.expectedPoll, cfg.CollectorConfig.PollInterval,
					"PollInterval должно быть установлено корректно")
				assert.Equal(t, tt.expectedReport, cfg.CollectorConfig.ReportInterval,
					"ReportInterval должно быть установлено корректно")
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
				ServerConfig: ServerConfig{
					Schema:     "http",
					ServerAddr: "localhost:8080",
				},
				CollectorConfig: CollectorConfig{
					PollInterval:   2 * time.Second,
					ReportInterval: 10 * time.Second,
				},
			},
		},
		{
			name:    "Custom server address via flag (env not set)",
			args:    []string{"cmd", "-a", "test:9090"},
			env:     nil,
			wantErr: false,
			expected: &Config{
				ServerConfig: ServerConfig{
					Schema:     "http",
					ServerAddr: "test:9090",
				},
				CollectorConfig: CollectorConfig{
					PollInterval:   2 * time.Second,
					ReportInterval: 10 * time.Second,
				},
			},
		},
		{
			name:    "Custom intervals via flags (env not set)",
			args:    []string{"cmd", "-p", "5", "-r", "20"},
			env:     nil,
			wantErr: false,
			expected: &Config{
				ServerConfig: ServerConfig{
					Schema:     "http",
					ServerAddr: "localhost:8080",
				},
				CollectorConfig: CollectorConfig{
					PollInterval:   5 * time.Second,
					ReportInterval: 20 * time.Second,
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
				ServerConfig: ServerConfig{
					Schema:     "http",
					ServerAddr: "env-addr:9000", // берётся из env
				},
				CollectorConfig: CollectorConfig{
					PollInterval:   2 * time.Second,
					ReportInterval: 10 * time.Second,
				},
			},
		},
		{
			name: "Env overrides flags for intervals",
			args: []string{"cmd", "-p", "1", "-r", "2"},
			env: map[string]string{
				"POLL_INTERVAL":   "30",
				"REPORT_INTERVAL": "60",
			},
			wantErr: false,
			expected: &Config{
				ServerConfig: ServerConfig{
					Schema:     "http",
					ServerAddr: "localhost:8080",
				},
				CollectorConfig: CollectorConfig{
					PollInterval:   30 * time.Second, // из env
					ReportInterval: 60 * time.Second, // из env
				},
			},
		},
		{
			name: "Invalid env value for POLL_INTERVAL and REPORT_INTERVAL (fallback to flag)",
			args: []string{"cmd", "-p", "10", "-r", "30"},
			env: map[string]string{
				"POLL_INTERVAL":   "not-a-number",
				"REPORT_INTERVAL": "not-a-number too",
			},
			wantErr: false, // ошибка в env не должна ломать парсинг — используется флаг
			expected: &Config{
				ServerConfig: ServerConfig{
					Schema:     "http",
					ServerAddr: "localhost:8080",
				},
				CollectorConfig: CollectorConfig{
					PollInterval:   10 * time.Second, // из флага
					ReportInterval: 30 * time.Second, // из флага
				},
			},
		},
		{
			name: "Invalid env value for REPORT_INTERVAL (fallback to default)",
			args: []string{"cmd"}, // нет флага -r
			env: map[string]string{
				"REPORT_INTERVAL": "-5", // отрицательное значение, но сначала проверим парсинг
			},
			wantErr: true, // applyAndValidateIntervals должен вернуть ошибку из-за отрицательного значения
		},
		{
			name:    "Negative interval via flag (validation fails)",
			args:    []string{"cmd", "-p", "-1"},
			env:     nil,
			wantErr: true,
		},
		{
			name: "Empty env value ignored (fallback to flag)",
			args: []string{"cmd", "-p", "15"},
			env: map[string]string{
				"POLL_INTERVAL": "", // пустая строка — игнорируется
			},
			wantErr: false,
			expected: &Config{
				ServerConfig: ServerConfig{
					Schema:     "http",
					ServerAddr: "localhost:8080",
				},
				CollectorConfig: CollectorConfig{
					PollInterval:   15 * time.Second, // из флага
					ReportInterval: 10 * time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			originalEnv := make(map[string]*string)
			for _, key := range []string{"ADDRESS", "POLL_INTERVAL", "REPORT_INTERVAL"} {
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
				return
			}

			require.NoError(t, err, "ParseFlags() не должна возвращать ошибку")

			// Проверяем ServerConfig
			assert.Equal(t, tt.expected.ServerConfig.Schema, cfg.ServerConfig.Schema,
				"ServerConfig.Schema должно совпадать с ожидаемым")
			assert.Equal(t, tt.expected.ServerConfig.ServerAddr, cfg.ServerConfig.ServerAddr,
				"ServerConfig.ServerAddr должно совпадать с ожидаемым")

			// Проверяем CollectorConfig
			assert.Equal(t, tt.expected.CollectorConfig.PollInterval, cfg.CollectorConfig.PollInterval,
				"CollectorConfig.PollInterval должно совпадать с ожидаемым")
			assert.Equal(t, tt.expected.CollectorConfig.ReportInterval, cfg.CollectorConfig.ReportInterval,
				"CollectorConfig.ReportInterval должно совпадать с ожидаемым")
		})
	}
}
