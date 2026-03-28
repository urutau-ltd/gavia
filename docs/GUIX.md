# Guix Packaging

This project ships a local Guix package definition in [`guix.scm`](../guix.scm).

## Build command

From the repository root:

```bash
guix build -f ./guix.scm
```

Or through the existing make target:

```bash
make pkg
```

The result is a store path like:

```text
/gnu/store/...-gavia-dev-dev
```

## Package shape

The package currently defined is:

- `gavia-dev`

It is built with `go-build-system` and uses the module import path declared in
[`go.mod`](../go.mod):

```text
codeberg.org/urutau-ltd/gavia
```

That import path is important. The package must not be built as
`github.com/urutau-ltd/gavia`, because the source tree and imports no longer
match that module path.

## Go inputs

The Guix package explicitly carries the Go dependencies that matter for GOPATH
builds:

- `go-codeberg-org-urutau-ltd-aile-v2`
- `go-github-com-google-uuid`
- `go-modernc-org-sqlite`

These are declared in [`guix.scm`](../guix.scm) so the package can build under
Guix without relying on networked module resolution.

## Why tests are disabled in Guix

The `gavia-dev` package sets:

```scheme
#:tests? #f
```

This is intentional.

Guix's `go-build-system` builds in GOPATH mode and forces `GO111MODULE=off`.
Under that mode, the app binary compiles correctly, but the route-level test
suite fails during the Guix `check` phase with `404` responses on mounted Aile
routes. The same source tree passes its normal module-aware test run outside
that Guix GOPATH check path.

The canonical test command for the repository remains:

```bash
env CGO_ENABLED=0 GOCACHE=/tmp/go-build go test ./...
```

In other words:

- Guix package build validates that the application compiles and installs
- normal repository tests validate application behavior

## Maintenance notes

If the module path changes again, update all three of these together:

- [`go.mod`](../go.mod)
- `#:import-path` in [`guix.scm`](../guix.scm)
- `#:unpack-path` in [`guix.scm`](../guix.scm)

If Guix later gains a module-aware Go packaging path for this package shape, or
the route tests stop failing under GOPATH mode, revisit `#:tests? #f`.
