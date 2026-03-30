package ui

import (
	"html"
	"html/template"
)

func BoxAlertHTML(baseClass, kind, msg string) template.HTML {
	className := "info"
	switch kind {
	case "ok", "bad", "warn", "info":
		className = kind
	}

	role := "status"
	if className == "bad" {
		role = "alert"
	}

	return template.HTML(
		`<div class="` + html.EscapeString(baseClass) + ` box ` + className + `" role="` + role + `">` +
			html.EscapeString(msg) +
			`</div>`,
	)
}

func BannerHTML(baseClass, kind, msg string) template.HTML {
	return BoxAlertHTML(baseClass, kind, msg)
}
