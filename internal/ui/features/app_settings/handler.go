package appsettings

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"codeberg.org/urutau-ltd/aile/v2/x/htmx"
	"codeberg.org/urutau-ltd/gavia/internal/auth"
	"codeberg.org/urutau-ltd/gavia/internal/backup"
	accountsetting "codeberg.org/urutau-ltd/gavia/internal/models/account_setting"
	appsetting "codeberg.org/urutau-ltd/gavia/internal/models/app_setting"
	expenseentry "codeberg.org/urutau-ltd/gavia/internal/models/expense_entry"
	operatingsystem "codeberg.org/urutau-ltd/gavia/internal/models/operating_system"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger      *slog.Logger
	tmpl        *template.Template
	appRepo     *appsetting.AppSettingsRepository
	accountRepo *accountsetting.AccountSettingsRepository
	expenseRepo *expenseentry.ExpenseEntryRepository
	osRepo      *operatingsystem.OperatingSystemRepository
	backup      *backup.Service
	auth        *auth.Service
}

type expenseFormData struct {
	Title      string
	Category   string
	Amount     string
	Currency   string
	OccurredOn string
	Notes      string
}

type pageData struct {
	ui.BaseData
	Settings           *appsetting.AppSettings
	OSChoices          []string
	Expenses           []*expenseentry.ExpenseEntry
	ExpenseForm        expenseFormData
	NoticeHTML         template.HTML
	ErrorHTML          template.HTML
	Editing            bool
	CanExportEncrypted bool
}

