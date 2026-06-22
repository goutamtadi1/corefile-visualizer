# Phase 2b: Plugin reference panel + metadata + unknown-plugin lint — Design

**Date:** 2026-06-22
**Status:** Approved
**Builds on:** Phase 1, CLI, Phase 2a (request-flow), all merged to `main`

## Overview

Make plugins informative and validated:

1. **Per-plugin metadata** (Go, authoritative) — a one-line summary and an
   official doc link for each in-tree plugin.
2. **Plugin reference panel** (frontend) — clicking a plugin name in the
   structure tree or the request-flow diagram shows that plugin's summary and
   doc link in a side panel.
3. **Unknown-plugin lint** (Go) — a Corefile directive that is not a recognized
   CoreDNS plugin produces a warning.

The two-pane layout (Corefile editor left, generated views right) already landed
on this branch in commit `81e398a`; the new reference panel slots into the
right-hand views pane as a fourth view.

## Why unknown-plugin lint, not ordering lint

CoreDNS sorts a server block's plugins by `plugin.cfg` order at load time, so the
order plugins appear in the Corefile never affects behavior — an "out of order"
warning would be misleading noise. The genuinely meaningful structural check is
an **unknown plugin**: a directive whose name is not a recognized CoreDNS plugin,
which CoreDNS itself would fail to load. That is the lint this phase adds.

## Non-Goals (this spec)

- No full per-plugin **directive schemas** (sub-directive documentation) — summary
  + doc link only; deferred.
- No changes to the `analyze` contract beyond what already exists; the catalog is
  delivered by a separate one-shot call.
- No new heavy frontend dependency.

## Architecture

Plugin metadata is authoritative in Go and exposed to the frontend once at load.

```
internal/plugins:  Order (exec order) + Meta{Summary,DocURL} per plugin
        │
        ├─ Catalog() ──► WASM pluginCatalog() ──► frontend (reference panel)
        └─ Rank() ─────► validate (unknown-plugin lint) ──► diagnostics
```

### `internal/plugins` — metadata

```go
type Meta struct {
    Summary string `json:"summary"`
    DocURL  string `json:"docUrl"`
}
```

- A package-level `meta map[string]Meta` with an entry for **every** name in
  `Order`. `Summary` is a one-line description sourced from each plugin's official
  coredns.io text (pinned, not invented). `DocURL` is
  `https://coredns.io/plugins/<name>/` where such a page exists, and **empty** for
  the handful without one (e.g. `cancel`, `timeouts`, `multisocket`, `on`) so the
  panel never links to a 404.
- `func Catalog() map[string]Meta` returns the metadata (a copy is not required;
  callers must not mutate — documented).
- A test asserts `meta` and `Order` cover exactly the same set of names (drift in
  either direction fails the build).

### WASM — `pluginCatalog()`

`cmd/wasm/main.go` registers a second JS global, `pluginCatalog()`, returning
`json.Marshal(plugins.Catalog())` — a static, one-shot call. `analyze` is
unchanged. The frontend calls `pluginCatalog()` once after `loadWasm()`.

### `internal/validate` — unknown-plugin rule

`validate` begins importing `plugins`. For each server block, each **top-level**
directive whose name is not known (`plugins.Rank` returns `ok == false`) yields one
`Diagnostic`:

- Severity: `warning`
- Message: `unknown plugin "<name>" — not a recognized CoreDNS plugin`
- Line: the directive's line
- Deduped per (server block, plugin name) so repeated unknowns warn once.

Existing rules (duplicate zone, empty block) are unchanged.

## Frontend

- **`web/src/lib/wasm.js`** gains `loadPluginCatalog()` — calls
  `globalThis.pluginCatalog()` and returns the parsed map (throws if the engine
  isn't loaded, mirroring `analyzeCorefile`).
- **`web/src/lib/selection.js`** — a Svelte `writable` store `selectedPlugin`
  (plugin name string, or null).
- **`Directives.svelte`** (tree) and **`RequestFlow.svelte`** (flow) — plugin
  names become clickable; clicking sets `selectedPlugin`. They stay otherwise
  display-only and keep their existing test IDs/behavior.
- **`web/src/lib/PluginReference.svelte`** — props: `catalog` (the map) and reads
  the `selectedPlugin` store. Renders:
  - empty state (`data-testid="reference-empty"`) when nothing is selected,
  - for a selected name in the catalog: the name, its summary, and a doc link
    (only when `docUrl` is non-empty),
  - for a selected name **not** in the catalog: an "unrecognized plugin" note.
- **`App.svelte`** loads the catalog on mount (after wasm) into state and renders
  `<PluginReference {catalog} />` as the fourth child of the right-pane `.views`.

## Data Flow

1. On mount: `loadWasm()` → `loadPluginCatalog()` stores the static catalog.
2. Per edit: `analyze` returns `Result` (now possibly with unknown-plugin
   diagnostics) as before.
3. User clicks a plugin in the tree or flow → `selectedPlugin` updates →
   `PluginReference` shows that plugin's reference from the catalog.

## Error Handling

- Selected plugin missing from the catalog → "unrecognized plugin" note (not an
  error). This is also what produces the validation warning.
- `docUrl` empty → render the summary with no link.
- Catalog load failure → `PluginReference` shows its empty state; the rest of the
  app is unaffected (mirrors the existing wasm-failure resilience).

## Testing

- **`internal/plugins`:** `meta`↔`Order` name-set parity; `Catalog()` returns an
  entry with a non-empty summary for a known plugin; doc URLs that are present
  follow the `coredns.io/plugins/<name>/` pattern.
- **`internal/validate`:** an unknown plugin yields one warning at its line; known
  plugins yield none; a repeated unknown in one block warns once; existing
  duplicate-zone/empty-block tests still pass.
- **WASM/engine:** end-to-end check that `pluginCatalog()` returns a non-empty map
  including a known plugin's summary.
- **Frontend:** `PluginReference` renders summary+link for a known plugin, the
  unrecognized note for an unknown, and the empty state when nothing is selected;
  clicking a plugin in `Directives`/`RequestFlow` updates `selectedPlugin`.

## Decisions

- **Lint:** unknown-plugin (warning), not ordering-violation.
- **Metadata depth:** one-line summary + doc link; no directive schemas.
- **Doc URL:** `coredns.io/plugins/<name>/` where it exists, empty otherwise.
- **Source of truth:** Go `internal/plugins`; frontend gets it via a one-shot
  `pluginCatalog()` WASM call.
- **Selection:** shared `selectedPlugin` store; clickable in both tree and flow.
- **Layout:** two-pane split already implemented (`81e398a`); panel is a fourth
  right-pane view.
- **Branch:** `phase-2b`, off `main`.
