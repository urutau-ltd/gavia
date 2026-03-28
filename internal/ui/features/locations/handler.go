package locations

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

	"codeberg.org/urutau-ltd/gavia/internal/models/location"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

// Handler coordinates HTTP endpoints, HTML templates and repository calls for the locations feature.
// It lives in the UI layer to isolate transport decisions from persistence code.
type Handler struct {
	logger       *slog.Logger
	tmpl         *template.Template
	locationRepo *location.LocationRepository
}

// pageData is the template contract for locations screens.
// Keeping this consolidated makes list/editor rendering predictable and easier to extend.
type pageData struct {
	ui.BaseData
	Locations  []*location.Location
	Location   *location.Location
	SearchTerm string
	Limit      int
	EditorMode string
	FormAction string
	FormSubmit string
	NoticeHTML template.HTML
	ErrorHTML  template.HTML
}

// NewHandler builds the locations feature handler and preloads all required templates.
// Parsing templates at startup avoids repeating work at request time.
func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/locations/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger:       l,
		tmpl:         t,
		locationRepo: location.NewLocationRepository(db),
	}
}

// Index serves GET /locations.
// It returns full HTML or only the list fragment depending on HTMX request headers.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list locations", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	ui.WriteHTMLHeader(w)
	if h.isListRequest(r) {
		h.renderTemplate(w, "location-list", data)
		return
	}

	if h.isEditorRequest(r) {
		h.renderTemplate(w, "location-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

// New serves GET /locations/new and prepares the editor in create mode.
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list locations", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	data.EditorMode = "new"
	data.FormAction = "/locations"
	data.FormSubmit = "Create location"
	data.Location = &location.Location{}

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "location-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

// Show serves GET /locations/{id} and loads detail data into the editor panel.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list locations", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	l, err := h.locationRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load location", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load location.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if l == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Location not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "detail"
	data.Location = l

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "location-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

// Edit serves GET /locations/{id}/edit with current values preloaded in the form.
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list locations", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	l, err := h.locationRepo.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to load location for edit", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to load location for editing.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if l == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Location not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	data.EditorMode = "edit"
	data.Location = l
	data.FormAction = fmt.Sprintf("/locations/%s/edit", l.Id)
	data.FormSubmit = "Update location"

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "location-editor-panel", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

// Create handles POST /locations and persists a new location.
// On success it returns a response that updates both editor and table when HTMX is used.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list locations", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	name := strings.TrimSpace(r.Form.Get("name"))
	city := strings.TrimSpace(r.Form.Get("city"))
	country := strings.TrimSpace(r.Form.Get("country"))
	notes := strings.TrimSpace(r.Form.Get("notes"))

	data.EditorMode = "new"
	data.FormAction = "/locations"
	data.FormSubmit = "Create location"
	data.Location = &location.Location{
		Name:    name,
		City:    ui.OptionalString(city),
		Country: ui.OptionalString(country),
		Notes:   ui.OptionalString(notes),
	}

	if name == "" {
		data.ErrorHTML = bannerHTML("bad", "Name is required.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.locationRepo.Create(r.Context(), data.Location); err != nil {
		h.logger.Error("Failed to create location", "err", err)
		status := http.StatusInternalServerError
		msg := "Unable to create location."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "A location with that name already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedLocations, listErr := h.locationRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh location list", "err", listErr)
	} else {
		data.Locations = updatedLocations
	}

	refreshed, getErr := h.locationRepo.GetByID(r.Context(), data.Location.Id)
	if getErr != nil {
		h.logger.Error("Failed to reload location after create", "id", data.Location.Id, "err", getErr)
	} else if refreshed != nil {
		data.Location = refreshed
	}

	data.EditorMode = "detail"
	data.NoticeHTML = bannerHTML("ok", "Location created successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "location-editor-response", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

// Update handles POST /locations/{id}/edit and persists location changes.
// It first checks existence to keep not-found and write errors distinct.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list locations", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	name := strings.TrimSpace(r.Form.Get("name"))
	city := strings.TrimSpace(r.Form.Get("city"))
	country := strings.TrimSpace(r.Form.Get("country"))
	notes := strings.TrimSpace(r.Form.Get("notes"))

	data.EditorMode = "edit"
	data.FormAction = fmt.Sprintf("/locations/%s/edit", id)
	data.FormSubmit = "Update location"
	data.Location = &location.Location{
		Id:      id,
		Name:    name,
		City:    ui.OptionalString(city),
		Country: ui.OptionalString(country),
		Notes:   ui.OptionalString(notes),
	}

	if name == "" {
		data.ErrorHTML = bannerHTML("bad", "Name is required.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	current, findErr := h.locationRepo.GetByID(r.Context(), id)
	if findErr != nil {
		h.logger.Error("Failed to load location for update", "id", id, "err", findErr)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to validate location before update.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if current == nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("warn", "Location not found.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusNotFound)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.locationRepo.Update(r.Context(), data.Location); err != nil {
		h.logger.Error("Failed to update location", "id", id, "err", err)
		status := http.StatusInternalServerError
		msg := "Unable to update location."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "A location with that name already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	refreshed, getErr := h.locationRepo.GetByID(r.Context(), id)
	if getErr != nil {
		h.logger.Error("Failed to reload location after update", "id", id, "err", getErr)
	} else if refreshed != nil {
		data.Location = refreshed
	}

	updatedLocations, listErr := h.locationRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh location list", "err", listErr)
	} else {
		data.Locations = updatedLocations
	}

	data.EditorMode = "detail"
	data.NoticeHTML = bannerHTML("ok", "Location updated successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "location-editor-response", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

// Delete handles DELETE /locations/{id} and refreshes list state after removal.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list locations", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	if err := h.locationRepo.Delete(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete location", "id", id, "err", err)
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to delete location.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "location-editor-response", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedLocations, listErr := h.locationRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr != nil {
		h.logger.Error("Failed to refresh location list", "err", listErr)
	} else {
		data.Locations = updatedLocations
	}

	data.EditorMode = "flash"
	data.NoticeHTML = bannerHTML("ok", "Location deleted successfully.")

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "location-editor-response", data)
		return
	}

	h.renderTemplate(w, "base", data)
}

// loadPageData resolves table state (filters, limits, records) for location views.
func (h *Handler) loadPageData(r *http.Request, start time.Time) (pageData, error) {
	searchTerm, limit := ui.ParseListState(r)
	locations, err := h.locationRepo.GetAll(r.Context(), searchTerm, limit)
	if err != nil {
		return pageData{}, err
	}

	return pageData{
		BaseData:   ui.NewBaseData(r, "Locations", start),
		Locations:  locations,
		SearchTerm: searchTerm,
		Limit:      limit,
		EditorMode: "empty",
		FormAction: "/locations",
		FormSubmit: "Create location",
	}, nil
}

// renderTemplate executes a named template and centralizes render-error logging.
func (h *Handler) renderTemplate(w http.ResponseWriter, tmpl string, data any) {
	if err := h.tmpl.ExecuteTemplate(w, tmpl, data); err != nil {
		h.logger.Error("Error rendering template", "template", tmpl, "err", err)
	}
}

// isListRequest detects HTMX calls meant to replace only the locations table body.
func (h *Handler) isListRequest(r *http.Request) bool {
	return ui.IsHTMXListRequest(
		r,
		"locations-body",
		[]string{"location-limit", "location-search"},
		[]string{"limit", "q"},
	)
}

// isEditorRequest detects HTMX calls that target the location editor panel.
func (h *Handler) isEditorRequest(r *http.Request) bool {
	return ui.IsHTMXEditorRequest(r, "location-editor")
}

// bannerHTML builds a sanitized alert block compatible with Missing.css status classes.
func bannerHTML(kind, msg string) template.HTML {
	className := "info"
	switch kind {
	case "ok", "bad", "warn", "info":
		className = kind
	}

	escaped := html.EscapeString(msg)
	return template.HTML(`<p class="location-alert ` + className + `">` + escaped + `</p>`)
}