func NewHandler(
	logger *slog.Logger,
	uiFS fs.FS,
	appRepo *appsetting.AppSettingsRepository,
	accountRepo *accountsetting.AccountSettingsRepository,
	expenseRepo *expenseentry.ExpenseEntryRepository,
	osRepo *operatingsystem.OperatingSystemRepository,
	backupService *backup.Service,
	authService *auth.Service,
) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/app_settings/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger:      logger,
		tmpl:        t,
		appRepo:     appRepo,
		accountRepo: accountRepo,
		expenseRepo: expenseRepo,
		osRepo:      osRepo,
		backup:      backupService,
		auth:        authService,
	}
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	settings, canExportEncrypted, expenses, err := h.loadPageState(r)
	if err != nil {
		http.Error(w, "Unable to load app settings.", http.StatusInternalServerError)
		return
	}

	data := pageData{
		BaseData:           ui.NewBaseData(r, "App settings", start),
		Settings:           settings,
		Expenses:           expenses,
		ExpenseForm:        defaultExpenseForm(settings.DashboardCurrency),
		NoticeHTML:         appSettingsNotice(r),
		Editing:            false,
		CanExportEncrypted: canExportEncrypted,
	}

	ui.WriteHTMLHeader(w)
	h.render(w, data)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	settings, canExportEncrypted, expenses, err := h.loadPageState(r)
	if err != nil {
		http.Error(w, "Unable to load app settings.", http.StatusInternalServerError)
		return
	}

	data := pageData{
		BaseData:           ui.NewBaseData(r, "App settings", start),
		Settings:           settings,
		OSChoices:          h.osChoices(r),
		Expenses:           expenses,
		ExpenseForm:        defaultExpenseForm(settings.DashboardCurrency),
		Editing:            true,
		CanExportEncrypted: canExportEncrypted,
	}

	ui.WriteHTMLHeader(w)
	h.render(w, data)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form payload.", http.StatusBadRequest)
		return
	}

	start := time.Now()
	current, canExportEncrypted, expenses, err := h.loadPageState(r)
	if err != nil {
		http.Error(w, "Unable to load app settings.", http.StatusInternalServerError)
		return
	}

	osChoices := h.osChoices(r)
	dueSoonAmount, err := strconv.Atoi(strings.TrimSpace(r.Form.Get("dashboard_due_soon_amount")))
	if err != nil || dueSoonAmount < 0 {
		data := pageData{
			BaseData:           ui.NewBaseData(r, "App settings", start),
			Settings:           current,
			OSChoices:          osChoices,
			Expenses:           expenses,
			ExpenseForm:        defaultExpenseForm(current.DashboardCurrency),
			Editing:            true,
			CanExportEncrypted: canExportEncrypted,
			ErrorHTML:          ui.BannerHTML("settings-alert", "bad", "Dashboard due-soon amount must be zero or greater."),
		}
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, data)
		return
	}

	settings := &appsetting.AppSettings{
		ID:                          "app",
		ShowVersionFooter:           r.Form.Get("show_version_footer") == "1",
		DefaultServerOS:             normalizeText(r.Form.Get("default_server_os"), "Linux"),
		DefaultCurrency:             normalizeCurrency(r.Form.Get("default_currency")),
		DashboardCurrency:           normalizeCurrency(r.Form.Get("dashboard_currency")),
		DashboardDueSoonAmount:      dueSoonAmount,
		DashboardExpenseHistoryJSON: "[]",
	}
	if current != nil {
		settings.CreatedAt = current.CreatedAt
	}

	data := pageData{
		BaseData:           ui.NewBaseData(r, "App settings", start),
		Settings:           settings,
		OSChoices:          osChoices,
		Expenses:           expenses,
		ExpenseForm:        defaultExpenseForm(settings.DashboardCurrency),
		Editing:            true,
		CanExportEncrypted: canExportEncrypted,
	}

	if settings.DefaultCurrency == "" || settings.DashboardCurrency == "" {
		data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "Currency codes are required.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, data)
		return
	}

	if err := h.appRepo.Update(r.Context(), settings); err != nil {
		h.logger.Error("Failed to update app settings", "err", err)
		data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "Unable to update app settings.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		h.render(w, data)
		return
	}

	ui.SetShowVersionFooter(settings.ShowVersionFooter)
	data.NoticeHTML = ui.BannerHTML("settings-alert", "ok", "App settings updated successfully.")
	ui.WriteHTMLHeader(w)
	h.render(w, data)
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
		h.logger.Error("Failed to export backup", "encrypted", encrypted, "err", err)
		http.Error(w, "Unable to export backup.", http.StatusInternalServerError)
		return
	}

	filename := "gavia-backup.json"
	if encrypted {
		filename = "gavia-backup.encrypted.json"
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write(payload)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form payload.", http.StatusBadRequest)
		return
	}

	payload := []byte(strings.TrimSpace(r.Form.Get("import_payload")))
	if len(payload) == 0 {
		http.Error(w, "Import JSON is required.", http.StatusBadRequest)
		return
	}

	recoveryKey := r.Form.Get("import_recovery_key")
	snapshot, err := h.backup.ParseImport(payload, recoveryKey)
	if err != nil {
		http.Error(w, "Unable to parse backup JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.backup.Import(r.Context(), snapshot); err != nil {
		h.logger.Error("Failed to import backup", "err", err)
		http.Error(w, "Unable to import the backup snapshot.", http.StatusInternalServerError)
		return
	}

	ui.SetShowVersionFooter(snapshot.AppSettings.ShowVersionFooter)
	if err := h.auth.EndSession(w, r); err != nil {
		h.logger.Error("Failed to clear session after import", "err", err)
		http.Error(w, "Backup imported, but the current session could not be cleared.", http.StatusInternalServerError)
		return
	}

	target := "/login?notice=imported"
	if htmx.IsRequest(r) {
		htmx.Redirect(w, target)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form payload.", http.StatusBadRequest)
		return
	}

	start := time.Now()
	settings, canExportEncrypted, expenses, err := h.loadPageState(r)
	if err != nil {
		http.Error(w, "Unable to load app settings.", http.StatusInternalServerError)
		return
	}

	form := expenseFormData{
		Title:      strings.TrimSpace(r.Form.Get("title")),
		Category:   normalizeText(r.Form.Get("category"), "manual"),
		Amount:     strings.TrimSpace(r.Form.Get("amount")),
		Currency:   normalizeCurrency(r.Form.Get("currency")),
		OccurredOn: normalizeText(r.Form.Get("occurred_on"), time.Now().Format(time.DateOnly)),
		Notes:      strings.TrimSpace(r.Form.Get("notes")),
	}

	data := pageData{
		BaseData:           ui.NewBaseData(r, "App settings", start),
		Settings:           settings,
		Expenses:           expenses,
		ExpenseForm:        form,
		NoticeHTML:         appSettingsNotice(r),
		Editing:            false,
		CanExportEncrypted: canExportEncrypted,
	}

	if form.Title == "" {
		data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "Expense title is required.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, data)
		return
	}

	amount, err := strconv.ParseFloat(form.Amount, 64)
	if err != nil || amount <= 0 {
		data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "Expense amount must be greater than zero.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, data)
		return
	}

	entry := &expenseentry.ExpenseEntry{
		Title:      form.Title,
		Category:   form.Category,
		Amount:     amount,
		Currency:   normalizeCurrency(form.Currency),
		OccurredOn: form.OccurredOn,
		Notes:      optionalString(form.Notes),
	}
	if entry.Currency == "" {
		entry.Currency = settings.DashboardCurrency
	}

	if err := h.expenseRepo.Create(r.Context(), entry); err != nil {
		h.logger.Error("Failed to create expense entry", "err", err)
		data.ErrorHTML = ui.BannerHTML("settings-alert", "bad", "Unable to create the expense entry.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		h.render(w, data)
		return
	}

	http.Redirect(w, r, "/app-settings?notice=expense-created", http.StatusSeeOther)
}

func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "Expense entry id is required.", http.StatusBadRequest)
		return
	}

	if err := h.expenseRepo.Delete(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete expense entry", "id", id, "err", err)
		http.Error(w, "Unable to delete the expense entry.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app-settings?notice=expense-deleted", http.StatusSeeOther)
}

