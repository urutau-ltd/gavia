package ledger

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	appsetting "codeberg.org/urutau-ltd/gavia/internal/models/app_setting"
	expenseentry "codeberg.org/urutau-ltd/gavia/internal/models/expense_entry"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger      *slog.Logger
	tmpl        *template.Template
	appRepo     *appsetting.AppSettingsRepository
	expenseRepo *expenseentry.ExpenseEntryRepository
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
	Settings     *appsetting.AppSettings
	Expenses     []*expenseentry.ExpenseEntry
	ExpenseStats []expenseStat
	ExpenseForm  expenseFormData
	NoticeHTML   template.HTML
	ErrorHTML    template.HTML
}

func NewHandler(
	logger *slog.Logger,
	uiFS fs.FS,
	appRepo *appsetting.AppSettingsRepository,
	expenseRepo *expenseentry.ExpenseEntryRepository,
) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/ledger/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger:      logger,
		tmpl:        t,
		appRepo:     appRepo,
		expenseRepo: expenseRepo,
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	settings, expenses, expenseStats, err := h.loadPageState(r)
	if err != nil {
		http.Error(w, "Unable to load ledger.", http.StatusInternalServerError)
		return
	}

	data := pageData{
		BaseData:     ui.NewBaseData(r, "Ledger", start),
		Settings:     settings,
		Expenses:     expenses,
		ExpenseStats: expenseStats,
		ExpenseForm:  defaultExpenseForm(settings.DefaultCurrency),
		NoticeHTML:   ledgerNotice(r),
	}

	ui.WriteHTMLHeader(w)
	h.render(w, data)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form payload.", http.StatusBadRequest)
		return
	}

	start := time.Now()
	settings, expenses, expenseStats, err := h.loadPageState(r)
	if err != nil {
		http.Error(w, "Unable to load ledger.", http.StatusInternalServerError)
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
		BaseData:     ui.NewBaseData(r, "Ledger", start),
		Settings:     settings,
		Expenses:     expenses,
		ExpenseStats: expenseStats,
		ExpenseForm:  form,
	}

	if form.Title == "" {
		data.ErrorHTML = ui.BannerHTML("ledger-alert", "bad", "Entry title is required.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, data)
		return
	}

	amount, err := strconv.ParseFloat(form.Amount, 64)
	if err != nil || amount <= 0 {
		data.ErrorHTML = ui.BannerHTML("ledger-alert", "bad", "Entry amount must be greater than zero.")
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
		h.logger.Error("Failed to create ledger entry", "err", err)
		data.ErrorHTML = ui.BannerHTML("ledger-alert", "bad", "Unable to create the ledger entry.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		h.render(w, data)
		return
	}

	http.Redirect(w, r, "/ledger?notice=created", http.StatusSeeOther)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "Ledger entry id is required.", http.StatusBadRequest)
		return
	}

	if err := h.expenseRepo.Delete(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete ledger entry", "id", id, "err", err)
		http.Error(w, "Unable to delete the ledger entry.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/ledger?notice=deleted", http.StatusSeeOther)
}

func (h *Handler) render(w http.ResponseWriter, data pageData) {
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Failed to render ledger", "err", err)
	}
}

func (h *Handler) loadPageState(r *http.Request) (*appsetting.AppSettings, []*expenseentry.ExpenseEntry, []expenseStat, error) {
	settings, err := h.appRepo.Get(r.Context())
	if err != nil {
		return nil, nil, nil, err
	}
	if settings == nil {
		settings = appsetting.Defaults()
	}

	expenses, err := h.expenseRepo.GetRecent(r.Context(), 50)
	if err != nil {
		return nil, nil, nil, err
	}

	allEntries, err := h.expenseRepo.GetAll(r.Context())
	if err != nil {
		return nil, nil, nil, err
	}

	return settings, expenses, buildExpenseStats(allEntries), nil
}

func ledgerNotice(r *http.Request) template.HTML {
	switch r.URL.Query().Get("notice") {
	case "created":
		return ui.BannerHTML("ledger-alert", "ok", "Ledger entry created successfully.")
	case "deleted":
		return ui.BannerHTML("ledger-alert", "ok", "Ledger entry deleted successfully.")
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
