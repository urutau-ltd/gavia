package licenses

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger *slog.Logger
	tmpl   *template.Template
}

func NewHandler(logger *slog.Logger, uiFS fs.FS) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/licenses/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger: logger,
		tmpl:   t,
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ui.BaseData
	}{
		BaseData: ui.NewBaseData(r, "Licenses", time.Now()),
	}

	ui.WriteHTMLHeader(w)
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Failed to render licenses page", "err", err)
	}
}
