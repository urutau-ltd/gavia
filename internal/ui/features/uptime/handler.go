package uptime

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger *slog.Logger
	tmpl   *template.Template
	repo   *uptimemonitor.Repository
}

type monitorRow struct {
	ID         string
	Name       string
	TargetURL  string
	StatusTone string
	StatusText string
	StatusMeta string
	Enabled    bool
}

type resultRow struct {
	CheckedAt  string
	StatusTone string
	StatusText string
	Latency    string
	Detail     string
}

type summaryStat struct {
	Label string
	Value string
	Hint  string
}

type resultChartPayload struct {
	Labels  []string `json:"labels"`
	Status  []int    `json:"status"`
	Latency []*int   `json:"latency"`
	Up      int      `json:"up"`
	Down    int      `json:"down"`
}

type pageData struct {
	ui.BaseData
	Monitors        []monitorRow
	Selected        *uptimemonitor.Monitor
	Results         []resultRow
	Summary         *uptimemonitor.Summary
	SelectedStats   []summaryStat
	ChartsJSON      template.JS
	NoticeHTML      template.HTML
	ErrorHTML       template.HTML
	EditingExisting bool
}

func NewHandler(logger *slog.Logger, uiFS fs.FS, repo *uptimemonitor.Repository) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/uptime/views/*.html",
		"components/*.html",
	))

	return &Handler{
		logger: logger,
		tmpl:   t,
		repo:   repo,
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	h.renderPage(w, r, "")
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	h.renderPage(w, r, r.PathValue("id"))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form payload.", http.StatusBadRequest)
		return
	}

	monitor, err := parseMonitorForm(nil, r)
	if err != nil {
		h.renderError(w, r, "", err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(r.Context(), monitor); err != nil {
		h.logger.Error("Failed to create uptime monitor", "err", err)
		h.renderError(w, r, "", "Unable to create the uptime monitor.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/uptime/"+monitor.ID+"?notice=created", http.StatusSeeOther)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form payload.", http.StatusBadRequest)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	current, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Unable to load the selected monitor.", http.StatusInternalServerError)
		return
	}
	if current == nil {
		http.NotFound(w, r)
		return
	}

	monitor, err := parseMonitorForm(current, r)
	if err != nil {
		h.renderError(w, r, id, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Update(r.Context(), monitor); err != nil {
		h.logger.Error("Failed to update uptime monitor", "id", id, "err", err)
		h.renderError(w, r, id, "Unable to update the uptime monitor.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/uptime/"+id+"?notice=updated", http.StatusSeeOther)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "Monitor id is required.", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete uptime monitor", "id", id, "err", err)
		http.Error(w, "Unable to delete the uptime monitor.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/uptime?notice=deleted", http.StatusSeeOther)
}

func (h *Handler) renderPage(w http.ResponseWriter, r *http.Request, selectedID string) {
	start := time.Now()
	monitors, err := h.repo.GetAll(r.Context(), 0)
	if err != nil {
		http.Error(w, "Unable to load uptime monitors.", http.StatusInternalServerError)
		return
	}

	selected, results, err := h.loadSelected(r, selectedID)
	if err != nil {
		http.Error(w, "Unable to load the selected uptime monitor.", http.StatusInternalServerError)
		return
	}

	summary, err := h.repo.GetSummary(r.Context())
	if err != nil {
		http.Error(w, "Unable to load uptime summary.", http.StatusInternalServerError)
		return
	}

	chartsJSON, err := buildChartsJSON(results)
	if err != nil {
		http.Error(w, "Unable to build uptime charts.", http.StatusInternalServerError)
		return
	}

	data := pageData{
		BaseData:        ui.NewBaseData(r, "Uptime", start),
		Monitors:        buildMonitorRows(monitors),
		Selected:        selected,
		Results:         buildResultRows(results),
		Summary:         summary,
		SelectedStats:   buildSelectedStats(results),
		ChartsJSON:      template.JS(chartsJSON),
		NoticeHTML:      uptimeNotice(r),
		EditingExisting: selected != nil && selected.ID != "",
	}

	if selected == nil {
		data.Selected = defaultMonitor()
	}

	ui.WriteHTMLHeader(w)
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Failed to render uptime page", "err", err)
	}
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, selectedID, message string, status int) {
	start := time.Now()
	monitors, _ := h.repo.GetAll(r.Context(), 0)
	selected, results, _ := h.loadSelected(r, selectedID)
	summary, _ := h.repo.GetSummary(r.Context())
	chartsJSON, _ := buildChartsJSON(results)
	if selected == nil {
		selected = defaultMonitor()
	}

	data := pageData{
		BaseData:        ui.NewBaseData(r, "Uptime", start),
		Monitors:        buildMonitorRows(monitors),
		Selected:        selected,
		Results:         buildResultRows(results),
		Summary:         summary,
		SelectedStats:   buildSelectedStats(results),
		ChartsJSON:      template.JS(chartsJSON),
		ErrorHTML:       ui.BannerHTML("uptime-alert", "bad", message),
		EditingExisting: selected.ID != "",
	}

	ui.WriteHTMLHeader(w)
	w.WriteHeader(status)
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Failed to render uptime error state", "err", err)
	}
}

func (h *Handler) loadSelected(r *http.Request, selectedID string) (*uptimemonitor.Monitor, []*uptimemonitor.Result, error) {
	selectedID = strings.TrimSpace(selectedID)
	if selectedID == "" {
		return nil, nil, nil
	}

	selected, err := h.repo.GetByID(r.Context(), selectedID)
	if err != nil || selected == nil {
		return selected, nil, err
	}

	results, err := h.repo.GetRecentResults(r.Context(), selectedID, 20)
	return selected, results, err
}

func parseMonitorForm(current *uptimemonitor.Monitor, r *http.Request) (*uptimemonitor.Monitor, error) {
	monitor := defaultMonitor()
	if current != nil {
		monitor = current
	}

	monitor.Name = strings.TrimSpace(r.Form.Get("name"))
	monitor.TargetURL = strings.TrimSpace(r.Form.Get("target_url"))
	monitor.Enabled = r.Form.Get("enabled") == "1"
	monitor.Notes = optionalString(r.Form.Get("notes"))

	expectedStatus, err := parsePositiveInt(r.Form.Get("expected_status"), 200)
	if err != nil {
		return monitor, err
	}
	intervalSeconds, err := parsePositiveInt(r.Form.Get("check_interval_seconds"), 300)
	if err != nil {
		return monitor, err
	}
	timeoutMS, err := parsePositiveInt(r.Form.Get("timeout_ms"), 5000)
	if err != nil {
		return monitor, err
	}

	monitor.ExpectedStatus = expectedStatus
	monitor.CheckIntervalSeconds = intervalSeconds
	monitor.TimeoutMS = timeoutMS

	if monitor.Name == "" {
		return monitor, errors.New("Name is required.")
	}

	parsedURL, err := neturl.ParseRequestURI(monitor.TargetURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return monitor, errors.New("A valid target URL is required.")
	}

	return monitor, nil
}

func buildMonitorRows(items []*uptimemonitor.MonitorStatus) []monitorRow {
	rows := make([]monitorRow, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}

		row := monitorRow{
			ID:        item.ID,
			Name:      item.Name,
			TargetURL: item.TargetURL,
			Enabled:   item.Enabled,
		}

		switch {
		case !item.Enabled:
			row.StatusTone = "warn"
			row.StatusText = "Disabled"
		case item.LastOK == nil:
			row.StatusTone = "warn"
			row.StatusText = "Pending"
		case *item.LastOK:
			row.StatusTone = "ok"
			row.StatusText = "Up"
		default:
			row.StatusTone = "bad"
			row.StatusText = "Down"
		}

		if item.LastCheckedAt != nil {
			row.StatusMeta = item.LastCheckedAt.Format("2006-01-02 15:04")
		}

		rows = append(rows, row)
	}

	return rows
}

func buildResultRows(items []*uptimemonitor.Result) []resultRow {
	rows := make([]resultRow, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}

		row := resultRow{
			CheckedAt: item.CheckedAt.Format("2006-01-02 15:04:05"),
		}
		if item.OK {
			row.StatusTone = "ok"
			row.StatusText = "Up"
		} else {
			row.StatusTone = "bad"
			row.StatusText = "Down"
		}
		if item.LatencyMS != nil {
			row.Latency = formatLatency(*item.LatencyMS)
		}
		if item.ErrorText != nil {
			row.Detail = *item.ErrorText
		} else if item.StatusCode != nil {
			row.Detail = "HTTP " + strings.TrimSpace(http.StatusText(*item.StatusCode))
			if row.Detail == "HTTP" {
				row.Detail = "HTTP status recorded"
			}
		}

		rows = append(rows, row)
	}

	return rows
}

func uptimeNotice(r *http.Request) template.HTML {
	switch r.URL.Query().Get("notice") {
	case "created":
		return ui.BannerHTML("uptime-alert", "ok", "Uptime monitor created successfully.")
	case "updated":
		return ui.BannerHTML("uptime-alert", "ok", "Uptime monitor updated successfully.")
	case "deleted":
		return ui.BannerHTML("uptime-alert", "ok", "Uptime monitor deleted successfully.")
	default:
		return ""
	}
}

func defaultMonitor() *uptimemonitor.Monitor {
	return &uptimemonitor.Monitor{
		Kind:                 "http",
		ExpectedStatus:       200,
		CheckIntervalSeconds: 300,
		TimeoutMS:            5000,
		Enabled:              true,
	}
}

func parsePositiveInt(raw string, fallback int) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback, nil
	}

	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return 0, errors.New("Numeric fields must be greater than zero.")
	}

	return number, nil
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func formatLatency(value int) string {
	if value <= 0 {
		return ""
	}
	return fmt.Sprintf("%dms", value)
}

func buildSelectedStats(items []*uptimemonitor.Result) []summaryStat {
	if len(items) == 0 {
		return nil
	}

	upCount := 0
	downCount := 0
	latencyTotal := 0
	latencyCount := 0
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.OK {
			upCount++
		} else {
			downCount++
		}
		if item.LatencyMS != nil && *item.LatencyMS > 0 {
			latencyTotal += *item.LatencyMS
			latencyCount++
		}
	}

	availability := 0.0
	if upCount+downCount > 0 {
		availability = float64(upCount) / float64(upCount+downCount) * 100
	}

	averageLatency := "Unavailable"
	if latencyCount > 0 {
		averageLatency = fmt.Sprintf("%dms", latencyTotal/latencyCount)
	}

	lastStatus := "Pending"
	if latest := items[0]; latest != nil {
		if latest.OK {
			lastStatus = "Up"
		} else {
			lastStatus = "Down"
		}
	}

	return []summaryStat{
		{Label: "Availability", Value: fmt.Sprintf("%.1f%%", availability), Hint: "Recent checks marked as up."},
		{Label: "Average latency", Value: averageLatency, Hint: "Mean latency across recent successful checks."},
		{Label: "Last status", Value: lastStatus, Hint: "Most recent monitor result."},
		{Label: "Recent checks", Value: fmt.Sprintf("%d", upCount+downCount), Hint: "Samples used in the chart below."},
	}
}

func buildChartsJSON(items []*uptimemonitor.Result) (string, error) {
	payload := resultChartPayload{}
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item == nil {
			continue
		}

		payload.Labels = append(payload.Labels, item.CheckedAt.Format("01-02 15:04"))
		if item.OK {
			payload.Status = append(payload.Status, 1)
			payload.Up++
		} else {
			payload.Status = append(payload.Status, 0)
			payload.Down++
		}
		payload.Latency = append(payload.Latency, item.LatencyMS)
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
