package ui

import (
	"net/http"
	"strconv"
	"strings"

	"codeberg.org/urutau-ltd/aile/v2/x/htmx"
)

const (
	DefaultListLimit = 10
	MaxListLimit     = 100
)

// WriteHTMLHeader enforces a consistent HTML content type for template responses.
func WriteHTMLHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

func OptionalString(value string) *string {
	if value == "" {
		return nil
	}

	return new(value)
}

// ParseListState reads shared query/form controls used by list screens.
func ParseListState(r *http.Request) (string, int) {
	_ = r.ParseForm()
	searchTerm := strings.TrimSpace(r.Form.Get("q"))
	limit := parseLimit(r.Form.Get("limit"))
	return searchTerm, limit
}

// IsHTMXListRequest detects HTMX requests that target a list fragment.
func IsHTMXListRequest(r *http.Request, target string, triggerIDs []string, triggerNames []string) bool {
	if !htmx.IsRequest(r) || htmx.IsBoosted(r) {
		return false
	}

	if htmx.TargetIs(r, target) {
		return true
	}

	if len(triggerIDs) > 0 && htmx.TriggerIs(r, triggerIDs...) {
		return true
	}

	return len(triggerNames) > 0 && htmx.TriggerNameIs(r, triggerNames...)
}

// IsHTMXEditorRequest detects HTMX requests that target a side editor panel.
func IsHTMXEditorRequest(r *http.Request, target string) bool {
	return htmx.IsRequest(r) && !htmx.IsBoosted(r) && htmx.TargetIs(r, target)
}

func parseLimit(raw string) int {
	if raw == "" {
		return DefaultListLimit
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return DefaultListLimit
	}

	if limit > MaxListLimit {
		return MaxListLimit
	}

	return limit
}
