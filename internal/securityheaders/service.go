package securityheaders

import (
	"net/http"

	"codeberg.org/urutau-ltd/aile/v2"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Middleware() aile.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := w.Header()
			headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")
			headers.Set("Permissions-Policy", "accelerometer=(), autoplay=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")
			headers.Set("Content-Security-Policy", "base-uri 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none'")

			if r != nil && r.TLS != nil {
				headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}
