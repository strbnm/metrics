package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	models "github.com/strbnm/metrics/internal/model"
	"github.com/strbnm/metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strbnm/metrics/internal/repository"
)

func TestHandler_UpdateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name   string
		url    string
		method string
		want   want
	}{
		{
			name:   "positive test #1",
			url:    "/update/gauge/Alloc/1.0001",
			method: http.MethodPost,
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "positive test #2",
			url:    "/update/counter/PollCount/100",
			method: http.MethodPost,
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #3 - not allowed path",
			url:    "/create/counter/PollCount/100",
			method: http.MethodPost,
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #4 - not allowed metric type",
			url:    "/update/histogram/SomeHist/100",
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    "Invalid metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #5 - without metric name",
			url:    "/update/gauge/100",
			method: http.MethodPost,
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #6 - without metric value",
			url:    "/update/gauge/Alloc",
			method: http.MethodPost,
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #7 - invalid metric value",
			url:    "/update/gauge/Alloc/invalid_value",
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    "Invalid metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #8 - invalid metric value",
			url:    "/update/counter/PullCount/invalid_value",
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    "Invalid metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #9 - invalid method",
			url:    "/update/counter/PullCount/invalid_value",
			method: http.MethodGet,
			want: want{
				code:        405,
				response:    "", //chi не возвращает стандартные ответы для 405
				contentType: "", //chi не возвращает стандартные ответы для 405
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.url, nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()
			repo := repository.NewMemStorage()
			srv := service.NewMetricsService(repo)
			h := NewHandler(srv)

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
		name   string
		url    string
		method string
		want   want
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
		},
		{
			name:   "negative test #3 - not allowed path",
			url:    "/value/counter/PollCount/100",
			method: http.MethodGet,
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #4 - not allowed metric type",
			url:    "/value/histogram/PollCount",
			method: http.MethodGet,
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
		},
		{
			name:   "negative test #6 - invalid method",
			url:    "/value/gauge/Alloc",
			method: http.MethodPost,
			want: want{
				code:        405,
				response:    "",
				contentType: "",
			},
		},
	}
	repo := repository.NewMemStorage()
	srv := service.NewMetricsService(repo)
	err := srv.UpdateMetric(models.Gauge, "Alloc", "1.000001")
	require.NoError(t, err)
	err = srv.UpdateMetric(models.Counter, "PollCount", "100")
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.url, nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()

			h := NewHandler(srv)

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
	var err error
	repo := repository.NewMemStorage()
	for _, metric := range []models.Metrics{
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
	} {
		err = repo.Save(metric)
		require.NoError(t, err)
	}
	srv := service.NewMetricsService(repo)
	expectedBody := "Alloc - 1.000001\nHeapAlloc - 10.1000015\nPollCount - 100\n"
	expectedContentType := "text/html; charset=utf-8"

	t.Run("positive test - get list all metrics", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		// создаём новый Recorder
		w := httptest.NewRecorder()

		h := NewHandler(srv)

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
		name   string
		url    string
		method string
		body   []byte
		want   want
	}{
		{
			name:   "positive test #1",
			url:    "/update",
			body:   []byte(`{"type":"gauge","id":"Alloc","value":1.0001}`),
			method: http.MethodPost,
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "positive test #2",
			url:    "/update",
			body:   []byte(`{"type":"counter","id":"PollCount","delta":100}`),
			method: http.MethodPost,
			want: want{
				code:        200,
				response:    "OK",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #3 - not allowed path",
			url:    "/create",
			method: http.MethodPost,
			want: want{
				code:        404,
				response:    "404 page not found",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #4 - not allowed metric type",
			url:    "/update",
			body:   []byte(`{"type":"histogram","id":"PollCount","delta":100}`),
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    `field "type": invalid value "histogram", must be "counter" or "gauge"`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #5 - without metric name",
			url:    "/update",
			body:   []byte(`{"type":"counter","delta":100}`),
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    `field "id": required`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #6 - without metric value",
			url:    "/update",
			body:   []byte(`{"type":"counter","id":"PollCount"}`),
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    "Metric value is required",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #7 - invalid metric value",
			url:    "/update",
			body:   []byte(`{"type":"counter","id":"PollCount","delta":100.12}`),
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    "JSON parse error: json: cannot unmarshal number 100.12 into Go struct field MetricAlias.delta of type int64",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #8 - invalid metric value",
			url:    "/update",
			body:   []byte(`{"type":"gauge","id":"PollCount","value":"100"}`),
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    "JSON parse error: json: cannot unmarshal string into Go struct field MetricAlias.value of type float64",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #9 - invalid method",
			url:    "/update",
			method: http.MethodGet,
			want: want{
				code:        405,
				response:    "", //chi не возвращает стандартные ответы для 405
				contentType: "", //chi не возвращает стандартные ответы для 405
			},
		},
		{
			name:   "negative test #10 - without metric value",
			url:    "/update",
			body:   []byte(`{"type":"counter","id":"PollCount","value":100.1}`),
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    "Metric value is required",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #11 - without metric value",
			url:    "/update",
			body:   []byte(`{"type":"gauge","id":"Alloc","delta":100}`),
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    "Metric value is required",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.url, bytes.NewBuffer(test.body))
			// создаём новый Recorder
			w := httptest.NewRecorder()
			repo := repository.NewMemStorage()
			srv := service.NewMetricsService(repo)
			h := NewHandler(srv)

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
		name   string
		url    string
		body   []byte
		method string
		want   want
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
		},
		{
			name:   "negative test #6 - invalid method",
			url:    "/value",
			method: http.MethodGet,
			want: want{
				code:        405,
				contentType: "",
			},
		},
	}
	repo := repository.NewMemStorage()
	srv := service.NewMetricsService(repo)
	err := srv.UpdateMetric(testGaugeMetric.MType, testGaugeMetric.ID, fmt.Sprint(*testGaugeMetric.Value))
	require.NoError(t, err)
	err = srv.UpdateMetric(testCounterMetric.MType, testCounterMetric.ID, fmt.Sprint(*testCounterMetric.Delta))
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.url, bytes.NewBuffer(test.body))
			// создаём новый Recorder
			w := httptest.NewRecorder()

			h := NewHandler(srv)

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
				err = json.NewDecoder(res.Body).Decode(&actual)

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

//go:fix inline
func ptrInt64(v int64) *int64 { return new(v) }

//go:fix inline
func ptrFloat64(v float64) *float64 { return new(v) }
