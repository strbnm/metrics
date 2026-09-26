package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/service"
	"github.com/strbnm/metrics/internal/service/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandler_UpdateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name       string
		url        string
		method     string
		expectSave bool
		want       want
	}{
		{
			name:       "positive test #1",
			url:        "/update/gauge/Alloc/1.0001",
			method:     http.MethodPost,
			expectSave: true,
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "positive test #2",
			url:        "/update/counter/PollCount/100",
			method:     http.MethodPost,
			expectSave: true,
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #3 - not allowed path",
			url:        "/create/counter/PollCount/100",
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #4 - not allowed metric type",
			url:        "/update/histogram/SomeHist/100",
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    "Invalid metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #5 - without metric name",
			url:        "/update/gauge/100",
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #6 - without metric value",
			url:        "/update/gauge/Alloc",
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #7 - invalid metric value",
			url:        "/update/gauge/Alloc/invalid_value",
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    "Invalid metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #8 - invalid metric value",
			url:        "/update/counter/PullCount/invalid_value",
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    "Invalid metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #9 - invalid method",
			url:        "/update/counter/PullCount/invalid_value",
			method:     http.MethodGet,
			expectSave: false,
			want: want{
				code:        405,
				response:    "", //chi не возвращает стандартные ответы для 405
				contentType: "", //chi не возвращает стандартные ответы для 405
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mock.NewMockRepository(ctrl)

			if test.expectSave {
				mockRepo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(nil)
			}

			request := httptest.NewRequest(test.method, test.url, nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()
			testLogger := zap.NewNop().Sugar()
			srv := service.NewMetricsService(mockRepo, testLogger)
			h := NewHandler(srv, nil, testLogger)

			r := chi.NewRouter()
			r.Route("/", func(r chi.Router) {
				r.Get("/", h.ListAllMetricsHandler)
				r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
				r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
			})

			r.ServeHTTP(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.Equal(t, test.want.response, string(resBody))
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestHandler_ValueHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name    string
		url     string
		method  string
		want    want
		getFunc func(m *mock.MockRepository)
	}{
		{
			name:   "positive test #1",
			url:    "/value/gauge/Alloc",
			method: http.MethodGet,
			want: want{
				code:        200,
				response:    "1.000001",
				contentType: "text/plain; charset=utf-8",
			},
			getFunc: func(m *mock.MockRepository) {
				val := 1.000001
				m.EXPECT().
					Get(gomock.Any(), "Alloc", models.Gauge).
					Return(models.Metrics{
						ID:    "Alloc",
						MType: models.Gauge,
						Value: &val,
					}, nil)
			},
		},
		{
			name:   "positive test #2",
			url:    "/value/counter/PollCount",
			method: http.MethodGet,
			want: want{
				code:        200,
				response:    "100",
				contentType: "text/plain; charset=utf-8",
			},
			getFunc: func(m *mock.MockRepository) {
				delta := int64(100)
				m.EXPECT().
					Get(gomock.Any(), "PollCount", models.Counter).
					Return(models.Metrics{
						ID:    "PollCount",
						MType: models.Counter,
						Delta: &delta,
					}, nil)
			},
		},
		{
			name:    "negative test #3 - not allowed path",
			url:     "/value/counter/PollCount/100",
			method:  http.MethodGet,
			getFunc: nil, // chi вернёт 404 до хендлера
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:    "negative test #4 - not allowed metric type",
			url:     "/value/histogram/PollCount",
			method:  http.MethodGet,
			getFunc: nil, // хендлер отдаёт 400 до вызова Get
			want: want{
				code:        400,
				response:    "Invalid metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #5 - unknown metric name",
			url:    "/value/gauge/SomeName",
			method: http.MethodGet,
			want: want{
				code:        404,
				response:    "metric not found\n",
				contentType: "text/plain; charset=utf-8",
			},
			getFunc: func(m *mock.MockRepository) {
				m.EXPECT().
					Get(gomock.Any(), "SomeName", models.Gauge).
					Return(models.Metrics{}, errors.New("metric not found"))
			},
		},
		{
			name:    "negative test #6 - invalid method",
			url:     "/value/gauge/Alloc",
			method:  http.MethodPost,
			getFunc: nil, // chi вернёт 405 до хендлера
			want: want{
				code:        405,
				response:    "",
				contentType: "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mock.NewMockRepository(ctrl)

			if test.getFunc != nil {
				test.getFunc(mockRepo)
			}

			testLogger := zap.NewNop().Sugar()
			srv := service.NewMetricsService(mockRepo, testLogger)
			h := NewHandler(srv, nil, testLogger)

			r := chi.NewRouter()
			r.Route("/", func(r chi.Router) {
				r.Get("/", h.ListAllMetricsHandler)
				r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
				r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
			})

			request := httptest.NewRequest(test.method, test.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, test.want.code, res.StatusCode)

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, test.want.response, string(resBody))
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func Test_generateMetricsText(t *testing.T) {
	type args struct {
		metrics []models.Metrics
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "positive test #1",
			args: args{
				metrics: []models.Metrics{
					{
						MType: models.Gauge,
						ID:    "Alloc",
						Value: func(v float64) *float64 { return &v }(1.000001),
					},
					{
						MType: models.Counter,
						ID:    "PollCount",
						Delta: func(v int64) *int64 { return &v }(100),
					},
					{
						MType: "InvalidMetricType",
						ID:    "SomeName",
						Delta: func(v int64) *int64 { return &v }(100),
						Value: func(v float64) *float64 { return &v }(1.000001),
					},
					{
						MType: models.Gauge,
						ID:    "withoutGaugeValue",
						Value: nil,
					},
					{
						MType: models.Gauge,
						ID:    "withoutCounterValue",
						Delta: nil,
					},
				},
			},
			want: "Alloc - 1.000001\nPollCount - 100\nSomeName - unknown\nwithoutCounterValue - NaN\nwithoutGaugeValue - NaN\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, generateMetricsText(tt.args.metrics), "generateMetricsText(%v)", tt.args.metrics)
		})
	}
}

func TestHandler_ListAllMetricsHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockRepository(ctrl)

	mockRepo.EXPECT().
		List(gomock.Any()).
		Return([]models.Metrics{
			{
				MType: models.Gauge,
				ID:    "Alloc",
				Value: func(v float64) *float64 { return &v }(1.000001),
			},
			{
				MType: models.Counter,
				ID:    "PollCount",
				Delta: func(v int64) *int64 { return &v }(100),
			},
			{
				MType: models.Gauge,
				ID:    "HeapAlloc",
				Value: func(v float64) *float64 { return &v }(10.1000015),
			},
		}, nil)

	testLogger := zap.NewNop().Sugar()
	srv := service.NewMetricsService(mockRepo, testLogger)
	h := NewHandler(srv, nil, testLogger)

	expectedBody := "Alloc - 1.000001\nHeapAlloc - 10.1000015\nPollCount - 100\n"
	expectedContentType := "text/html; charset=utf-8"

	t.Run("positive test - get list all metrics", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Route("/", func(r chi.Router) {
			r.Get("/", h.ListAllMetricsHandler)
			r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
			r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
		})

		r.ServeHTTP(w, request)

		res := w.Result()
		assert.Equal(t, 200, res.StatusCode)

		defer res.Body.Close()
		resBody, readErr := io.ReadAll(res.Body)

		require.NoError(t, readErr)
		assert.Equal(t, expectedBody, string(resBody))
		assert.Equal(t, expectedContentType, res.Header.Get("Content-Type"))
	})
}

func TestHandler_UpdateJSONHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name       string
		url        string
		method     string
		expectSave bool
		body       []byte
		want       want
	}{
		{
			name:       "positive test #1",
			url:        "/update",
			body:       []byte(`{"type":"gauge","id":"Alloc","value":1.0001}`),
			method:     http.MethodPost,
			expectSave: true,
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "positive test #2",
			url:        "/update",
			body:       []byte(`{"type":"counter","id":"PollCount","delta":100}`),
			method:     http.MethodPost,
			expectSave: true,
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #3 - not allowed path",
			url:        "/create",
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        404,
				response:    "404 page not found",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #4 - not allowed metric type",
			url:        "/update",
			body:       []byte(`{"type":"histogram","id":"PollCount","delta":100}`),
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    `field "type": invalid value "histogram", must be "counter" or "gauge"`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #5 - without metric name",
			url:        "/update",
			body:       []byte(`{"type":"counter","delta":100}`),
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    `field "id": required`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #6 - without metric value",
			url:        "/update",
			body:       []byte(`{"type":"counter","id":"PollCount"}`),
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    "Metric value is required",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #7 - invalid metric value",
			url:        "/update",
			body:       []byte(`{"type":"counter","id":"PollCount","delta":100.12}`),
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    "JSON parse error: json: cannot unmarshal number 100.12 into Go struct field MetricAlias.delta of type int64",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #8 - invalid metric value",
			url:        "/update",
			body:       []byte(`{"type":"gauge","id":"PollCount","value":"100"}`),
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    "JSON parse error: json: cannot unmarshal string into Go struct field MetricAlias.value of type float64",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #9 - invalid method",
			url:        "/update",
			method:     http.MethodGet,
			expectSave: false,
			want: want{
				code:        405,
				response:    "", //chi не возвращает стандартные ответы для 405
				contentType: "", //chi не возвращает стандартные ответы для 405
			},
		},
		{
			name:       "negative test #10 - without metric value",
			url:        "/update",
			body:       []byte(`{"type":"counter","id":"PollCount","value":100.1}`),
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    "Metric value is required",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:       "negative test #11 - without metric value",
			url:        "/update",
			body:       []byte(`{"type":"gauge","id":"Alloc","delta":100}`),
			method:     http.MethodPost,
			expectSave: false,
			want: want{
				code:        400,
				response:    "Metric value is required",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mock.NewMockRepository(ctrl)

			if test.expectSave {
				mockRepo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(nil)
			}
			request := httptest.NewRequest(test.method, test.url, bytes.NewBuffer(test.body))
			// создаём новый Recorder
			w := httptest.NewRecorder()
			testLogger := zap.NewNop().Sugar()
			srv := service.NewMetricsService(mockRepo, testLogger)
			h := NewHandler(srv, nil, testLogger)

			r := chi.NewRouter()
			r.Route("/", func(r chi.Router) {
				r.Get("/", h.ListAllMetricsHandler)
				r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
				r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
				r.Post("/update", h.UpdateJSONHandler)
			})

			r.ServeHTTP(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.Equal(t, test.want.response, strings.TrimSpace(string(resBody)))
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestHandler_ValueJSONHandler(t *testing.T) {
	testGaugeMetric := models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
		Value: ptrFloat64(1.00001),
	}
	testCounterMetric := models.Metrics{
		ID:    "PollCount",
		MType: models.Counter,
		Delta: ptrInt64(100),
	}

	type want struct {
		code        int
		metric      *models.Metrics
		contentType string
	}
	tests := []struct {
		name    string
		url     string
		body    []byte
		method  string
		want    want
		getFunc func(m *mock.MockRepository)
	}{
		{
			name:   "positive test #1",
			url:    "/value",
			body:   []byte(`{"type":"gauge","id":"Alloc"}`),
			method: http.MethodPost,
			want: want{
				code:        200,
				metric:      &testGaugeMetric,
				contentType: "application/json; charset=utf-8",
			},
			getFunc: func(m *mock.MockRepository) {
				m.EXPECT().
					Get(gomock.Any(), "Alloc", models.Gauge).
					Return(testGaugeMetric, nil)
			},
		},
		{
			name:   "positive test #2",
			url:    "/value",
			body:   []byte(`{"type":"counter","id":"PollCount"}`),
			method: http.MethodPost,
			want: want{
				code:        200,
				metric:      &testCounterMetric,
				contentType: "application/json; charset=utf-8",
			},
			getFunc: func(m *mock.MockRepository) {
				m.EXPECT().
					Get(gomock.Any(), "PollCount", models.Counter).
					Return(testCounterMetric, nil)
			},
		},
		{
			name:   "negative test #3 - not allowed path",
			url:    "/values",
			body:   []byte(`{"type":"gauge","id":"Alloc"}`),
			method: http.MethodPost,
			want: want{
				code:        404,
				contentType: "text/plain; charset=utf-8",
			},
			getFunc: nil, // chi вернёт 404 до хендлера
		},
		{
			name:   "negative test #4 - not allowed metric type",
			url:    "/value",
			body:   []byte(`{"type":"histogram","id":"Alloc"}`),
			method: http.MethodPost,
			want: want{
				code:        400,
				contentType: "text/plain; charset=utf-8",
			},
			getFunc: nil, // chi вернёт 400 до вызова Get
		},
		{
			name:   "negative test #5 - unknown metric name",
			url:    "/value",
			body:   []byte(`{"type":"gauge","id":"Alloc1"}`),
			method: http.MethodPost,
			want: want{
				code:        404,
				contentType: "text/plain; charset=utf-8",
			},
			getFunc: func(m *mock.MockRepository) {
				m.EXPECT().
					Get(gomock.Any(), "Alloc1", models.Gauge).
					Return(models.Metrics{}, errors.New("metric not found"))
			},
		},
		{
			name:   "negative test #6 - invalid method",
			url:    "/value",
			method: http.MethodGet,
			want: want{
				code:        405,
				contentType: "",
			},
			getFunc: nil, // chi вернёт 405 до хендлера
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mock.NewMockRepository(ctrl)

			if test.getFunc != nil {
				test.getFunc(mockRepo)
			}

			testLogger := zap.NewNop().Sugar()
			srv := service.NewMetricsService(mockRepo, testLogger)
			h := NewHandler(srv, nil, testLogger)

			request := httptest.NewRequest(test.method, test.url, bytes.NewBuffer(test.body))
			// создаём новый Recorder
			w := httptest.NewRecorder()

			r := chi.NewRouter()
			r.Route("/", func(r chi.Router) {
				r.Get("/", h.ListAllMetricsHandler)
				r.Get("/value/{metricType}/{metricName}", h.ValueHandler)
				r.Post("/update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)
				r.Post("/update", h.UpdateJSONHandler)
				r.Post("/value", h.ValueJSONHandler)
			})

			r.ServeHTTP(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.code, res.StatusCode)
			if res.StatusCode == 200 {
				// получаем и проверяем тело запроса
				defer res.Body.Close()
				var actual models.Metrics
				err := json.NewDecoder(res.Body).Decode(&actual)

				require.NoError(t, err)
				assert.Equal(t, test.want.metric.ID, actual.ID)
				assert.Equal(t, test.want.metric.MType, actual.MType)
				assert.Equal(t, test.want.metric.Value, actual.Value)
				assert.Equal(t, test.want.metric.Delta, actual.Delta)
			}
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestHandler_UpdateBatchJSONHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name        string
		url         string
		method      string
		expectBatch bool
		body        []byte
		want        want
	}{
		{
			name:        "positive test #1 - mixed batch",
			url:         "/updates",
			method:      http.MethodPost,
			expectBatch: true,
			body:        []byte(`[{"type":"gauge","id":"Alloc","value":1.0001},{"type":"counter","id":"PollCount","delta":100}]`),
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "positive test #2 - single gauge",
			url:         "/updates",
			method:      http.MethodPost,
			expectBatch: true,
			body:        []byte(`[{"type":"gauge","id":"HeapAlloc","value":42.5}]`),
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "positive test #3 - multiple counters",
			url:         "/updates",
			method:      http.MethodPost,
			expectBatch: true,
			body:        []byte(`[{"type":"counter","id":"PollCount","delta":10},{"type":"counter","id":"Requests","delta":5}]`),
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "negative test #4 - empty batch",
			url:         "/updates",
			method:      http.MethodPost,
			expectBatch: false,
			body:        []byte(`[]`),
			want: want{
				code:        400,
				response:    "empty batch",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "negative test #5 - invalid JSON",
			url:         "/updates",
			method:      http.MethodPost,
			expectBatch: false,
			body:        []byte(`{not valid json`),
			want: want{
				code:        400,
				response:    "invalid request body",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "negative test #6 - invalid metric type",
			url:         "/updates",
			method:      http.MethodPost,
			expectBatch: false,
			body:        []byte(`[{"type":"histogram","id":"PollCount","delta":100}]`),
			want: want{
				code:        400,
				response:    "invalid request body",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "negative test #7 - empty metric name",
			url:         "/updates",
			method:      http.MethodPost,
			expectBatch: false,
			body:        []byte(`[{"type":"counter","id":"","delta":100}]`),
			want: want{
				code:        400,
				response:    "invalid request body",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "negative test #8 - internal server error",
			url:         "/updates",
			method:      http.MethodPost,
			expectBatch: true,
			body:        []byte(`[{"type":"gauge","id":"Alloc","value":1.0}]`),
			want: want{
				code:        500,
				response:    "Internal Server Error",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "negative test #9 - not allowed path",
			url:         "/create",
			method:      http.MethodPost,
			expectBatch: false,
			want: want{
				code:        404,
				response:    "404 page not found",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "negative test #10 - invalid method",
			url:         "/updates",
			method:      http.MethodGet,
			expectBatch: false,
			want: want{
				code:        405,
				response:    "",
				contentType: "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRepo := mock.NewMockRepository(ctrl)

			if test.expectBatch {
				call := mockRepo.EXPECT().
					SaveBatch(gomock.Any(), gomock.Any()).
					Return(nil)

				if test.want.code == 500 {
					call.Return(errors.New("db connection lost"))
				}
			}

			var bodyReader io.Reader
			if test.body != nil {
				bodyReader = bytes.NewReader(test.body)
			}
			request := httptest.NewRequest(test.method, test.url, bodyReader)
			w := httptest.NewRecorder()
			testLogger := zap.NewNop().Sugar()
			srv := service.NewMetricsService(mockRepo, testLogger)
			h := NewHandler(srv, nil, testLogger)

			r := chi.NewRouter()
			r.Route("/", func(r chi.Router) {
				r.Post("/updates", h.UpdateBatchJSONHandler)
			})

			r.ServeHTTP(w, request)

			res := w.Result()
			assert.Equal(t, test.want.code, res.StatusCode)

			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, test.want.response, strings.TrimSpace(string(resBody)))
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

//go:fix inline
func ptrInt64(v int64) *int64 { return new(v) }

//go:fix inline
func ptrFloat64(v float64) *float64 { return new(v) }
