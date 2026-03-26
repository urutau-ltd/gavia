package servers

import (
	"context"
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
	ipmodel "codeberg.org/urutau-ltd/gavia/internal/models/ip"
	labelmodel "codeberg.org/urutau-ltd/gavia/internal/models/label"
	"codeberg.org/urutau-ltd/gavia/internal/models/location"
	operatingsystem "codeberg.org/urutau-ltd/gavia/internal/models/operating_system"
	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
	servermodel "codeberg.org/urutau-ltd/gavia/internal/models/server"
	"codeberg.org/urutau-ltd/gavia/internal/models/serverlink"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger       *slog.Logger
	tmpl         *template.Template
	serverRepo   *servermodel.Repository
	ipRepo       *ipmodel.Repository
	labelRepo    *labelmodel.Repository
	linkRepo     *serverlink.Repository
	osRepo       *operatingsystem.OperatingSystemRepository
	locationRepo *location.LocationRepository
	providerRepo *provider.ProviderRepository
	appRepo      *appsetting.AppSettingsRepository
}

type pageData struct {
	ui.BaseData
	Servers         []*servermodel.Server
	Server          *servermodel.Server
	OSOptions       []ui.SelectOption
	LocationOptions []ui.SelectOption
	ProviderOptions []ui.SelectOption
	IPOptions       []ui.AssignmentOption
	LabelOptions    []ui.AssignmentOption
	AssignedIPs     []*ipmodel.IP
	AssignedLabels  []*labelmodel.Label
	DefaultCurrency string
	DefaultOSID     string
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
		"features/servers/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger:       l,
		tmpl:         t,
		serverRepo:   servermodel.NewRepository(db),
		ipRepo:       ipmodel.NewRepository(db),
		labelRepo:    labelmodel.NewRepository(db),
		linkRepo:     serverlink.NewRepository(db),
		osRepo:       operatingsystem.NewOperatingSystemRepository(db),
		locationRepo: location.NewLocationRepository(db),
		providerRepo: provider.NewProviderRepository(db),
		appRepo:      appsetting.NewAppSettingsRepository(db),
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list servers", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	ui.WriteHTMLHeader(w)
	if h.isListRequest(r) {
		h.renderTemplate(w, "server-list", data)
		return
	}

	if h.isEditorRequest(r) {
		h.renderTemplate(w, "server-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to load server form", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	data.EditorMode = "new"
	data.FormAction = "/servers"
	data.FormSubmit = "Create server"
	data.Server = &servermodel.Server{
		OSID:     ui.OptionalString(data.DefaultOSID),
		Currency: data.DefaultCurrency,
	}

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "server-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list servers", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.serverRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load server", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load server.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if item == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Server not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "detail"
	data.Server = item
	if err := h.loadServerAssignments(r.Context(), &data, item.ID); err != nil {
		h.logger.Error("Failed to load server assignments", "id", id, "err", err)
		h.renderError(w, r, data, http.StatusInternalServerError, "Unable to load server assignments.")
		return
	}

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "server-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list servers", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.serverRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load server for edit", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load server for editing.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if item == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Server not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "edit"
	data.Server = item
	data.FormAction = fmt.Sprintf("/servers/%s/edit", item.ID)
	data.FormSubmit = "Update server"
	if err := h.loadServerAssignments(r.Context(), &data, item.ID); err != nil {
		h.logger.Error("Failed to load server assignments", "id", id, "err", err)
		h.renderError(w, r, data, http.StatusInternalServerError, "Unable to load server assignments.")
		return
	}

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "server-editor-panel", data)
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
		h.logger.Error("Failed to list servers", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	ipIDs := selectedFormValues(r.Form["ip_ids"])
	labelIDs := selectedFormValues(r.Form["label_ids"])

	item, formErr := buildServerFromForm(r, "", data.DefaultCurrency)
	data.EditorMode = "new"
	data.FormAction = "/servers"
	data.FormSubmit = "Create server"
	data.Server = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.serverRepo.Create(r.Context(), item); err != nil {
		h.logger.Error("Failed to create server", "err", err)
		status := http.StatusInternalServerError
		msg := "Unable to create server."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "A server with that hostname already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	if err := h.linkRepo.ReplaceServerIPs(r.Context(), item.ID, ipIDs); err != nil {
		h.logger.Error("Failed to assign server IP addresses", "id", item.ID, "err", err)
		h.renderError(w, r, data, http.StatusInternalServerError, "Server created, but IP assignments could not be saved.")
		return
	}
	if err := h.linkRepo.ReplaceServerLabels(r.Context(), item.ID, labelIDs); err != nil {
		h.logger.Error("Failed to assign server labels", "id", item.ID, "err", err)
		h.renderError(w, r, data, http.StatusInternalServerError, "Server created, but label assignments could not be saved.")
		return
	}

	updatedItems, listErr := h.serverRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh server list", "err", listErr)
	} else {
		data.Servers = updatedItems
	}

	data.EditorMode = "detail"
	if refreshed, getErr := h.serverRepo.GetByID(r.Context(), item.ID); getErr == nil && refreshed != nil {
		data.Server = refreshed
	}
	if data.Server != nil {
		if err := h.loadServerAssignments(r.Context(), &data, data.Server.ID); err != nil {
			h.logger.Error("Failed to reload server assignments", "id", item.ID, "err", err)
		}
	}
	data.NoticeHTML = bannerHTML("ok", "Server created successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "server-editor-response", data)
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
		h.logger.Error("Failed to list servers", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	ipIDs := selectedFormValues(r.Form["ip_ids"])
	labelIDs := selectedFormValues(r.Form["label_ids"])

	id := r.PathValue("id")
	item, formErr := buildServerFromForm(r, id, data.DefaultCurrency)
	data.EditorMode = "edit"
	data.FormAction = fmt.Sprintf("/servers/%s/edit", id)
	data.FormSubmit = "Update server"
	data.Server = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	current, findErr := h.serverRepo.GetByID(r.Context(), id)
	if findErr != nil {
		h.logger.Error("Failed to load server for update", "id", id, "err", findErr)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to validate server before update.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if current == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Server not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.serverRepo.Update(r.Context(), item); err != nil {
		h.logger.Error("Failed to update server", "id", id, "err", err)
		status := http.StatusInternalServerError
		msg := "Unable to update server."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "A server with that hostname already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	if err := h.linkRepo.ReplaceServerIPs(r.Context(), item.ID, ipIDs); err != nil {
		h.logger.Error("Failed to update server IP assignments", "id", item.ID, "err", err)
		h.renderError(w, r, data, http.StatusInternalServerError, "Server updated, but IP assignments could not be saved.")
		return
	}
	if err := h.linkRepo.ReplaceServerLabels(r.Context(), item.ID, labelIDs); err != nil {
		h.logger.Error("Failed to update server label assignments", "id", item.ID, "err", err)
		h.renderError(w, r, data, http.StatusInternalServerError, "Server updated, but label assignments could not be saved.")
		return
	}

	refreshed, getErr := h.serverRepo.GetByID(r.Context(), id)
	if getErr != nil {
		h.logger.Error("Failed to reload server after update", "id", id, "err", getErr)
	} else if refreshed != nil {
		data.Server = refreshed
	}

	updatedItems, listErr := h.serverRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh server list", "err", listErr)
	} else {
		data.Servers = updatedItems
	}

	data.EditorMode = "detail"
	if data.Server != nil {
		if err := h.loadServerAssignments(r.Context(), &data, data.Server.ID); err != nil {
			h.logger.Error("Failed to reload server assignments", "id", id, "err", err)
		}
	}
	data.NoticeHTML = bannerHTML("ok", "Server updated successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "server-editor-response", data)
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
		h.logger.Error("Failed to list servers", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	if err := h.serverRepo.Delete(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete server", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to delete server.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "server-editor-response", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedItems, listErr := h.serverRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh server list", "err", listErr)
	} else {
		data.Servers = updatedItems
	}

	data.EditorMode = "flash"
	data.NoticeHTML = bannerHTML("ok", "Server deleted successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "server-editor-response", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

func (h *Handler) loadPageData(r *http.Request, start time.Time) (pageData, error) {
	searchTerm, limit := ui.ParseListState(r)
	items, err := h.serverRepo.GetAll(r.Context(), searchTerm, limit)
	if err != nil {
		return pageData{}, err
	}

	operatingSystems, err := h.osRepo.GetAll(r.Context(), "", 100)
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

	allIPs, err := h.ipRepo.GetAll(r.Context(), "", 100)
	if err != nil {
		return pageData{}, err
	}

	allLabels, err := h.labelRepo.GetAll(r.Context(), "", 100)
	if err != nil {
		return pageData{}, err
	}

	settings, err := h.appRepo.Get(r.Context())
	if err != nil {
		return pageData{}, err
	}

	defaultCurrency := "MXN"
	defaultOSID := ""
	if settings != nil {
		defaultCurrency = settings.DefaultCurrency
		defaultOSID = ui.FindSelectValueByLabel(
			operatingSystems,
			settings.DefaultServerOS,
			func(item *operatingsystem.OperatingSystem) string { return item.Id },
			func(item *operatingsystem.OperatingSystem) string { return item.Name },
		)
	}

	return pageData{
		BaseData:        ui.NewBaseData(r, "Servers", start),
		Servers:         items,
		OSOptions:       ui.BuildSelectOptions(operatingSystems, func(item *operatingsystem.OperatingSystem) string { return item.Id }, func(item *operatingsystem.OperatingSystem) string { return item.Name }),
		LocationOptions: ui.BuildSelectOptions(locations, func(item *location.Location) string { return item.Id }, func(item *location.Location) string { return item.Name }),
		ProviderOptions: ui.BuildSelectOptions(providers, func(item *provider.Provider) string { return item.Id }, func(item *provider.Provider) string { return item.Name }),
		IPOptions: ui.BuildAssignmentOptions(
			allIPs,
			map[string]struct{}{},
			func(item *ipmodel.IP) string { return item.ID },
			func(item *ipmodel.IP) string { return item.Address },
			func(item *ipmodel.IP) string { return strings.ToUpper(item.Type) },
		),
		LabelOptions: ui.BuildAssignmentOptions(
			allLabels,
			map[string]struct{}{},
			func(item *labelmodel.Label) string { return item.ID },
			func(item *labelmodel.Label) string { return item.Name },
			func(item *labelmodel.Label) string { return item.NotesValue() },
		),
		DefaultCurrency: defaultCurrency,
		DefaultOSID:     defaultOSID,
		SearchTerm:      searchTerm,
		Limit:           limit,
		EditorMode:      "empty",
		FormAction:      "/servers",
		FormSubmit:      "Create server",
	}, nil
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, data pageData, status int, message string) {
	data.EditorMode = "flash"
	data.ErrorHTML = bannerHTML("bad", message)
	ui.WriteHTMLHeader(w)
	w.WriteHeader(status)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "server-editor-panel", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) renderTemplate(w http.ResponseWriter, tmpl string, data any) {
	if err := h.tmpl.ExecuteTemplate(w, tmpl, data); err != nil {
		h.logger.Error("Error rendering template", "template", tmpl, "err", err)
	}
}

func (h *Handler) isListRequest(r *http.Request) bool {
	return ui.IsHTMXListRequest(
		r,
		"servers-body",
		[]string{"server-limit", "server-search"},
		[]string{"limit", "q"},
	)
}

func (h *Handler) isEditorRequest(r *http.Request) bool {
	return ui.IsHTMXEditorRequest(r, "server-editor")
}

func buildServerFromForm(r *http.Request, id string, defaultCurrency string) (*servermodel.Server, error) {
	hostname := strings.TrimSpace(r.Form.Get("hostname"))
	serverType := strings.TrimSpace(r.Form.Get("type"))
	osID := strings.TrimSpace(r.Form.Get("os_id"))
	locationID := strings.TrimSpace(r.Form.Get("location_id"))
	providerID := strings.TrimSpace(r.Form.Get("provider_id"))
	notes := strings.TrimSpace(r.Form.Get("notes"))

	item := &servermodel.Server{
		ID:         id,
		Hostname:   hostname,
		Type:       serverType,
		OSID:       ui.OptionalString(osID),
		LocationID: ui.OptionalString(locationID),
		ProviderID: ui.OptionalString(providerID),
		Currency:   ui.NormalizeCurrency(r.Form.Get("currency"), defaultCurrency),
		Notes:      ui.OptionalString(notes),
	}

	cpuCores, err := ui.ParseOptionalInt(r.Form.Get("cpu_cores"))
	if err != nil {
		return item, fmt.Errorf("CPU cores must be a valid integer")
	}
	item.CPUCores = cpuCores

	memoryGB, err := ui.ParseOptionalInt(r.Form.Get("memory_gb"))
	if err != nil {
		return item, fmt.Errorf("Memory in GB must be a valid integer")
	}
	item.MemoryGB = memoryGB

	diskGB, err := ui.ParseOptionalInt(r.Form.Get("disk_gb"))
	if err != nil {
		return item, fmt.Errorf("Disk in GB must be a valid integer")
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
	case item.Hostname == "":
		return item, fmt.Errorf("Hostname is required")
	case item.Type == "":
		return item, fmt.Errorf("Type is required")
	default:
		return item, nil
	}
}

func (h *Handler) loadServerAssignments(ctx context.Context, data *pageData, serverID string) error {
	selectedIPs, err := h.linkRepo.GetAssignedIPIDs(ctx, serverID)
	if err != nil {
		return err
	}
	selectedLabels, err := h.linkRepo.GetAssignedLabelIDs(ctx, serverID)
	if err != nil {
		return err
	}

	allIPs, err := h.ipRepo.GetAll(ctx, "", 100)
	if err != nil {
		return err
	}
	allLabels, err := h.labelRepo.GetAll(ctx, "", 100)
	if err != nil {
		return err
	}
	assignedIPs, err := h.linkRepo.GetServerIPs(ctx, serverID)
	if err != nil {
		return err
	}
	assignedLabels, err := h.linkRepo.GetServerLabels(ctx, serverID)
	if err != nil {
		return err
	}

	data.IPOptions = ui.BuildAssignmentOptions(
		allIPs,
		selectedIPs,
		func(item *ipmodel.IP) string { return item.ID },
		func(item *ipmodel.IP) string { return item.Address },
		func(item *ipmodel.IP) string { return strings.ToUpper(item.Type) },
	)
	data.LabelOptions = ui.BuildAssignmentOptions(
		allLabels,
		selectedLabels,
		func(item *labelmodel.Label) string { return item.ID },
		func(item *labelmodel.Label) string { return item.Name },
		func(item *labelmodel.Label) string { return item.NotesValue() },
	)
	data.AssignedIPs = assignedIPs
	data.AssignedLabels = assignedLabels
	return nil
}

func selectedFormValues(values []string) []string {
	selected := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		selected = append(selected, value)
	}

	return selected
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
