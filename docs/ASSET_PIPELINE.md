# Asset Pipeline

This document explains the current frontend asset model. The project does not
use a JavaScript bundler or a CSS preprocessor.

## Overview

The frontend asset model is intentionally explicit:

- vendored third-party files are served from [`static/js/`](../static/js/)
- app-owned page scripts also live in [`static/js/`](../static/js/)
- templates load only the scripts needed by each page

There is no intermediate source directory and no generated bundle output.

## Vendored libraries

These files are served directly as static assets:

- [`static/js/htmx.js`](../static/js/htmx.js)
- [`static/js/hyperscript.js`](../static/js/hyperscript.js)
- [`static/js/chartjs.js`](../static/js/chartjs.js)

`chartjs.js` is the standalone UMD build so page scripts can use `window.Chart`
without module resolution in the browser.

## App-owned scripts

Current page-local scripts:

- [`static/js/app-settings.js`](../static/js/app-settings.js)
- [`static/js/dashboard.js`](../static/js/dashboard.js)
- [`static/js/uptime.js`](../static/js/uptime.js)

These are edited directly and loaded only in the templates that need them.

## Editing workflow

When changing page JavaScript:

1. edit the relevant file under [`static/js/`](../static/js/)
2. refresh the browser

When changing vendored libraries:

1. replace the file under `static/js/`
2. verify the page still loads it directly

## Chart.js note

The project expects the standalone UMD build for Chart.js because page scripts
access it through `window.Chart`. Using an ESM-only file in `static/js/chartjs.js`
would break the current page-level integration.
