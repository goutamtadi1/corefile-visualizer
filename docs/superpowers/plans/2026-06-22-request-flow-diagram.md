# Request-flow Diagram (Phase 2a) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** For each server block, compute and display the query's plugin execution chain in true CoreDNS `plugin.cfg` order, as a third frontend view.

**Architecture:** A new `internal/plugins` package embeds CoreDNS's pinned `plugin.cfg` ordering and builds, per server block, a `[]FlowStep` (distinct plugins, known ones sorted by execution rank, unknown ones appended and flagged). `engine.Run` enriches each `ServerBlock` with its `Flow`; it travels through the existing WASM→JSON contract to a new lightweight Svelte view.

**Tech Stack:** Go 1.26; existing Svelte + Vite + Vitest frontend (native CSS/SVG, no new dependency).

## Global Constraints

- Go module path: `github.com/gtadi/corefile-visualizer`; Go 1.26.
- `plugin.cfg` ordering is sourced **verbatim from CoreDNS v1.12.0** (`https://github.com/coredns/coredns/blob/v1.12.0/plugin.cfg`); the exact list is in this plan — do not re-derive or reorder it.
- Execution order = `plugin.cfg` order, NOT Corefile declaration order.
- Repeated directives of the same plugin collapse to a single chain step (first occurrence).
- Unknown plugins (not in `plugin.cfg`) are appended after known ones, in declaration order, flagged `Known:false`.
- `validate` is unchanged (no new diagnostics this phase).
- Rendering uses native Svelte/CSS/SVG — no new frontend dependency.
- The JSON contract addition is `ServerBlock.Flow []FlowStep` with `FlowStep{Name string, Known bool}`.
- Conventional commit messages; no Claude co-author.
- Branch: `phase-2` (already created off `main`).

## File Structure

```
internal/model/model.go            # (modify) add FlowStep type + ServerBlock.Flow field
internal/model/model_test.go       # (modify) cover Flow in the JSON contract
internal/plugins/plugins.go        # (create) Order, Rank, BuildFlow
internal/plugins/plugins_test.go   # (create)
internal/engine/engine.go          # (modify) enrich each ServerBlock with Flow
internal/engine/engine_test.go     # (modify) assert Flow populated in execution order
web/src/lib/types.js               # (modify) add FlowStep typedef + flow on ServerBlock
web/src/lib/RequestFlow.svelte     # (create) the chain view
web/test/RequestFlow.test.js       # (create)
web/src/App.svelte                 # (modify) render RequestFlow as a third view
```

**Scope:** Single cohesive feature (the diagram + the `plugin.cfg` ordering slice). Reference panel, per-plugin directive metadata, and ordering-violation lint remain deferred.

---

### Task 1: Model — FlowStep type and ServerBlock.Flow field

**Files:**
- Modify: `internal/model/model.go`
- Modify: `internal/model/model_test.go`

**Interfaces:**
- Produces: `model.FlowStep{ Name string (json:"name"); Known bool (json:"known") }` and `model.ServerBlock.Flow []FlowStep (json:"flow")`. Consumed by `plugins.BuildFlow` (Task 2), `engine.Run` (Task 3), and the frontend (Task 4).

- [ ] **Step 1: Write the failing test**

Add to `internal/model/model_test.go`:

```go
func TestServerBlockFlowJSON(t *testing.T) {
	sb := ServerBlock{
		Keys: []string{"."},
		Line: 1,
		Directives: []Directive{{Name: "whoami", Line: 2}},
		Flow: []FlowStep{
			{Name: "errors", Known: true},
			{Name: "myplugin", Known: false},
		},
	}
	b, err := json.Marshal(sb)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	want := `{"keys":["."],"line":1,"directives":[{"name":"whoami","line":2}],"flow":[{"name":"errors","known":true},{"name":"myplugin","known":false}]}`
	if got != want {
		t.Errorf("Flow contract changed:\n got=%s\nwant=%s", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/model/ -run TestServerBlockFlowJSON`
Expected: FAIL — `unknown field 'Flow' in struct literal` / `undefined: FlowStep`.

- [ ] **Step 3: Add the type and field**

In `internal/model/model.go`, add the `Flow` field to `ServerBlock` (after `Directives`):

