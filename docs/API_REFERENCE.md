# API Reference (Where + Why)

This reference documents the current project APIs from two angles:

1. External APIs: HTTP surface consumed by browser/HTMX.
2. Internal APIs: Go packages consumed by other project modules.

It intentionally prioritizes **where** each API lives and **why** it exists.

## 1) External HTTP API

### 1.1 UI Routes (full page + HTMX fragments)

| Route                  | Method   | Where (implementation)                                           | Why this route exists                                                               |
| ---------------------- | -------- | ---------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| `/dashboard`           | `GET`    | `internal/ui/features/dashboard/handler.go` (`(*Handler).Index`) | Landing summary page; keeps a low-friction health check for UI + DB reads.          |
| `/providers`           | `GET`    | `internal/ui/features/providers/handler.go` (`Index`)            | Providers listing page and HTMX list refresh endpoint.                              |
| `/providers/new`       | `GET`    | `providers/handler.go` (`New`)                                   | Opens provider creation panel without full reload (HTMX target `#provider-editor`). |
| `/providers`           | `POST`   | `providers/handler.go` (`Create`)                                | Creates provider and returns editor + list refresh contract.                        |
| `/providers/{id}`      | `GET`    | `providers/handler.go` (`Show`)                                  | Opens provider detail panel by ID.                                                  |
| `/providers/{id}/edit` | `GET`    | `providers/handler.go` (`Edit`)                                  | Opens provider edit form in panel.                                                  |
| `/providers/{id}/edit` | `POST`   | `providers/handler.go` (`Update`)                                | Persists provider updates and refreshes list/panel state.                           |
| `/providers/{id}`      | `DELETE` | `providers/handler.go` (`Delete`)                                | Deletes provider from panel actions.                                                |
| `/locations`           | `GET`    | `internal/ui/features/locations/handler.go` (`Index`)            | Locations listing page and HTMX list refresh endpoint.                              |
| `/locations/new`       | `GET`    | `locations/handler.go` (`New`)                                   | Opens location creation panel without full reload.                                  |
| `/locations`           | `POST`   | `locations/handler.go` (`Create`)                                | Creates location and returns editor + list refresh contract.                        |
| `/locations/{id}`      | `GET`    | `locations/handler.go` (`Show`)                                  | Opens location detail panel by ID.                                                  |
| `/locations/{id}/edit` | `GET`    | `locations/handler.go` (`Edit`)                                  | Opens location edit form in panel.                                                  |
| `/locations/{id}/edit` | `POST`   | `locations/handler.go` (`Update`)                                | Persists location updates and refreshes list/panel state.                           |
| `/locations/{id}`      | `DELETE` | `locations/handler.go` (`Delete`)                                | Deletes location from panel actions.                                                |
| `/static/{path...}`    | `GET`    | `main.go` route mount                                            | Serves CSS/JS/assets required by Missing.css, htmx, hyperscript and images.         |

### 1.2 HTMX Fragment Contracts (internal but externally consumed by browser)

These are not Go-exported symbols, but they are API contracts between server and
UI attributes.

| Contract                             | Where                                                                          | Why                                                                                            |
| ------------------------------------ | ------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| `HX-Target: providers-body`          | `providers/handler.go` (`isListRequest`)                                       | Prevents returning table rows for unrelated boosted navigations.                               |
| `HX-Target: provider-editor`         | `providers/handler.go` (`isEditorRequest`)                                     | Isolates editor panel rendering from full page rendering.                                      |
| `HX-Target: locations-body`          | `locations/handler.go` (`isListRequest`)                                       | Same isolation principle for locations list refresh.                                           |
| `HX-Target: location-editor`         | `locations/handler.go` (`isEditorRequest`)                                     | Same isolation principle for locations editor.                                                 |
| `hx-swap-oob="outerHTML"` on `tbody` | `providers/views/provider-editor.html`, `locations/views/location-editor.html` | Allows create/update/delete response to refresh list region in one response without manual JS. |

## 2) Internal APIs (Go)

## 2.1 `main` package

### Exported API

- None (entrypoint package).

### Non-exported API

| Symbol                     | Where     | Why                                                                                           |
| -------------------------- | --------- | --------------------------------------------------------------------------------------------- |
| `newLogger() *slog.Logger` | `main.go` | Centralizes app log shape for consistent readable operations logs and easier grepping.        |
| `main()`                   | `main.go` | Single composition root: DB setup, migrations, middleware, route wiring, app lifecycle hooks. |

## 2.2 `internal/database`

### Exported API

