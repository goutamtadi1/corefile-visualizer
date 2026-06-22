# Phase 2a: Request-flow diagram — Design

**Date:** 2026-06-22
**Status:** Approved
**Builds on:** Phase 1 MVP (merged to `main`)

## Overview

A third visualization view: for each server block, show the query flowing through
its plugins in **true CoreDNS execution order**. CoreDNS runs a block's plugins in
the canonical order defined by its `plugin.cfg`, not the order they appear in the
Corefile — this diagram makes that ordering visible.

This is the first slice of Phase 2. It introduces only the `plugin.cfg` ordering
data (a small part of the eventual plugin registry). The plugin reference panel,
full per-plugin directive metadata, and plugin-aware/ordering-violation
validation are explicitly deferred to a later Phase 2 spec.

## Goals

- For each server block, render its plugins as a linear chain in execution order:
  `request → [plugin] → [plugin] → … → response`.
- Order known plugins by their `plugin.cfg` rank; collapse repeated directives of
  the same plugin into a single chain step.
- Visually flag plugins not present in `plugin.cfg` as unrecognized.

## Non-Goals (this spec)

- No plugin reference panel, no per-plugin directive schemas.
- No new validation rules (unknown-plugin / ordering-violation lint stays
  deferred); `validate` is unchanged.
- No modeling of fallthrough, conditional branching, or per-plugin internal
  behavior — the flow is the ordered chain only.
- No heavy diagramming dependency (rendered with native Svelte/CSS/SVG).

## Architecture

The ordering is authoritative in Go and travels to the frontend through the
existing JSON contract.

```
Corefile ─► analyzer (structure) ─► engine.Run ─► model.Result ─► frontend
                                       │  enriches each ServerBlock with Flow
                                       └─ plugins.BuildFlow(directives)
                                          using plugins.Order (plugin.cfg)
```

### New Go package: `internal/plugins`

Holds the canonical `plugin.cfg` execution order and the flow builder.

- `Order []string` — the plugin names in `plugin.cfg` order (index = priority).
  **Sourced verbatim** from CoreDNS's `plugin.cfg` at a pinned release (recorded
  in the source file's doc comment with the exact CoreDNS version/commit and the
  source URL `https://github.com/coredns/coredns/blob/<ref>/plugin.cfg`). Not
  hand-guessed.
- `func Rank(name string) (int, bool)` — returns the plugin's index in `Order`
  and whether it is known.
- `func BuildFlow(directives []model.Directive) []model.FlowStep` — produces the
  block's chain: collect the **distinct** top-level plugin names (first
  occurrence wins; repeated directives like two `file`s collapse to one step),
  partition into known vs unknown, sort known by `Rank`, then append unknown
  names in declaration order. Each step is flagged `Known`.

### Engine composition

`engine.Run`, after `analyzer.Analyze` succeeds, sets `sb.Flow =
plugins.BuildFlow(sb.Directives)` for each server block, then runs `validate` as
today. The analyzer remains purely structural (no plugin knowledge).

### Model addition (the contract)

`ServerBlock` gains a field:

```go
Flow []FlowStep `json:"flow"`
```

with the new type:

```go
type FlowStep struct {
    Name  string `json:"name"`
    Known bool   `json:"known"`
}
```

This serializes through the existing WASM `analyze` → JSON path automatically; the
frontend `types.js` gains matching typedefs.

## Frontend

A new `web/src/lib/RequestFlow.svelte` (prop: `corefile`) renders, per server
block, a labeled horizontal chain:

- A leading `request` node, each plugin as a box joined by `→` arrows, and a
  trailing `response` node.
- Boxes wrap to the next line on narrow widths.
- Steps with `known === false` get a distinct "unrecognized" style.
- Nodes show the plugin name only (arguments/details stay in the structure tree).
- An empty state when `corefile` is null.

It is wired into `App.svelte` as a third view alongside `StructureTree` and
`ValidationPanel`, reading `result?.corefile`.

## Data Flow

1. Text → WASM `analyze` → `engine.Run` parses, builds `Flow` per block, validates.
2. JSON `Result` now carries `serverBlocks[].flow`.
3. Frontend renders the tree, validation panel, and the new request-flow view from
   the same `Result`.

## Error Handling

- Unknown plugins: not an error; appended to the chain and flagged `Known:false`.
- Empty server block: produces an empty `Flow` (just `request → response`); no
  error.
- Parse error: `Corefile` is nil as today; the flow view shows its empty state.

## Testing

- **`internal/plugins`:** `Rank` for a known plugin (correct index) and an unknown
  (ok=false); `BuildFlow` orders known plugins by `plugin.cfg` rank, collapses a
  repeated directive into one step, appends an unknown plugin in declaration order
  with `Known:false`, and returns an empty slice for no directives.
- **`internal/engine`:** `Run` populates `Flow` on each server block for a
  multi-plugin Corefile (asserting execution order differs from declaration order
  for a known case).
- **`internal/model`:** the `Flow`/`FlowStep` JSON field names are covered by the
  existing contract test (extend it to include a populated `Flow`).
- **Frontend:** `RequestFlow` renders steps in order, marks an unknown step, and
  shows the empty state when `corefile` is null.

## Decisions

- **Ordering semantic:** `plugin.cfg` execution order (not declaration order).
- **Repeated directives:** collapse to a single chain step per plugin name.
- **Unknown plugins:** appended after known plugins, flagged `Known:false`.
- **Computation site:** Go (`internal/plugins` + `engine.Run`), authoritative and
  reusable for later ordering-violation lint.
- **Rendering:** lightweight native Svelte/CSS/SVG; horizontal `request → … →
  response` chain, wrapping.
- **Branch:** `phase-2`, off `main`.
