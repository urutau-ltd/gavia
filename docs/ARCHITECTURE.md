# Architecture Overview

This document explains how the current application is composed and where the
main responsibilities live.

## Runtime shape

The application starts in [`main.go`](../main.go), but the composition logic is
split across small helpers in the `main` package:

- [`main.go`](../main.go): process entrypoint, config loading, logger setup, DB
  bootstrap, embedded FS mounting, and `app.Run`
- [`bootstrap.go`](../bootstrap.go): repository construction, service
  construction, handler construction, middleware wiring, and lifecycle hooks
- [`routing.go`](../routing.go): route interfaces and public route mounting

This keeps `main.go` short without hiding runtime behavior in a separate
framework.

## Main layers

### 1. Database layer

The database package lives in [`internal/database/`](../internal/database/).

Key responsibilities:

- open the SQLite connection
- apply pragmas
- run embedded migrations
- seed deterministic reference data

Important files:

- [`internal/database/database.go`](../internal/database/database.go)
- [`internal/database/seed.go`](../internal/database/seed.go)
- [`internal/database/migrations/`](../internal/database/migrations/)

### 2. Model and repository layer

Each domain area owns its repository package under
[`internal/models/`](../internal/models/).

Examples:

- providers: [`internal/models/provider/`](../internal/models/provider/)
- locations: [`internal/models/location/`](../internal/models/location/)
- account settings: [`internal/models/account_setting/`](../internal/models/account_setting/)
- uptime monitors: [`internal/models/uptime_monitor/`](../internal/models/uptime_monitor/)

Repositories are intentionally explicit and SQL-centric. The app does not use an
ORM.

### 3. Service layer

Services coordinate operations that are broader than a single repository.

Current service packages:

- [`internal/auth/`](../internal/auth/): account lookup, sessions, redirects,
  API token auth, and middleware
- [`internal/backup/`](../internal/backup/): JSON export/import plus optional
  encrypted backups
- [`internal/finance/`](../internal/finance/): FX sampling and currency
  conversion
- [`internal/observability/`](../internal/observability/): runtime sampling
- [`internal/uptime/`](../internal/uptime/): scheduled HTTP checks
- [`internal/security/`](../internal/security/): password hashing, token
  hashing, ML-KEM recovery keys, HKDF, and AES-GCM backup encryption

### 4. UI layer

The UI lives in [`internal/ui/`](../internal/ui/).

Important parts:

- [`internal/ui/layout/base.html`](../internal/ui/layout/base.html): shared page
  shell
- [`internal/ui/components/`](../internal/ui/components/): footer and shared
  fragments
- [`internal/ui/common.go`](../internal/ui/common.go): shared page metadata and
  footer diagnostics
- [`internal/ui/request.go`](../internal/ui/request.go): shared HTMX and HTML
  helpers
- [`internal/ui/features/`](../internal/ui/features/): one package per feature

The UI is server-rendered. HTMX is used for partial updates, not to replace the
page model entirely.

## Routing model

Routes are mounted in [`routing.go`](../routing.go).

The app currently uses three route styles:

- Manual pages:
  `dashboard`, `login`, `logout`, `licenses`, `uptime`, and API endpoints
- Collection resources mounted with Aile `x/resource`:
  `providers`, `locations`, `os`, `ips`, `dns`, `labels`, `domains`,
  `hostings`, `servers`, and `subscriptions`
- Singleton resources mounted with Aile `x/resource`:
  `account-settings`, `app-settings`

This split is intentional:

- repetitive CRUD modules benefit from `resource.MountCollection`
- settings pages fit `resource.MountSingleton`
- special workflows stay manual

## Background work

Three services run in the background after startup:

- FX sampler
- runtime diagnostics sampler
- uptime monitor scheduler

Lifecycle registration lives in [`bootstrap.go`](../bootstrap.go), and shutdown
is coordinated through a shared cancellable context plus DB close on process
stop.

## Authentication model

Authentication is single-account, local-first, and explicit.

Pieces involved:

- account settings singleton stores username, password hash, API token hash,
  recovery public key, and avatar
- session records live in SQLite
- session cookie name is `gavia_session`
- API routes accept session auth or token auth where appropriate
- CSRF token cookie name is `gavia_csrf`

Relevant files:

- [`internal/auth/service.go`](../internal/auth/service.go)
- [`internal/csrf/service.go`](../internal/csrf/service.go)
- [`internal/models/account_setting/`](../internal/models/account_setting/)
- [`internal/models/session/`](../internal/models/session/)

## Security model

The app does not use email-based recovery.

Current primitives:

- password hashing: PBKDF2 with SHA3-256
- API/session token hashing: SHA3-256
- recovery key pair: ML-KEM-768
- encrypted backups: ML-KEM-768 + HKDF(SHA3-256) + AES-GCM
- browser CSRF protection: `http.NewCrossOriginProtection()` plus a
  double-submit token for compatibility with older browsers and requests that
  do not send modern fetch metadata headers

Implementation:

- [`internal/security/security.go`](../internal/security/security.go)
- [`internal/csrf/service.go`](../internal/csrf/service.go)

## Frontend asset model

The project ships:

- vendored static libraries in [`static/js/`](../static/js/)
- small app-owned page scripts in [`static/js/`](../static/js/)

The asset pipeline is documented in
[`docs/ASSET_PIPELINE.md`](./ASSET_PIPELINE.md).

## Generics in the codebase

Generics are used sparingly and only where they reduce real repetition without
hiding behavior:

- [`internal/database/seed.go`](../internal/database/seed.go):
  `seedMany[T]` reduces repeated seed loops
- [`internal/models/dashboard_summary/overview.go`](../internal/models/dashboard_summary/overview.go):
  `AggregateByLabel[T]` groups due and expense data without duplicating logic

The project deliberately avoids turning handlers, repositories, and route wiring
into a generic meta-framework. That would reduce local clarity more than it
would reduce code.

## Tables without CRUD UI

Not every table in the schema should have a browser CRUD screen.

Background and internal state tables such as `user_sessions`,
`exchange_rate_samples`, `runtime_samples`, and `uptime_monitor_results` are
kept internal on purpose. They exist to support authentication, diagnostics,
currency sampling, and monitor history, not to expose manual record editing in
the UI.