```go
type ServerBlock struct {
	Keys       []string    `json:"keys"`
	Line       int         `json:"line"`
	Directives []Directive `json:"directives"`
	Flow       []FlowStep  `json:"flow"`
}
```

And add the new type (place it directly after the `Directive` type):

```go
// FlowStep is one plugin in a server block's request-execution chain. Steps are
// ordered by CoreDNS plugin.cfg execution order; Known is false for plugins not
// present in plugin.cfg.
type FlowStep struct {
	Name  string `json:"name"`
	Known bool   `json:"known"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/model/`
Expected: PASS — all model tests, including `TestServerBlockFlowJSON` and the existing `TestResultJSONFieldNames` (unchanged: it marshals no `ServerBlock`, so the new field does not affect its output).

- [ ] **Step 5: Commit**

```bash
git add internal/model/
git commit -m "feat: add FlowStep type and ServerBlock.Flow to the model"
```

---

### Task 2: Plugins package — plugin.cfg order, Rank, BuildFlow

**Files:**
- Create: `internal/plugins/plugins.go`
- Test: `internal/plugins/plugins_test.go`

**Interfaces:**
- Consumes: `model.Directive`, `model.FlowStep` (Task 1).
- Produces:
  - `plugins.Order []string` — plugin.cfg order (index = priority).
  - `func plugins.Rank(name string) (int, bool)` — index in `Order`, and whether known.
  - `func plugins.BuildFlow(directives []model.Directive) []model.FlowStep` — distinct top-level plugin names; known sorted by `Rank` ascending; unknown appended in first-occurrence order; each flagged. Returns a non-nil (possibly empty) slice.

- [ ] **Step 1: Write the failing test**

`internal/plugins/plugins_test.go`:

```go
package plugins

