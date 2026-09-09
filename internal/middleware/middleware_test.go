package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/strbnm/metrics/internal/logger"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggingResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	lw := NewLoggingResponseWriter(rec)

	data := []byte("hello")

	n, err := lw.Write(data)

	require.NoError(t, err)
	require.Equal(t, len(data), n)
	require.Equal(t, len(data), lw.responseData.size)
	require.Equal(t, data, rec.Body.Bytes())
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	lw := NewLoggingResponseWriter(rec)

	lw.WriteHeader(http.StatusCreated)

	require.Equal(t, http.StatusCreated, lw.responseData.status)
	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestLoggingResponseWriter_WriteHeaderAndWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	lw := NewLoggingResponseWriter(rec)

	lw.WriteHeader(http.StatusCreated)

	data := []byte("hello world")

	n, err := lw.Write(data)

	require.NoError(t, err)
	require.Equal(t, len(data), n)
	require.Equal(t, http.StatusCreated, lw.responseData.status)
	require.Equal(t, len(data), lw.responseData.size)
	require.Equal(t, data, rec.Body.Bytes())
}

func TestLoggingResponseWriter_Write_AccumulatesSize(t *testing.T) {
	rec := httptest.NewRecorder()
	lw := NewLoggingResponseWriter(rec)

	_, err := lw.Write([]byte("hello"))
	require.NoError(t, err)

	_, err = lw.Write([]byte(" world"))
	require.NoError(t, err)

	require.Equal(t, 11, lw.responseData.size)
	require.Equal(t, "hello world", rec.Body.String())
}

func TestWithLogging(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	oldLogger := logger.Log

	logger.Log = zap.New(core).Sugar()
	t.Cleanup(func() {
		logger.Log = oldLogger
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello"))
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/test?foo=bar",
		nil,
	)
	rec := httptest.NewRecorder()

	middleware := WithLogging(handler)
	middleware.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "hello", rec.Body.String())

	require.Equal(t, 1, logs.Len())

	entry := logs.All()[0]

	require.Equal(t, "uri", entry.Context[0].Key)
	require.Equal(t, "/test?foo=bar", entry.Context[0].String)

	require.Equal(t, "method", entry.Context[1].Key)
	require.Equal(t, http.MethodGet, entry.Context[1].String)

	require.Equal(t, "status", entry.Context[2].Key)
	require.Equal(t, int64(http.StatusCreated), entry.Context[2].Integer)

	require.Equal(t, "duration", entry.Context[3].Key)

	require.Equal(t, "size", entry.Context[4].Key)
	require.Equal(t, int64(5), entry.Context[4].Integer)
}
