# Tempel Snippets (Emacs)

This repository includes [Emacs Tempel](https://github.com/minad/tempel)
templates to speed up repetitive CRUD code.

## Where

- Snippet file: `etc/snippets/tempel/templates.eld`
- Emacs integration: `.dir-locals.el`

The `.dir-locals.el` block resolves the repo root and prepends this file to
`tempel-path` automatically when you open any file in the project.

## Why

The snippets focus on repeated patterns in this codebase:

- Go handler/repository scaffolds
- HTMX template skeletons (`index`, `list`, `editor` partials)
- CSS grid layout pattern for list + editor views
- Docstring starter oriented to "where" and "why"

This keeps new CRUD features consistent with providers/locations and reduces
manual boilerplate.

## Included snippet names

### `go-mode`

- `gdoc`: Go doc comment scaffold (where/why style)
- `gindex`: Index handler with HTMX/full-page branch pattern
- `ghandler`: handler struct + constructor scaffold
- `grepo`: repository scaffold with `GetAll` query pattern

### `html-mode` / `nhtml-mode` / `web-mode`

- `htmxindex`: main CRUD view skeleton
- `htmxlist`: table-body partial with row actions
- `htmxeditor`: right-side editor panel partial

### `css-mode`

- `crudcss`: responsive list/editor layout block

## Usage

1. Open a file in this repository from Emacs.
2. Run `M-x tempel-insert`.
3. Choose one of the snippet names above.

If `tempel-path` was already configured globally, the repo file is prepended,
not replaced.
