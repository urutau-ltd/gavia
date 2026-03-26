package labels

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

	labelmodel "codeberg.org/urutau-ltd/gavia/internal/models/label"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger *slog.Logger
	tmpl   *template.Template
	repo   *labelmodel.Repository
}

type pageData struct {
	ui.BaseData
	Labels     []*labelmodel.Label
	Label      *labelmodel.Label
	SearchTerm string
	Limit      int
	EditorMode string
	FormAction string
	FormSubmit string
	NoticeHTML template.HTML
	ErrorHTML  template.HTML
}

func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/labels/views/*.html",
		"components/*.html",
	))

	return &Handler{logger: l, tmpl: t, repo: labelmodel.NewRepository(db)}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	ui.WriteHTMLHeader(w)
	if h.isListRequest(r) {
		h.renderTemplate(w, "label-list", data)
		return
	}
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "label-editor-panel", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	data.EditorMode = "new"
	data.FormAction = "/labels"
	data.FormSubmit = "Create label"
	data.Label = &labelmodel.Label{}
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "label-editor-panel", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	h.renderOne(w, r, false)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	h.renderOne(w, r, true)
}

func (h *Handler) renderOne(w http.ResponseWriter, r *http.Request, editing bool) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	id := r.PathValue("id")
	item, err := h.repo.GetByID(r.Context(), id)
	if err != nil || item == nil {
		status := http.StatusBadRequest
		msg := "Unable to load the label."
		if item == nil && err == nil {
			status = http.StatusNotFound
			msg = "Label not found."
		}
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "label-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	if editing {
		data.EditorMode = "edit"
		data.FormAction = fmt.Sprintf("/labels/%s/edit", item.ID)
		data.FormSubmit = "Update label"
	} else {
		data.EditorMode = "detail"
	}
	data.Label = item
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "label-editor-panel", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, "", true)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, r.PathValue("id"), false)
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request, id string, creating bool) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	item := &labelmodel.Label{
		ID:    id,
		Name:  strings.TrimSpace(r.Form.Get("name")),
		Notes: ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
	}
	if creating {
		data.EditorMode = "new"
		data.FormAction = "/labels"
		data.FormSubmit = "Create label"
	} else {
		data.EditorMode = "edit"
		data.FormAction = fmt.Sprintf("/labels/%s/edit", id)
		data.FormSubmit = "Update label"
	}
	data.Label = item
	if item.Name == "" {
		data.ErrorHTML = bannerHTML("bad", "Name is required")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "label-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	if !creating {
		current, err := h.repo.GetByID(r.Context(), id)
		if err != nil || current == nil {
			status := http.StatusBadRequest
			msg := "Unable to validate the label before update."
			if current == nil && err == nil {
				status = http.StatusNotFound
				msg = "Label not found."
			}
			data.EditorMode = "flash"
			data.ErrorHTML = bannerHTML("bad", msg)
			ui.WriteHTMLHeader(w)
			w.WriteHeader(status)
			if h.isEditorRequest(r) {
				h.renderTemplate(w, "label-editor-panel", data)
				return
			}
			h.renderTemplate(w, "base", data)
			return
		}
	}
	var saveErr error
	if creating {
		saveErr = h.repo.Create(r.Context(), item)
	} else {
		saveErr = h.repo.Update(r.Context(), item)
	}
	if saveErr != nil {
		status := http.StatusInternalServerError
		msg := "Unable to save the label."
		if strings.Contains(strings.ToLower(saveErr.Error()), "unique") {
			status = http.StatusConflict
			msg = "That label already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "label-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	refreshed, _ := h.repo.GetByID(r.Context(), item.ID)
	if refreshed != nil {
		data.Label = refreshed
	}
	data.Labels, _ = h.repo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	data.EditorMode = "detail"
	if creating {
		data.NoticeHTML = bannerHTML("ok", "Label created successfully.")
	} else {
		data.NoticeHTML = bannerHTML("ok", "Label updated successfully.")
	}
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "label-editor-response", data)
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
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err := h.repo.Delete(r.Context(), r.PathValue("id")); err != nil {
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", "Unable to delete the label.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "label-editor-response", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	data.Labels, _ = h.repo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	data.EditorMode = "flash"
	data.NoticeHTML = bannerHTML("ok", "Label deleted successfully.")
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "label-editor-response", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) loadPageData(r *http.Request, start time.Time) (pageData, error) {
	searchTerm, limit := ui.ParseListState(r)
	items, err := h.repo.GetAll(r.Context(), searchTerm, limit)
	if err != nil {
		return pageData{}, err
	}
	return pageData{
		BaseData:   ui.NewBaseData(r, "Labels", start),
		Labels:     items,
		SearchTerm: searchTerm,
		Limit:      limit,
		EditorMode: "empty",
		FormAction: "/labels",
		FormSubmit: "Create label",
	}, nil
}

func (h *Handler) renderTemplate(w http.ResponseWriter, tmpl string, data any) {
	if err := h.tmpl.ExecuteTemplate(w, tmpl, data); err != nil {
		h.logger.Error("Error rendering template", "template", tmpl, "err", err)
	}
}

func (h *Handler) isListRequest(r *http.Request) bool {
	return ui.IsHTMXListRequest(r, "labels-body", []string{"label-limit", "label-search"}, []string{"limit", "q"})
}

func (h *Handler) isEditorRequest(r *http.Request) bool {
	return ui.IsHTMXEditorRequest(r, "label-editor")
}

func bannerHTML(kind, msg string) template.HTML {
	className := "info"
	switch kind {
	case "ok", "bad", "warn", "info":
		className = kind
	}
	return template.HTML(`<p class="crud-alert ` + className + `">` + html.EscapeString(msg) + `</p>`)
}
