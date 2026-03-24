# CRUD Extension Playbook (Where + Why)

This playbook explains **where to extend** the project and **why each extension
point exists** so a newcomer can add the next CRUD module from
`001_create_tables.sql`.

It is intentionally architecture-oriented (not a step-by-step code tutorial).

## 1) Architectural map

## 1.1 Persistence boundary

- Where: `internal/models/<entity>/<entity>.go`
- Why: repository packages isolate SQL decisions from HTTP/UI concerns.
- Contract expected by UI handlers:
  - `New<Entity>Repository(db *sql.DB)`
  - `Create`, `GetByID`, `GetAll(search, limit)`, `Update`, `Delete`

If your next table supports list filtering, include search in `GetAll` now, not
later. Handlers already assume list refresh uses query state.

## 1.2 HTTP/UI boundary

- Where: `internal/ui/features/<entities>/handler.go`
- Why: each feature package owns route behavior, template rendering, and HTMX
  fragment contracts.
- Contract expected by route mount:
  - `NewHandler(logger, uiFS, db)`
  - `Index`, `New`, `Show`, `Edit`, `Create`, `Update`, `Delete`

Keep **fragment guards** (`isListRequest`, `isEditorRequest`) per feature. This
avoids HTMX collisions when `hx-boost` is enabled globally in layout.

## 1.3 Templates boundary

- Where: `internal/ui/features/<entities>/views/`
- Why: separates page shell, table rows, and side editor panel so HTMX swaps can
  be targeted precisely.
- Required templates by naming convention:
  - `index.html` => full page shell
  - `<entity>-list.html` => table row fragment
  - `<entity>-editor.html` => panel content + OOB list refresh fragment

The naming and target IDs are API contracts between HTML attributes and handler
guards.

## 1.4 Route composition boundary

- Where: `main.go`
- Why: single composition root for every runtime dependency and public URL
  surface.

New feature route registration belongs here, not inside other packages, to keep
startup visibility and operational debugging simple.

## 1.5 Visual system boundary

- Where: `internal/ui/layout/base.html` + `static/css/gavia.css`
- Why: base layout loads Missing.css + htmx + hyperscript globally; `gavia.css`
  provides feature-specific refinements.

Do not copy external CSS frameworks into feature templates. Use Missing.css
vocabulary first, then minimal local overrides.

## 2) HTMX and HyperScript contracts to preserve

## 2.1 Why this project uses fragments

List + editor are independent fragments so user actions
(search/create/edit/delete) can update only the affected region and keep page
context.

## 2.2 Required fragment IDs

For each feature, define unique IDs to avoid cross-module collisions:

- list target: `#<entities>-body`
- editor target: `#<entity>-editor`

Handler guards must match those IDs:

- list guard checks `HX-Target == <entities>-body`
- editor guard checks `HX-Target == <entity>-editor`

## 2.3 OOB refresh strategy

For create/update/delete responses, include:

- editor HTML (normal target swap)
- list `<tbody ... hx-swap-oob="outerHTML">` (out-of-band swap)

Why: one server response keeps UI state coherent and avoids extra request
choreography.

## 2.4 HyperScript usage policy

Where used now:

- row visual feedback during deletes (`add .is-removing`)
- panel loading state (`add/remove .is-loading`)

Why: small declarative behaviors without introducing custom JS lifecycle code.

Keep HyperScript snippets local to the element they control.

## 3) Missing.css integration policy

## 3.1 Preferred primitives

Use Missing.css primitives in templates first:

- `tool-bar`
- `flex-switch`
- `width:100%`
- semantic ARIA attributes (`role="toolbar"`)

Why: the design system already solves responsive behavior and accessibility
defaults.

## 3.2 Local CSS scope

Use namespaced selectors per feature (`.providers-*`, `.locations-*`).

Why: avoids visual regressions when multiple CRUD modules coexist.

## 4) Error and validation policy

Where:

- handler methods (`Create`, `Update`, `Delete`, `Show`, `Edit`)

Why:

- keep user-facing status in HTML banners
- keep operational detail in logs

Rules to preserve:

- required fields validated before repository call
- unique-constraint collisions mapped to `409 Conflict`
- malformed/unknown IDs return controlled `400/404` panel responses

## 5) Logging policy

Where:

- app-level logger shape: `main.go:newLogger`
- request-level logs: `aile/x/logger` middleware

Why:

- app logs need consistent, grep-friendly structure (`t`, `lvl`, `msg`, `src`)
- request middleware keeps transport trace visibility per endpoint

When adding new feature handlers, always log:

- storage failures (with `err`)
- ID/context for mutations (`id`, filters when relevant)

## 6) Pattern for the next table in migrations

For each new table in `internal/database/migrations/001_create_tables.sql`:

1. Create repository package under `internal/models/<entity>/`.
2. Create feature package under `internal/ui/features/<entities>/`.
3. Add `views/index.html`, `<entity>-list.html`, `<entity>-editor.html`.
4. Register routes in `main.go`.
5. Add scoped CSS block in `static/css/gavia.css`.
6. Add API entries in `docs/API_REFERENCE.md`.

Why this order: DB boundary first, UI boundary second, route exposure last. It
reduces half-wired states and keeps integration testable by layer.

## 7) Current feature references

Use these as canonical examples:

- Providers: `internal/models/provider/` + `internal/ui/features/providers/`
- Locations: `internal/models/location/` + `internal/ui/features/locations/`

Both already implement the full pattern expected by this playbook.

## 8) Design references used in current CRUD modules

- Missing.css Layout docs: `https://missing.style/docs/layout/`
- Missing.css Flexbox docs: `https://missing.style/docs/flex/`
- Missing.css ARIA docs (toolbar role): `https://missing.style/docs/aria/`
- Missing.css Utilities docs: `https://missing.style/docs/utils/`
- HyperScript docs (event handlers, add/remove/toggle, transitions, queueing):
  `https://hyperscript.org/docs/`