func (h *Handler) render(w http.ResponseWriter, data pageData) {
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Failed to render app settings", "err", err)
	}
}

func (h *Handler) loadPageState(r *http.Request) (*appsetting.AppSettings, bool, []*expenseentry.ExpenseEntry, error) {
	settings, err := h.appRepo.Get(r.Context())
	if err != nil {
		return nil, false, nil, err
	}
	if settings == nil {
		settings = appsetting.Defaults()
	}

	account, err := h.accountRepo.Get(r.Context())
	if err != nil {
		return nil, false, nil, err
	}

	expenses, err := h.expenseRepo.GetRecent(r.Context(), 10)
	if err != nil {
		return nil, false, nil, err
	}

	canExportEncrypted := account != nil && strings.TrimSpace(account.RecoveryPublicKey) != ""
	return settings, canExportEncrypted, expenses, nil
}

func (h *Handler) osChoices(r *http.Request) []string {
	items, err := h.osRepo.GetAll(r.Context(), "", 0)
	if err != nil {
		h.logger.Error("Failed to load operating system choices", "err", err)
		return nil
	}

	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.Name)
	}
	return values
}

func normalizeText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func appSettingsNotice(r *http.Request) template.HTML {
	switch r.URL.Query().Get("notice") {
	case "imported":
		return ui.BannerHTML("settings-alert", "ok", "Backup imported successfully.")
	case "expense-created":
		return ui.BannerHTML("settings-alert", "ok", "Expense entry created successfully.")
	case "expense-deleted":
		return ui.BannerHTML("settings-alert", "ok", "Expense entry deleted successfully.")
	}

	return ""
}

func defaultExpenseForm(currency string) expenseFormData {
	return expenseFormData{
		Category:   "manual",
		Currency:   normalizeCurrency(currency),
		OccurredOn: time.Now().Format(time.DateOnly),
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return &value
}
