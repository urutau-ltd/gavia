package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"codeberg.org/urutau-ltd/aile/v2"
)

type stubDashboardHandler struct{}

func (stubDashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("dashboard"))
}

type stubCollectionHandler struct {
	prefix string
}

func (h stubCollectionHandler) Index(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(h.prefix + " index"))
}

func (h stubCollectionHandler) New(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(h.prefix + " new"))
}

func (h stubCollectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(h.prefix + " create"))
}

func (h stubCollectionHandler) Show(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(h.prefix + " show " + r.PathValue("id")))
}

func (h stubCollectionHandler) Edit(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(h.prefix + " edit " + r.PathValue("id")))
}

func (h stubCollectionHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(h.prefix + " update " + r.PathValue("id")))
}

func (h stubCollectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

type stubSingletonHandler struct {
	prefix string
}

func (h stubSingletonHandler) Show(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(h.prefix + " show"))
}

func (h stubSingletonHandler) Edit(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(h.prefix + " edit"))
}

func (h stubSingletonHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(h.prefix + " update"))
}

type stubLoginHandler struct{}

func (stubLoginHandler) Show(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("login show"))
}

func (stubLoginHandler) Submit(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("login submit"))
}

type stubLogoutHandler struct{}

func (stubLogoutHandler) Perform(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("logout"))
}

type stubAppSettingsHandler struct{}

func (stubAppSettingsHandler) Show(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("app settings show"))
}

func (stubAppSettingsHandler) Edit(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("app settings edit"))
}

func (stubAppSettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("app settings update"))
}

func (stubAppSettingsHandler) Export(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("app settings export"))
}

func (stubAppSettingsHandler) Import(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("app settings import"))
}

func (stubAppSettingsHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("app settings create expense"))
}

func (stubAppSettingsHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("app settings delete expense " + r.PathValue("id")))
}

type stubBackupAPIHandler struct{}

func (stubBackupAPIHandler) Export(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("backup api export"))
}

func (stubBackupAPIHandler) Import(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("backup api import"))
}

type stubDashboardAPIHandler struct{}

func (stubDashboardAPIHandler) Summary(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("dashboard api summary"))
}

type stubLicensesHandler struct{}

func (stubLicensesHandler) Index(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("licenses"))
}

type stubUptimeHandler struct{}

func (stubUptimeHandler) Index(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("uptime index"))
}

func (stubUptimeHandler) Show(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("uptime show " + r.PathValue("id")))
}

func (stubUptimeHandler) Create(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("uptime create"))
}

func (stubUptimeHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("uptime update " + r.PathValue("id")))
}

func (stubUptimeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("uptime delete " + r.PathValue("id")))
}

