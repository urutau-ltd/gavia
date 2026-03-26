package ips

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

	ipmodel "codeberg.org/urutau-ltd/gavia/internal/models/ip"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger *slog.Logger
	tmpl   *template.Template
	ipRepo *ipmodel.Repository
}

type pageData struct {
	ui.BaseData
	IPs         []*ipmodel.IP
	IP          *ipmodel.IP
	TypeOptions []ui.SelectOption
	SearchTerm  string
	Limit       int
	EditorMode  string
	FormAction  string
	FormSubmit  string
	NoticeHTML  template.HTML
	ErrorHTML   template.HTML
}

func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/ips/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger: l,
		tmpl:   t,
		ipRepo: ipmodel.NewRepository(db),
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list IP addresses", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	ui.WriteHTMLHeader(w)
	if h.isListRequest(r) {
		h.renderTemplate(w, "ip-list", data)
		return
	}
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "ip-editor-panel", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to load IP form", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	data.EditorMode = "new"
	data.FormAction = "/ips"
	data.FormSubmit = "Create IP address"
	data.IP = &ipmodel.IP{Type: "ipv4"}

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "ip-editor-panel", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list IP addresses", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.ipRepo.GetByID(r.Context(), id)
	if err != nil {
		h.renderError(w, r, data, http.StatusBadRequest, "Unable to load the IP address.")
		return
	}
	if item == nil {
		h.renderError(w, r, data, http.StatusNotFound, "IP address not found.")
		return
	}

	data.EditorMode = "detail"
	data.IP = item
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "ip-editor-panel", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	data, err := h.loadPageData(r, start)
	if err != nil {
		h.logger.Error("Failed to list IP addresses", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, err := h.ipRepo.GetByID(r.Context(), id)
	if err != nil {
		h.renderError(w, r, data, http.StatusBadRequest, "Unable to load the IP address for editing.")
		return
	}
	if item == nil {
		h.renderError(w, r, data, http.StatusNotFound, "IP address not found.")
		return
	}

	data.EditorMode = "edit"
	data.IP = item
	data.FormAction = fmt.Sprintf("/ips/%s/edit", item.ID)
	data.FormSubmit = "Update IP address"

	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "ip-editor-panel", data)
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
		h.logger.Error("Failed to list IP addresses", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	item, formErr := buildIPFromForm(r, "")
	data.EditorMode = "new"
	data.FormAction = "/ips"
	data.FormSubmit = "Create IP address"
	data.IP = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "ip-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	if err := h.ipRepo.Create(r.Context(), item); err != nil {
		status := http.StatusInternalServerError
		msg := "Unable to create the IP address."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "That IP address already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "ip-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	updatedItems, listErr := h.ipRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr == nil {
		data.IPs = updatedItems
	}

	data.EditorMode = "detail"
	data.IP = item
	data.NoticeHTML = bannerHTML("ok", "IP address created successfully.")
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "ip-editor-response", data)
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
		h.logger.Error("Failed to list IP addresses", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	item, formErr := buildIPFromForm(r, id)
	data.EditorMode = "edit"
	data.FormAction = fmt.Sprintf("/ips/%s/edit", id)
	data.FormSubmit = "Update IP address"
	data.IP = item
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "ip-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	current, findErr := h.ipRepo.GetByID(r.Context(), id)
	if findErr != nil || current == nil {
		status := http.StatusBadRequest
		msg := "Unable to validate the IP address before update."
		if current == nil && findErr == nil {
			status = http.StatusNotFound
			msg = "IP address not found."
		}
		h.renderError(w, r, data, status, msg)
		return
	}

	if err := h.ipRepo.Update(r.Context(), item); err != nil {
		status := http.StatusInternalServerError
		msg := "Unable to update the IP address."
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			status = http.StatusConflict
			msg = "That IP address already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "ip-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}

	refreshed, getErr := h.ipRepo.GetByID(r.Context(), id)
	if getErr == nil && refreshed != nil {
		data.IP = refreshed
	}
	updatedItems, listErr := h.ipRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr == nil {
		data.IPs = updatedItems
	}

	data.EditorMode = "detail"
	data.NoticeHTML = bannerHTML("ok", "IP address updated successfully.")
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "ip-editor-response", data)
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
		h.logger.Error("Failed to list IP addresses", "err", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := r.PathValue("id")
	if err := h.ipRepo.Delete(r.Context(), id); err != nil {
		h.renderError(w, r, data, http.StatusBadRequest, "Unable to delete the IP address.")
		return
	}

	updatedItems, listErr := h.ipRepo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	if listErr == nil {
		data.IPs = updatedItems
	}
	data.EditorMode = "flash"
	data.NoticeHTML = bannerHTML("ok", "IP address deleted successfully.")
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "ip-editor-response", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) loadPageData(r *http.Request, start time.Time) (pageData, error) {
	searchTerm, limit := ui.ParseListState(r)
	items, err := h.ipRepo.GetAll(r.Context(), searchTerm, limit)
	if err != nil {
		return pageData{}, err
	}

	return pageData{
		BaseData: ui.NewBaseData(r, "IP Addresses", start),
		IPs:      items,
		TypeOptions: []ui.SelectOption{
			{Value: "ipv4", Label: "IPv4"},
			{Value: "ipv6", Label: "IPv6"},
		},
		SearchTerm: searchTerm,
		Limit:      limit,
		EditorMode: "empty",
		FormAction: "/ips",
		FormSubmit: "Create IP address",
	}, nil
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, data pageData, status int, message string) {
	data.EditorMode = "flash"
	data.ErrorHTML = bannerHTML("bad", message)
	ui.WriteHTMLHeader(w)
	w.WriteHeader(status)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "ip-editor-panel", data)
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
	return ui.IsHTMXListRequest(r, "ips-body", []string{"ip-limit", "ip-search"}, []string{"limit", "q"})
}

func (h *Handler) isEditorRequest(r *http.Request) bool {
	return ui.IsHTMXEditorRequest(r, "ip-editor")
}

func buildIPFromForm(r *http.Request, id string) (*ipmodel.IP, error) {
	item := &ipmodel.IP{
		ID:      id,
		Address: strings.TrimSpace(r.Form.Get("address")),
		Type:    strings.TrimSpace(r.Form.Get("type")),
		City:    ui.OptionalString(strings.TrimSpace(r.Form.Get("city"))),
		Country: ui.OptionalString(strings.TrimSpace(r.Form.Get("country"))),
		Org:     ui.OptionalString(strings.TrimSpace(r.Form.Get("org"))),
		ASN:     ui.OptionalString(strings.TrimSpace(r.Form.Get("asn"))),
		ISP:     ui.OptionalString(strings.TrimSpace(r.Form.Get("isp"))),
		Notes:   ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
	}

	switch {
	case item.Address == "":
		return item, fmt.Errorf("Address is required")
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
	return template.HTML(`<p class="crud-alert ` + className + `">` + html.EscapeString(msg) + `</p>`)
}
