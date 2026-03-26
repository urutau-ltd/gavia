package dns

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

	"codeberg.org/urutau-ltd/gavia/internal/models/dnsrecord"
	domainmodel "codeberg.org/urutau-ltd/gavia/internal/models/domain"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger     *slog.Logger
	tmpl       *template.Template
	repo       *dnsrecord.Repository
	domainRepo *domainmodel.Repository
}

type pageData struct {
	ui.BaseData
	Records       []*dnsrecord.DNSRecord
	Record        *dnsrecord.DNSRecord
	TypeOptions   []ui.SelectOption
	DomainOptions []ui.SelectOption
	SearchTerm    string
	Limit         int
	EditorMode    string
	FormAction    string
	FormSubmit    string
	NoticeHTML    template.HTML
	ErrorHTML     template.HTML
}

func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/dns/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger:     l,
		tmpl:       t,
		repo:       dnsrecord.NewRepository(db),
		domainRepo: domainmodel.NewRepository(db),
	}
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
		h.renderTemplate(w, "dns-list", data)
		return
	}
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "dns-editor-panel", data)
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
	data.FormAction = "/dns"
	data.FormSubmit = "Create DNS record"
	data.Record = &dnsrecord.DNSRecord{Type: "A"}
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "dns-editor-panel", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
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
		msg := "Unable to load the DNS record."
		if item == nil && err == nil {
			status = http.StatusNotFound
			msg = "DNS record not found."
		}
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "dns-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	data.EditorMode = "detail"
	data.Record = item
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "dns-editor-panel", data)
		return
	}
	h.renderTemplate(w, "base", data)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
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
		msg := "Unable to load the DNS record for editing."
		if item == nil && err == nil {
			status = http.StatusNotFound
			msg = "DNS record not found."
		}
		data.EditorMode = "flash"
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "dns-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	data.EditorMode = "edit"
	data.Record = item
	data.FormAction = fmt.Sprintf("/dns/%s/edit", item.ID)
	data.FormSubmit = "Update DNS record"
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "dns-editor-panel", data)
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
	item, formErr := buildDNSFromForm(r, id)
	data.Record = item
	if creating {
		data.EditorMode = "new"
		data.FormAction = "/dns"
		data.FormSubmit = "Create DNS record"
	} else {
		data.EditorMode = "edit"
		data.FormAction = fmt.Sprintf("/dns/%s/edit", id)
		data.FormSubmit = "Update DNS record"
	}
	if formErr != nil {
		data.ErrorHTML = bannerHTML("bad", formErr.Error())
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "dns-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	if !creating {
		current, err := h.repo.GetByID(r.Context(), id)
		if err != nil || current == nil {
			status := http.StatusBadRequest
			msg := "Unable to validate the DNS record before update."
			if current == nil && err == nil {
				status = http.StatusNotFound
				msg = "DNS record not found."
			}
			data.EditorMode = "flash"
			data.ErrorHTML = bannerHTML("bad", msg)
			ui.WriteHTMLHeader(w)
			w.WriteHeader(status)
			if h.isEditorRequest(r) {
				h.renderTemplate(w, "dns-editor-panel", data)
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
		msg := "Unable to save the DNS record."
		if strings.Contains(strings.ToLower(saveErr.Error()), "unique") {
			status = http.StatusConflict
			msg = "That DNS record already exists."
		}
		data.ErrorHTML = bannerHTML("bad", msg)
		ui.WriteHTMLHeader(w)
		w.WriteHeader(status)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "dns-editor-panel", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	refreshed, _ := h.repo.GetByID(r.Context(), item.ID)
	if refreshed != nil {
		data.Record = refreshed
	}
	data.Records, _ = h.repo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	data.EditorMode = "detail"
	if creating {
		data.NoticeHTML = bannerHTML("ok", "DNS record created successfully.")
	} else {
		data.NoticeHTML = bannerHTML("ok", "DNS record updated successfully.")
	}
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "dns-editor-response", data)
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
		data.ErrorHTML = bannerHTML("bad", "Unable to delete the DNS record.")
		ui.WriteHTMLHeader(w)
		w.WriteHeader(http.StatusBadRequest)
		if h.isEditorRequest(r) {
			h.renderTemplate(w, "dns-editor-response", data)
			return
		}
		h.renderTemplate(w, "base", data)
		return
	}
	data.Records, _ = h.repo.GetAll(r.Context(), data.SearchTerm, data.Limit)
	data.EditorMode = "flash"
	data.NoticeHTML = bannerHTML("ok", "DNS record deleted successfully.")
	ui.WriteHTMLHeader(w)
	if h.isEditorRequest(r) {
		h.renderTemplate(w, "dns-editor-response", data)
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

	domains, err := h.domainRepo.GetAll(r.Context(), "", 100)
	if err != nil {
		return pageData{}, err
	}
	return pageData{
		BaseData: ui.NewBaseData(r, "DNS Records", start),
		Records:  items,
		TypeOptions: []ui.SelectOption{
			{Value: "A", Label: "A"},
			{Value: "AAAA", Label: "AAAA"},
			{Value: "CNAME", Label: "CNAME"},
			{Value: "MX", Label: "MX"},
			{Value: "TXT", Label: "TXT"},
			{Value: "NS", Label: "NS"},
			{Value: "SOA", Label: "SOA"},
			{Value: "SRV", Label: "SRV"},
		},
		DomainOptions: ui.BuildSelectOptions(domains, func(item *domainmodel.Domain) string { return item.ID }, func(item *domainmodel.Domain) string { return item.Domain }),
		SearchTerm:    searchTerm,
		Limit:         limit,
		EditorMode:    "empty",
		FormAction:    "/dns",
		FormSubmit:    "Create DNS record",
	}, nil
}

func (h *Handler) renderTemplate(w http.ResponseWriter, tmpl string, data any) {
	if err := h.tmpl.ExecuteTemplate(w, tmpl, data); err != nil {
		h.logger.Error("Error rendering template", "template", tmpl, "err", err)
	}
}

func (h *Handler) isListRequest(r *http.Request) bool {
	return ui.IsHTMXListRequest(r, "dns-body", []string{"dns-limit", "dns-search"}, []string{"limit", "q"})
}

func (h *Handler) isEditorRequest(r *http.Request) bool {
	return ui.IsHTMXEditorRequest(r, "dns-editor")
}

func buildDNSFromForm(r *http.Request, id string) (*dnsrecord.DNSRecord, error) {
	item := &dnsrecord.DNSRecord{
		ID:       id,
		Type:     strings.TrimSpace(r.Form.Get("type")),
		Hostname: strings.TrimSpace(r.Form.Get("hostname")),
		DomainID: ui.OptionalString(strings.TrimSpace(r.Form.Get("domain_id"))),
		Address:  strings.TrimSpace(r.Form.Get("address")),
		Notes:    ui.OptionalString(strings.TrimSpace(r.Form.Get("notes"))),
	}
	switch {
	case item.Type == "":
		return item, fmt.Errorf("Type is required")
	case item.Hostname == "":
		return item, fmt.Errorf("Hostname is required")
	case item.Address == "":
		return item, fmt.Errorf("Address is required")
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
