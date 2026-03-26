package csrf

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"codeberg.org/urutau-ltd/gavia/internal/security"
)

const (
	CookieName    = "gavia_csrf"
	HeaderName    = "X-CSRF-Token"
	FormFieldName = "_csrf"
	cookieTTL     = 14 * 24 * time.Hour
)

type tokenContextKey struct{}

type Service struct {
	protection *http.CrossOriginProtection
}

func NewService() *Service {
	protection := http.NewCrossOriginProtection()
	return &Service{protection: protection}
}

func (s *Service) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := s.ensureToken(w, r)
			if err != nil {
				http.Error(w, "Unable to initialize CSRF protection.", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), tokenContextKey{}, token)
			r = r.WithContext(ctx)

			if shouldBypass(r) {
				next.ServeHTTP(w, r)
				return
			}

			if err := s.protection.Check(r); err != nil {
				http.Error(w, "Cross-origin request rejected.", http.StatusForbidden)
				return
			}

			if !tokenMatches(r, token) {
				http.Error(w, "CSRF token rejected.", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func TokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(tokenContextKey{}).(string)
	return token
}

func (s *Service) ensureToken(w http.ResponseWriter, r *http.Request) (string, error) {
	if cookie, err := r.Cookie(CookieName); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token, nil
		}
	}

	token, _, _, err := security.GenerateToken()
	if err != nil {
		return "", err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
		Expires:  time.Now().Add(cookieTTL),
		MaxAge:   int(cookieTTL.Seconds()),
	})

	return token, nil
}

func shouldBypass(r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/static/") {
		return true
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}

	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		return true
	}

	return strings.TrimSpace(r.Header.Get("X-API-Token")) != ""
}

func tokenMatches(r *http.Request, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return false
	}

	submitted := strings.TrimSpace(r.Header.Get(HeaderName))
	if submitted == "" {
		_ = r.ParseForm()
		submitted = strings.TrimSpace(r.Form.Get(FormFieldName))
	}
	if submitted == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(expected), []byte(submitted)) == 1
}
