package providers

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
)

type Handler struct {
	logger       *slog.Logger
	tmpl         *template.Template
	providerRepo *provider.ProviderRepository
}

func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/providers/views/index.html",
		"components/*.html",
	))

	return &Handler{
		logger:       l,
		tmpl:         t,
		providerRepo: provider.NewProviderRepository(db),
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Loading Providers module...")

	providers, err := h.providerRepo.GetAll(r.Context())

	if err != nil {
		h.logger.Error("Failed to get providers!")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.logger.Info(`Database retrieved providers: `,
		slog.Int("count", len(providers)))

	start := time.Now()

	data := map[string]any{
		"Title":         "Providers",
		"Providers":     providers,
		"ProviderCount": len(providers),
		"FooterData": map[string]any{
			"RenderTime":  fmt.Sprintf("%.2fs", time.Since(start).Seconds()),
			"AileVersion": "v1.1.0",
			"GoVersion":   strings.Trim(runtime.Version(), "go"),
		},
	}

	h.logger.Info(`Page data: `, slog.Any("data", data))
	w.Header().Set("Content-Type", "text/html")

	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Error at rendering ", "err", err)
	}
}
