package ui

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"codeberg.org/urutau-ltd/aile/v2"
	"codeberg.org/urutau-ltd/gavia/internal/auth"
	"codeberg.org/urutau-ltd/gavia/internal/csrf"
)

// BaseData is the shared top-level payload embedded by feature pages.
// It exists so every view can receive a consistent title and footer metadata
// without each handler rebuilding the same structure.
type BaseData struct {
	Title      string
	Section    string
	CSRFToken  string
	Nav        NavData
	FooterData FooterData
}

type NavData struct {
	HasAccount      bool
	IsAuthenticated bool
	SetupRequired   bool
	Username        string
	AvatarPath      string
}

// FooterData centralizes runtime diagnostics shown in layout/footer templates.
// Keeping this in one struct makes footer output predictable across all pages.
type FooterData struct {
	Visible     bool
	AppVersion  string
	RenderTime  string
	AileVersion string
	GoVersion   string
}

var appVersion = "dev"
var showVersionFooter atomic.Bool

// SetAppVersion stores the running app version so shared templates can expose
// build metadata without every handler wiring it manually.
func SetAppVersion(version string) {
	version = strings.TrimSpace(version)
	if version == "" {
		return
	}

	appVersion = version
}

func SetShowVersionFooter(visible bool) {
	showVersionFooter.Store(visible)
}

// NewBaseData builds the common payload used by UI handlers.
// It is called right before template rendering so footer diagnostics reflect
// the actual request cost seen by the user.
func NewBaseData(r *http.Request, title string, start time.Time) BaseData {
	viewer := auth.ViewerFromContext(r.Context())

	return BaseData{
		Title:   title,
		Section: navSection(r.URL.Path),
		CSRFToken: csrf.TokenFromContext(
			r.Context(),
		),
		Nav: NavData{
			HasAccount:      viewer.HasAccount,
			IsAuthenticated: viewer.IsAuthenticated,
			SetupRequired:   viewer.SetupRequired,
			Username:        viewer.Username,
			AvatarPath:      viewer.AvatarPath,
		},
		FooterData: FooterData{
			Visible:     showVersionFooter.Load(),
			AppVersion:  appVersion,
			RenderTime:  formatRenderTime(time.Since(start)),
			AileVersion: aile.Version,
			GoVersion: strings.Trim(
				runtime.Version(),
				"go",
			),
		},
	}
}

func init() {
	showVersionFooter.Store(true)
}

// formatRenderTime chooses a human scale (ns/µs/ms/s) to avoid displaying
// misleading values like "0.00s" for fast requests.
func formatRenderTime(elapsed time.Duration) string {
	switch {
	case elapsed < time.Microsecond:
		return fmt.Sprintf("%dns", elapsed.Nanoseconds())
	case elapsed < time.Millisecond:
		return fmt.Sprintf("%.2fµs", float64(elapsed)/float64(time.Microsecond))
	case elapsed < time.Second:
		return fmt.Sprintf("%.2fms", float64(elapsed)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%.2fs", elapsed.Seconds())
	}
}

func navSection(path string) string {
	switch {
	case path == "/dashboard":
		return "dashboard"
	case strings.HasPrefix(path, "/providers"):
		return "providers"
	case strings.HasPrefix(path, "/locations"):
		return "locations"
	case strings.HasPrefix(path, "/os"):
		return "os"
	case strings.HasPrefix(path, "/ips"):
		return "ips"
	case strings.HasPrefix(path, "/dns"):
		return "dns"
	case strings.HasPrefix(path, "/labels"):
		return "labels"
	case strings.HasPrefix(path, "/domains"):
		return "domains"
	case strings.HasPrefix(path, "/hostings"):
		return "hostings"
	case strings.HasPrefix(path, "/servers"):
		return "servers"
	case strings.HasPrefix(path, "/subscriptions"):
		return "subscriptions"
	case strings.HasPrefix(path, "/ledger"):
		return "ledger"
	case strings.HasPrefix(path, "/account-settings"):
		return "account-settings"
	case strings.HasPrefix(path, "/app-settings"):
		return "app-settings"
	case strings.HasPrefix(path, "/uptime"):
		return "uptime"
	case strings.HasPrefix(path, "/login"):
		return "login"
	case strings.HasPrefix(path, "/javascript-license-info"):
		return "licenses"
	case strings.HasPrefix(path, "/licenses"):
		return "licenses"
	default:
		return ""
	}
}
