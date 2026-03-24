package dashboard

import (
	"database/sql"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"codeberg.org/urutau-ltd/gavia/internal/models/location"
	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger       *slog.Logger
	tmpl         *template.Template
	locationRepo *location.LocationRepository
	providerRepo *provider.ProviderRepository
}

type statCard struct {
	Label string
	Value int
	Hint  string
	Tone  string
}

type moduleCard struct {
	Name    string
	Status  string
	Summary string
	Href    string
	Started bool
}

func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/dashboard/views/index.html",
		"components/*.html",
	))

	return &Handler{
		logger:       l,
		locationRepo: location.NewLocationRepository(db),
		tmpl:         t,
		providerRepo: provider.NewProviderRepository(db),
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	providers, err := h.providerRepo.GetAll(r.Context(), "", 5)
	if err != nil {
		h.logger.Error("Failed to load dashboard providers", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	locations, err := h.locationRepo.GetAll(r.Context(), "", 5)
	if err != nil {
		h.logger.Error("Failed to load dashboard locations", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	providerCount, err := h.providerRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count providers", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	locationCount, err := h.locationRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count locations", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	modules := []moduleCard{
		{
			Name:    "Providers",
			Status:  "ready",
			Summary: "CRUD base available with list, search and side editor.",
			Href:    "/providers",
			Started: true,
		},
		{
			Name:    "Locations",
			Status:  "ready",
			Summary: "CRUD base available with search and responsive list layout.",
			Href:    "/locations",
			Started: true,
		},
		{
			Name:    "Operating systems",
			Status:  "next",
			Summary: "Table exists in migrations, but the UI module has not started yet.",
			Started: false,
		},
		{
			Name:    "Domains",
			Status:  "next",
			Summary: "Good candidate for the next CRUD after providers.",
			Started: false,
		},
		{
			Name:    "Hostings",
			Status:  "next",
			Summary: "Already modeled in SQL and ready for HTMX scaffolding.",
			Started: false,
		},
		{
			Name:    "Servers",
			Status:  "next",
			Summary: "Schema exists and can follow the same repository + editor pattern.",
			Started: false,
		},
		{
			Name:    "IPs",
			Status:  "backlog",
			Summary: "Schema is present for inventory enrichment later on.",
			Started: false,
		},
		{
			Name:    "DNS records",
			Status:  "backlog",
			Summary: "Table is ready for a focused CRUD once core inventory is in place.",
			Started: false,
		},
	}

	startedModules := 0
	for _, module := range modules {
		if module.Started {
			startedModules++
		}
	}

	stats := []statCard{
		{
			Label: "Providers",
			Value: providerCount,
			Hint:  "Records available right now.",
			Tone:  "ok",
		},
		{
			Label: "Locations",
			Value: locationCount,
			Hint:  "Places already tracked in the inventory.",
			Tone:  "info",
		},
		{
			Label: "Started modules",
			Value: startedModules,
			Hint:  "CRUD areas already wired end-to-end.",
			Tone:  "warn",
		},
		{
			Label: "Pending modules",
			Value: len(modules) - startedModules,
			Hint:  "Schemas that still need their UI package.",
			Tone:  "bad",
		},
	}

	data := struct {
		ui.BaseData
		Providers     []*provider.Provider
		Locations     []*location.Location
		ProviderCount int
		LocationCount int
		Stats         []statCard
		Modules       []moduleCard
	}{
		BaseData:      ui.NewBaseData("Dashboard", start),
		Providers:     providers,
		Locations:     locations,
		ProviderCount: providerCount,
		LocationCount: locationCount,
		Stats:         stats,
		Modules:       modules,
	}

	h.logger.Info("Dashboard loaded",
		"provider_count", providerCount,
		"location_count", locationCount,
		"started_modules", startedModules,
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Error rendering dashboard", "err", err)
	}
}
