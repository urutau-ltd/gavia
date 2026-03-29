# API Reference

This document describes the current public HTTP surface and the most important
internal interfaces used to compose the app.

## HTTP routes

### Public page shell

| Route                      | Method        | Purpose                                           |
| -------------------------- | ------------- | ------------------------------------------------- |
| `/`                        | `GET`         | Redirect to `/dashboard`                          |
| `/dashboard`               | `GET`         | Main dashboard page                               |
| `/login`                   | `GET`         | Login and recovery page                           |
| `/login`                   | `POST`        | Login submit or password recovery submit          |
| `/logout`                  | `GET`, `POST` | End the current session                           |
| `/licenses`                | `GET`         | Static project licenses and attribution page      |
| `/javascript-license-info` | `GET`         | Public JavaScript license labels page for LibreJS |

### Collection routes

These are mounted through Aile `x/resource`.

#### Providers

| Route                  | Method          |
| ---------------------- | --------------- |
| `/providers`           | `GET`, `POST`   |
| `/providers/new`       | `GET`           |
| `/providers/{id}`      | `GET`, `DELETE` |
| `/providers/{id}/edit` | `GET`, `POST`   |

#### Locations

| Route                  | Method          |
| ---------------------- | --------------- |
| `/locations`           | `GET`, `POST`   |
| `/locations/new`       | `GET`           |
| `/locations/{id}`      | `GET`, `DELETE` |
| `/locations/{id}/edit` | `GET`, `POST`   |

#### Operating systems

| Route           | Method          |
| --------------- | --------------- |
| `/os`           | `GET`, `POST`   |
| `/os/new`       | `GET`           |
| `/os/{id}`      | `GET`, `DELETE` |
| `/os/{id}/edit` | `GET`, `POST`   |

#### IP addresses

| Route            | Method          |
| ---------------- | --------------- |
| `/ips`           | `GET`, `POST`   |
| `/ips/new`       | `GET`           |
| `/ips/{id}`      | `GET`, `DELETE` |
| `/ips/{id}/edit` | `GET`, `POST`   |

#### DNS records

| Route            | Method          |
| ---------------- | --------------- |
| `/dns`           | `GET`, `POST`   |
| `/dns/new`       | `GET`           |
| `/dns/{id}`      | `GET`, `DELETE` |
| `/dns/{id}/edit` | `GET`, `POST`   |

#### Labels

| Route               | Method          |
| ------------------- | --------------- |
| `/labels`           | `GET`, `POST`   |
| `/labels/new`       | `GET`           |
| `/labels/{id}`      | `GET`, `DELETE` |
| `/labels/{id}/edit` | `GET`, `POST`   |

#### Domains

| Route                | Method          |
| -------------------- | --------------- |
| `/domains`           | `GET`, `POST`   |
| `/domains/new`       | `GET`           |
| `/domains/{id}`      | `GET`, `DELETE` |
| `/domains/{id}/edit` | `GET`, `POST`   |

#### Hostings

| Route                 | Method          |
| --------------------- | --------------- |
| `/hostings`           | `GET`, `POST`   |
| `/hostings/new`       | `GET`           |
| `/hostings/{id}`      | `GET`, `DELETE` |
| `/hostings/{id}/edit` | `GET`, `POST`   |

#### Servers

| Route                | Method          |
| -------------------- | --------------- |
| `/servers`           | `GET`, `POST`   |
| `/servers/new`       | `GET`           |
| `/servers/{id}`      | `GET`, `DELETE` |
| `/servers/{id}/edit` | `GET`, `POST`   |

#### Subscriptions

| Route                      | Method          |
| -------------------------- | --------------- |
| `/subscriptions`           | `GET`, `POST`   |
| `/subscriptions/new`       | `GET`           |
| `/subscriptions/{id}`      | `GET`, `DELETE` |
| `/subscriptions/{id}/edit` | `GET`, `POST`   |

