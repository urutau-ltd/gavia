package subscriptions

import (
	"database/sql"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	appsetting "codeberg.org/urutau-ltd/gavia/internal/models/app_setting"
	"codeberg.org/urutau-ltd/gavia/internal/models/subscription"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger           *slog.Logger
	tmpl             *template.Template
	subscriptionRepo *subscription.Repository
	appRepo          *appsetting.AppSettingsRepository
}

type pageData struct {
	ui.BaseData
	Subscriptions   []*subscription.Subscription
	Subscription    *subscription.Subscription
	DefaultCurrency string
	SearchTerm      string
	Limit           int
	EditorMode      string
	FormAction      string
	FormSubmit      string
	NoticeHTML      template.HTML
	ErrorHTML       template.HTML
}

func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/subscriptions/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger:           l,
		tmpl:             t,
		subscriptionRepo: subscription.NewRepository(db),
		appRepo:          appsetting.NewAppSettingsRepository(db),
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list subscriptions", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	ui.WriteHTMLHeader(w)
	if h.isListRequest(r) {
		h.renderTemplate(w, "subscription-list", data)
		return
	}

	if h.isEditorRequest(r) {
		h.renderTemplate(w, "subscription-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to load subscription form", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	data.EditorMode = "new"
	data.FormAction = "/subscriptions"
	data.FormSubmit = "Create subscription"
	data.Subscription = &subscription.Subscription{
		Currency: data.DefaultCurrency,
	}

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "subscription-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list subscriptions", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.subscriptionRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load subscription", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load subscription.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if item == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Subscription not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "detail"
	data.Subscription = item

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "subscription-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list subscriptions", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.subscriptionRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load subscription for edit", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load subscription for editing.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if item == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Subscription not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "edit"
	data.Subscription = item
	data.FormAction = fmt.Sprintf("/subscriptions/%s/edit", item.ID)
	data.FormSubmit = "Update subscription"

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "subscription-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list subscriptions", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	item, formErr := buildSubscriptionFromForm(r, "", data.DefaultCurrency)
	data.EditorMode = "new"
	data.FormAction = "/subscriptions"
	data.FormSubmit = "Create subscription"
	data.Subscription = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.subscriptionRepo.Create(r.Context(), item); err != nil {
		h.logger.Error("Failed to create subscription", "err", err)
		status := http.StatusInternalServerError
		msg := "Unable to create subscription."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "A subscription with that name already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedItems, listErr := h.subscriptionRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh subscription list", "err", listErr)
	} else {
		data.Subscriptions = updatedItems
	}

	data.EditorMode = "detail"
	data.Subscription = item
	data.NoticeHTML = bannerHTML("ok", "Subscription created successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "subscription-editor-response", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list subscriptions", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, formErr := buildSubscriptionFromForm(r, id, data.DefaultCurrency)
	data.EditorMode = "edit"
	data.FormAction = fmt.Sprintf("/subscriptions/%s/edit", id)
	data.FormSubmit = "Update subscription"
	data.Subscription = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	current, findErr := h.subscriptionRepo.GetByID(r.Context(), id)
	if findErr != nil {
		h.logger.Error("Failed to load subscription for update", "id", id, "err", findErr)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to validate subscription before update.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if current == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Subscription not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.subscriptionRepo.Update(r.Context(), item); err != nil {
		h.logger.Error("Failed to update subscription", "id", id, "err", err)
		status := http.StatusInternalServerError
		msg := "Unable to update subscription."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "A subscription with that name already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	refreshed, getErr := h.subscriptionRepo.GetByID(r.Context(), id)
	if getErr != nil {
		h.logger.Error("Failed to reload subscription after update", "id", id, "err", getErr)
	} else if refreshed != nil {
		data.Subscription = refreshed
	}

	updatedItems, listErr := h.subscriptionRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh subscription list", "err", listErr)
	} else {
		data.Subscriptions = updatedItems
	}

	data.EditorMode = "detail"
	data.NoticeHTML = bannerHTML("ok", "Subscription updated successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "subscription-editor-response", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list subscriptions", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	if err := h.subscriptionRepo.Delete(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete subscription", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to delete subscription.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "subscription-editor-response", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedItems, listErr := h.subscriptionRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh subscription list", "err", listErr)
	} else {
		data.Subscriptions = updatedItems
	}

	data.EditorMode = "flash"
	data.NoticeHTML = bannerHTML("ok", "Subscription deleted successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "subscription-editor-response", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) loadPageData(r *http.Request, start time.Time) (pageData, error) {
	searchTerm, limit := ui.ParseListState(r)
	items, err := h.subscriptionRepo.GetAll(r.Context(), searchTerm, limit)
	if err != nil {
		return pageData{}, err
	}

	settings, err := h.appRepo.Get(r.Context())
	if err != nil {
		return pageData{}, err
	}

	defaultCurrency := "MXN"
	if settings != nil {
		defaultCurrency = settings.DefaultCurrency
	}

	return pageData{
		BaseData:        ui.NewBaseData(r, "Subscriptions", start),
		Subscriptions:   items,
		DefaultCurrency: defaultCurrency,
		SearchTerm:      searchTerm,
		Limit:           limit,
		EditorMode:      "empty",
		FormAction:      "/subscriptions",
		FormSubmit:      "Create subscription",
	}, nil
}

func (h *Handler) renderTemplate(w http.ResponseWriter, tmpl string, data any) {
	if err := h.tmpl.ExecuteTemplate(w, tmpl, data); err != nil {
		h.logger.Error("Error rendering template", "template", tmpl, "err", err)
	}
}

func (h *Handler) isListRequest(r *http.Request) bool {
	return ui.IsHTMXListRequest(
		r,
		"subscriptions-body",
		[]string{"subscription-limit", "subscription-search"},
		[]string{"limit", "q"},
	)
}

func (h *Handler) isEditorRequest(r *http.Request) bool {
	return ui.IsHTMXEditorRequest(r, "subscription-editor")
}

func buildSubscriptionFromForm(r *http.Request, id string, defaultCurrency string) (*subscription.Subscription, error) {
	name := strings.TrimSpace(r.Form.Get("name"))
	subscriptionType := strings.TrimSpace(r.Form.Get("type"))
	price, err := ui.ParseOptionalFloat(r.Form.Get("price"))
	if err != nil {
		return &subscription.Subscription{
			ID:            id,
			Name:          name,
			Type:          subscriptionType,
			Currency:      ui.NormalizeCurrency(r.Form.Get("currency"), defaultCurrency),
			RenewalPeriod: ui.OptionalString(strings.TrimSpace(r.Form.Get("renewal_period"))),
			Notes:         ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
		}, fmt.Errorf("Price must be a valid decimal value")
	}

	dueDate, err := ui.ParseOptionalDate(r.Form.Get("due_date"))
	if err != nil {
		return &subscription.Subscription{
			ID:            id,
			Name:          name,
			Type:          subscriptionType,
			Price:         price,
			Currency:      ui.NormalizeCurrency(r.Form.Get("currency"), defaultCurrency),
			RenewalPeriod: ui.OptionalString(strings.TrimSpace(r.Form.Get("renewal_period"))),
			Notes:         ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
		}, fmt.Errorf("Due date must use the YYYY-MM-DD format")
	}

	sinceDate, err := ui.ParseOptionalDate(r.Form.Get("since_date"))
	if err != nil {
		return &subscription.Subscription{
			ID:            id,
			Name:          name,
			Type:          subscriptionType,
			Price:         price,
			Currency:      ui.NormalizeCurrency(r.Form.Get("currency"), defaultCurrency),
			DueDate:       dueDate,
			RenewalPeriod: ui.OptionalString(strings.TrimSpace(r.Form.Get("renewal_period"))),
			Notes:         ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
		}, fmt.Errorf("Since date must use the YYYY-MM-DD format")
	}

	item := &subscription.Subscription{
		ID:            id,
		Name:          name,
		Type:          subscriptionType,
		Price:         price,
		Currency:      ui.NormalizeCurrency(r.Form.Get("currency"), defaultCurrency),
		DueDate:       dueDate,
		SinceDate:     sinceDate,
		RenewalPeriod: ui.OptionalString(strings.TrimSpace(r.Form.Get("renewal_period"))),
		Notes:         ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
	}

	switch {
	case item.Name == "":
		return item, fmt.Errorf("Name is required")
	case item.Type == "":
		return item, fmt.Errorf("Type is required")
	default:
		return item, nil
	}
}

func bannerHTML(kind, msg string) template.HTML {
	className := "info"
	switch kind {
	case "ok", "bad", "warn", "info":
		className = kind
	}

	escaped := html.EscapeString(msg)
	return template.HTML(`<p class="crud-alert ` + className + `">` + escaped + `</p>`)
}
