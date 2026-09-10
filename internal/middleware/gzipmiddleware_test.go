package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Хелперы ---

// gzipData сжимает данные и возвращает []byte
func gzipData(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(data)
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

// gunzipData распаковывает данные и возвращает []byte
func gunzipData(t *testing.T, data []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer reader.Close()
	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	return decompressed
}

// --- Тесты compressWriter ---

func TestCompressWriter_Write_SetsContentEncoding(t *testing.T) {
	rec := httptest.NewRecorder()
	cw := newCompressWriter(rec,
		map[string]bool{
			"application/json": true,
			"text/html":        true,
		},
	)

	cw.Header().Set("Content-Type", "text/html")

	_, err := cw.Write([]byte("hello world"))
	require.NoError(t, err)
	require.NoError(t, cw.Close())

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	decompressed := gunzipData(t, rec.Body.Bytes())
	assert.Equal(t, "hello world", string(decompressed))
}

func TestCompressWriter_WriteHeader_SetsEncodingForSuccessCodes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		statusCode  int
		expectSet   bool
	}{
		{"200 OK", "text/html", http.StatusOK, true},
		{"201 Created", "text/html", http.StatusCreated, true},
		{"301 Redirect", "text/html", http.StatusMovedPermanently, false},
		{"404 Not Found", "text/html", http.StatusNotFound, false},
		{"500 Server Error", "text/html", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			cw := newCompressWriter(rec,
				map[string]bool{
					"application/json": true,
					"text/html":        true,
				},
			)
			cw.Header().Set("Content-Type", tt.contentType)
			cw.WriteHeader(tt.statusCode)
			require.NoError(t, cw.Close())

			if tt.expectSet {
				assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
			} else {
				assert.Empty(t, rec.Header().Get("Content-Encoding"))
			}
		})
	}
}

// --- Тесты compressReader ---

func TestCompressReader_Read(t *testing.T) {
	original := []byte(`{"id":"m1","type":"counter","delta":42}`)
	compressed := gzipData(t, original)

	reader := io.NopCloser(bytes.NewReader(compressed))
	cr, err := newCompressReader(reader)
	require.NoError(t, err)
	defer cr.Close()

	decompressed, err := io.ReadAll(cr)
	require.NoError(t, err)
	assert.Equal(t, original, decompressed)
}

func TestCompressReader_InvalidGzip(t *testing.T) {
	reader := io.NopCloser(bytes.NewReader([]byte("not gzip data")))
	cr, err := newCompressReader(reader)
	require.Error(t, err)
	assert.Nil(t, cr)
}

// --- Тесты GzipMiddleware ---

// Простой хендлер, который эхом возвращает тело запроса как JSON
func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})
}

func TestGzipMiddleware_CompressesResponse(t *testing.T) {
	handler := GzipMiddleware(echoHandler())
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Клиент запрашивает gzip
	req, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Сервер должен вернуть сжатый ответ
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

}

func TestGzipMiddleware_DecompressesRequest(t *testing.T) {
	// Хендлер проверяет, что получил РАСПАКОВАННЫЕ данные
	var receivedBody string
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	ts := httptest.NewServer(handler)
	defer ts.Close()

	originalJSON := `{"id":"m1","type":"counter","delta":10}`
	compressed := gzipData(t, []byte(originalJSON))

	req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(compressed))
	require.NoError(t, err)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Хендлер должен получить распакованный JSON
	assert.Equal(t, originalJSON, receivedBody)
}

func TestGzipMiddleware_BothCompressAndDecompress(t *testing.T) {
	handler := GzipMiddleware(echoHandler())
	ts := httptest.NewServer(handler)
	defer ts.Close()

	originalJSON := `{"id":"g1","type":"gauge","value":3.14}`
	compressedRequest := gzipData(t, []byte(originalJSON))

	req, err := http.NewRequest("POST", ts.URL, bytes.NewReader(compressedRequest))
	require.NoError(t, err)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Ответ тоже должен быть сжат
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	decompressedResponse := gunzipData(t, func() []byte {
		b, _ := io.ReadAll(resp.Body)
		return b
	}())

	assert.Equal(t, originalJSON, string(decompressedResponse))
}

func TestGzipMiddleware_NoCompressionWithoutAcceptEncoding(t *testing.T) {
	handler := GzipMiddleware(echoHandler())
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader([]byte(`{"test":true}`)))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Нет Accept-Encoding → ответ не сжат
	assert.Empty(t, resp.Header.Get("Content-Encoding"))

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, `{"test":true}`, string(body))
}

func TestGzipMiddleware_InvalidGzipRequest_Returns500(t *testing.T) {
	handler := GzipMiddleware(echoHandler())
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Отправляем мусор с заголовком Content-Encoding: gzip
	req, err := http.NewRequest("POST", ts.URL, bytes.NewReader([]byte("not gzip at all")))
	require.NoError(t, err)
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestGzipMiddleware_PassesThroughWithoutEncoding(t *testing.T) {
	handler := GzipMiddleware(echoHandler())
	ts := httptest.NewServer(handler)
	defer ts.Close()

	originalJSON := `{"id":"m1","type":"counter","delta":5}`

	// Ни Accept-Encoding, ни Content-Encoding
	req, err := http.NewRequest("POST", ts.URL, bytes.NewReader([]byte(originalJSON)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, resp.Header.Get("Content-Encoding"))

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, originalJSON, string(body))
}
