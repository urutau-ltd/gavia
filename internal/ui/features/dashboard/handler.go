package dashboard

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"codeberg.org/urutau-ltd/gavia/internal/finance"
	appsetting "codeberg.org/urutau-ltd/gavia/internal/models/app_setting"
	dashboardsummary "codeberg.org/urutau-ltd/gavia/internal/models/dashboard_summary"
	"codeberg.org/urutau-ltd/gavia/internal/models/dnsrecord"
	"codeberg.org/urutau-ltd/gavia/internal/models/domain"
	exchangerate "codeberg.org/urutau-ltd/gavia/internal/models/exchange_rate"
	expenseentry "codeberg.org/urutau-ltd/gavia/internal/models/expense_entry"
	"codeberg.org/urutau-ltd/gavia/internal/models/hosting"
	ipmodel "codeberg.org/urutau-ltd/gavia/internal/models/ip"
	labelmodel "codeberg.org/urutau-ltd/gavia/internal/models/label"
	"codeberg.org/urutau-ltd/gavia/internal/models/location"
	operatingsystem "codeberg.org/urutau-ltd/gavia/internal/models/operating_system"
	"codeberg.org/urutau-ltd/gavia/internal/models/provider"
	runtimesample "codeberg.org/urutau-ltd/gavia/internal/models/runtime_sample"
	servermodel "codeberg.org/urutau-ltd/gavia/internal/models/server"
	"codeberg.org/urutau-ltd/gavia/internal/models/subscription"
	uptimemonitor "codeberg.org/urutau-ltd/gavia/internal/models/uptime_monitor"
	"codeberg.org/urutau-ltd/gavia/internal/ui"
)

type Handler struct {
	logger       *slog.Logger
	tmpl         *template.Template
	appRepo      *appsetting.AppSettingsRepository
	dueRepo      *dashboardsummary.Repository
	rateRepo     *exchangerate.Repository
	expenseRepo  *expenseentry.ExpenseEntryRepository
	osRepo       *operatingsystem.OperatingSystemRepository
	ipRepo       *ipmodel.Repository
	dnsRepo      *dnsrecord.Repository
	labelRepo    *labelmodel.Repository
	domainRepo   *domain.Repository
	hostingRepo  *hosting.Repository
	serverRepo   *servermodel.Repository
	subRepo      *subscription.Repository
	locationRepo *location.LocationRepository
	providerRepo *provider.ProviderRepository
	runtimeRepo  *runtimesample.Repository
	uptimeRepo   *uptimemonitor.Repository
}

type statCard struct {
	Label string
	Value int
	Hint  string
	Tone  string
}

type inventoryItem struct {
	Label string
	Count int
	Hint  string
	Href  string
	Share float64
}

type dueSummary struct {
	Items []dashboardsummary.DueItem
	Total float64
}

type breakdownItem struct {
	Label string
	Hint  string
	Value string
	Share float64
}

type conversionItem struct {
	Currency  string
	Value     string
	Available bool
}

type diagnosticItem struct {
	Label string
	Value string
	Hint  string
}

type uptimeRow struct {
	Name       string
	StatusTone string
	StatusText string
	TargetURL  string
}

type chartPayload struct {
	ExpenseHistory struct {
		Labels []string  `json:"labels"`
		MXN    []float64 `json:"mxn"`
		USD    []float64 `json:"usd"`
		XMR    []float64 `json:"xmr"`
	} `json:"expense_history"`
	InventoryDistribution struct {
		Labels []string `json:"labels"`
		Counts []int    `json:"counts"`
	} `json:"inventory_distribution"`
	FXHistory struct {
		Labels   []string   `json:"labels"`
		MXNToUSD []*float64 `json:"mxn_to_usd"`
		MXNToXMR []*float64 `json:"mxn_to_xmr"`
	} `json:"fx_history"`
	RuntimeHistory struct {
		Labels            []string  `json:"labels"`
		HeapAllocMB       []float64 `json:"heap_alloc_mb"`
		Goroutines        []int     `json:"goroutines"`
		DBOpenConnections []int     `json:"db_open_connections"`
	} `json:"runtime_history"`
	UptimeStatus struct {
		Labels []string `json:"labels"`
		Counts []int    `json:"counts"`
	} `json:"uptime_status"`
}

