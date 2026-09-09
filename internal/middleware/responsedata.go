package middleware

import "net/http"

type (
	responseData struct {
		status int
		size   int
	}

	LoggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	if r.responseData.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{
		ResponseWriter: w,
		responseData: &responseData{
			status: 0,
			size:   0,
		},
	}
}
