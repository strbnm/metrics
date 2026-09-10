package middleware

import (
	"net/http"
	"time"

	"github.com/strbnm/metrics/internal/logger"
)

func WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lw := NewLoggingResponseWriter(w)
		h.ServeHTTP(lw, r)

		duration := time.Since(start)

		logger.Log.Infow(
			"HTTP request",
			"uri", r.RequestURI,
			"method", r.Method,
			"status", lw.responseData.status,
			"duration", duration,
			"size", lw.responseData.size,
		)
	}
	return http.HandlerFunc(logFn)
}
