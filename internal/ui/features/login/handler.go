package login

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"codeberg.org/urutau-ltd/aile/v2/x/htmx"
	"codeberg.org/urutau-ltd/gavia/internal/auth"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger *slog.Logger
	tmpl   *template.Template
	auth   *auth.Service
}

type pageData struct {
	ui.BaseData
	NoticeHTML       template.HTML
	ErrorHTML        template.HTML
	Username         string
	RecoveryUsername string
	RecoveryKey      string
}

func NewHandler(logger *slog.Logger, uiFS fs.FS, authService *auth.Service) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/login/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger: logger,
		tmpl:   t,
		auth:   authService,
	}
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data := pageData{
		BaseData:   ui.NewBaseData(r, "Login", start),
		NoticeHTML: loginNotice(r),
	}

	ui.WriteHTMLHeader(w)
	h.render(w, data)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form payload.", http.StatusBadRequest)
		return
	}

	switch r.Form.Get("intent") {
	case "recover":
		h.recover(w, r)
	default:
		h.login(w, r)
	}
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	username := r.Form.Get("username")
	password := r.Form.Get("password")

	data := pageData{
		BaseData: ui.NewBaseData(r, "Login", start),
		Username: username,
	}

	account, err := h.auth.Authenticate(r.Context(), username, password)
	if err != nil {
		h.logger.Error("Failed to authenticate account", "err", err)
		data.ErrorHTML = ui.BannerHTML("auth-alert", "bad", "Unable to validate the login request.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		h.render(w, data)
		return
	}

	if account == nil {
		data.ErrorHTML = ui.BannerHTML("auth-alert", "bad", "Invalid username or password.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusUnauthorized)
		h.render(w, data)
		return
	}

	if err := h.auth.StartSession(w, r); err != nil {
		h.logger.Error("Failed to create login session", "err", err)
		data.ErrorHTML = ui.BannerHTML("auth-alert", "bad", "Unable to start a new session.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		h.render(w, data)
		return
	}

	redirectAfterLogin(w, r, auth.DashboardPath)
}

func (h *Handler) recover(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	username := r.Form.Get("recovery_username")
	recoveryKey := r.Form.Get("recovery_key")
	newPassword := r.Form.Get("new_password")
	confirmPassword := r.Form.Get("confirm_password")

	data := pageData{
		BaseData:         ui.NewBaseData(r, "Login", start),
		RecoveryUsername: username,
		RecoveryKey:      recoveryKey,
	}

	if newPassword != confirmPassword {
		data.ErrorHTML = ui.BannerHTML("auth-alert", "bad", "The new password confirmation did not match.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, data)
		return
	}

	if _, err := h.auth.ResetPasswordWithRecovery(r.Context(), username, recoveryKey, newPassword); err != nil {
		data.ErrorHTML = ui.BannerHTML("auth-alert", "bad", err.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, data)
		return
	}

	if err := h.auth.ClearAllSessions(r.Context()); err != nil {
		h.logger.Error("Failed to clear sessions after password recovery", "err", err)
		data.ErrorHTML = ui.BannerHTML("auth-alert", "bad", "Password updated, but old sessions could not be invalidated.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		h.render(w, data)
		return
	}

	if err := h.auth.StartSession(w, r); err != nil {
		h.logger.Error("Failed to start session after recovery", "err", err)
		data.ErrorHTML = ui.BannerHTML("auth-alert", "bad", "Password updated, but the new session could not be created.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		h.render(w, data)
		return
	}

	redirectAfterLogin(w, r, "/account-settings?notice=password-recovered")
}

func (h *Handler) render(w http.ResponseWriter, data pageData) {
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Failed to render login view", "err", err)
	}
}

func redirectAfterLogin(w http.ResponseWriter, r *http.Request, target string) {
	if htmx.IsRequest(r) {
		htmx.Redirect(w, target)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}

func loginNotice(r *http.Request) template.HTML {
	switch r.URL.Query().Get("notice") {
	case "logged-out":
		return ui.BoxAlertHTML("auth-alert", "ok", "You have been signed out.")
	case "imported":
		return ui.BoxAlertHTML("auth-alert", "ok", "Backup imported. Please sign in again.")
	default:
		return ""
	}
}
