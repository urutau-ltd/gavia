;;; templates.eld --- Tempel snippets for Gavia -*- mode: lisp-data; lexical-binding: t; -*-
;;;
;;;
;;; Commentary:
;;; These snippets focus on repetitive CRUD work used in this repository.
;;; They are intentionally opinionated toward HTMX + html/template + Missing.css.
;;;
;;; Code:


;; -----------------------------------------------------------------------------
;; Go snippets
;; -----------------------------------------------------------------------------
go-mode

(gdoc
 "// " (p "Identifier") " explains where this code is used and why it exists."
 :ann "Go doc where/why"
 :doc "Docstring scaffold focused on where and why")

(gindex
 "// Index serves GET /" (p "resource path" "providers") "." n
 "// It returns either the full page or only the HTMX list fragment." n
 "func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {" n
 > "start := time.Now()" n
 > "data, err := h.loadPageData(r, start)" n
 > "if err != nil {" n
 > > "h.logger.Error(\"Failed to list records\", \"err\", err)" n
 > > "http.Error(w, \"Not found\", http.StatusNotFound)" n
 > > "return" n
 > "}" n n
 > "writeHTMLHeader(w)" n
 > "if h.isListRequest(r) {" n
 > > "h.renderTemplate(w, \"" (p "list template" "provider-list") "\", data)" n
 > > "return" n
 > "}" n n
 > "if h.isEditorRequest(r) {" n
 > > "h.renderTemplate(w, \"" (p "editor template" "provider-editor-panel") "\", data)" n
 > > "return" n
 > "}" n n
 > "h.renderTemplate(w, \"base\", data)" n
 "}"
 :ann "Handler Index"
 :doc "CRUD list endpoint with HTMX/full-page branching")

(ghandler
 "type Handler struct {" n
 > "logger       *slog.Logger" n
 > "tmpl         *template.Template" n
 > (p "repo field" "providerRepo") " *" (p "repo type" "provider.ProviderRepository") n
 "}" n n
 "func NewHandler(l *slog.Logger, uiFS fs.FS, db *sql.DB) *Handler {" n
 > "t := template.Must(template.ParseFS(uiFS," n
 > > "\"layout/base.html\"," n
 > > "\"features/" (p "feature" "providers") "/views/*.html\"," n
 > > "\"components/*.html\"," n
 > "))" n n
 > "return &Handler{" n
 > > "logger:       l," n
 > > "tmpl:         t," n
 > > (p "repo assignment" "providerRepo: provider.NewProviderRepository(db),") n
 > "}" n
 "}"
 :ann "Handler struct/new"
 :doc "Handler + constructor scaffold for a new feature")

(grepo
 "type " (p "Entity" "Provider") "Repository struct {" n
 > "db *sql.DB" n
 "}" n n
 "func New" (p "Entity" "Provider") "Repository(db *sql.DB) *" (p "Entity" "Provider") "Repository {" n
 > "return &" (p "Entity" "Provider") "Repository{db: db}" n
 "}" n n
 "func (r *" (p "Entity" "Provider") "Repository) GetAll(ctx context.Context, searchTerm string, limit int) ([]*" (p "Entity" "Provider") ", error) {" n
 > "query := `SELECT ... FROM " (p "table" "providers") " WHERE 1=1`" n
 > "var args []any" n n
 > "if searchTerm != \"\" {" n
 > > "query += \" AND name LIKE ?\"" n
 > > "args = append(args, \"%\"+searchTerm+\"%\")" n
 > "}" n n
 > "if limit > 0 {" n
 > > "query += \" LIMIT ?\"" n
 > > "args = append(args, limit)" n
 > "}" n n
 > "rows, err := r.db.QueryContext(ctx, query, args...)" n
 > "if err != nil {" n
 > > "return nil, err" n
 > "}" n
 > "defer rows.Close()" n n
 > "var items []*" (p "Entity" "Provider") n
 > "for rows.Next() {" n
 > > "item := &" (p "Entity" "Provider") "{}" n
 > > "if err := rows.Scan(/* fields */); err != nil {" n
 > > > "return nil, err" n
 > > "}" n
 > > "items = append(items, item)" n
 > "}" n
 > "return items, rows.Err()" n
 "}"
 :ann "Repository scaffold"
 :doc "Repository skeleton with GetAll pattern")