func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {
	t := template.Must(template.ParseFS(uiFS,
		"layout/base.html",
		"features/dashboard/views/index.html",
		"components/*.html",
	))

	return &Handler{
		logger:       l,
		appRepo:      appsetting.NewAppSettingsRepository(db),
		dueRepo:      dashboardsummary.NewRepository(db),
		rateRepo:     exchangerate.NewRepository(db),
		expenseRepo:  expenseentry.NewExpenseEntryRepository(db),
		osRepo:       operatingsystem.NewOperatingSystemRepository(db),
		ipRepo:       ipmodel.NewRepository(db),
		dnsRepo:      dnsrecord.NewRepository(db),
		labelRepo:    labelmodel.NewRepository(db),
		domainRepo:   domain.NewRepository(db),
		hostingRepo:  hosting.NewRepository(db),
		serverRepo:   servermodel.NewRepository(db),
		subRepo:      subscription.NewRepository(db),
		locationRepo: location.NewLocationRepository(db),
		tmpl:         t,
		providerRepo: provider.NewProviderRepository(db),
		runtimeRepo:  runtimesample.NewRepository(db),
		uptimeRepo:   uptimemonitor.NewRepository(db),
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

	osCount, err := h.osRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count operating systems", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	ipCount, err := h.ipRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count IP addresses", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	dnsCount, err := h.dnsRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count DNS records", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	labelCount, err := h.labelRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count labels", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	domainCount, err := h.domainRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count domains", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	hostingCount, err := h.hostingRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count hostings", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	serverCount, err := h.serverRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count servers", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	subscriptionCount, err := h.subRepo.Count(r.Context())
	if err != nil {
		h.logger.Error("Failed to count subscriptions", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	settings, err := h.appRepo.Get(r.Context())
	if err != nil {
		h.logger.Error("Failed to load app settings for dashboard", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}
	if settings == nil {
		settings = appsetting.Defaults()
	}

	dueItems, err := h.dueRepo.UpcomingDue(r.Context(), settings.DashboardDueSoonAmount)
	if err != nil {
		h.logger.Error("Failed to load due-soon items", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	expenses, err := h.expenseRepo.GetRecent(r.Context(), 5)
	if err != nil {
		h.logger.Error("Failed to load recent expense entries", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}
	expenses = filterSpendEntries(expenses)

	allExpenses, err := h.expenseRepo.GetAll(r.Context())
	if err != nil {
		h.logger.Error("Failed to load expense breakdown", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}
	allExpenses = filterSpendEntries(allExpenses)

	runtimeSamples, err := h.runtimeRepo.GetRecent(r.Context(), 32)
	if err != nil {
		h.logger.Error("Failed to load runtime samples", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	mxnUSDLatest, err := h.rateRepo.GetLatest(r.Context(), "MXN", "USD")
	if err != nil {
		h.logger.Error("Failed to load MXN/USD rate", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	xmrUSDLatest, err := h.rateRepo.GetLatest(r.Context(), "XMR", "USD")
	if err != nil {
		h.logger.Error("Failed to load XMR/USD rate", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	mxnUSDSeries, err := h.rateRepo.GetRecent(r.Context(), "MXN", "USD", 32)
	if err != nil {
		h.logger.Error("Failed to load MXN/USD history", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	xmrUSDSeries, err := h.rateRepo.GetRecent(r.Context(), "XMR", "USD", 32)
	if err != nil {
		h.logger.Error("Failed to load XMR/USD history", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	uptimeSummary, err := h.uptimeRepo.GetSummary(r.Context())
	if err != nil {
		h.logger.Error("Failed to load uptime summary", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	uptimeStatuses, err := h.uptimeRepo.GetAll(r.Context(), 5)
	if err != nil {
		h.logger.Error("Failed to load uptime statuses", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	inventoryItems := buildInventoryItems([]inventoryItem{
		{Label: "Providers", Count: providerCount, Hint: "Vendors and registrars in the catalog.", Href: "/providers"},
		{Label: "Locations", Count: locationCount, Hint: "Cities and regions used across the inventory.", Href: "/locations"},
		{Label: "Operating systems", Count: osCount, Hint: "Selectable operating system catalog entries.", Href: "/os"},
		{Label: "IP addresses", Count: ipCount, Hint: "Public and private network endpoints.", Href: "/ips"},
		{Label: "DNS records", Count: dnsCount, Hint: "Tracked hostnames, records, and linked domains.", Href: "/dns"},
		{Label: "Labels", Count: labelCount, Hint: "Reusable tags for grouping infrastructure.", Href: "/labels"},
		{Label: "Domains", Count: domainCount, Hint: "Billable domain renewals and ownership records.", Href: "/domains"},
		{Label: "Hostings", Count: hostingCount, Hint: "Hosting plans with provider and domain linkage.", Href: "/hostings"},
		{Label: "Servers", Count: serverCount, Hint: "Compute nodes with OS, billing, IPs, and labels.", Href: "/servers"},
		{Label: "Subscriptions", Count: subscriptionCount, Hint: "Recurring SaaS or service subscriptions.", Href: "/subscriptions"},
	})

	catalogCount := providerCount + locationCount + osCount
	serviceAssetCount := domainCount + hostingCount + serverCount + subscriptionCount
	networkRecordCount := ipCount + dnsCount + labelCount
	totalInventoryCount := catalogCount + networkRecordCount + serviceAssetCount
	uptimeMonitorCount := 0
	if uptimeSummary != nil {
		uptimeMonitorCount = uptimeSummary.Total
	}

	converter := finance.NewConverter(compactRateSamples(mxnUSDLatest, xmrUSDLatest))
	chartsJSON, err := buildChartsJSON(
		allExpenses,
		settings.DefaultCurrency,
		converter,
		mxnUSDSeries,
		xmrUSDSeries,
		runtimeSamples,
		uptimeSummary,
		inventoryItems,
	)
	if err != nil {
		h.logger.Error("Failed to build dashboard chart data", "err", err)
		http.Error(w, "Not found", http.StatusInternalServerError)
		return
	}

	stats := []statCard{
		{Label: "Catalog records", Value: catalogCount, Hint: "Providers, locations, and operating systems.", Tone: "ok"},
		{Label: "Infrastructure assets", Value: serviceAssetCount, Hint: "Domains, hostings, servers, and subscriptions.", Tone: "info"},
		{Label: "Network records", Value: networkRecordCount, Hint: "IP addresses, DNS records, and labels.", Tone: "warn"},
		{Label: "Uptime monitors", Value: uptimeMonitorCount, Hint: "Checks currently configured in the app.", Tone: "bad"},
	}

	data := struct {
		ui.BaseData
		Providers        []*provider.Provider
		Locations        []*location.Location
		Stats            []statCard
		InventoryItems   []inventoryItem
		InventoryTotal   int
		AppSettings      *appsetting.AppSettings
		DueSoon          dueSummary
		DueBreakdown     []breakdownItem
		ExpenseBreakdown []breakdownItem
		ConvertedTotals  []conversionItem
		Diagnostics      []diagnosticItem
		UptimeSummary    *uptimemonitor.Summary
		UptimeRows       []uptimeRow
		ChartsJSON       template.JS
		Expenses         []*expenseentry.ExpenseEntry
	}{
		BaseData:         ui.NewBaseData(r, "Dashboard", start),
		Providers:        providers,
		Locations:        locations,
		Stats:            stats,
		InventoryItems:   inventoryItems,
		InventoryTotal:   totalInventoryCount,
		AppSettings:      settings,
		DueSoon:          buildDueSummary(dueItems, settings.DashboardCurrency, settings.DefaultCurrency, converter),
		DueBreakdown:     buildDueBreakdown(dueItems, settings.DashboardCurrency, settings.DefaultCurrency, converter),
		ExpenseBreakdown: buildExpenseBreakdown(allExpenses),
		ConvertedTotals:  buildConvertedTotals(dueItems, settings.DefaultCurrency, converter),
		Diagnostics:      buildDiagnostics(runtimeSamples),
		UptimeSummary:    uptimeSummary,
		UptimeRows:       buildUptimeRows(uptimeStatuses),
		ChartsJSON:       template.JS(chartsJSON),
		Expenses:         expenses,
	}

	h.logger.Info("Dashboard loaded",
		"provider_count", providerCount,
		"location_count", locationCount,
		"inventory_total", totalInventoryCount,
		"service_assets", serviceAssetCount,
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		h.logger.Error("Error rendering dashboard", "err", err)
	}
}

func buildDueSummary(
	items []*dashboardsummary.DueItem,
	targetCurrency string,
	defaultCurrency string,
	converter *finance.Converter,
) dueSummary {
	summary := dueSummary{Items: make([]dashboardsummary.DueItem, 0, len(items))}
	for _, item := range items {
		if item == nil {
			continue
		}
		summary.Items = append(summary.Items, *item)
		if item.Price != nil {
			if converted, ok := convertAmount(*item.Price, item.Currency, targetCurrency, defaultCurrency, converter); ok {
				summary.Total += converted
			}
		}
	}
	return summary
}

func buildDueBreakdown(
	items []*dashboardsummary.DueItem,
	targetCurrency string,
	defaultCurrency string,
	converter *finance.Converter,
) []breakdownItem {
	buckets := dashboardsummary.AggregateByLabel(
		items,
		func(item *dashboardsummary.DueItem) string {
			if item == nil {
				return ""
			}
			return item.SourceType
		},
		func(item *dashboardsummary.DueItem) float64 {
			if item == nil || item.Price == nil {
				return 0
			}
			converted, ok := convertAmount(*item.Price, item.Currency, targetCurrency, defaultCurrency, converter)
			if !ok {
				return 0
			}
			return converted
		},
	)

	targetCurrency = strings.ToUpper(strings.TrimSpace(targetCurrency))
	return buildBreakdownItems(buckets, func(bucket dashboardsummary.AmountBucket) string {
		return fmt.Sprintf("%s %.2f", targetCurrency, bucket.Amount)
	}, "item")
}

func buildExpenseBreakdown(items []*expenseentry.ExpenseEntry) []breakdownItem {
	buckets := dashboardsummary.AggregateByLabel(
		items,
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
			return fmt.Sprintf("%s · %s", category, currency)
		},
		func(item *expenseentry.ExpenseEntry) float64 {
			if item == nil {
				return 0
			}
			return item.Amount
		},
	)

	return buildBreakdownItems(buckets, func(bucket dashboardsummary.AmountBucket) string {
		return fmt.Sprintf("%.2f", bucket.Amount)
	}, "entry")
}

func filterSpendEntries(items []*expenseentry.ExpenseEntry) []*expenseentry.ExpenseEntry {
	filtered := make([]*expenseentry.ExpenseEntry, 0, len(items))
	for _, item := range items {
		if item == nil || !item.IsSpendLike() {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func buildInventoryItems(items []inventoryItem) []inventoryItem {
	if len(items) == 0 {
		return nil
	}

	result := append([]inventoryItem(nil), items...)
	total := 0
	for _, item := range result {
		total += item.Count
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Label < result[j].Label
		}
		return result[i].Count > result[j].Count
	})

	if total <= 0 {
		return result
	}

	for i := range result {
		result[i].Share = float64(result[i].Count) / float64(total) * 100
	}

	return result
}

func buildInventoryDistributionChart(items []inventoryItem) struct {
	Labels []string `json:"labels"`
	Counts []int    `json:"counts"`
} {
	result := struct {
		Labels []string `json:"labels"`
		Counts []int    `json:"counts"`
	}{}

	for _, item := range items {
		result.Labels = append(result.Labels, item.Label)
		result.Counts = append(result.Counts, item.Count)
	}

	return result
}

func buildBreakdownItems(
	buckets []dashboardsummary.AmountBucket,
	valueLabel func(dashboardsummary.AmountBucket) string,
	noun string,
) []breakdownItem {
	if len(buckets) == 0 {
		return nil
	}

	maxAmount := 0.0
	for _, bucket := range buckets {
		if bucket.Amount > maxAmount {
			maxAmount = bucket.Amount
		}
	}

	items := make([]breakdownItem, 0, len(buckets))
	for _, bucket := range buckets {
		share := 0.0
		if maxAmount > 0 {
			share = bucket.Amount / maxAmount * 100
		}
		items = append(items, breakdownItem{
			Label: bucket.Label,
			Hint:  countLabel(bucket.Count, noun),
			Value: valueLabel(bucket),
			Share: share,
		})
	}

	return items
}

func buildConvertedTotals(items []*dashboardsummary.DueItem, defaultCurrency string, converter *finance.Converter) []conversionItem {
	targets := []string{"MXN", "USD", "XMR"}
	totals := make([]conversionItem, 0, len(targets))
	for _, target := range targets {
		total := 0.0
		available := false
		for _, item := range items {
			if item == nil || item.Price == nil {
				continue
			}
			converted, ok := convertAmount(*item.Price, item.Currency, target, defaultCurrency, converter)
			if !ok {
				continue
			}
			total += converted
			available = true
		}

		entry := conversionItem{Currency: target, Available: available}
		if available {
			if target == "XMR" {
				entry.Value = fmt.Sprintf("%.6f", total)
			} else {
				entry.Value = fmt.Sprintf("%.2f", total)
			}
		} else {
			entry.Value = "Unavailable"
		}
		totals = append(totals, entry)
	}
	return totals
}

func buildDiagnostics(samples []*runtimesample.Sample) []diagnosticItem {
	if len(samples) == 0 || samples[0] == nil {
		return nil
	}

	latest := samples[0]
	return []diagnosticItem{
		{Label: "Goroutines", Value: fmt.Sprintf("%d", latest.Goroutines), Hint: "Current runtime concurrency."},
		{Label: "Heap alloc", Value: formatMB(latest.HeapAllocBytes), Hint: "Allocated heap."},
		{Label: "DB connections", Value: fmt.Sprintf("%d", latest.DBOpenConnections), Hint: "Current sqlite pool usage."},
		{Label: "Next GC", Value: formatMB(latest.NextGCBytes), Hint: "Approximate next GC target."},
		{Label: "CPUs", Value: fmt.Sprintf("%d", latest.CPUCount), Hint: "Runtime CPU visibility."},
	}
}

func buildUptimeRows(items []*uptimemonitor.MonitorStatus) []uptimeRow {
	rows := make([]uptimeRow, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}

		row := uptimeRow{Name: item.Name, TargetURL: item.TargetURL}
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
		rows = append(rows, row)
	}
	return rows
}

func buildChartsJSON(
	expenses []*expenseentry.ExpenseEntry,
	defaultCurrency string,
	converter *finance.Converter,
	mxnUSDSeries []*exchangerate.Sample,
	xmrUSDSeries []*exchangerate.Sample,
	runtimeSamples []*runtimesample.Sample,
	uptimeSummary *uptimemonitor.Summary,
	inventoryItems []inventoryItem,
) (string, error) {
	payload := chartPayload{}
	payload.ExpenseHistory = buildExpenseHistoryChart(expenses, defaultCurrency, converter, mxnUSDSeries, xmrUSDSeries)
	payload.InventoryDistribution = buildInventoryDistributionChart(inventoryItems)
	payload.FXHistory = buildFXHistoryChart(mxnUSDSeries, xmrUSDSeries)
	payload.RuntimeHistory = buildRuntimeHistoryChart(runtimeSamples)
	payload.UptimeStatus = buildUptimeStatusChart(uptimeSummary)

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func buildExpenseHistoryChart(
	expenses []*expenseentry.ExpenseEntry,
	defaultCurrency string,
	converter *finance.Converter,
	mxnUSDSeries []*exchangerate.Sample,
	xmrUSDSeries []*exchangerate.Sample,
) struct {
	Labels []string  `json:"labels"`
	MXN    []float64 `json:"mxn"`
	USD    []float64 `json:"usd"`
	XMR    []float64 `json:"xmr"`
} {
	type totals struct{ mxn, usd, xmr float64 }

	grouped := make(map[string]totals)
	labelSet := make(map[string]bool)
	var labels []string
	datedConverter := buildDatedConverter(mxnUSDSeries, xmrUSDSeries, converter)

	for _, item := range expenses {
		if item == nil {
			continue
		}
		date := strings.TrimSpace(item.OccurredOn)
		if date == "" {
			continue
		}
		if !labelSet[date] {
			labels = append(labels, date)
			labelSet[date] = true
		}

		currency := strings.TrimSpace(item.Currency)
		if currency == "" {
			currency = defaultCurrency
		}

		total := grouped[date]
		if converted, ok := convertAmountAt(date, item.Amount, currency, "MXN", defaultCurrency, converter, datedConverter); ok {
			total.mxn += converted
		}
		if converted, ok := convertAmountAt(date, item.Amount, currency, "USD", defaultCurrency, converter, datedConverter); ok {
			total.usd += converted
		}
		if converted, ok := convertAmountAt(date, item.Amount, currency, "XMR", defaultCurrency, converter, datedConverter); ok {
			total.xmr += converted
		}
		grouped[date] = total
	}

	sort.Strings(labels)
	result := struct {
		Labels []string  `json:"labels"`
		MXN    []float64 `json:"mxn"`
		USD    []float64 `json:"usd"`
		XMR    []float64 `json:"xmr"`
	}{Labels: labels}
	for _, label := range labels {
		total := grouped[label]
		result.MXN = append(result.MXN, total.mxn)
		result.USD = append(result.USD, total.usd)
		result.XMR = append(result.XMR, total.xmr)
	}
	return result
}

type datedConverter struct {
	dates      []string
	converters map[string]*finance.Converter
	fallback   *finance.Converter
}

func buildDatedConverter(
	mxnUSDSeries []*exchangerate.Sample,
	xmrUSDSeries []*exchangerate.Sample,
	fallback *finance.Converter,
) *datedConverter {
	type rateSet struct {
		mxnUSD *exchangerate.Sample
		xmrUSD *exchangerate.Sample
	}

	grouped := make(map[string]*rateSet)
	for _, sample := range reverseRateSamples(mxnUSDSeries) {
		date := sample.ObservedAt.Format(time.DateOnly)
		if grouped[date] == nil {
			grouped[date] = &rateSet{}
		}
		grouped[date].mxnUSD = sample
	}

	for _, sample := range reverseRateSamples(xmrUSDSeries) {
		date := sample.ObservedAt.Format(time.DateOnly)
		if grouped[date] == nil {
			grouped[date] = &rateSet{}
		}
		grouped[date].xmrUSD = sample
	}

	if len(grouped) == 0 {
		return &datedConverter{fallback: fallback}
	}

	dates := make([]string, 0, len(grouped))
	for date := range grouped {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	converters := make(map[string]*finance.Converter, len(dates))
	var current rateSet
	for _, date := range dates {
		rates := grouped[date]
		if rates.mxnUSD != nil {
			current.mxnUSD = rates.mxnUSD
		}
		if rates.xmrUSD != nil {
			current.xmrUSD = rates.xmrUSD
		}
		converters[date] = finance.NewConverter(compactRateSamples(current.mxnUSD, current.xmrUSD))
	}

	return &datedConverter{
		dates:      dates,
		converters: converters,
		fallback:   fallback,
	}
}

func convertAmountAt(
	observedDate string,
	amount float64,
	fromCurrency string,
	toCurrency string,
	defaultCurrency string,
	fallback *finance.Converter,
	timeline *datedConverter,
) (float64, bool) {
	if timeline == nil || len(timeline.dates) == 0 {
		return convertAmount(amount, fromCurrency, toCurrency, defaultCurrency, fallback)
	}

	index := sort.SearchStrings(timeline.dates, observedDate)
	switch {
	case index < len(timeline.dates) && timeline.dates[index] == observedDate:
		return convertAmount(amount, fromCurrency, toCurrency, defaultCurrency, timeline.converters[timeline.dates[index]])
	case index == 0:
		return convertAmount(amount, fromCurrency, toCurrency, defaultCurrency, timeline.converters[timeline.dates[0]])
	default:
		return convertAmount(amount, fromCurrency, toCurrency, defaultCurrency, timeline.converters[timeline.dates[index-1]])
	}
}

func buildFXHistoryChart(
	mxnUSDSeries []*exchangerate.Sample,
	xmrUSDSeries []*exchangerate.Sample,
) struct {
	Labels   []string   `json:"labels"`
	MXNToUSD []*float64 `json:"mxn_to_usd"`
	MXNToXMR []*float64 `json:"mxn_to_xmr"`
} {
	type ratePoint struct {
		mxnUSD *float64
		xmrUSD *float64
	}

	grouped := make(map[string]ratePoint)
	labelSet := make(map[string]bool)
	var labels []string

	for _, sample := range reverseRateSamples(mxnUSDSeries) {
		date := sample.ObservedAt.Format(time.DateOnly)
		point := grouped[date]
		value := sample.Rate
		point.mxnUSD = &value
		grouped[date] = point
		if !labelSet[date] {
			labels = append(labels, date)
			labelSet[date] = true
		}
	}

	for _, sample := range reverseRateSamples(xmrUSDSeries) {
		date := sample.ObservedAt.Format(time.DateOnly)
		point := grouped[date]
		value := sample.Rate
		point.xmrUSD = &value
		grouped[date] = point
		if !labelSet[date] {
			labels = append(labels, date)
			labelSet[date] = true
		}
	}

	sort.Strings(labels)
	result := struct {
		Labels   []string   `json:"labels"`
		MXNToUSD []*float64 `json:"mxn_to_usd"`
		MXNToXMR []*float64 `json:"mxn_to_xmr"`
	}{Labels: labels}
	for _, label := range labels {
		point := grouped[label]
		result.MXNToUSD = append(result.MXNToUSD, point.mxnUSD)
		if point.mxnUSD != nil && point.xmrUSD != nil && *point.xmrUSD > 0 {
			value := *point.mxnUSD / *point.xmrUSD
			result.MXNToXMR = append(result.MXNToXMR, &value)
		} else {
			result.MXNToXMR = append(result.MXNToXMR, nil)
		}
	}

	return result
}

func buildRuntimeHistoryChart(samples []*runtimesample.Sample) struct {
	Labels            []string  `json:"labels"`
	HeapAllocMB       []float64 `json:"heap_alloc_mb"`
	Goroutines        []int     `json:"goroutines"`
	DBOpenConnections []int     `json:"db_open_connections"`
} {
	result := struct {
		Labels            []string  `json:"labels"`
		HeapAllocMB       []float64 `json:"heap_alloc_mb"`
		Goroutines        []int     `json:"goroutines"`
		DBOpenConnections []int     `json:"db_open_connections"`
	}{}

	for i := len(samples) - 1; i >= 0; i-- {
		sample := samples[i]
		if sample == nil {
			continue
		}
		result.Labels = append(result.Labels, sample.ObservedAt.Format("15:04"))
		result.HeapAllocMB = append(result.HeapAllocMB, bytesToMB(sample.HeapAllocBytes))
		result.Goroutines = append(result.Goroutines, sample.Goroutines)
		result.DBOpenConnections = append(result.DBOpenConnections, sample.DBOpenConnections)
	}

	return result
}

func buildUptimeStatusChart(summary *uptimemonitor.Summary) struct {
	Labels []string `json:"labels"`
	Counts []int    `json:"counts"`
} {
	result := struct {
		Labels []string `json:"labels"`
		Counts []int    `json:"counts"`
	}{
		Labels: []string{"Up", "Down", "Unknown", "Disabled"},
	}
	if summary == nil {
		result.Counts = []int{0, 0, 0, 0}
		return result
	}
	result.Counts = []int{summary.Up, summary.Down, summary.Unknown, summary.Disabled}
	return result
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

func reverseRateSamples(items []*exchangerate.Sample) []*exchangerate.Sample {
	result := make([]*exchangerate.Sample, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		if items[i] != nil {
			result = append(result, items[i])
		}
	}
	return result
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

func bytesToMB(value uint64) float64 {
	return float64(value) / 1024 / 1024
}

func formatMB(value uint64) string {
	return fmt.Sprintf("%.2f MB", bytesToMB(value))
}

func countLabel(count int, noun string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", count, noun)
}
