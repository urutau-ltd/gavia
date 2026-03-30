package securityheaders

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareSetsSecurityHeaders(t *testing.T) {
	service := NewService()
	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("expected Referrer-Policy header, got %q", got)
	}

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}

	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected X-Frame-Options DENY, got %q", got)
	}

	if got := rec.Header().Get("Permissions-Policy"); got == "" {
		t.Fatal("expected Permissions-Policy header to be present")
	}

	if got := rec.Header().Get("Content-Security-Policy"); got != "base-uri 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none'" {
		t.Fatalf("unexpected Content-Security-Policy header: %q", got)
	}

	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("expected HSTS header to be absent on plain HTTP, got %q", got)
	}
}

func TestMiddlewareSetsHSTSOnTLSRequests(t *testing.T) {
	service := NewService()
	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains" {
		t.Fatalf("expected HSTS header, got %q", got)
	}
}
