package main

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	domainmodel "codeberg.org/urutau-ltd/gavia/internal/models/domain"
	"codeberg.org/urutau-ltd/gavia/internal/models/location"
	operatingsystem "codeberg.org/urutau-ltd/gavia/internal/models/operating_system"
	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/dns"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/domains"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/hostings"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/ips"
	providersui "codeberg.org/urutau-ltd/gavia/internal/ui/features/providers"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/servers"
	"codeberg.org/urutau-ltd/gavia/internal/ui/features/subscriptions"
)

func TestNewCRUDFormsRenderExpectedOptions(t *testing.T) {
	db := openFlowTestDB(t)
	runFlowMigrations(t, db)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uiRoot, err := fs.Sub(UIFS, "internal/ui")
	if err != nil {
		t.Fatalf("fs.Sub returned error: %v", err)
	}

	providerRepo := provider.NewProviderRepository(db)
	providers, err := providerRepo.GetAll(t.Context(), "", 1)
	if err != nil {
		t.Fatalf("GetAll returned error: %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("expected seeded providers to exist")
	}

	locationRepo := location.NewLocationRepository(db)
	loc := &location.Location{Name: "Monterrey"}
	if err := locationRepo.Create(t.Context(), loc); err != nil {
		t.Fatalf("Create location returned error: %v", err)
	}

	osRepo := operatingsystem.NewOperatingSystemRepository(db)
	osItem := &operatingsystem.OperatingSystem{Name: "Ubuntu 24.04"}
	if err := osRepo.Create(t.Context(), osItem); err != nil {
		t.Fatalf("Create operating system returned error: %v", err)
	}

	domainRepo := domainmodel.NewRepository(db)
	domainItem := &domainmodel.Domain{
		Domain:     "example.com",
		ProviderID: &providers[0].Id,
		Currency:   "USD",
		DueDate:    stringPointer(time.Now().UTC().Format(time.DateOnly)),
	}
	if err := domainRepo.Create(t.Context(), domainItem); err != nil {
		t.Fatalf("Create domain returned error: %v", err)
	}

	tests := []struct {
		name     string
		run      func(http.ResponseWriter, *http.Request)
		path     string
		target   string
		mustHave []string
	}{
		{
			name:   "servers new form includes operating systems",
			run:    servers.NewHandler(logger, uiRoot, db).New,
			path:   "/servers/new",
			target: "server-editor",
			mustHave: []string{
				"Create server",
				"Ubuntu 24.04",
				"Monterrey",
			},
		},
		{
			name:   "ips new form includes type options and helper text",
			run:    ips.NewHandler(logger, uiRoot, db).New,
			path:   "/ips/new",
			target: "ip-editor",
			mustHave: []string{
				"IPv4",
				"IPv6",
				`Origin" is built from organization or country`,
			},
		},
		{
			name:   "dns new form includes linked domains",
			run:    dns.NewHandler(logger, uiRoot, db).New,
			path:   "/dns/new",
			target: "dns-editor",
			mustHave: []string{
				"Create DNS record",
				"example.com",
			},
		},
		{
			name:   "domains new form includes providers",
			run:    domains.NewHandler(logger, uiRoot, db).New,
			path:   "/domains/new",
			target: "domain-editor",
			mustHave: []string{
				"Create domain",
				"Cloudflare Registrar",
			},
		},
		{
			name:   "hostings new form includes related options",
			run:    hostings.NewHandler(logger, uiRoot, db).New,
			path:   "/hostings/new",
			target: "hosting-editor",
			mustHave: []string{
				"Create hosting",
				"example.com",
				"Monterrey",
			},
		},
		{
			name:   "subscriptions new form includes constrained currency choices",
			run:    subscriptions.NewHandler(logger, uiRoot, db).New,
			path:   "/subscriptions/new",
			target: "subscription-editor",
			mustHave: []string{
				"Renewal cadence",
				`option value="MXN"`,
				`option value="USD"`,
				`option value="XMR"`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("HX-Request", "true")
			req.Header.Set("HX-Target", tc.target)
			rec := httptest.NewRecorder()

			tc.run(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			body := rec.Body.String()
			for _, want := range tc.mustHave {
				if !strings.Contains(body, want) {
					t.Fatalf("expected response body to include %q, got %q", want, body)
				}
			}
		})
	}
}

func TestProvidersCreateHTMXResponseWrapsListBodyInHiddenTable(t *testing.T) {
	db := openFlowTestDB(t)
	runFlowMigrations(t, db)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uiRoot, err := fs.Sub(UIFS, "internal/ui")
	if err != nil {
		t.Fatalf("fs.Sub returned error: %v", err)
	}

	form := url.Values{
		"name":    {"Urutau Limited"},
		"website": {"https://urutau-ltd.org/"},
	}

	req := httptest.NewRequest(http.MethodPost, "/providers", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "provider-editor")
	rec := httptest.NewRecorder()

	providersui.NewHandler(logger, uiRoot, db).Create(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"Provider created successfully.",
		`<table hidden aria-hidden="true">`,
		`<tbody id="providers-body" hx-swap-oob="outerHTML">`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected HTMX create response to include %q, got %q", want, body)
		}
	}
}

func stringPointer(value string) *string {
	return &value
}
