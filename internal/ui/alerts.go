package ui

import (
	"html"
	"html/template"
)

func BannerHTML(baseClass, kind, msg string) template.HTML {
	className := "info"
	switch kind {
	case "ok", "bad", "warn", "info":
		className = kind
	}

	escaped := html.EscapeString(msg)
	role := "status"
	if className == "bad" {
		role = "alert"
	}

	return template.HTML(
		`<aside class="` + html.EscapeString(baseClass) + ` gavia-banner ` + className + `" role="` + role + `">` +
			`<div class="gavia-banner-copy">` + escaped + `</div>` +
			`<button type="button" class="plain gavia-banner-dismiss" aria-label="Dismiss message"` +
			` _="on click add .is-dismissing to closest .gavia-banner then wait 160ms then remove closest .gavia-banner">Dismiss</button>` +
			`</aside>`,
	)
}