;; -----------------------------------------------------------------------------
;; HTML/HTMX snippets
;; -----------------------------------------------------------------------------
html-mode nhtml-mode web-mode

(htmxindex
 "{{define \"content\"}}" n
 "<section class=\"" (p "view class" "providers-view") "\">" n
 > "<header class=\"" (p "head class" "providers-head") "\">" n
 > > "<h2>{{.Title}}</h2>" n
 > "</header>" n n
 > "<section class=\"" (p "grid class" "providers-grid") "\">" n
 > > "<article class=\"box width:100%\">" n
 > > > "<table class=\"width:100%\">" n
 > > > > "<tbody id=\"" (p "tbody id" "providers-body") "\">" n
 > > > > > "{{template \"" (p "list template" "provider-list") "\" .}}" n
 > > > > "</tbody>" n
 > > > "</table>" n
 > > "</article>" n n
 > > "{{template \"" (p "editor template" "provider-editor-panel") "\" .}}" n
 > "</section>" n
 "</section>" n
 "{{end}}"
 :ann "HTMX index view"
 :doc "Main CRUD page with list + editor partials")

(htmxlist
 "{{define \"" (p "list template name" "provider-list") "\"}}" n
 "{{- range ." (p "collection field" "Providers") "}}" n
 "<tr id=\"" (p "row id prefix" "provider-row") "-{{." (p "id field" "Id") "}}\">" n
 > "<td>{{." (p "main label" "Name") "}}</td>" n
 > "<td class=\"" (p "actions class" "provider-actions") "\">" n
 > > "<a href=\"/" (p "resource" "providers") "/{{." (p "id field" "Id") "}}\"" n
 > > "   hx-get=\"/" (p "resource" "providers") "/{{." (p "id field" "Id") "}}\"" n
 > > "   hx-target=\"#" (p "editor id" "provider-editor") "\"" n
 > > "   hx-swap=\"outerHTML\">Ver</a>" n
 > "</td>" n
 "</tr>" n
 "{{- else}}" n
 "<tr><td colspan=\"3\"><em>No hay registros.</em></td></tr>" n
 "{{- end}}" n
 "{{end}}"
 :ann "HTMX list partial"
 :doc "Table-body partial with row actions")

(htmxeditor
 "{{define \"" (p "panel template" "provider-editor-panel") "\"}}" n
 "<section id=\"" (p "editor id" "provider-editor") "\" class=\"box " (p "editor class" "provider-editor") "\">" n
 > "{{template \"" (p "content template" "provider-editor-content") "\" .}}" n
 "</section>" n
 "{{end}}"
 :ann "HTMX editor panel"
 :doc "Editor panel partial scaffold")


;; -----------------------------------------------------------------------------
;; CSS snippet
;; -----------------------------------------------------------------------------
css-mode

(crudcss
 "." (p "view class" "providers-view") " {" n
 > "display: grid;" n
 > "gap: var(--gap);" n
 "}" n n
 "." (p "grid class" "providers-grid") " {" n
 > "display: grid;" n
 > "gap: var(--gap);" n
 > "grid-template-columns: minmax(0, 1fr);" n
 "}" n n
 "@media (min-width: " (p "breakpoint" "1080px") ") {" n
 > "." (p "grid class" "providers-grid") " {" n
 > > "grid-template-columns: minmax(0, 2.3fr) minmax(22rem, 1fr);" n
 > "}" n
 "}"
 :ann "CRUD view css"
 :doc "Base responsive grid CSS for list/editor layout")