func TestMountRoutes(t *testing.T) {
	app := aile.MustNew()
	err := mountRoutes(app, appHandlers{
		dashboard:       stubDashboardHandler{},
		provider:        stubCollectionHandler{prefix: "providers"},
		location:        stubCollectionHandler{prefix: "locations"},
		os:              stubCollectionHandler{prefix: "os"},
		ip:              stubCollectionHandler{prefix: "ips"},
		dns:             stubCollectionHandler{prefix: "dns"},
		label:           stubCollectionHandler{prefix: "labels"},
		domain:          stubCollectionHandler{prefix: "domains"},
		hosting:         stubCollectionHandler{prefix: "hostings"},
		server:          stubCollectionHandler{prefix: "servers"},
		subscription:    stubCollectionHandler{prefix: "subscriptions"},
		accountSettings: stubSingletonHandler{prefix: "account settings"},
		appSettings:     stubAppSettingsHandler{},
		login:           stubLoginHandler{},
		logout:          stubLogoutHandler{},
		backupAPI:       stubBackupAPIHandler{},
		dashboardAPI:    stubDashboardAPIHandler{},
		licenses:        stubLicensesHandler{},
		uptime:          stubUptimeHandler{},
	})
	if err != nil {
		t.Fatalf("mountRoutes returned error: %v", err)
	}

	state, err := app.Build(context.Background())
	if err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		statusCode int
		location   string
		body       string
	}{
		{
			name:       "root redirect",
			method:     http.MethodGet,
			path:       "/",
			statusCode: http.StatusSeeOther,
			location:   "/dashboard",
		},
		{
			name:       "dashboard route",
			method:     http.MethodGet,
			path:       "/dashboard",
			statusCode: http.StatusOK,
			body:       "dashboard",
		},
		{
			name:       "providers new route",
			method:     http.MethodGet,
			path:       "/providers/new",
			statusCode: http.StatusOK,
			body:       "providers new",
		},
		{
			name:       "locations create route",
			method:     http.MethodPost,
			path:       "/locations",
			statusCode: http.StatusCreated,
			body:       "locations create",
		},
		{
			name:       "operating systems edit route",
			method:     http.MethodGet,
			path:       "/os/42/edit",
			statusCode: http.StatusOK,
			body:       "os edit 42",
		},
		{
			name:       "providers delete route",
			method:     http.MethodDelete,
			path:       "/providers/42",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "domains show route",
			method:     http.MethodGet,
			path:       "/domains/42",
			statusCode: http.StatusOK,
			body:       "domains show 42",
		},
		{
			name:       "ips new route",
			method:     http.MethodGet,
			path:       "/ips/new",
			statusCode: http.StatusOK,
			body:       "ips new",
		},
		{
			name:       "dns edit route",
			method:     http.MethodGet,
			path:       "/dns/42/edit",
			statusCode: http.StatusOK,
			body:       "dns edit 42",
		},
		{
			name:       "labels delete route",
			method:     http.MethodDelete,
			path:       "/labels/42",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "hostings new route",
			method:     http.MethodGet,
			path:       "/hostings/new",
			statusCode: http.StatusOK,
			body:       "hostings new",
		},
		{
			name:       "servers update route",
			method:     http.MethodPost,
			path:       "/servers/42/edit",
			statusCode: http.StatusOK,
			body:       "servers update 42",
		},
		{
			name:       "subscriptions delete route",
			method:     http.MethodDelete,
			path:       "/subscriptions/42",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "account settings edit route",
			method:     http.MethodGet,
			path:       "/account-settings/edit",
			statusCode: http.StatusOK,
			body:       "account settings edit",
		},
		{
			name:       "app settings export route",
			method:     http.MethodGet,
			path:       "/app-settings/export",
			statusCode: http.StatusOK,
			body:       "app settings export",
		},
		{
			name:       "login submit route",
			method:     http.MethodPost,
			path:       "/login",
			statusCode: http.StatusOK,
			body:       "login submit",
		},
		{
			name:       "logout route",
			method:     http.MethodPost,
			path:       "/logout",
			statusCode: http.StatusOK,
			body:       "logout",
		},
		{
			name:       "expense create route",
			method:     http.MethodPost,
			path:       "/app-settings/expenses",
			statusCode: http.StatusOK,
			body:       "app settings create expense",
		},
		{
			name:       "expense delete route",
			method:     http.MethodPost,
			path:       "/app-settings/expenses/42/delete",
			statusCode: http.StatusOK,
			body:       "app settings delete expense 42",
		},
		{
			name:       "backup api export route",
			method:     http.MethodGet,
			path:       "/api/v1/backup/export",
			statusCode: http.StatusOK,
			body:       "backup api export",
		},
		{
			name:       "dashboard api summary route",
			method:     http.MethodGet,
			path:       "/api/v1/dashboard/summary",
			statusCode: http.StatusOK,
			body:       "dashboard api summary",
		},
		{
			name:       "licenses route",
			method:     http.MethodGet,
			path:       "/licenses",
			statusCode: http.StatusOK,
			body:       "licenses",
		},
		{
			name:       "uptime index route",
			method:     http.MethodGet,
			path:       "/uptime",
			statusCode: http.StatusOK,
			body:       "uptime index",
		},
		{
			name:       "uptime update route",
			method:     http.MethodPost,
			path:       "/uptime/42/edit",
			statusCode: http.StatusOK,
			body:       "uptime update 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			state.Handler.ServeHTTP(rec, req)

			if rec.Code != tt.statusCode {
				t.Fatalf("expected status %d, got %d", tt.statusCode, rec.Code)
			}

			if tt.location != "" && rec.Header().Get("Location") != tt.location {
				t.Fatalf("expected redirect to %q, got %q", tt.location, rec.Header().Get("Location"))
			}

			if tt.body != "" && rec.Body.String() != tt.body {
				t.Fatalf("expected body %q, got %q", tt.body, rec.Body.String())
			}
		})
	}
}
