package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestSign(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		key     string
		want    string
	}{
		{
			name:    "simple payload",
			payload: []byte("hello world"),
			key:     "secret",
			want:    signExpected([]byte("hello world"), "secret"),
		},
		{
			name:    "empty payload",
			payload: []byte{},
			key:     "secret",
			want:    signExpected([]byte{}, "secret"),
		},
		{
			name:    "empty key",
			payload: []byte("hello world"),
			key:     "",
			want:    signExpected([]byte("hello world"), ""),
		},
		{
			name:    "unicode payload",
			payload: []byte("привет, мир!"),
			key:     "ключ",
			want:    signExpected([]byte("привет, мир!"), "ключ"),
		},
		{
			name:    "binary payload",
			payload: []byte{0x00, 0x01, 0xFF, 0xFE},
			key:     "binary-key",
			want:    signExpected([]byte{0x00, 0x01, 0xFF, 0xFE}, "binary-key"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sign(tt.payload, tt.key)
			if got != tt.want {
				t.Errorf("Sign() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSignDeterministic(t *testing.T) {
	payload := []byte("same data")
	key := "same-key"
	first := Sign(payload, key)
	second := Sign(payload, key)
	if first != second {
		t.Errorf("Sign() is not deterministic: %q != %q", first, second)
	}
}

func TestSignDifferentKeys(t *testing.T) {
	payload := []byte("hello")
	h1 := Sign(payload, "key1")
	h2 := Sign(payload, "key2")
	if h1 == h2 {
		t.Errorf("different keys should produce different hashes")
	}
}

func TestSignDifferentPayloads(t *testing.T) {
	key := "key"
	h1 := Sign([]byte("hello"), key)
	h2 := Sign([]byte("world"), key)
	if h1 == h2 {
		t.Errorf("different payloads should produce different hashes")
	}
}

func TestVerify(t *testing.T) {
	validPayload := []byte(`{"id":"test","value":42}`)
	validKey := "secret"
	validHash := Sign(validPayload, validKey)

	tamperedPayload := []byte(`{"id":"test","value":43}`)

	tests := []struct {
		name       string
		payload    []byte
		key        string
		expectHash string
		want       bool
	}{
		{
			name:       "valid signature",
			payload:    validPayload,
			key:        validKey,
			expectHash: validHash,
			want:       true,
		},
		{
			name:       "tampered payload",
			payload:    tamperedPayload,
			key:        validKey,
			expectHash: validHash,
			want:       false,
		},
		{
			name:       "wrong key",
			payload:    validPayload,
			key:        "wrong-key",
			expectHash: validHash,
			want:       false,
		},
		{
			name:       "empty key",
			payload:    validPayload,
			key:        "",
			expectHash: validHash,
			want:       false,
		},
		{
			name:       "empty expected hash",
			payload:    validPayload,
			key:        validKey,
			expectHash: "",
			want:       false,
		},
		{
			name:       "empty payload with valid signature",
			payload:    []byte{},
			key:        validKey,
			expectHash: Sign([]byte{}, validKey),
			want:       true,
		},
		{
			name:       "malformed hash (not hex)",
			payload:    validPayload,
			key:        validKey,
			expectHash: "not-a-hex-string",
			want:       false,
		},
		{
			name:       "hash with wrong length",
			payload:    validPayload,
			key:        validKey,
			expectHash: "deadbeef",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Verify(tt.payload, tt.key, tt.expectHash)
			if got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

// signExpected — эталонная реализация, независимая от Sign.
// Используется, чтобы тест не сводился к сравнению функции с самой собой.
func signExpected(payload []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
