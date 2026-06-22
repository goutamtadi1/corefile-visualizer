# CoreDNS Corefile Visualizer — Design

**Date:** 2026-06-22
**Status:** Approved

## Overview

A fully client-side web application for visualizing [CoreDNS](https://coredns.io)
Corefiles. A user pastes or uploads a Corefile; a Go engine compiled to
WebAssembly parses and analyzes it entirely in the browser (no data leaves the
user's machine); a Svelte UI renders the result as a structure tree, a
request-flow diagram, validation results, and an inline plugin reference.

The app is static and hostable for free on GitHub Pages. It is open source under
Apache-2.0 (matching CoreDNS and its parser).

## Goals

1. **Structure tree** — a clear hierarchical view of server blocks → zones/ports
   → plugins in declaration order, so a complex Corefile is readable at a glance.
2. **Request-flow diagram** — show the plugin execution order (the path a DNS
   query takes through the plugin chain) per server block.
3. **Validation & lint** — flag syntax errors, unknown/misordered plugins,
   duplicate zones, and common misconfigurations, with line references.
4. **Plugin reference** — inline descriptions and directive docs for each
   recognized plugin, surfaced on click/hover.

## Non-Goals (v1)

- Executing or simulating actual DNS resolution.
- Editing/round-tripping the Corefile back out (it is read + analyze only).
- Covering external/third-party plugins beyond the official in-tree set.
- Any server-side processing or persistence.

## Architecture

```
Corefile text ──► [WASM engine] ──► JSON model ──► [Svelte stores] ──► views
   (editor)        (Go, in-browser)                                   ├─ Structure tree
                                                                      ├─ Request-flow diagram
                                                                      ├─ Validation panel
                                                                      └─ Plugin reference
```

**Go engine (compiled to WASM):** wraps `github.com/coredns/caddy` — the real
CoreDNS configuration parser (Apache-2.0) — so parsing matches CoreDNS exactly.
On top of the raw parse it builds a structured model and runs semantic checks.
It exposes a single JS-callable function `analyze(text) → JSON`.

**Frontend (Svelte + Vite):** an editor pane (CodeMirror editor + file upload +
bundled sample Corefiles), loads `main.wasm` via `wasm_exec.js`, calls `analyze`
debounced on input, and renders four views from the returned model.
The request-flow diagram uses Svelte Flow (`@xyflow/svelte`); the tree and
reference panels are native Svelte components.

## Engine Internals

Each unit has one clear purpose and is independently testable.

- **`internal/model`** — structured, JSON-serializable types:
  `ServerBlock → keys (zones/ports) → directives (plugin name, args, line)`,
  plus a `Diagnostic` type (severity, message, line/col). This is the contract
  between the engine and the frontend.
- **`internal/analyzer`** — wraps the caddy parser and walks its tokens into the
  `model` types. Owns nothing about plugin semantics; purely structural.
- **`internal/plugins`** — curated metadata registry for the ~30 official
  in-tree CoreDNS plugins: name, one-line summary, directive schema, and
  `plugin.cfg` execution-order priority. Pure data + lookup helpers.
- **`internal/validate`** — semantic lint layered on the model using the plugin
  registry: unknown plugins (flagged, non-fatal), plugin-ordering violations vs
  `plugin.cfg`, duplicate zones, malformed blocks, and known missing/invalid
  arguments. Emits `Diagnostic`s.
- **`cmd/wasm/main.go`** — `syscall/js` glue: registers `analyze`, marshals the
  model + diagnostics to JSON, and surfaces parse panics as diagnostics.

## Data Flow

1. User types/pastes/uploads Corefile text in the editor.
2. Frontend debounces and calls `analyze(text)` on the WASM module.
3. Engine parses (caddy), builds the model, runs validation, returns JSON:
   `{ serverBlocks: [...], diagnostics: [...] }`.
4. Frontend deserializes into Svelte stores; the four views derive from that
   single model. Plugin reference content is keyed by plugin name against a
   frontend copy (or WASM-provided subset) of the plugin registry.

## Repo Layout

```
go.mod
cmd/wasm/main.go            # syscall/js glue
internal/
  model/                   # structured types + JSON
  analyzer/                # parse wrapper → model
  plugins/                 # curated in-tree plugin metadata registry
  validate/                # semantic lint rules
web/                       # Svelte/Vite app
  src/
  public/wasm/             # main.wasm, wasm_exec.js (built artifacts)
scripts/build-wasm.sh      # GOOS=js GOARCH=wasm build into web/public/wasm
.github/workflows/         # build + deploy to GitHub Pages
README.md
LICENSE                    # Apache-2.0
```

## Build & Hosting

- `scripts/build-wasm.sh` runs `GOOS=js GOARCH=wasm go build` to emit
  `main.wasm` and copies Go's `wasm_exec.js` into `web/public/wasm/`.
- Vite builds the Svelte app into a static bundle that includes the WASM assets.
- A GitHub Actions workflow builds the WASM, builds the Vite site, and deploys
  the static output to GitHub Pages.

## Testing

- **Go:** table-driven tests over Corefile fixtures with golden JSON for
  `analyzer` and `validate`; unit tests for `plugins` lookups.
- **Frontend:** Vitest component tests for the tree, diagram, validation, and
  reference views; shared sample Corefiles as fixtures.

## Phasing

- **Phase 1 (MVP):** `model` + `analyzer` + WASM build + Svelte editor, structure
  tree, and validation panel. Establishes the end-to-end text → WASM → render
  pipeline.
- **Phase 2:** request-flow diagram + plugin reference panel + the full in-tree
  plugin metadata registry feeding `validate`.
- **Phase 3:** polish — bundled sample Corefiles, shareable URL (Corefile encoded
  in the URL hash), and the GitHub Pages deploy workflow.

## Decisions

- **Corefile type:** CoreDNS configuration Corefile (not a process core dump).
- **Execution model:** Go → WASM, fully client-side. No backend.
- **Frontend:** Svelte + Vite.
- **Parser:** reuse the real CoreDNS parser (`github.com/coredns/caddy`).
- **Plugin scope (v1):** curated official in-tree plugins; unknown plugins shown
  but flagged.
- **License:** Apache-2.0.
- **Editor:** CodeMirror.
