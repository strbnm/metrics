package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
				response:    "Invalid gauge value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #7 - invalid metric value",
			url:    "/update/counter/PullCount/invalid_value",
			method: http.MethodPost,
			want: want{
				code:        400,
				response:    "Invalid counter value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "negative test #8 - invalid method",
			url:    "/update/counter/PullCount/invalid_value",
			method: http.MethodGet,
			want: want{
				code:        405,
				response:    "Method Not Allowed\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.url, nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()
			repo := repository.NewMemStorage()
			h := NewHandler(repo)

			mux := http.NewServeMux()
			mux.HandleFunc("POST /update/{metricType}/{metricName}/{metricValue}", h.UpdateHandler)

			mux.ServeHTTP(w, request)

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
