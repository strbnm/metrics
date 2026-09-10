package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type compressWriter struct {
	w            http.ResponseWriter
	zw           *gzip.Writer
	allowedTypes map[string]bool
	compress     bool // Флаг: сжимать или нет
	checked      bool // Флаг: уже проверили Content-Type
}

func newCompressWriter(w http.ResponseWriter, allowed map[string]bool) *compressWriter {
	return &compressWriter{
		w:            w,
		zw:           gzip.NewWriter(w),
		allowedTypes: allowed,
	}
}

// checkContentType проверяет Content-Type один раз и выставляет флаг compress
func (c *compressWriter) checkContentType() {
	if c.checked {
		return
	}
	c.checked = true

	ct := c.w.Header().Get("Content-Type")
	// отсекаем параметры: "application/json; charset=utf-8" → "application/json"
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = ct[:idx]
	}
	ct = strings.TrimSpace(ct)
	c.compress = c.allowedTypes[ct]
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	c.checkContentType()
	if !c.compress {
		// тип не подходит — пишем напрямую, без сжатия
		return c.w.Write(p)
	}
	c.w.Header().Set("Content-Encoding", "gzip")
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	c.checkContentType()
	if c.compress && statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer только если было сжатие
func (c *compressWriter) Close() error {
	if c.compress {
		return c.zw.Close()
	}
	return nil
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
