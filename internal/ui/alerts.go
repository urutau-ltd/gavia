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
	return template.HTML(`<p class="` + html.EscapeString(baseClass) + ` ` + className + `">` + escaped + `</p>`)
}