| Symbol                                                 | Where                           | Why                                                                           |
| ------------------------------------------------------ | ------------------------------- | ----------------------------------------------------------------------------- |
| `Client(dbPath string) (*sql.DB, error)`               | `internal/database/database.go` | Provides singleton DB handle for whole process; avoids duplicate pools.       |
| `SetPragmas(db *sql.DB) error`                         | `internal/database/database.go` | Enforces SQLite runtime characteristics expected by the app (FKs, WAL, sync). |
| `RunMigrations(db *sql.DB, logger *slog.Logger) error` | `internal/database/database.go` | Applies embedded SQL migrations once, with migration tracking table.          |
| `SeedProviders(db *sql.DB) error`                      | `internal/database/seed.go`     | Provides baseline provider dataset for first-run UX and development.          |

### Non-exported API

| Symbol                      | Where         | Why                                                                                  |
| --------------------------- | ------------- | ------------------------------------------------------------------------------------ |
| `migrationsFS`              | `database.go` | Embeds migration SQLs into binary for consistent deploy artifact.                    |
| `client`, `once`, `initErr` | `database.go` | Implements `Client` singleton lifecycle.                                             |
| `seedProvider`              | `seed.go`     | Internal shape for static seed payload; intentionally hidden from package consumers. |

## 2.3 `internal/models/provider`

### Exported API

| Symbol                                                  | Where                                  | Why                                                                      |
| ------------------------------------------------------- | -------------------------------------- | ------------------------------------------------------------------------ |
| `type Provider`                                         | `internal/models/provider/provider.go` | Canonical provider entity used by repositories + templates.              |
| `type ProviderRepository`                               | `provider.go`                          | DB boundary for provider aggregate (single responsibility: persistence). |
| `NewProviderRepository(db *sql.DB) *ProviderRepository` | `provider.go`                          | Controlled repository construction with injected DB dependency.          |
| `(*ProviderRepository).Create(...)`                     | `provider.go`                          | Provider insertion with generated UUID v7.                               |
| `(*ProviderRepository).GetByID(...)`                    | `provider.go`                          | Single-provider lookup with ID validation.                               |
| `(*ProviderRepository).GetAll(...)`                     | `provider.go`                          | List/search/limit read path for screens and HTMX updates.                |
| `(*ProviderRepository).Update(...)`                     | `provider.go`                          | Provider mutation boundary and update timestamp policy.                  |
| `(*ProviderRepository).Delete(...)`                     | `provider.go`                          | Provider delete boundary with UUID validation.                           |

### Non-exported API

- None in this package (all methods/types are exported today).

## 2.4 `internal/models/location`

### Exported API

| Symbol                                                  | Where                                  | Why                                                     |
| ------------------------------------------------------- | -------------------------------------- | ------------------------------------------------------- |
| `type Location`                                         | `internal/models/location/location.go` | Canonical location entity for handlers/templates.       |
| `type LocationRepository`                               | `location.go`                          | DB boundary for locations persistence and lookup.       |
| `NewLocationRepository(db *sql.DB) *LocationRepository` | `location.go`                          | Constructor with explicit DB dependency.                |
| `(*LocationRepository).Create(...)`                     | `location.go`                          | Inserts location with generated UUID v7.                |
| `(*LocationRepository).GetByID(...)`                    | `location.go`                          | Returns one location by ID with UUID guardrails.        |
| `(*LocationRepository).GetAll(...)`                     | `location.go`                          | Supports list/search by name/city/country + limit.      |
| `(*LocationRepository).Update(...)`                     | `location.go`                          | Persists editable location fields and update timestamp. |
| `(*LocationRepository).Delete(...)`                     | `location.go`                          | Removes location by ID with validation.                 |

### Non-exported API

- None in this package (all methods/types are exported today).

## 2.5 `internal/ui/common`

### Exported API

| Symbol                                                | Where                   | Why                                                                   |
| ----------------------------------------------------- | ----------------------- | --------------------------------------------------------------------- |
| `type BaseData`                                       | `internal/ui/common.go` | Shared payload shape used by multiple feature handlers.               |
| `type FooterData`                                     | `common.go`             | Shared footer metadata contract for templates.                        |
| `NewBaseData(title string, start time.Time) BaseData` | `common.go`             | Standardizes page title + footer diagnostics (render time, versions). |

### Non-exported API

- None in this package.

## 2.6 `internal/ui/features/dashboard`

### Exported API

| Symbol                     | Where                  | Why                                                             |
| -------------------------- | ---------------------- | --------------------------------------------------------------- |
| `type Handler`             | `dashboard/handler.go` | Route-level orchestration for dashboard UI.                     |
| `NewHandler(...) *Handler` | `dashboard/handler.go` | Builds dashboard handler with template+repository dependencies. |
| `(*Handler).Index(...)`    | `dashboard/handler.go` | Dashboard read endpoint.                                        |

