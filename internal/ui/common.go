package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"codeberg.org/urutau-ltd/aile/v2"
)

// BaseData is the shared top-level payload embedded by feature pages.
// It exists so every view can receive a consistent title and footer metadata
// without each handler rebuilding the same structure.
type BaseData struct {
	Title      string
	FooterData FooterData
}

// FooterData centralizes runtime diagnostics shown in layout/footer templates.
// Keeping this in one struct makes footer output predictable across all pages.
type FooterData struct {
	AppVersion  string
	RenderTime  string
	AileVersion string
	GoVersion   string
}

var appVersion = "dev"

// SetAppVersion stores the running app version so shared templates can expose
// build metadata without every handler wiring it manually.
func SetAppVersion(version string) {
	version = strings.TrimSpace(version)
	if version == "" {
		return
	}

	appVersion = version
}

// NewBaseData builds the common payload used by UI handlers.
// It is called right before template rendering so footer diagnostics reflect
// the actual request cost seen by the user.
func NewBaseData(title string, start time.Time) BaseData {
	return BaseData{
		Title: title,
		FooterData: FooterData{
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
