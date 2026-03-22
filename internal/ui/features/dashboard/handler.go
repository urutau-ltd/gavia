package dashboard

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"runtime"
	"time"
)

type Handler struct {
	logger *slog.Logger
	tmpl   *template.Template
}

func NewHandler(l *slog.Logger, uiFS fs.FS) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/dashboard/views/index.html",
		"components/*.html",
	))

	return &Handler{
		logger: l,
		tmpl:   t,
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Loading Dashboard module...")

	start := time.Now()

	data := map[string]any{
		"Title": "Panel de Control",
		"FooterData": map[string]any{
			"RenderTime":  fmt.Sprintf("%.4fs", time.Since(start).Seconds()),
			"AileVersion": "v1.1.0",
			"GoVersion":   runtime.Version(),
		},
	}

	w.Header().Set("Content-Type", "text/html")

	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Error at rendering ", "err", err)
	}
}
