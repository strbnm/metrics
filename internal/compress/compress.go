package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
)

const CompressionLevel = gzip.DefaultCompression

var AllowedTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, CompressionLevel)
	if err != nil {
		return nil, err
	}
	if _, err := gz.Write(data); err != nil {
		err := gz.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func NewWriter(w io.Writer) (*gzip.Writer, error) {
	return gzip.NewWriterLevel(w, CompressionLevel)
}

func NewReader(r io.Reader) (*gzip.Reader, error) {
	return gzip.NewReader(r)
}

// IsCompressible проверяет Content-Type без параметров
func IsCompressible(contentType string) bool {
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)
	return AllowedTypes[contentType]
}
