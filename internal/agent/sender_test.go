package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "github.com/strbnm/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSender(t *testing.T) {
	tests := []struct {
		name      string
		serverURL string
		wantErr   bool
		errMsg    string
		checkFunc func(t *testing.T, sender *Sender, err error)
	}{
		{
			name:      "positive test #1 - valid URL",
			serverURL: "http://localhost:8080",
			wantErr:   false,
			checkFunc: func(t *testing.T, sender *Sender, err error) {
				require.NoError(t, err)
				assert.NotNil(t, sender)
				assert.Equal(t, "http://localhost:8080", sender.client.BaseURL)
				assert.NotNil(t, sender.client)
			},
		},
		{
			name:      "negative test #1 - empty URL",
			serverURL: "",
			wantErr:   true,
			errMsg:    "serverURL cannot be empty",
			checkFunc: func(t *testing.T, sender *Sender, err error) {
				require.Error(t, err)
				assert.Nil(t, sender)
				assert.Contains(t, err.Error(), "serverURL cannot be empty")
			},
		},
		{
			name:      "positive test #2 - URL with spaces (allowed)",
			serverURL: " http://localhost:8080 ",
			wantErr:   false,
			checkFunc: func(t *testing.T, sender *Sender, err error) {
				require.NoError(t, err)
				assert.NotNil(t, sender)
				// Конструктор сохраняет строку как есть, включая пробелы
				assert.Equal(t, " http://localhost:8080 ", sender.client.BaseURL)
				assert.NotNil(t, sender.client)
			},
		},
		{
			name:      "positive test #3 - very long URL",
			serverURL: strings.Repeat("a", 1000) + "://example.com",
			wantErr:   false,
			checkFunc: func(t *testing.T, sender *Sender, err error) {
				require.NoError(t, err)
				assert.NotNil(t, sender)
				assert.Len(t, sender.client.BaseURL, 1014) // 1000 'a' + 14 символов
				assert.NotNil(t, sender.client)
			},
		},
		{
			name:      "positive test #4 - special characters in URL",
			serverURL: "http://user:pass@localhost:8080/path?query=value#fragment",
			wantErr:   false,
			checkFunc: func(t *testing.T, sender *Sender, err error) {
				require.NoError(t, err)
				assert.NotNil(t, sender)
				assert.Equal(t, "http://user:pass@localhost:8080/path?query=value#fragment", sender.client.BaseURL)
				assert.NotNil(t, sender.client)
			},
		},
		{
			name:      "positive test #5 - IP address URL",
			serverURL: "http://192.168.1.1:8080",
			wantErr:   false,
			checkFunc: func(t *testing.T, sender *Sender, err error) {
				require.NoError(t, err)
				assert.NotNil(t, sender)
				assert.Equal(t, "http://192.168.1.1:8080", sender.client.BaseURL)
				assert.NotNil(t, sender.client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender, err := NewSender(tt.serverURL)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, sender)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, sender)
			}

			// Выполняем дополнительные проверки, определённые в checkFunc
			tt.checkFunc(t, sender, err)
		})
	}
}

func TestSender_Send(t *testing.T) {
	tests := []struct {
		name          string
		metrics       []models.Metrics
		serverHandler http.Handler
		wantErr       bool
		errContains   string
	}{
		{
			name: "positive test #1 - send single gauge metric",
			metrics: []models.Metrics{
				{
					ID:    "Alloc",
					MType: models.Gauge,
					Value: func(v float64) *float64 { return &v }(1.0001),
				},
			},
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/update", r.URL.Path)
				assert.Equal(t, "application/json; charset=utf-8", r.Header.Get("Content-Type"))

				var received models.Metrics
				// Читаем и парсим сразу
				err := json.NewDecoder(r.Body).Decode(&received)
				require.NoError(t, err)
				defer r.Body.Close()

				assert.Equal(t, models.Gauge, received.MType)
				assert.Equal(t, 1.0001, *received.Value)
				assert.Equal(t, "Alloc", received.ID)

				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "OK")
			}),
			wantErr: false,
		},
		{
			name: "positive test #2 - send multiple metrics",
			metrics: []models.Metrics{
				{
					ID:    "Alloc",
					MType: models.Gauge,
					Value: func(v float64) *float64 { return &v }(1.0001),
				},
				{
					ID:    "PollCount",
					MType: models.Counter,
					Delta: func(v int64) *int64 { return &v }(100),
				},
			},
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/update", r.URL.Path)
				assert.Equal(t, "application/json; charset=utf-8", r.Header.Get("Content-Type"))

				var received models.Metrics
				// Читаем и парсим сразу
				err := json.NewDecoder(r.Body).Decode(&received)
				require.NoError(t, err)
				defer r.Body.Close()

				switch received.MType {
				case models.Gauge:
					assert.Equal(t, models.Gauge, received.MType)
					assert.Equal(t, 1.0001, *received.Value)
					assert.Equal(t, "Alloc", received.ID)

					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, "OK")

				case models.Counter:
					assert.Equal(t, models.Counter, received.MType)
					assert.Equal(t, int64(100), *received.Delta)
					assert.Equal(t, "PollCount", received.ID)

					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, "OK")
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}),
			wantErr: false,
		},
		{
			name: "negative test #2 - server returns error status",
			metrics: []models.Metrics{
				{
					ID:    "ErrorMetric",
					MType: models.Gauge,
					Value: func(v float64) *float64 { return &v }(1.0),
				},
			},
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}),
			wantErr:     true,
			errContains: "server returned status: 500",
		},
		{
			name: "negative test #3 - http client error",
			metrics: []models.Metrics{
				{
					ID:    "ClientError",
					MType: models.Gauge,
					Value: func(v float64) *float64 { return &v }(1.0),
				},
			},
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Client error", http.StatusBadRequest)
			}),
			wantErr:     true,
			errContains: "failed to send metric ClientError",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(test.serverHandler)
			defer server.Close()

			sender, err := NewSender(server.URL)
			assert.NoError(t, err)

			sendErr := sender.Send(test.metrics)

			if test.wantErr {
				require.Error(t, sendErr)
				assert.Contains(t, sendErr.Error(), test.errContains)
			} else {
				require.NoError(t, sendErr)
			}
		})
	}
}
