package dashboardapi

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"codeberg.org/urutau-ltd/gavia/internal/finance"
	appsetting "codeberg.org/urutau-ltd/gavia/internal/models/app_setting"
	dashboardsummary "codeberg.org/urutau-ltd/gavia/internal/models/dashboard_summary"
	exchangerate "codeberg.org/urutau-ltd/gavia/internal/models/exchange_rate"
	expenseentry "codeberg.org/urutau-ltd/gavia/internal/models/expense_entry"
	"codeberg.org/urutau-ltd/gavia/internal/models/location"
	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
	runtimesample "codeberg.org/urutau-ltd/gavia/internal/models/runtime_sample"
	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
)

type Handler struct {
	logger       *slog.Logger
	appRepo      *appsetting.AppSettingsRepository
	dueRepo      *dashboardsummary.Repository
	rateRepo     *exchangerate.Repository
	expenseRepo  *expenseentry.ExpenseEntryRepository
	locationRepo *location.LocationRepository
	providerRepo *provider.ProviderRepository
	runtimeRepo  *runtimesample.Repository
	uptimeRepo   *uptimemonitor.Repository
}

type summaryResponse struct {
	GeneratedAt time.Time               `json:"generated_at"`
	Settings    *appsetting.AppSettings `json:"settings"`
	Stats       summaryStats            `json:"stats"`
	DueSoon     dueSoonSummary          `json:"due_soon"`
	Currency    currencySummary         `json:"currency"`
	Expenses    expenseSummary          `json:"expenses"`
	Diagnostics diagnosticsSummary      `json:"diagnostics"`
	Uptime      uptimeSummaryResponse   `json:"uptime"`
}

type summaryStats struct {
	ProviderCount int `json:"provider_count"`
	LocationCount int `json:"location_count"`
}

type dueSoonSummary struct {
	Items    []*dashboardsummary.DueItem     `json:"items"`
	Total    float64                         `json:"total"`
	BySource []dashboardsummary.AmountBucket `json:"by_source"`
}

type expenseSummary struct {
	Items      []*expenseentry.ExpenseEntry    `json:"items"`
	ByCategory []dashboardsummary.AmountBucket `json:"by_category"`
}

type currencySummary struct {
	Latest latestRates      `json:"latest"`
	Totals []currencyTotals `json:"totals"`
}

type latestRates struct {
	MXNToUSD *float64 `json:"mxn_to_usd"`
	MXNToXMR *float64 `json:"mxn_to_xmr"`
	XMRToUSD *float64 `json:"xmr_to_usd"`
}

type currencyTotals struct {
	Currency  string  `json:"currency"`
	Value     float64 `json:"value"`
	Available bool    `json:"available"`
}

type diagnosticsSummary struct {
	Latest *runtimesample.Sample `json:"latest"`
}

type uptimeSummaryResponse struct {
	Summary  *uptimemonitor.Summary         `json:"summary"`
	Monitors []*uptimemonitor.MonitorStatus `json:"monitors"`
}

