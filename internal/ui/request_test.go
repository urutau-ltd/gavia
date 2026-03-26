package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseListState(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/providers?q=cloud&limit=500", nil)

	search, limit := ParseListState(req)

	if search != "cloud" {
		t.Fatalf("expected search term %q, got %q", "cloud", search)
	}

	if limit != MaxListLimit {
		t.Fatalf("expected capped limit %d, got %d", MaxListLimit, limit)
	}
}

func TestIsHTMXListRequest(t *testing.T) {
	t.Run("matches target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/providers", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", "#providers-body")

		if !IsHTMXListRequest(req, "providers-body", []string{"provider-search"}, []string{"q"}) {
			t.Fatal("expected list request to match target")
		}
	})

	t.Run("matches trigger name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/providers", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Trigger-Name", "q")

		if !IsHTMXListRequest(req, "providers-body", []string{"provider-search"}, []string{"q"}) {
			t.Fatal("expected list request to match trigger name")
		}
	})

	t.Run("ignores boosted requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/providers", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Boosted", "true")
		req.Header.Set("HX-Target", "#providers-body")

		if IsHTMXListRequest(req, "providers-body", []string{"provider-search"}, []string{"q"}) {
			t.Fatal("expected boosted request to be ignored")
		}
	})
}

func TestIsHTMXEditorRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/providers/new", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "provider-editor")

	if !IsHTMXEditorRequest(req, "provider-editor") {
		t.Fatal("expected editor request to match target")
	}
}
