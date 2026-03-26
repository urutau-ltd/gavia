package hostings

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
	"codeberg.org/urutau-ltd/gavia/internal/models/hosting"
	"codeberg.org/urutau-ltd/gavia/internal/models/location"
	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger       *slog.Logger
	tmpl         *template.Template
	hostingRepo  *hosting.Repository
	locationRepo *location.LocationRepository
	providerRepo *provider.ProviderRepository
	domainRepo   *domain.Repository
	appRepo      *appsetting.AppSettingsRepository
}

type pageData struct {
	ui.BaseData
	Hostings        []*hosting.Hosting
	Hosting         *hosting.Hosting
	LocationOptions []ui.SelectOption
	ProviderOptions []ui.SelectOption
	DomainOptions   []ui.SelectOption
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
		"features/hostings/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger:       l,
		tmpl:         t,
		hostingRepo:  hosting.NewRepository(db),
		locationRepo: location.NewLocationRepository(db),
		providerRepo: provider.NewProviderRepository(db),
		domainRepo:   domain.NewRepository(db),
		appRepo:      appsetting.NewAppSettingsRepository(db),
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list hostings", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	ui.WriteHTMLHeader(w)
	if h.isListRequest(r) {
		h.renderTemplate(w, "hosting-list", data)
		return
	}

	if h.isEditorRequest(r) {
		h.renderTemplate(w, "hosting-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to load hosting form", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	data.EditorMode = "new"
	data.FormAction = "/hostings"
	data.FormSubmit = "Create hosting"
	data.Hosting = &hosting.Hosting{
		Currency: data.DefaultCurrency,
	}

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "hosting-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list hostings", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.hostingRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load hosting", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load hosting.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if item == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Hosting not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "detail"
	data.Hosting = item

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "hosting-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list hostings", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.hostingRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load hosting for edit", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load hosting for editing.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if item == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Hosting not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "edit"
	data.Hosting = item
	data.FormAction = fmt.Sprintf("/hostings/%s/edit", item.ID)
	data.FormSubmit = "Update hosting"

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "hosting-editor-panel", data)
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
		h.logger.Error("Failed to list hostings", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	item, formErr := buildHostingFromForm(r, "", data.DefaultCurrency)
	data.EditorMode = "new"
	data.FormAction = "/hostings"
	data.FormSubmit = "Create hosting"
	data.Hosting = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.hostingRepo.Create(r.Context(), item); err != nil {
		h.logger.Error("Failed to create hosting", "err", err)
		data.ErrorHTML = bannerHTML("bad", "Unable to create hosting.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedItems, listErr := h.hostingRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh hosting list", "err", listErr)
	} else {
		data.Hostings = updatedItems
	}

	data.EditorMode = "detail"
	data.Hosting = item
	data.NoticeHTML = bannerHTML("ok", "Hosting created successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "hosting-editor-response", data)
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
		h.logger.Error("Failed to list hostings", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, formErr := buildHostingFromForm(r, id, data.DefaultCurrency)
	data.EditorMode = "edit"
	data.FormAction = fmt.Sprintf("/hostings/%s/edit", id)
	data.FormSubmit = "Update hosting"
	data.Hosting = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	current, findErr := h.hostingRepo.GetByID(r.Context(), id)
	if findErr != nil {
		h.logger.Error("Failed to load hosting for update", "id", id, "err", findErr)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to validate hosting before update.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if current == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Hosting not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.hostingRepo.Update(r.Context(), item); err != nil {
		h.logger.Error("Failed to update hosting", "id", id, "err", err)
		data.ErrorHTML = bannerHTML("bad", "Unable to update hosting.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusInternalServerError)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	refreshed, getErr := h.hostingRepo.GetByID(r.Context(), id)
	if getErr != nil {
		h.logger.Error("Failed to reload hosting after update", "id", id, "err", getErr)
	} else if refreshed != nil {
		data.Hosting = refreshed
	}

	updatedItems, listErr := h.hostingRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh hosting list", "err", listErr)
	} else {
		data.Hostings = updatedItems
	}

	data.EditorMode = "detail"
	data.NoticeHTML = bannerHTML("ok", "Hosting updated successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "hosting-editor-response", data)
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
		h.logger.Error("Failed to list hostings", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	if err := h.hostingRepo.Delete(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete hosting", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to delete hosting.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "hosting-editor-response", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedItems, listErr := h.hostingRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh hosting list", "err", listErr)
	} else {
		data.Hostings = updatedItems
	}

	data.EditorMode = "flash"
	data.NoticeHTML = bannerHTML("ok", "Hosting deleted successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "hosting-editor-response", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) loadPageData(r *http.Request, start time.Time) (pageData, error) {
	searchTerm, limit := ui.ParseListState(r)
	items, err := h.hostingRepo.GetAll(r.Context(), searchTerm, limit)
	if err != nil {
		return pageData{}, err
	}

	locations, err := h.locationRepo.GetAll(r.Context(), "", 100)
	if err != nil {
		return pageData{}, err
	}

	providers, err := h.providerRepo.GetAll(r.Context(), "", 100)
	if err != nil {
		return pageData{}, err
	}

	domains, err := h.domainRepo.GetAll(r.Context(), "", 100)
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
		BaseData:        ui.NewBaseData(r, "Hostings", start),
		Hostings:        items,
		LocationOptions: ui.BuildSelectOptions(locations, func(item *location.Location) string { return item.Id }, func(item *location.Location) string { return item.Name }),
		ProviderOptions: ui.BuildSelectOptions(providers, func(item *provider.Provider) string { return item.Id }, func(item *provider.Provider) string { return item.Name }),
		DomainOptions:   ui.BuildSelectOptions(domains, func(item *domain.Domain) string { return item.ID }, func(item *domain.Domain) string { return item.Domain }),
		DefaultCurrency: defaultCurrency,
		SearchTerm:      searchTerm,
		Limit:           limit,
		EditorMode:      "empty",
		FormAction:      "/hostings",
		FormSubmit:      "Create hosting",
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
		"hostings-body",
		[]string{"hosting-limit", "hosting-search"},
		[]string{"limit", "q"},
	)
}

func (h *Handler) isEditorRequest(r *http.Request) bool {
	return ui.IsHTMXEditorRequest(r, "hosting-editor")
}

func buildHostingFromForm(r *http.Request, id string, defaultCurrency string) (*hosting.Hosting, error) {
	name := strings.TrimSpace(r.Form.Get("name"))
	hostingType := strings.TrimSpace(r.Form.Get("type"))
	locationID := strings.TrimSpace(r.Form.Get("location_id"))
	providerID := strings.TrimSpace(r.Form.Get("provider_id"))
	domainID := strings.TrimSpace(r.Form.Get("domain_id"))
	notes := strings.TrimSpace(r.Form.Get("notes"))

	item := &hosting.Hosting{
		ID:         id,
		Name:       name,
		Type:       hostingType,
		LocationID: ui.OptionalString(locationID),
		ProviderID: ui.OptionalString(providerID),
		DomainID:   ui.OptionalString(domainID),
		Currency:   ui.NormalizeCurrency(r.Form.Get("currency"), defaultCurrency),
		Notes:      ui.OptionalString(notes),
	}

	diskGB, err := ui.ParseOptionalInt(r.Form.Get("disk_gb"))
	if err != nil {
		return item, fmt.Errorf("Disk size must be a valid integer")
	}
	item.DiskGB = diskGB

	price, err := ui.ParseOptionalFloat(r.Form.Get("price"))
	if err != nil {
		return item, fmt.Errorf("Price must be a valid decimal value")
	}
	item.Price = price

	dueDate, err := ui.ParseOptionalDate(r.Form.Get("due_date"))
	if err != nil {
		return item, fmt.Errorf("Due date must use the YYYY-MM-DD format")
	}
	item.DueDate = dueDate

	sinceDate, err := ui.ParseOptionalDate(r.Form.Get("since_date"))
	if err != nil {
		return item, fmt.Errorf("Since date must use the YYYY-MM-DD format")
	}
	item.SinceDate = sinceDate

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
