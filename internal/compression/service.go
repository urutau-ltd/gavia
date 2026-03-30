package compression

import (
	"compress/gzip"
	"net/http"
	"path/filepath"
	"strings"

	"codeberg.org/urutau-ltd/aile/v2"
	xcompress "codeberg.org/urutau-ltd/aile/v2/x/compress"
)

var skippedExtensions = map[string]struct{}{
	".7z":    {},
	".avi":   {},
	".avif":  {},
	".bz2":   {},
	".gif":   {},
	".gz":    {},
	".ico":   {},
	".jpeg":  {},
	".jpg":   {},
	".m4a":   {},
	".m4v":   {},
	".mkv":   {},
	".mov":   {},
	".mp3":   {},
	".mp4":   {},
	".ogg":   {},
	".pdf":   {},
	".png":   {},
	".rar":   {},
	".tar":   {},
	".tgz":   {},
	".wav":   {},
	".webm":  {},
	".webp":  {},
	".woff":  {},
	".woff2": {},
	".xz":    {},
	".zip":   {},
}

type Service struct {
	middleware aile.Middleware
}

func NewService() *Service {
	return &Service{
		middleware: xcompress.Middleware(xcompress.Config{
			Level:   gzip.BestCompression,
			MinSize: 256,
		}),
	}
}

func (s *Service) Middleware() aile.Middleware {
	return func(next http.Handler) http.Handler {
		compressed := s.middleware(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldSkipCompression(r) {
				next.ServeHTTP(w, r)
				return
			}

			compressed.ServeHTTP(w, r)
		})
	}
}

func shouldSkipCompression(r *http.Request) bool {
	if r == nil {
		return true
	}

	if r.Method == http.MethodHead {
		return true
	}

	if r.Header.Get("Range") != "" {
		return true
	}

	ext := strings.ToLower(filepath.Ext(r.URL.Path))
	_, skip := skippedExtensions[ext]
	return skip
}
