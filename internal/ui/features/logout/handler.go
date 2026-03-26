package logout

import (
	"log/slog"
	"net/http"

	"codeberg.org/urutau-ltd/aile/v2/x/htmx"
	"codeberg.org/urutau-ltd/gavia/internal/auth"
)

type Handler struct {
	logger *slog.Logger
	auth   *auth.Service
}

func NewHandler(logger *slog.Logger, authService *auth.Service) *Handler {
	return &Handler{
		logger: logger,
		auth:   authService,
	}
}

func (h *Handler) Perform(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.EndSession(w, r); err != nil {
		h.logger.Error("Failed to end session", "err", err)
		http.Error(w, "Unable to close the session.", http.StatusInternalServerError)
		return
	}

	target := "/login?notice=logged-out"
	if htmx.IsRequest(r) {
		htmx.Redirect(w, target)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}
