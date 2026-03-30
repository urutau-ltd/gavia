package ui

import (
	"net/http"

	vexhtmx "codeberg.org/urutau-ltd/vexilo/htmx"
	vexrequeststate "codeberg.org/urutau-ltd/vexilo/requeststate"
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
	state := vexrequeststate.Parse(r, DefaultListLimit, MaxListLimit)
	return state.Query, state.Limit
}

// IsHTMXListRequest detects HTMX requests that target a list fragment.
func IsHTMXListRequest(r *http.Request, target string, triggerIDs []string, triggerNames []string) bool {
	return vexhtmx.IsListRequest(r, target, triggerIDs, triggerNames)
}

// IsHTMXEditorRequest detects HTMX requests that target a side editor panel.
func IsHTMXEditorRequest(r *http.Request, target string) bool {
	return vexhtmx.IsEditorRequest(r, target)
}
