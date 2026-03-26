package domains

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
	"codeberg.org/urutau-ltd/gavia/internal/models/domain"
	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger       *slog.Logger
	tmpl         *template.Template
	domainRepo   *domain.Repository
	providerRepo *provider.ProviderRepository
	appRepo      *appsetting.AppSettingsRepository
}

type pageData struct {
	ui.BaseData
	Domains         []*domain.Domain
	Domain          *domain.Domain
	ProviderOptions []ui.SelectOption
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
		"features/domains/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger:       l,
		tmpl:         t,
		domainRepo:   domain.NewRepository(db),
		providerRepo: provider.NewProviderRepository(db),
		appRepo:      appsetting.NewAppSettingsRepository(db),
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list domains", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	ui.WriteHTMLHeader(w)
	if h.isListRequest(r) {
		h.renderTemplate(w, "domain-list", data)
		return
	}

	if h.isEditorRequest(r) {
		h.renderTemplate(w, "domain-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to load domain form", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	data.EditorMode = "new"
	data.FormAction = "/domains"
	data.FormSubmit = "Create domain"
	data.Domain = &domain.Domain{
		Currency: data.DefaultCurrency,
	}

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "domain-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list domains", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.domainRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load domain", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load domain.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if item == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Domain not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "detail"
	data.Domain = item

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "domain-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list domains", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.domainRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load domain for edit", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load domain for editing.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if item == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Domain not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "edit"
	data.Domain = item
	data.FormAction = fmt.Sprintf("/domains/%s/edit", item.ID)
	data.FormSubmit = "Update domain"

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "domain-editor-panel", data)
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
		h.logger.Error("Failed to list domains", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	item, formErr := buildDomainFromForm(r, "", data.DefaultCurrency)
	data.EditorMode = "new"
	data.FormAction = "/domains"
	data.FormSubmit = "Create domain"
	data.Domain = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.domainRepo.Create(r.Context(), item); err != nil {
		h.logger.Error("Failed to create domain", "err", err)
		status := http.StatusInternalServerError
		msg := "Unable to create domain."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "A domain with that name already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedItems, listErr := h.domainRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh domain list", "err", listErr)
	} else {
		data.Domains = updatedItems
	}

	data.EditorMode = "detail"
	data.Domain = item
	data.NoticeHTML = bannerHTML("ok", "Domain created successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "domain-editor-response", data)
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
		h.logger.Error("Failed to list domains", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, formErr := buildDomainFromForm(r, id, data.DefaultCurrency)
	data.EditorMode = "edit"
	data.FormAction = fmt.Sprintf("/domains/%s/edit", id)
	data.FormSubmit = "Update domain"
	data.Domain = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	current, findErr := h.domainRepo.GetByID(r.Context(), id)
	if findErr != nil {
		h.logger.Error("Failed to load domain for update", "id", id, "err", findErr)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to validate domain before update.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if current == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Domain not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.domainRepo.Update(r.Context(), item); err != nil {
		h.logger.Error("Failed to update domain", "id", id, "err", err)
		status := http.StatusInternalServerError
		msg := "Unable to update domain."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "A domain with that name already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	refreshed, getErr := h.domainRepo.GetByID(r.Context(), id)
	if getErr != nil {
		h.logger.Error("Failed to reload domain after update", "id", id, "err", getErr)
	} else if refreshed != nil {
		data.Domain = refreshed
	}

	updatedItems, listErr := h.domainRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh domain list", "err", listErr)
	} else {
		data.Domains = updatedItems
	}

	data.EditorMode = "detail"
	data.NoticeHTML = bannerHTML("ok", "Domain updated successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "domain-editor-response", data)
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
		h.logger.Error("Failed to list domains", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	if err := h.domainRepo.Delete(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete domain", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to delete domain.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "domain-editor-response", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedItems, listErr := h.domainRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh domain list", "err", listErr)
	} else {
		data.Domains = updatedItems
	}

	data.EditorMode = "flash"
	data.NoticeHTML = bannerHTML("ok", "Domain deleted successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "domain-editor-response", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) loadPageData(r *http.Request, start time.Time) (pageData, error) {
	searchTerm, limit := ui.ParseListState(r)
	items, err := h.domainRepo.GetAll(r.Context(), searchTerm, limit)
	if err != nil {
		return pageData{}, err
	}

	providers, err := h.providerRepo.GetAll(r.Context(), "", 100)
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
		BaseData:        ui.NewBaseData(r, "Domains", start),
		Domains:         items,
		ProviderOptions: ui.BuildSelectOptions(providers, func(item *provider.Provider) string { return item.Id }, func(item *provider.Provider) string { return item.Name }),
		DefaultCurrency: defaultCurrency,
		SearchTerm:      searchTerm,
		Limit:           limit,
		EditorMode:      "empty",
		FormAction:      "/domains",
		FormSubmit:      "Create domain",
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
		"domains-body",
		[]string{"domain-limit", "domain-search"},
		[]string{"limit", "q"},
	)
}

func (h *Handler) isEditorRequest(r *http.Request) bool {
	return ui.IsHTMXEditorRequest(r, "domain-editor")
}

func buildDomainFromForm(r *http.Request, id string, defaultCurrency string) (*domain.Domain, error) {
	name := strings.TrimSpace(r.Form.Get("domain"))
	providerID := strings.TrimSpace(r.Form.Get("provider_id"))
	dueDate, err := ui.ParseOptionalDate(r.Form.Get("due_date"))
	if err != nil {
		return &domain.Domain{
			ID:         id,
			Domain:     name,
			ProviderID: ui.OptionalString(providerID),
			DueDate:    dueDate,
			Currency:   ui.NormalizeCurrency(r.Form.Get("currency"), defaultCurrency),
			Notes:      ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
		}, fmt.Errorf("Due date must use the YYYY-MM-DD format")
	}

	price, err := ui.ParseOptionalFloat(r.Form.Get("price"))
	if err != nil {
		return &domain.Domain{
			ID:         id,
			Domain:     name,
			ProviderID: ui.OptionalString(providerID),
			DueDate:    dueDate,
			Currency:   ui.NormalizeCurrency(r.Form.Get("currency"), defaultCurrency),
			Notes:      ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
		}, fmt.Errorf("Price must be a valid decimal value")
	}

	item := &domain.Domain{
		ID:         id,
		Domain:     name,
		ProviderID: ui.OptionalString(providerID),
		DueDate:    dueDate,
		Currency:   ui.NormalizeCurrency(r.Form.Get("currency"), defaultCurrency),
		Price:      price,
		Notes:      ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
	}

	if item.Domain == "" {
		return item, fmt.Errorf("Domain is required")
	}

	return item, nil
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