### Non-exported API

- None in this package.

## 2.7 `internal/ui/features/providers`

### Exported API

| Symbol                                                | Where                  | Why                                                                  |
| ----------------------------------------------------- | ---------------------- | -------------------------------------------------------------------- |
| `type Handler`                                        | `providers/handler.go` | Feature-level orchestrator for providers CRUD UI and HTMX contracts. |
| `NewHandler(...) *Handler`                            | `providers/handler.go` | Dependency-injected constructor for providers feature.               |
| `(*Handler).Index/New/Show/Edit/Create/Update/Delete` | `providers/handler.go` | Public route methods mounted from `main.go`.                         |

### Non-exported API

| Symbol                         | Where                  | Why                                                                  |
| ------------------------------ | ---------------------- | -------------------------------------------------------------------- |
| `type pageData`                | `providers/handler.go` | Internal view-model to keep template payload stable and explicit.    |
| `(*Handler).loadPageData`      | `providers/handler.go` | Single source for list/search state loading across route methods.    |
| `(*Handler).renderTemplate`    | `providers/handler.go` | Centralized template rendering error logging.                        |
| `(*Handler).isListRequest`     | `providers/handler.go` | Guards HTMX fragment route behavior to avoid boost/nav collisions.   |
| `(*Handler).isEditorRequest`   | `providers/handler.go` | Same guard for editor panel target.                                  |
| `parseListState`, `parseLimit` | `providers/handler.go` | Shared query/form parsing policy for list views.                     |
| `bannerHTML`                   | `providers/handler.go` | Controlled banner markup output as `template.HTML`.                  |
| `writeHTMLHeader`              | `providers/handler.go` | Guarantees content-type consistency for full and fragment responses. |

## 2.8 `internal/ui/features/locations`

### Exported API

| Symbol                                                | Where                  | Why                                                                  |
| ----------------------------------------------------- | ---------------------- | -------------------------------------------------------------------- |
| `type Handler`                                        | `locations/handler.go` | Feature-level orchestrator for locations CRUD UI and HTMX contracts. |
| `NewHandler(...) *Handler`                            | `locations/handler.go` | Dependency-injected constructor for locations feature.               |
| `(*Handler).Index/New/Show/Edit/Create/Update/Delete` | `locations/handler.go` | Public route methods mounted from `main.go`.                         |

### Non-exported API

| Symbol                         | Where                  | Why                                                                  |
| ------------------------------ | ---------------------- | -------------------------------------------------------------------- |
| `type pageData`                | `locations/handler.go` | Internal view-model for templates and HTMX responses.                |
| `(*Handler).loadPageData`      | `locations/handler.go` | Central place to keep query/list behavior consistent across actions. |
| `(*Handler).renderTemplate`    | `locations/handler.go` | Uniform template error logging and render path.                      |
| `(*Handler).isListRequest`     | `locations/handler.go` | HTMX target guard for list fragment isolation.                       |
| `(*Handler).isEditorRequest`   | `locations/handler.go` | HTMX target guard for editor fragment isolation.                     |
| `parseListState`, `parseLimit` | `locations/handler.go` | Shared list query normalization and guardrails.                      |
| `bannerHTML`                   | `locations/handler.go` | Controlled banner markup output for location module.                 |
| `writeHTMLHeader`              | `locations/handler.go` | Ensures HTML content-type in fragment/full responses.                |

## 3) External Library APIs consumed by this project

| Library              | Where used                                        | Why                                                                |
| -------------------- | ------------------------------------------------- | ------------------------------------------------------------------ |
| `aile`               | `main.go`                                         | HTTP app container, route registration, lifecycle hooks.           |
| `aile/x/combine`     | `main.go`                                         | Middleware composition without manual nesting.                     |
| `aile/x/logger`      | `main.go`                                         | Request logging middleware integrated with app logger.             |
| `aile/x/request_id`  | `main.go`                                         | Per-request traceability via request ID propagation.               |
| `modernc.org/sqlite` | `internal/database/database.go`                   | SQLite driver for embedded DB use-case.                            |
| `htmx`               | `internal/ui/layout/base.html`, feature templates | Incremental page updates via server-rendered HTML fragments.       |
| `hyperscript`        | feature templates + footer component              | Declarative client-side behavior without custom JS bundle.         |
| `missing.css`        | layout + feature templates + gavia.css overrides  | Design system baseline and utility vocabulary (layout/ARIA/forms). |
