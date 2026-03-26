package accountsettings

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"codeberg.org/urutau-ltd/gavia/internal/auth"
	accountsetting "codeberg.org/urutau-ltd/gavia/internal/models/account_setting"
	"codeberg.org/urutau-ltd/gavia/internal/security"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

var avatars = []string{
	"/static/img/avatar-1.svg",
	"/static/img/avatar-2.svg",
	"/static/img/avatar-3.svg",
	"/static/img/avatar-4.svg",
	"/static/img/avatar-5.svg",
}

type Handler struct {
	logger      *slog.Logger
	tmpl        *template.Template
	accountRepo *accountsetting.AccountSettingsRepository
	auth        *auth.Service
}

type avatarOption struct {
	Path     string
	Selected bool
}

type pageData struct {
	ui.BaseData
	Account              *accountsetting.AccountSettings
	Avatars              []avatarOption
	NoticeHTML           template.HTML
	ErrorHTML            template.HTML
	GeneratedAPIToken    string
	GeneratedRecoveryKey string
	SetupMode            bool
	Editing              bool
}

func NewHandler(
	logger *slog.Logger,
	uiFS fs.FS,
	accountRepo *accountsetting.AccountSettingsRepository,
	authService *auth.Service,
) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/account_settings/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger:      logger,
		tmpl:        t,
		accountRepo: accountRepo,
		auth:        authService,
	}
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	account, err := h.accountRepo.Get(r.Context())
	if err != nil {
		http.Error(w, "Unable to load account settings.", http.StatusInternalServerError)
		return
	}

	if account == nil {
		http.Redirect(w, r, auth.SetupPath, http.StatusSeeOther)
		return
	}

	data := pageData{
		BaseData:   ui.NewBaseData(r, "Account settings", start),
		Account:    account,
		Avatars:    avatarOptions(account.AvatarPath),
		NoticeHTML: accountNotice(r),
		Editing:    false,
	}

	ui.WriteHTMLHeader(w)
	h.render(w, "base", data)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	account, err := h.accountRepo.Get(r.Context())
	if err != nil {
		http.Error(w, "Unable to load account settings.", http.StatusInternalServerError)
		return
	}

	if account == nil {
		account = &accountsetting.AccountSettings{
			ID:         "account",
			AvatarPath: avatars[0],
		}
	}

	data := pageData{
		BaseData:  ui.NewBaseData(r, "Account settings", start),
		Account:   account,
		Avatars:   avatarOptions(account.AvatarPath),
		SetupMode: account.CreatedAt.IsZero(),
		Editing:   true,
	}

	ui.WriteHTMLHeader(w)
	h.render(w, "base", data)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form payload.", http.StatusBadRequest)
		return
	}

	start := time.Now()
	current, err := h.accountRepo.Get(r.Context())
	if err != nil {
		http.Error(w, "Unable to load account settings.", http.StatusInternalServerError)
		return
	}

	setupMode := current == nil
	username := strings.TrimSpace(r.Form.Get("username"))
	avatarPath := sanitizeAvatarPath(r.Form.Get("avatar_path"))
	password := r.Form.Get("password")
	confirmPassword := r.Form.Get("confirm_password")
	regenerateToken := r.Form.Get("regenerate_api_token") == "1"
	rotateRecovery := r.Form.Get("rotate_recovery_key") == "1"

	account := &accountsetting.AccountSettings{
		ID:         "account",
		Username:   username,
		AvatarPath: avatarPath,
	}

	if current != nil {
		*account = *current
		account.Username = username
		account.AvatarPath = avatarPath
	}

	data := pageData{
		BaseData:  ui.NewBaseData(r, "Account settings", start),
		Account:   account,
		Avatars:   avatarOptions(avatarPath),
		SetupMode: setupMode,
		Editing:   true,
	}

	if username == "" {
		h.logger.Warn("Rejected account settings update", "reason", "missing_username", "setup_mode", setupMode)
		data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "Username is required.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, "base", data)
		return
	}

	if password != confirmPassword {
		h.logger.Warn("Rejected account settings update", "reason", "password_mismatch", "setup_mode", setupMode)
		data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "The password confirmation did not match.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, "base", data)
		return
	}

	if setupMode && strings.TrimSpace(password) == "" {
		h.logger.Warn("Rejected initial account setup", "reason", "missing_password")
		data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "The first administrator password is required.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, "base", data)
		return
	}

	if strings.TrimSpace(password) != "" {
		passwordHash, err := security.HashPassword(password)
		if err != nil {
			h.logger.Warn("Rejected account settings password", "reason", err.Error(), "setup_mode", setupMode)
			data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", err.Error())
			ui.WriteHTMLHeader(w)
			w.WriteHeader(http.StatusBadRequest)
			h.render(w, "base", data)
			return
		}
		account.PasswordHash = passwordHash
	}

	if setupMode || strings.TrimSpace(account.APITokenHash) == "" || regenerateToken {
		plainToken, hashedToken, hint, err := security.GenerateToken()
		if err != nil {
			http.Error(w, "Unable to generate a new API token.", http.StatusInternalServerError)
			return
		}
		account.APITokenHash = hashedToken
		account.APITokenHint = hint
		data.GeneratedAPIToken = plainToken
	}

	if setupMode || strings.TrimSpace(account.RecoveryPublicKey) == "" || rotateRecovery {
		publicKey, recoveryKey, err := security.GenerateRecoveryKeyPair()
		if err != nil {
			http.Error(w, "Unable to generate a recovery key.", http.StatusInternalServerError)
			return
		}
		account.RecoveryPublicKey = publicKey
		data.GeneratedRecoveryKey = recoveryKey
	}

	if setupMode {
		if err := h.accountRepo.Create(r.Context(), account); err != nil {
			h.logger.Error("Failed to create initial account settings", "err", err)
			data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "Unable to create the administrator account.")
			ui.WriteHTMLHeader(w)
			w.WriteHeader(http.StatusInternalServerError)
			h.render(w, "base", data)
			return
		}

		if err := h.auth.StartSession(w, r); err != nil {
			h.logger.Error("Failed to start setup session", "err", err)
			data.ErrorHTML = ui.BannerHTML("settings-alert", "warn", "Account created, but the initial session could not be started. Please sign in manually.")
			ui.WriteHTMLHeader(w)
			w.WriteHeader(http.StatusInternalServerError)
			h.render(w, "base", data)
			return
		}

		data.SetupMode = false
		data.Account = account
		data.Avatars = avatarOptions(account.AvatarPath)
		data.NoticeHTML = ui.BannerHTML("settings-alert", "ok", "Administrator account created successfully.")
		ui.WriteHTMLHeader(w)
		h.render(w, "base", data)
		return
	}

	if err := h.accountRepo.Update(r.Context(), account); err != nil {
		h.logger.Error("Failed to update account settings", "err", err)
		data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "Unable to update account settings.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		h.render(w, "base", data)
		return
	}

	data.Account = account
	data.Avatars = avatarOptions(account.AvatarPath)
	data.NoticeHTML = ui.BannerHTML("settings-alert", "ok", "Account settings updated successfully.")
	ui.WriteHTMLHeader(w)
	h.render(w, "base", data)
}

func (h *Handler) render(w http.ResponseWriter, tmpl string, data pageData) {
	if err := h.tmpl.ExecuteTemplate(w, tmpl, data); err != nil {
		h.logger.Error("Failed to render account settings", "template", tmpl, "err", err)
	}
}

func accountNotice(r *http.Request) template.HTML {
	if r.URL.Query().Get("notice") == "password-recovered" {
		return ui.BannerHTML("settings-alert", "ok", "Password recovered successfully.")
	}

	return ""
}

func avatarOptions(selected string) []avatarOption {
	options := make([]avatarOption, 0, len(avatars))
	for _, path := range avatars {
		options = append(options, avatarOption{
			Path:     path,
			Selected: path == sanitizeAvatarPath(selected),
		})
	}

	return options
}

func sanitizeAvatarPath(value string) string {
	value = strings.TrimSpace(value)
	for _, candidate := range avatars {
		if value == candidate {
			return value
		}
	}

	return avatars[0]
}