func NewHandler(logger *slog.Logger, db *sql.DB) *Handler {
	return &Handler{
		logger:       logger,
		appRepo:      appsetting.NewAppSettingsRepository(db),
		dueRepo:      dashboardsummary.NewRepository(db),
		rateRepo:     exchangerate.NewRepository(db),
		expenseRepo:  expenseentry.NewExpenseEntryRepository(db),
		locationRepo: location.NewLocationRepository(db),
		providerRepo: provider.NewProviderRepository(db),
		runtimeRepo:  runtimesample.NewRepository(db),
		uptimeRepo:   uptimemonitor.NewRepository(db),
	}
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	settings, err := h.appRepo.Get(r.Context())
	if err != nil {
		http.Error(w, "Unable to load app settings.", http.StatusInternalServerError)
		return
	}
	if settings == nil {
		settings = appsetting.Defaults()
	}

	providerCount, err := h.providerRepo.Count(r.Context())
	if err != nil {
		http.Error(w, "Unable to count providers.", http.StatusInternalServerError)
		return
	}

	locationCount, err := h.locationRepo.Count(r.Context())
	if err != nil {
		http.Error(w, "Unable to count locations.", http.StatusInternalServerError)
		return
	}

	dueItems, err := h.dueRepo.UpcomingDue(r.Context(), settings.DashboardDueSoonAmount)
	if err != nil {
		http.Error(w, "Unable to load due-soon items.", http.StatusInternalServerError)
		return
	}

	expenses, err := h.expenseRepo.GetRecent(r.Context(), 5)
	if err != nil {
		http.Error(w, "Unable to load expense entries.", http.StatusInternalServerError)
		return
	}

	allExpenses, err := h.expenseRepo.GetAll(r.Context())
	if err != nil {
		http.Error(w, "Unable to load expense totals.", http.StatusInternalServerError)
		return
	}

	runtimeSamples, err := h.runtimeRepo.GetRecent(r.Context(), 1)
	if err != nil {
		http.Error(w, "Unable to load runtime diagnostics.", http.StatusInternalServerError)
		return
	}

	uptimeSummary, err := h.uptimeRepo.GetSummary(r.Context())
	if err != nil {
		http.Error(w, "Unable to load uptime summary.", http.StatusInternalServerError)
		return
	}

	uptimeMonitors, err := h.uptimeRepo.GetAll(r.Context(), 5)
	if err != nil {
		http.Error(w, "Unable to load uptime monitors.", http.StatusInternalServerError)
		return
	}

	mxnUSDLatest, err := h.rateRepo.GetLatest(r.Context(), "MXN", "USD")
	if err != nil {
		http.Error(w, "Unable to load MXN to USD rate.", http.StatusInternalServerError)
		return
	}

	xmrUSDLatest, err := h.rateRepo.GetLatest(r.Context(), "XMR", "USD")
	if err != nil {
		http.Error(w, "Unable to load XMR to USD rate.", http.StatusInternalServerError)
		return
	}

	converter := finance.NewConverter(compactRateSamples(mxnUSDLatest, xmrUSDLatest))
	currency := buildCurrencySummary(dueItems, settings.DefaultCurrency, converter, mxnUSDLatest, xmrUSDLatest)

	response := summaryResponse{
		GeneratedAt: time.Now().UTC(),
		Settings:    settings,
		Stats: summaryStats{
			ProviderCount: providerCount,
			LocationCount: locationCount,
		},
		DueSoon: dueSoonSummary{
			Items:    dueItems,
			Total:    sumDueItems(dueItems),
			BySource: dashboardsummary.AggregateByLabel(dueItems, dueLabel, dueAmount),
		},
		Currency: currency,
		Expenses: expenseSummary{
			Items: expenses,
			ByCategory: dashboardsummary.AggregateByLabel(
				allExpenses,
				func(item *expenseentry.ExpenseEntry) string {
					if item == nil {
						return ""
					}
					category := strings.TrimSpace(item.Category)
					if category == "" {
						category = "Uncategorized"
					}

					currency := strings.ToUpper(strings.TrimSpace(item.Currency))
					if currency == "" {
						currency = "MXN"
					}

					return category + " · " + currency
				},
				func(item *expenseentry.ExpenseEntry) float64 {
					if item == nil {
						return 0
					}
					return item.Amount
				},
			),
		},
		Diagnostics: diagnosticsSummary{
			Latest: firstRuntimeSample(runtimeSamples),
		},
		Uptime: uptimeSummaryResponse{
			Summary:  uptimeSummary,
			Monitors: uptimeMonitors,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode dashboard summary response", "err", err)
	}
}

func sumDueItems(items []*dashboardsummary.DueItem) float64 {
	total := 0.0
	for _, item := range items {
		if item == nil || item.Price == nil {
			continue
		}
		total += *item.Price
	}
	return total
}

func dueLabel(item *dashboardsummary.DueItem) string {
	if item == nil {
		return ""
	}
	return item.SourceType
}

func dueAmount(item *dashboardsummary.DueItem) float64 {
	if item == nil || item.Price == nil {
		return 0
	}
	return *item.Price
}

func compactRateSamples(items ...*exchangerate.Sample) []*exchangerate.Sample {
	result := make([]*exchangerate.Sample, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, item)
		}
	}
	return result
}

func buildCurrencySummary(
	items []*dashboardsummary.DueItem,
	defaultCurrency string,
	converter *finance.Converter,
	mxnUSDLatest *exchangerate.Sample,
	xmrUSDLatest *exchangerate.Sample,
) currencySummary {
	summary := currencySummary{
		Latest: latestRates{
			MXNToUSD: sampleRate(mxnUSDLatest),
			XMRToUSD: sampleRate(xmrUSDLatest),
		},
		Totals: make([]currencyTotals, 0, 3),
	}

	if summary.Latest.MXNToUSD != nil && summary.Latest.XMRToUSD != nil && *summary.Latest.XMRToUSD > 0 {
		value := *summary.Latest.MXNToUSD / *summary.Latest.XMRToUSD
		summary.Latest.MXNToXMR = &value
	}

	for _, currency := range []string{"MXN", "USD", "XMR"} {
		total := 0.0
		available := false
		for _, item := range items {
			if item == nil || item.Price == nil {
				continue
			}
			converted, ok := convertAmount(*item.Price, item.Currency, currency, defaultCurrency, converter)
			if !ok {
				continue
			}
			total += converted
			available = true
		}

		summary.Totals = append(summary.Totals, currencyTotals{
			Currency:  currency,
			Value:     total,
			Available: available,
		})
	}

	return summary
}

func sampleRate(sample *exchangerate.Sample) *float64 {
	if sample == nil {
		return nil
	}
	value := sample.Rate
	return &value
}

func convertAmount(
	amount float64,
	fromCurrency string,
	toCurrency string,
	defaultCurrency string,
	converter *finance.Converter,
) (float64, bool) {
	fromCurrency = strings.ToUpper(strings.TrimSpace(fromCurrency))
	if fromCurrency == "" {
		fromCurrency = strings.ToUpper(strings.TrimSpace(defaultCurrency))
	}
	toCurrency = strings.ToUpper(strings.TrimSpace(toCurrency))
	if fromCurrency == "" || toCurrency == "" {
		return 0, false
	}
	if fromCurrency == toCurrency {
		return amount, true
	}
	if converter == nil {
		return 0, false
	}
	return converter.Convert(amount, fromCurrency, toCurrency)
}

func firstRuntimeSample(samples []*runtimesample.Sample) *runtimesample.Sample {
	if len(samples) == 0 {
		return nil
	}
	return samples[0]
}
