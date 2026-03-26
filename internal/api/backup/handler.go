package backupapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"codeberg.org/urutau-ltd/gavia/internal/backup"
	accountsetting "codeberg.org/urutau-ltd/gavia/internal/models/account_setting"
)

type Handler struct {
	logger      *slog.Logger
	backup      *backup.Service
	accountRepo *accountsetting.AccountSettingsRepository
}

func NewHandler(
	logger *slog.Logger,
	backupService *backup.Service,
	accountRepo *accountsetting.AccountSettingsRepository,
) *Handler {
	return &Handler{
		logger:      logger,
		backup:      backupService,
		accountRepo: accountRepo,
	}
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	encrypted := r.URL.Query().Get("encrypted") == "1"

	var (
		payload []byte
		err     error
	)

	if encrypted {
		account, accountErr := h.accountRepo.Get(r.Context())
		if accountErr != nil {
			http.Error(w, "Unable to load account settings for encrypted export.", http.StatusInternalServerError)
			return
		}
		if account == nil || strings.TrimSpace(account.RecoveryPublicKey) == "" {
			http.Error(w, "Encrypted export requires a configured recovery key.", http.StatusBadRequest)
			return
		}

		payload, err = h.backup.ExportEncryptedJSON(r.Context(), account.RecoveryPublicKey)
	} else {
		payload, err = h.backup.ExportJSON(r.Context())
	}

	if err != nil {
		h.logger.Error("Failed to export backup over API", "encrypted", encrypted, "err", err)
		http.Error(w, "Unable to export backup.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(payload)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read the request body.", http.StatusBadRequest)
		return
	}

	snapshot, err := h.backup.ParseImport(payload, r.Header.Get("X-Recovery-Key"))
	if err != nil {
		http.Error(w, "Unable to parse backup JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.backup.Import(r.Context(), snapshot); err != nil {
		h.logger.Error("Failed to import backup over API", "err", err)
		http.Error(w, "Unable to import the backup snapshot.", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"status": "imported",
		"format": snapshot.Format,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode backup import response", "err", err)
	}
}
