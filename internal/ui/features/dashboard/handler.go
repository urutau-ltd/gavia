package dashboard

import (
	"database/sql"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger       *slog.Logger
	tmpl         *template.Template
	providerRepo *provider.ProviderRepository
}

func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/dashboard/views/index.html",
		"components/*.html",
	))

	return &Handler{
		logger:       l,
		tmpl:         t,
		providerRepo: provider.NewProviderRepository(db),
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Loading Dashboard module...")

	providers, err := h.providerRepo.GetAll(r.Context(), "", 5)

	if err != nil {
		h.logger.Error("Failed to get providers!")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	h.logger.Info(`Database retrieved providers: `, slog.Int("count", len(providers)))

	start := time.Now()

	data := struct {
		ui.BaseData
		Providers     []*provider.Provider
		ProviderCount int
	}{
		BaseData:      ui.NewBaseData("Dashboard", start),
		Providers:     providers,
		ProviderCount: len(providers),
	}

	h.logger.Info(`Page data: `, slog.Any("data", data))

	w.Header().Set("Content-Type", "text/html")

	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Error at rendering ", "err", err)
	}
}
