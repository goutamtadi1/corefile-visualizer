# CLI "pipe & visualize" — Design

**Date:** 2026-06-22
**Status:** Approved
**Builds on:** Phase 1 MVP (branch `phase-1-mvp`)

## Overview

A native command-line entrypoint that lets a user pipe (or pass) a CoreDNS
Corefile and immediately see it in the existing browser visualization:

```bash
cat Corefile | corefile-visualizer      # read from a piped stdin
corefile-visualizer ./Corefile           # or a file-path argument
corefile-visualizer --no-open ./Corefile # start the server but don't open a browser
corefile-visualizer --port 8080 ...       # pin the port (default: random free port)
```

The CLI reads the Corefile, starts a small local HTTP server that serves the
already-built Svelte/WASM app with the content pre-loaded, opens the user's
browser to it, prints the URL, and stays running until the user presses
**Ctrl-C**.

## Goals

- `cat Corefile | corefile-visualizer` opens the browser showing that Corefile's
  structure tree and validation — reusing the existing web UI unchanged in
  behavior.
- Also accept a file-path argument when stdin is not piped.
- `--no-open` (for headless/remote machines) and `--port` flags.
- Server stays alive until Ctrl-C, so the page can refresh and re-analyze edits.

## Non-Goals

- No terminal/TUI rendering of the visualization (browser only).
- No change to how the standalone web app (dev server / future GitHub Pages)
  behaves — the `/corefile` mechanism degrades gracefully to the existing SAMPLE.
- No watch/reload of the source file after start (the page is interactive on its
  own; re-piping is a fresh invocation).

## Architecture

A **new native Go binary** `cmd/corefile-visualizer` (distinct from the existing
`cmd/wasm` WASM target). It embeds the built web app and serves it locally.

```
stdin / file ─► [CLI: read content] ─► [local HTTP server] ─► browser opens ─► existing Svelte+WASM UI
                                          ├─ serves embedded web app (index.html, JS, main.wasm, wasm_exec.js)
                                          └─ GET /corefile → the piped text (text/plain)
```

**Why a local HTTP server rather than a `file://` page:** WebAssembly
instantiation (`WebAssembly.instantiateStreaming`) requires the `.wasm` asset to
be served with the correct `application/wasm` MIME type over HTTP; `file://`
loading is unreliable across browsers. A tiny embedded server sidesteps this and
allows page refresh.

## Content Delivery (the only frontend change)

The server exposes `GET /corefile` returning the raw Corefile text as
`text/plain`. On mount, the frontend attempts `fetch('corefile')` (relative to
`import.meta.env.BASE_URL`):

- **200 OK** → use the returned text as the initial editor content.
- **404 / network error** → fall back to the current `SAMPLE` constant.

This is the only change to the existing web app. In the standalone deployment
(dev server, GitHub Pages) there is no `/corefile` route, so the fetch fails and
the app behaves exactly as today.

## Build Coupling

`go:embed` patterns cannot contain `..`, so the binary cannot embed `web/dist`
directly from `cmd/corefile-visualizer/`. Instead:

- `internal/webui/` contains `embed.go` with `//go:embed all:dist` exposing the
  app as an `fs.FS` (function `FS()` returning the `dist` sub-filesystem).
- `internal/webui/dist/` is **gitignored**; the build script copies `web/dist`
  into it before `go build`.

Build order (wrapped in `scripts/build-cli.sh`):

1. `./scripts/build-wasm.sh` — WASM engine into `web/public/wasm/`.
2. `cd web && npm run build` — produces `web/dist/` (includes the wasm assets).
3. Copy `web/dist/` → `internal/webui/dist/`.
4. `go build -o bin/corefile-visualizer ./cmd/corefile-visualizer`.

Producing the CLI therefore needs the Node toolchain once at build time; end
users only run the resulting binary.

## Components

Each unit has one responsibility and is testable in isolation.

- **`internal/webui`** — `//go:embed all:dist` wrapper; `func FS() (fs.FS, error)`
  returning the embedded app's root filesystem. Pure embed + sub-FS.
- **`internal/cliserver`** — the server logic, browser-free and unit-testable:
  - `func ReadCorefile(stdin io.Reader, stdinIsPipe bool, fileArg string) (string, error)`
    — stdin if piped, else the file arg, else an error.
  - `func Handler(app fs.FS, corefile string) http.Handler` — serves the embedded
    app (static file server) and `GET /corefile` (text/plain, the content).
  - `func ListenLocal(port int) (net.Listener, error)` — binds `127.0.0.1:port`
    (port 0 = a random free port); returns the listener (so the chosen port is
    knowable for the printed URL).
- **`cmd/corefile-visualizer/main.go`** — flag parsing (`--no-open`, `--port`),
  wiring `webui` + `cliserver`, opening the browser (behind a swappable
  `openBrowser` func var so tests skip it), printing the URL, and serving until
  SIGINT/SIGTERM.

## Browser Opening

Platform dispatch in `main.go`: `open` (darwin), `xdg-open` (linux),
`rundll32 url.dll,FileProtocolHandler` / `cmd /c start` (windows). Failure to
open is non-fatal — the URL is already printed and the server keeps running.
`--no-open` skips the attempt entirely.

## Error Handling

- No stdin pipe and no file arg → print usage and exit non-zero.
- File arg that cannot be read → error to stderr, exit non-zero.
- Empty input is allowed (the app shows an empty editor; not an error).
- Port already in use (explicit `--port`) → error to stderr, exit non-zero.
- A Corefile parse error is NOT a CLI error — it is surfaced in the browser via
  the existing validation panel (the CLI does not parse; the WASM app does).

## Testing

- **Go (`internal/cliserver`):**
  - `ReadCorefile`: piped stdin wins; file arg used when not piped; missing both
    errors; unreadable file errors.
  - `Handler`: `GET /corefile` returns the exact piped text with
    `Content-Type: text/plain`; `GET /` serves `index.html`; an embedded asset
    (e.g. a known file) is served; an unknown path under the app returns the
    app's static behavior. Tested with `httptest`.
  - `ListenLocal`: port 0 yields a bound listener with a non-zero port.
- **Go (`internal/webui`):** `FS()` returns a filesystem containing `index.html`
  (uses a tiny test `dist` fixture so the test does not require a real build).
- **Frontend:** the initial-content loader uses fetched `/corefile` text when the
  fetch resolves with content, and falls back to `SAMPLE` on a failed/404 fetch
  (mocked `fetch`). This logic is extracted into a small testable function in
  `web/src/lib/`.

## Decisions

- **Output target:** browser (reuse the existing UI), not terminal.
- **Transport:** embedded local HTTP server (not `file://`).
- **Inputs:** piped stdin, or a file-path argument.
- **Flags:** `--no-open`, `--port` (default random free port).
- **Lifecycle:** stay running until Ctrl-C.
- **Frontend impact:** single graceful change — fetch `/corefile`, fall back to
  `SAMPLE`.
- **Branch:** `cli-pipe-visualize`, off `phase-1-mvp`.
