package compression

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareCompressesTextResponsesWithGzip(t *testing.T) {
	service := NewService()
	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(strings.Repeat("hello compressed world ", 20)))
	}))

	req := httptest.NewRequest(http.MethodGet, "http://gavia.test/dashboard", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("unexpected content encoding: got %q want %q", got, "gzip")
	}

	if got := rec.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("expected Vary to mention Accept-Encoding, got %q", got)
	}

	reader, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("gzip.NewReader returned error: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll returned error: %v", err)
	}

	if want := strings.Repeat("hello compressed world ", 20); string(body) != want {
		t.Fatalf("unexpected decompressed body: got %q want %q", string(body), want)
	}
}

func TestMiddlewareSkipsBinaryStaticExtensions(t *testing.T) {
	service := NewService()
	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("PNGDATA"))
	}))

	req := httptest.NewRequest(http.MethodGet, "http://gavia.test/static/img/logo.png", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("expected binary static response to skip compression, got %q", got)
	}

	if got := rec.Body.String(); got != "PNGDATA" {
		t.Fatalf("unexpected plain body: got %q want %q", got, "PNGDATA")
	}
}

func TestMiddlewareSkipsCompressionWithoutAcceptEncoding(t *testing.T) {
	service := NewService()
	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write([]byte(strings.Repeat("body{color:red;}", 20)))
	}))

	req := httptest.NewRequest(http.MethodGet, "http://gavia.test/static/css/gavia.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("expected response without Accept-Encoding gzip to remain uncompressed, got %q", got)
	}
}
