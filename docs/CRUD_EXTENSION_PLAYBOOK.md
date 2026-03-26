# CRUD Extension Playbook

This document explains how to add the next CRUD module without fighting the
current architecture.

## Expected shape

The project now uses three route styles:

- collection resources for repeated CRUD modules
- singleton resources for settings-like pages
- manual pages for special flows

Choose the shape first. Do not start by copying handlers blindly.

## Collection modules

Use a collection resource when the domain has multiple rows and the UI follows a
standard list/editor flow.

Current examples:

- providers
- locations
- operating systems

### Repository contract

Create a repository under `internal/models/<entity>/` with the usual explicit
methods:

- `Create`
- `GetByID`
- `GetAll`
- `Update`
- `Delete`
- `Count` if dashboard visibility is useful

### Handler contract

Create a feature package under `internal/ui/features/<entities>/` with explicit
methods:

- `Index`
- `New`
- `Create`
- `Show`
- `Edit`
- `Update`
- `Delete`

Keep handlers explicit. Do not hide CRUD operations behind a large generic UI
framework.

### Route mounting

Mount it with:

```go
resource.MountCollection(app, "/route", handler)
```

That wiring currently lives in [`routing.go`](../routing.go).

### HTMX helpers

Use shared helpers from [`internal/ui/request.go`](../internal/ui/request.go):

- `ui.ParseListState`
- `ui.IsHTMXListRequest`
- `ui.IsHTMXEditorRequest`
- `ui.WriteHTMLHeader`

Do not reintroduce manual `HX-*` header parsing in new modules.

## Singleton modules

Use a singleton resource when the table should behave like configuration, not a
repeated entity.

Current examples:

- account settings
- app settings

### Route shape

Singletons use:

- `GET /resource`
- `GET /resource/edit`
- `POST /resource/edit`

Mount them with:

```go
resource.MountSingleton(app, "/route", handler)
```

## Manual modules

Use a manual page when the flow does not fit collection or singleton semantics.

Current manual pages:

- dashboard
- login
- logout
- licenses
- uptime
- JSON API routes

Keep these as normal handlers and plain route registration.

## Templates

Template layout is still server-rendered Go HTML templates.

Collection modules usually use:

- `index.html`
- `<entity>-list.html`
- `<entity>-editor.html`

Singleton and manual pages can keep a single `index.html` if that is clearer.

## Styling

Start with Missing.css primitives and add small, local CSS in
[`static/css/gavia.css`](../static/css/gavia.css).

Do not add a new CSS framework. Do not generate CSS automatically.

## Generics policy

Generics are acceptable when they reduce repeated algorithmic code without
making the flow harder to read.

Good examples in this repository:

- [`seedMany[T]`](../internal/database/seed.go)
- [`AggregateByLabel[T]`](../internal/models/dashboard_summary/overview.go)

Bad candidates:

- generic HTTP handlers
- generic route builders for unrelated handler signatures
- generic template rendering layers

## Checklist for a new module

1. Add or update the SQL migration.
2. Create the repository package.
3. Create the UI feature package.
4. Add templates.
5. Mount the route in [`routing.go`](../routing.go).
6. Add CSS only if the existing patterns are not enough.
7. Add or update docs in [`docs/API_REFERENCE.md`](./API_REFERENCE.md).
8. Add tests for the route shape and critical behavior.
