package csrf

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMiddlewareSetsCSRFTokenCookieOnSafeRequest(t *testing.T) {
	service := NewService()
	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://gavia.test/form", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected GET to pass through with %d, got %d", http.StatusNoContent, rec.Code)
	}

	cookie := cookieByName(t, rec.Result().Cookies(), CookieName)
	if strings.TrimSpace(cookie.Value) == "" {
		t.Fatal("expected CSRF middleware to issue a non-empty token cookie")
	}
}

func TestMiddlewareRejectsUnsafeRequestWithoutToken(t *testing.T) {
	service := NewService()
	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "http://gavia.test/form", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected POST without CSRF token to be rejected with %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestMiddlewareAcceptsMatchingFormToken(t *testing.T) {
	service := NewService()
	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tokenCookie := issueTokenCookie(t, handler)
	form := url.Values{FormFieldName: {tokenCookie.Value}}

	req := httptest.NewRequest(http.MethodPost, "http://gavia.test/form", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(tokenCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected POST with matching CSRF token to pass through with %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestMiddlewareRejectsCrossOriginUnsafeRequest(t *testing.T) {
	service := NewService()
	handler := service.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tokenCookie := issueTokenCookie(t, handler)
	form := url.Values{FormFieldName: {tokenCookie.Value}}

	req := httptest.NewRequest(http.MethodPost, "http://gavia.test/form", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(tokenCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected cross-origin POST to be rejected with %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func issueTokenCookie(t *testing.T, handler http.Handler) *http.Cookie {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "http://gavia.test/form", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return cookieByName(t, rec.Result().Cookies(), CookieName)
}

func cookieByName(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("expected response to include cookie %q", name)
	return nil
}
