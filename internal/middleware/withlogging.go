package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

func WithLogging(log *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		logFn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lw := NewLoggingResponseWriter(w)
			h.ServeHTTP(lw, r)

			duration := time.Since(start)

			log.Infow(
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
}