### Singleton routes

These are mounted through Aile `x/resource`.

#### Account settings

| Route                    | Method        |
| ------------------------ | ------------- |
| `/account-settings`      | `GET`         |
| `/account-settings/edit` | `GET`, `POST` |

#### App settings

| Route                                | Method        |
| ------------------------------------ | ------------- |
| `/app-settings`                      | `GET`         |
| `/app-settings/edit`                 | `GET`, `POST` |
| `/app-settings/export`               | `GET`         |
| `/app-settings/import`               | `POST`        |
| `/app-settings/expenses`             | `POST`        |
| `/app-settings/expenses/{id}/delete` | `POST`        |

### Uptime module

| Route                 | Method | Purpose                      |
| --------------------- | ------ | ---------------------------- |
| `/uptime`             | `GET`  | Monitor list and create form |
| `/uptime`             | `POST` | Create monitor               |
| `/uptime/{id}`        | `GET`  | Monitor detail page          |
| `/uptime/{id}/edit`   | `POST` | Update monitor               |
| `/uptime/{id}/delete` | `POST` | Delete monitor               |

### JSON APIs

| Route                       | Method | Auth                 |
| --------------------------- | ------ | -------------------- |
| `/api/v1/backup/export`     | `GET`  | Session or API token |
| `/api/v1/backup/import`     | `POST` | Session or API token |
| `/api/v1/dashboard/summary` | `GET`  | Session or API token |

### Static assets

| Route               | Method | Purpose                                     |
| ------------------- | ------ | ------------------------------------------- |
| `/static/{path...}` | `GET`  | CSS, JS, images, vendored browser libraries |

## HTMX contracts

The app uses global `hx-boost="true"` in the base layout, then narrows fragment
behavior in handlers with shared helpers from
[`internal/ui/request.go`](../internal/ui/request.go).

Important patterns:

- list fragments use `ui.IsHTMXListRequest(...)`
- editor fragments use `ui.IsHTMXEditorRequest(...)`
- HTML responses use `ui.WriteHTMLHeader(...)`
- the base layout injects `X-CSRF-Token` for HTMX requests
- unsafe browser requests also require a matching `_csrf` form field or
  `X-CSRF-Token` header, enforced by
  [`internal/csrf/service.go`](../internal/csrf/service.go)

HTMX response helpers are used where redirect semantics matter:

- login
- logout
- app settings import
- auth middleware redirects

## Important internal constructors

### Composition root

- [`newRepositories`](../bootstrap.go)
- [`newServices`](../bootstrap.go)
- [`newHandlers`](../bootstrap.go)
- [`configureMiddleware`](../bootstrap.go)
- [`registerLifecycle`](../bootstrap.go)
- [`mountRoutes`](../routing.go)

### Database and persistence

- [`database.Client`](../internal/database/database.go)
- [`database.SetPragmas`](../internal/database/database.go)
- [`database.RunMigrations`](../internal/database/database.go)
- [`database.SeedReferenceData`](../internal/database/seed.go)

### Services

- [`auth.NewService`](../internal/auth/service.go)
- [`backup.NewService`](../internal/backup/service.go)
- [`finance.NewService`](../internal/finance/service.go)
- [`observability.NewService`](../internal/observability/service.go)
- [`uptime.NewService`](../internal/uptime/service.go)

### Useful shared helpers

- [`ui.NewBaseData`](../internal/ui/common.go)
- [`ui.WriteHTMLHeader`](../internal/ui/request.go)
- [`ui.ParseListState`](../internal/ui/request.go)
- [`ui.IsHTMXListRequest`](../internal/ui/request.go)
- [`ui.IsHTMXEditorRequest`](../internal/ui/request.go)
- [`dashboardsummary.AggregateByLabel[T]`](../internal/models/dashboard_summary/overview.go)
- [`seedMany[T]`](../internal/database/seed.go)
