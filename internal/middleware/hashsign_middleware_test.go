package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/strbnm/metrics/internal/signer"
)

// testHandler возвращает запрос обратно в ответ и ставит статус 200.
func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})
}

// chain последовательно оборачивает handler middleware.
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func TestHashSign_NoKey_PassesThrough(t *testing.T) {
	body := []byte(`{"metric":"cpu"}`)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h := HashSign("")(testHandler())
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != string(body) {
		t.Errorf("body = %q, want %q", rec.Body.String(), string(body))
	}
	if rec.Header().Get(HashHeader) != "" {
		t.Errorf("HashHeader should not be set when key is empty")
	}
}

func TestHashSign_ValidSignature(t *testing.T) {
	key := "secret"
	body := []byte(`{"metric":"cpu"}`)
	hash := signer.Sign(body, key)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(HashHeader, hash)
	rec := httptest.NewRecorder()

	h := HashSign(key)(testHandler())
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != string(body) {
		t.Errorf("body = %q, want %q", rec.Body.String(), string(body))
	}

	respHash := rec.Header().Get(HashHeader)
	if respHash == "" {
		t.Fatal("HashHeader not set in response")
	}
	expectedRespHash := signer.Sign(rec.Body.Bytes(), key)
	if respHash != expectedRespHash {
		t.Errorf("response hash = %q, want %q", respHash, expectedRespHash)
	}
}

func TestHashSign_InvalidSignature(t *testing.T) {
	key := "secret"
	body := []byte(`{"metric":"cpu"}`)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(HashHeader, "invalid-hash-value")
	rec := httptest.NewRecorder()

	h := HashSign(key)(testHandler())
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHashSign_TamperedPayload(t *testing.T) {
	key := "secret"
	original := []byte(`{"metric":"cpu"}`)
	tampered := []byte(`{"metric":"mem"}`)
	hash := signer.Sign(original, key)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tampered))
	req.Header.Set(HashHeader, hash)
	rec := httptest.NewRecorder()

	h := HashSign(key)(testHandler())
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHashSign_WrongKey(t *testing.T) {
	body := []byte(`{"metric":"cpu"}`)
	hash := signer.Sign(body, "other-key")

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(HashHeader, hash)
	rec := httptest.NewRecorder()

	h := HashSign("server-key")(testHandler())
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHashSign_ResponseHashMatches(t *testing.T) {
	key := "secret"
	body := []byte(`{"metric":"cpu","value":42}`)
	hash := signer.Sign(body, key)

	// handler возвращает изменённое тело — проверяем что хеш считается от ответа
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respBody := []byte(`{"status":"ok"}`)
		w.WriteHeader(http.StatusAccepted)
		w.Write(respBody)
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(HashHeader, hash)
	rec := httptest.NewRecorder()

	h := HashSign(key)(handler)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}

	respHash := rec.Header().Get(HashHeader)
	expected := signer.Sign(rec.Body.Bytes(), key)
	if respHash != expected {
		t.Errorf("response hash = %q, want %q", respHash, expected)
	}
}

func TestHashSign_PreservesInnerHeaders(t *testing.T) {
	key := "secret"
	body := []byte(`{"metric":"cpu"}`)
	hash := signer.Sign(body, key)

	// handler ставит свой заголовок
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(HashHeader, hash)
	rec := httptest.NewRecorder()

	h := HashSign(key)(handler)
	h.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type not preserved: %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get(HashHeader) == "" {
		t.Error("HashHeader not set")
	}
}

func TestHashSign_WithGzipMiddleware(t *testing.T) {
	key := "secret"
	rawBody := []byte(`{"metric":"cpu","value":42}`)

	// Сжимаем тело, как сделал бы агент
	compressedBody := gzipCompress(t, rawBody)
	hash := signer.Sign(compressedBody, key)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressedBody))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set(HashHeader, hash)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GzipMiddleware уже распаковал — handler видит сырое тело
		decompressed, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(decompressed)
	})

	// Порядок как в роутере: HashSign (внешняя) → Gzip (внутренняя)
	h := chain(handler, HashSign(key), GzipMiddleware)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Тело ответа должно быть сжато gzip
	decompressedResp := gzipDecompress(t, rec.Body.Bytes())
	if string(decompressedResp) != string(rawBody) {
		t.Errorf("decompressed response = %q, want %q", string(decompressedResp), string(rawBody))
	}

	// Хеш ответа считается от СЖАТОГО тела
	respHash := rec.Header().Get(HashHeader)
	expected := signer.Sign(rec.Body.Bytes(), key)
	if respHash != expected {
		t.Errorf("response hash = %q, want %q", respHash, expected)
	}

	// Content-Encoding должен быть сохранён
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Content-Encoding not preserved: %q", rec.Header().Get("Content-Encoding"))
	}
}

func TestHashSign_EmptyBody(t *testing.T) {
	key := "secret"
	body := []byte{}
	hash := signer.Sign(body, key)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(HashHeader, hash)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	h := HashSign(key)(handler)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHashSign_BodyRestoredForHandler(t *testing.T) {
	key := "secret"
	body := []byte(`{"metric":"cpu"}`)
	hash := signer.Sign(body, key)

	var handlerReceivedBody []byte

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerReceivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(HashHeader, hash)
	rec := httptest.NewRecorder()

	h := HashSign(key)(handler)
	h.ServeHTTP(rec, req)

	if string(handlerReceivedBody) != string(body) {
		t.Errorf("handler received body = %q, want %q", string(handlerReceivedBody), string(body))
	}
}

// --- helpers ---

func gzipCompress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func gzipDecompress(t *testing.T, data []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer zr.Close()
	result, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	return result
}
