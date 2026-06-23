# Remove the structure-tree view — Design

**Date:** 2026-06-22
**Status:** Approved
**Branch:** folded into `phase-2b` (unmerged)

## Overview

Remove the structure-tree visualization. It largely re-rendered the raw Corefile
as a list, overlapping with the raw editor (left pane). The request-flow diagram
already conveys structure in execution order, so the tree is redundant.

## Rationale

The four right-pane views and the raw editor now divide cleanly with no overlap:

- **Raw Corefile (left):** full detail — args, nested sub-directives, comments.
- **Request-flow diagram:** the plugin chain per server block in execution order.
- **Validation panel:** problems (parse errors, duplicate zones, unknown plugins).
- **Plugin reference:** summary + doc link for a selected plugin.

The structure tree added neither full detail (the raw has it) nor the execution
picture (the flow has it), so it goes.

## Changes

- Delete `web/src/lib/StructureTree.svelte` and `web/src/lib/Directives.svelte`
  (`Directives` is used only by `StructureTree`).
- Delete `web/test/StructureTree.test.js`.
- `web/src/App.svelte`: remove the `StructureTree` import and its render in the
  views pane.
- `web/src/app.css`: remove the now-unused tree selectors (`.tree`,
  `.directives`, `.keys`, `.args`, `.name`, `.name:hover`, `.block`). Keep flow
  (`.step`, `.endpoint`), validation, reference, card/`.views`, and base theme.

## Ripple / no capability lost

- The **plugin reference panel** previously opened from clicks in both the tree
  and the flow. After removal it opens **only from request-flow steps**. The flow
  lists every plugin in each block, so all plugins remain selectable.
  `RequestFlow`'s existing click-integration test covers this; the tree's click
  test is removed with its file.
- Args and nested sub-directives remain visible in the **raw Corefile** pane.

## Testing

- Frontend Vitest suite passes with `StructureTree.test.js` removed (RequestFlow,
  ValidationPanel, PluginReference, initialContent, selection remain green).
- `npm run build` succeeds with no unresolved imports.
- Manual: the right pane shows Validation → Request-flow → Reference; clicking a
  flow step still populates the reference panel.
