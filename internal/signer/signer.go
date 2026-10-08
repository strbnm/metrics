package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign вычисляет HMAC-SHA256 от тела payload с использованием ключа key
// и возвращает hex-представление хеша.
func Sign(payload []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// Verify проверяет, что expectedHash совпадает с вычисленным хешем payload.
func Verify(payload []byte, key, expectedHash string) bool {
	if key == "" || expectedHash == "" {
		return false
	}
	computed := Sign(payload, key)
	return hmac.Equal([]byte(computed), []byte(expectedHash))
}
