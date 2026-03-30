package appsettings

import (
	"html/template"
	"io"
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
	Title         string
	EntryType     string
	AccountName   string
	Category      string
	Counterparty  string
	Scope         string
	Amount        string
	Currency      string
	OccurredOn    string
	DueOn         string
	PaidOn        string
	PaymentMethod string
	Notes         string
}

type expenseStat struct {
	Label string
	Value string
	Hint  string
}

type pageData struct {
	ui.BaseData
	Settings           *appsetting.AppSettings
	OSChoices          []string
	Expenses           []*expenseentry.ExpenseEntry
	ExpenseStats       []expenseStat
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
	settings, canExportEncrypted, expenses, expenseStats, err := h.loadPageState(r)
	if err != nil {
		http.Error(w, "Unable to load app settings.", http.StatusInternalServerError)
		return
	}

	data := pageData{
		BaseData:           ui.NewBaseData(r, "App settings", start),
		Settings:           settings,
		Expenses:           expenses,
		ExpenseStats:       expenseStats,
		ExpenseForm:        defaultExpenseForm(settings.DefaultCurrency),
		NoticeHTML:         appSettingsNotice(r),
		Editing:            false,
		CanExportEncrypted: canExportEncrypted,
	}

	ui.WriteHTMLHeader(w)
	h.render(w, data)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	settings, canExportEncrypted, expenses, expenseStats, err := h.loadPageState(r)
	if err != nil {
		http.Error(w, "Unable to load app settings.", http.StatusInternalServerError)
		return
	}

	data := pageData{
		BaseData:           ui.NewBaseData(r, "App settings", start),
		Settings:           settings,
		OSChoices:          h.osChoices(r),
		Expenses:           expenses,
		ExpenseStats:       expenseStats,
		ExpenseForm:        defaultExpenseForm(settings.DefaultCurrency),
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
	current, canExportEncrypted, expenses, expenseStats, err := h.loadPageState(r)
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
			ExpenseStats:       expenseStats,
			ExpenseForm:        defaultExpenseForm(current.DefaultCurrency),
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
		ExpenseStats:       expenseStats,
		ExpenseForm:        defaultExpenseForm(settings.DefaultCurrency),
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
	const maxImportUploadSize = 32 << 20

	r.Body = http.MaxBytesReader(w, r.Body, maxImportUploadSize)
	if err := r.ParseMultipartForm(maxImportUploadSize); err != nil {
		http.Error(w, "Invalid backup upload payload.", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("import_file")
	if err != nil {
		http.Error(w, "Backup file is required.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	payload, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Unable to read the uploaded backup file.", http.StatusBadRequest)
		return
	}

	payload = []byte(strings.TrimSpace(string(payload)))
	if len(payload) == 0 {
		http.Error(w, "Backup file is empty.", http.StatusBadRequest)
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
	settings, canExportEncrypted, expenses, expenseStats, err := h.loadPageState(r)
	if err != nil {
		http.Error(w, "Unable to load app settings.", http.StatusInternalServerError)
		return
	}

	form := expenseFormData{
		Title:         strings.TrimSpace(r.Form.Get("title")),
		EntryType:     normalizeExpenseEntryType(r.Form.Get("entry_type")),
		AccountName:   normalizeText(r.Form.Get("account_name"), "cash"),
		Category:      normalizeText(r.Form.Get("category"), "manual"),
		Counterparty:  strings.TrimSpace(r.Form.Get("counterparty")),
		Scope:         normalizeExpenseScope(r.Form.Get("scope")),
		Amount:        strings.TrimSpace(r.Form.Get("amount")),
		Currency:      normalizeCurrency(r.Form.Get("currency")),
		OccurredOn:    normalizeText(r.Form.Get("occurred_on"), time.Now().Format(time.DateOnly)),
		DueOn:         strings.TrimSpace(r.Form.Get("due_on")),
		PaidOn:        strings.TrimSpace(r.Form.Get("paid_on")),
		PaymentMethod: strings.TrimSpace(r.Form.Get("payment_method")),
		Notes:         strings.TrimSpace(r.Form.Get("notes")),
	}

	data := pageData{
		BaseData:           ui.NewBaseData(r, "App settings", start),
		Settings:           settings,
		Expenses:           expenses,
		ExpenseStats:       expenseStats,
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
		Title:         form.Title,
		EntryType:     form.EntryType,
		AccountName:   form.AccountName,
		Category:      form.Category,
		Counterparty:  optionalString(form.Counterparty),
		Scope:         form.Scope,
		Amount:        amount,
		Currency:      normalizeCurrency(form.Currency),
		OccurredOn:    form.OccurredOn,
		DueOn:         optionalString(form.DueOn),
		PaidOn:        optionalString(form.PaidOn),
		PaymentMethod: optionalString(form.PaymentMethod),
		Notes:         optionalString(form.Notes),
	}
	if entry.Currency == "" {
		entry.Currency = settings.DefaultCurrency
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

func (h *Handler) loadPageState(r *http.Request) (*appsetting.AppSettings, bool, []*expenseentry.ExpenseEntry, []expenseStat, error) {
	settings, err := h.appRepo.Get(r.Context())
	if err != nil {
		return nil, false, nil, nil, err
	}
	if settings == nil {
		settings = appsetting.Defaults()
	}

	account, err := h.accountRepo.Get(r.Context())
	if err != nil {
		return nil, false, nil, nil, err
	}

	expenses, err := h.expenseRepo.GetRecent(r.Context(), 20)
	if err != nil {
		return nil, false, nil, nil, err
	}

	allEntries, err := h.expenseRepo.GetAll(r.Context())
	if err != nil {
		return nil, false, nil, nil, err
	}

	canExportEncrypted := account != nil && strings.TrimSpace(account.RecoveryPublicKey) != ""
	return settings, canExportEncrypted, expenses, buildExpenseStats(allEntries), nil
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
		return ui.BannerHTML("settings-alert", "ok", "Ledger entry created successfully.")
	case "expense-deleted":
		return ui.BannerHTML("settings-alert", "ok", "Ledger entry deleted successfully.")
	}

	return ""
}

func defaultExpenseForm(currency string) expenseFormData {
	return expenseFormData{
		EntryType:   "expense",
		AccountName: "cash",
		Category:    "manual",
		Scope:       "infrastructure",
		Currency:    normalizeCurrency(currency),
		OccurredOn:  time.Now().Format(time.DateOnly),
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return &value
}

func normalizeExpenseEntryType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "income", "transfer", "refund":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "expense"
	}
}

func normalizeExpenseScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "household", "office", "personal", "other":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "infrastructure"
	}
}

func buildExpenseStats(items []*expenseentry.ExpenseEntry) []expenseStat {
	counts := map[string]int{
		"expense":  0,
		"income":   0,
		"transfer": 0,
		"refund":   0,
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		counts[item.EntryTypeValue()]++
	}

	return []expenseStat{
		{Label: "Recorded entries", Value: strconv.Itoa(len(items)), Hint: "Everything currently stored in the internal ledger."},
		{Label: "Expenses", Value: strconv.Itoa(counts["expense"]), Hint: "Outgoing payments such as bills, groceries, or renewals."},
		{Label: "Income and refunds", Value: strconv.Itoa(counts["income"] + counts["refund"]), Hint: "Money received back or incoming cash flow."},
		{Label: "Transfers", Value: strconv.Itoa(counts["transfer"]), Hint: "Moves between your own accounts, wallets, or cash boxes."},
	}
}