import (
	"testing"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

func TestRankKnownAndUnknown(t *testing.T) {
	ie, ok := Rank("errors")
	il, _ := Rank("log")
	if !ok || ie >= il {
		t.Fatalf("expected errors known and ranked before log: errors=%d ok=%v log=%d", ie, ok, il)
	}
	if _, ok := Rank("definitely-not-a-plugin"); ok {
		t.Error("expected unknown plugin to report ok=false")
	}
}

func dirs(names ...string) []model.Directive {
	out := make([]model.Directive, len(names))
	for i, n := range names {
		out[i] = model.Directive{Name: n}
	}
	return out
}

func names(flow []model.FlowStep) []string {
	out := make([]string, len(flow))
	for i, f := range flow {
		out[i] = f.Name
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildFlowOrdersByPluginCfg(t *testing.T) {
	// Declaration order: log, errors, forward, cache.
	// plugin.cfg execution order: errors < log < cache < forward.
	flow := BuildFlow(dirs("log", "errors", "forward", "cache"))
	got := names(flow)
	want := []string{"errors", "log", "cache", "forward"}
	if !equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for _, f := range flow {
		if !f.Known {
			t.Errorf("expected %q known", f.Name)
		}
	}
}

func TestBuildFlowCollapsesRepeats(t *testing.T) {
	flow := BuildFlow(dirs("file", "file"))
	if got := names(flow); !equal(got, []string{"file"}) {
		t.Fatalf("repeats not collapsed: %v", got)
	}
}

func TestBuildFlowAppendsUnknownFlagged(t *testing.T) {
	flow := BuildFlow(dirs("whoami", "customplugin"))
	got := names(flow)
	if !equal(got, []string{"whoami", "customplugin"}) {
		t.Fatalf("order = %v, want [whoami customplugin]", got)
	}
	if !flow[0].Known {
		t.Error("whoami should be Known")
	}
	if flow[1].Known {
		t.Error("customplugin should be Known=false")
	}
}

func TestBuildFlowEmptyIsNonNil(t *testing.T) {
	flow := BuildFlow(nil)
	if flow == nil {
		t.Fatal("BuildFlow(nil) returned nil; want empty non-nil slice")
	}
	if len(flow) != 0 {
		t.Fatalf("want empty, got %v", names(flow))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/plugins/`
Expected: FAIL — `undefined: Rank`, `undefined: BuildFlow`.

- [ ] **Step 3: Write the implementation**

`internal/plugins/plugins.go`:

```go
// Package plugins holds the CoreDNS plugin execution order and builds a server
// block's request-flow chain. The order is taken verbatim from CoreDNS's
// plugin.cfg at v1.12.0:
// https://github.com/coredns/coredns/blob/v1.12.0/plugin.cfg
package plugins

import (
	"sort"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

// Order is the CoreDNS plugin.cfg execution order (index = priority). Sourced
// verbatim from CoreDNS v1.12.0; do not reorder.
var Order = []string{
	"root", "metadata", "geoip", "cancel", "tls", "timeouts", "multisocket",
	"reload", "nsid", "bufsize", "bind", "debug", "trace", "ready", "health",
	"pprof", "prometheus", "errors", "log", "dnstap", "local", "dns64", "acl",
	"any", "chaos", "loadbalance", "tsig", "cache", "rewrite", "header", "dnssec",
	"autopath", "minimal", "template", "transfer", "hosts", "route53", "azure",
	"clouddns", "k8s_external", "kubernetes", "file", "auto", "secondary", "etcd",
	"loop", "forward", "grpc", "erratic", "whoami", "on", "sign", "view",
}

var rankByName = func() map[string]int {
	m := make(map[string]int, len(Order))
	for i, name := range Order {
		m[name] = i
	}
	return m
}()

// Rank returns the plugin's index in Order and whether it is a known plugin.
func Rank(name string) (int, bool) {
	i, ok := rankByName[name]
	return i, ok
}

// BuildFlow returns the request-execution chain for a server block's top-level
// directives: distinct plugin names (first occurrence wins), known plugins
// sorted by plugin.cfg rank, then unknown plugins appended in declaration order.
// The result is always non-nil.
func BuildFlow(directives []model.Directive) []model.FlowStep {
	seen := map[string]bool{}
	var known []string
	var unknown []string
	for _, d := range directives {
		if seen[d.Name] {
			continue
		}
		seen[d.Name] = true
		if _, ok := Rank(d.Name); ok {
			known = append(known, d.Name)
		} else {
			unknown = append(unknown, d.Name)
		}
	}
	sort.SliceStable(known, func(i, j int) bool {
		ri, _ := Rank(known[i])
		rj, _ := Rank(known[j])
		return ri < rj
	})

	flow := make([]model.FlowStep, 0, len(known)+len(unknown))
	for _, n := range known {
		flow = append(flow, model.FlowStep{Name: n, Known: true})
	}
	for _, n := range unknown {
		flow = append(flow, model.FlowStep{Name: n, Known: false})
	}
	return flow
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/plugins/`
Expected: PASS (all five tests).

- [ ] **Step 5: Commit**

```bash
git add internal/plugins/
git commit -m "feat: add plugin.cfg ordering and request-flow builder"
```

---

### Task 3: Engine — enrich each server block with its flow

**Files:**
- Modify: `internal/engine/engine.go`
- Modify: `internal/engine/engine_test.go`

**Interfaces:**
- Consumes: `analyzer.Analyze` (existing), `plugins.BuildFlow` (Task 2), `model.ServerBlock.Flow` (Task 1), `validate.Validate` (existing).
- Produces: `engine.Run` now returns a `Result` whose every `ServerBlock` has `Flow` populated (execution order).

- [ ] **Step 1: Write the failing test**

Add to `internal/engine/engine_test.go`:

```go
func TestRunPopulatesFlowInExecutionOrder(t *testing.T) {
	// Declaration order is log then errors; plugin.cfg ranks errors before log.
	res := Run(". {\n    log\n    errors\n}\n")
	if res.Corefile == nil || len(res.Corefile.ServerBlocks) != 1 {
		t.Fatalf("corefile = %+v", res.Corefile)
	}
	flow := res.Corefile.ServerBlocks[0].Flow
	if len(flow) != 2 {
		t.Fatalf("flow = %+v, want 2 steps", flow)
	}
	if flow[0].Name != "errors" || flow[1].Name != "log" {
		t.Errorf("flow order = [%s %s], want [errors log]", flow[0].Name, flow[1].Name)
	}
	if !flow[0].Known || !flow[1].Known {
		t.Errorf("both steps should be Known: %+v", flow)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestRunPopulatesFlowInExecutionOrder`
Expected: FAIL — `flow = [], want 2 steps` (engine does not populate Flow yet).

- [ ] **Step 3: Implement the enrichment**

In `internal/engine/engine.go`, add the `plugins` import and set `Flow` on each server block before validating. The success path of `Run` becomes:

```go
	cf, err := analyzer.Analyze(input)
	if err != nil {
		return model.Result{
			Corefile: nil,
			Diagnostics: []model.Diagnostic{{
				Severity: model.SeverityError,
				Message:  err.Error(),
				Line:     0,
			}},
		}
	}
	for i := range cf.ServerBlocks {
		cf.ServerBlocks[i].Flow = plugins.BuildFlow(cf.ServerBlocks[i].Directives)
	}
	return model.Result{Corefile: cf, Diagnostics: validate.Validate(cf)}
```

Add to the import block:

```go
	"github.com/gtadi/corefile-visualizer/internal/plugins"
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/`
Expected: PASS — the new test plus the existing engine tests (`TestRunValid`, `TestRunParseErrorBecomesDiagnostic`).

- [ ] **Step 5: Commit**

```bash
git add internal/engine/
git commit -m "feat: enrich server blocks with request-flow in engine"
```

---

### Task 4: Frontend — RequestFlow view

**Files:**
- Modify: `web/src/lib/types.js`
- Create: `web/src/lib/RequestFlow.svelte`
- Create: `web/test/RequestFlow.test.js`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: the `Corefile` JSON shape with `serverBlocks[].flow` = array of `{ name, known }` (Tasks 1–3).
- Produces: `RequestFlow.svelte` with a `corefile` prop (`Corefile|null`); renders per-block chains and an empty state.

- [ ] **Step 1: Add the typedefs**

In `web/src/lib/types.js`, add a `FlowStep` typedef and a `flow` property on `ServerBlock`. Add this typedef block (before the `ServerBlock` typedef):

```js
/**
 * @typedef {Object} FlowStep
 * @property {string} name
 * @property {boolean} known
 */
```

And add the `flow` line to the existing `ServerBlock` typedef:

```js
/**
 * @typedef {Object} ServerBlock
 * @property {string[]} keys
 * @property {number} line
 * @property {Directive[]} directives
 * @property {FlowStep[]} [flow]
 */
```

- [ ] **Step 2: Write the failing test**

`web/test/RequestFlow.test.js`:

```js
import { render, screen } from '@testing-library/svelte'
import { describe, it, expect } from 'vitest'
import RequestFlow from '../src/lib/RequestFlow.svelte'

const corefile = {
  serverBlocks: [
    {
      keys: ['example.org:53'],
      line: 1,
      directives: [],
      flow: [
        { name: 'errors', known: true },
        { name: 'log', known: true },
        { name: 'customplugin', known: false },
      ],
    },
  ],
}

describe('RequestFlow', () => {
  it('renders flow steps in order', () => {
    render(RequestFlow, { corefile })
    const steps = screen.getAllByTestId('flow-step')
    expect(steps.map((s) => s.textContent.trim())).toEqual(['errors', 'log', 'customplugin'])
  })

  it('marks unknown steps', () => {
    render(RequestFlow, { corefile })
    const unknown = screen.getByText('customplugin')
    expect(unknown.className).toContain('unknown')
  })

  it('renders request and response endpoints', () => {
    render(RequestFlow, { corefile })
    expect(screen.getByText('request')).toBeInTheDocument()
    expect(screen.getByText('response')).toBeInTheDocument()
  })

  it('shows an empty state when corefile is null', () => {
    render(RequestFlow, { corefile: null })
    expect(screen.getByTestId('flow-empty')).toBeInTheDocument()
  })
})
```

- [ ] **Step 3: Run test to verify it fails**

Run from `web/`: `npm run test -- RequestFlow`
Expected: FAIL — cannot resolve `../src/lib/RequestFlow.svelte`.

- [ ] **Step 4: Write the component**

`web/src/lib/RequestFlow.svelte`:

```svelte
<script>
  /** @type {import('./types.js').Corefile|null} */
  export let corefile = null
</script>

{#if !corefile}
  <p data-testid="flow-empty">No Corefile parsed.</p>
{:else}
  <div class="flows">
    {#each corefile.serverBlocks as block}
      <section class="flow-block">
        <span class="keys">{block.keys.join(' ')}</span>
        <div class="chain">
          <span class="endpoint">request</span>
          {#each (block.flow ?? []) as step}
            <span class="arrow">→</span>
            <span class="step {step.known ? 'known' : 'unknown'}" data-testid="flow-step">{step.name}</span>
          {/each}
          <span class="arrow">→</span>
          <span class="endpoint">response</span>
        </div>
      </section>
    {/each}
  </div>
{/if}

<style>
  .chain {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem;
  }
  .step,
  .endpoint {
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    border: 1px solid currentColor;
    font-family: ui-monospace, monospace;
    font-size: 0.85rem;
  }
  .endpoint {
    opacity: 0.7;
  }
  .step.unknown {
    border-style: dashed;
    opacity: 0.7;
  }
  .arrow {
    opacity: 0.5;
  }
  .keys {
    font-weight: 600;
    display: block;
    margin-bottom: 0.3rem;
  }
  .flow-block {
    margin-bottom: 1rem;
  }
</style>
```

- [ ] **Step 5: Run test to verify it passes**

Run from `web/`: `npm run test -- RequestFlow`
Expected: PASS (4 tests).

- [ ] **Step 6: Wire it into App.svelte**

In `web/src/App.svelte`, add the import:

```js
  import RequestFlow from './lib/RequestFlow.svelte'
```

Inside the `<section class="views">`, below `<ValidationPanel .../>`, add:

```svelte
      <RequestFlow corefile={result?.corefile ?? null} />
```

- [ ] **Step 7: Run the full frontend suite and build**

Run from `web/`:
```bash
npm run test
npm run build
```
Expected: all Vitest tests pass (StructureTree, ValidationPanel, initialContent, RequestFlow); Vite build succeeds.

- [ ] **Step 8: Commit**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
git add web/src/ web/test/
git commit -m "feat: add request-flow diagram view"
```

---

### Task 5: Integration verification

**Files:** none (verification only).

**Interfaces:** Consumes the complete feature (Tasks 1–4).

- [ ] **Step 1: Run the full suites**

Run:
```bash
go test ./...
cd web && npm run test && npm run build
```
Expected: all Go tests pass (`model`, `plugins`, `engine`, `analyzer`, `validate`, `cliserver`, `webui`); all Vitest tests pass; Vite build succeeds.

- [ ] **Step 2: Verify flow is present in the WASM output end-to-end**

Run:
```bash
cd /Users/gtadi/workspace/corefile-visualizer
./scripts/build-wasm.sh
node -e '
const fs=require("fs"),path=require("path");
const dir="web/public/wasm";
new Function(fs.readFileSync(path.join(dir,"wasm_exec.js"),"utf8"))();
const go=new globalThis.Go();
WebAssembly.instantiate(fs.readFileSync(path.join(dir,"main.wasm")),go.importObject).then(({instance})=>{
  go.run(instance);
  const r=JSON.parse(globalThis.analyze(". {\n  log\n  errors\n}\n"));
  console.log("flow:", JSON.stringify(r.corefile.serverBlocks[0].flow));
});
'
```
Expected: `flow: [{"name":"errors","known":true},{"name":"log","known":true}]` — confirming the engine, WASM, and JSON contract carry the execution-ordered flow.

- [ ] **Step 3: Confirm no build artifacts are staged**

Run: `git status --short`
Expected: clean (no `web/public/wasm/*` binaries, no `web/dist`, no `node_modules`).

---

## Self-Review Notes

- **Spec coverage:** `internal/plugins` Order/Rank/BuildFlow with the pinned v1.12.0 list (Task 2); execution-order semantic + repeat-collapse + unknown-flagging (Task 2 tests); engine enrichment (Task 3); `ServerBlock.Flow`/`FlowStep` contract (Task 1); lightweight native Svelte view + App wiring (Task 4); `validate` untouched; end-to-end WASM check (Task 5).
- **Type/name consistency:** `FlowStep{Name,Known}` / `ServerBlock.Flow` / `plugins.Order` / `plugins.Rank` / `plugins.BuildFlow` are used identically across Tasks 1–3; the JSON names (`flow`, `name`, `known`) match between Task 1 (Go), Task 4 (`types.js`), and the component/test.
- **plugin.cfg list** is embedded verbatim from v1.12.0 (53 plugins) in Task 2; the doc comment cites the exact source ref.
- **Contract safety:** adding `Flow` (no `omitempty`) serializes as `"flow":null` for ServerBlocks the engine didn't build; `RequestFlow` guards with `block.flow ?? []`. The existing `TestResultJSONFieldNames` is unaffected (it marshals no ServerBlock).
