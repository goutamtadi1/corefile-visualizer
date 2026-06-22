# Plugin Reference Panel + Metadata + Unknown-plugin Lint + Theme (Phase 2b) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-plugin metadata (summary + doc link), a click-to-open plugin reference panel, an unknown-plugin validation warning, and a Kubernetes-blue + orange visual theme.

**Architecture:** `internal/plugins` gains an authoritative metadata map exposed to the frontend via a new one-shot `pluginCatalog()` WASM export. `validate` adds an unknown-plugin rule. The frontend adds a shared `selectedPlugin` store (set by clicking plugins in the tree or flow), a `PluginReference` view, and a theme applied through CSS custom properties.

**Tech Stack:** Go 1.26 (`syscall/js`, `embed` unaffected); existing Svelte + Vite + Vitest frontend (native CSS, no new dependency).

## Global Constraints

- Go module path: `github.com/gtadi/corefile-visualizer`; Go 1.26.
- Plugin metadata covers exactly the names in `plugins.Order` (53, pinned to CoreDNS v1.12.0) — a test enforces parity.
- Plugin summaries are the canonical coredns.io one-line descriptions (sourced, not invented).
- Doc URLs follow `https://coredns.io/plugins/<name>/`; the only plugin without a page is `on` (empty `DocURL`).
- Lint added this phase is **unknown-plugin** (warning), not ordering-violation.
- The catalog is delivered by a new `pluginCatalog()` WASM global; `analyze` is unchanged.
- No new frontend dependency; theme via CSS custom properties.
- Theme palette: Kubernetes blue `#326CE5` (dark `#2851B8`), orange `#ED8B00` (light `#F5A623`).
- Conventional commit messages; no Claude co-author.
- Branch: `phase-2b` (already created off `main`; the two-pane layout commit `81e398a` is already present).

## File Structure

```
internal/plugins/plugins.go        # (modify) add Meta type, meta map, Catalog()
internal/plugins/plugins_test.go   # (modify) parity + Catalog tests
cmd/wasm/main.go                   # (modify) register pluginCatalog() JS global
internal/validate/validate.go      # (modify) unknown-plugin rule
internal/validate/validate_test.go # (modify) unknown-plugin tests
web/src/lib/wasm.js                # (modify) loadPluginCatalog()
web/src/lib/selection.js           # (create) selectedPlugin store
web/src/lib/PluginReference.svelte # (create) reference panel
web/src/lib/Directives.svelte      # (modify) clickable plugin names
web/src/lib/RequestFlow.svelte     # (modify) clickable flow steps
web/src/App.svelte                 # (modify) load catalog, render PluginReference
web/test/PluginReference.test.js   # (create)
web/test/selection.test.js         # (create)
web/src/app.css                    # (modify) theme tokens + accents
```

**Scope:** plugin reference + metadata + unknown-plugin lint + theme. Per-plugin directive schemas remain deferred.

---

### Task 1: Plugin metadata + Catalog

**Files:**
- Modify: `internal/plugins/plugins.go`
- Modify: `internal/plugins/plugins_test.go`

**Interfaces:**
- Consumes: existing `plugins.Order`.
- Produces: `type plugins.Meta struct { Summary string (json:"summary"); DocURL string (json:"docUrl") }`; `func plugins.Catalog() map[string]Meta`. Consumed by Task 2 (WASM) and the frontend.

- [ ] **Step 1: Write the failing test**

Add to `internal/plugins/plugins_test.go`:

```go
func TestCatalogCoversOrderExactly(t *testing.T) {
	cat := Catalog()
	if len(cat) != len(Order) {
		t.Fatalf("catalog has %d entries, Order has %d", len(cat), len(Order))
	}
	for _, name := range Order {
		m, ok := cat[name]
		if !ok {
			t.Errorf("missing metadata for %q", name)
			continue
		}
		if m.Summary == "" {
			t.Errorf("empty summary for %q", name)
		}
	}
}

func TestCatalogDocURLPattern(t *testing.T) {
	cat := Catalog()
	for name, m := range cat {
		if m.DocURL == "" {
			continue // some plugins (e.g. "on") have no doc page
		}
		want := "https://coredns.io/plugins/" + name + "/"
		if m.DocURL != want {
			t.Errorf("docURL for %q = %q, want %q", name, m.DocURL, want)
		}
	}
}

func TestCatalogKnownPlugin(t *testing.T) {
	cat := Catalog()
	if cat["forward"].Summary == "" || cat["forward"].DocURL == "" {
		t.Errorf("forward metadata incomplete: %+v", cat["forward"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/plugins/ -run TestCatalog`
Expected: FAIL — `undefined: Catalog`.

- [ ] **Step 3: Add the Meta type, metadata map, and Catalog**

Add to `internal/plugins/plugins.go` (after the `BuildFlow` function). All summaries are the canonical coredns.io descriptions; `DocURL` is `https://coredns.io/plugins/<name>/` except `on` (no page):

```go
// Meta is reference metadata for a plugin: a one-line summary (from coredns.io)
// and a documentation URL (empty when the plugin has no coredns.io page).
type Meta struct {
	Summary string `json:"summary"`
	DocURL  string `json:"docUrl"`
}

func doc(name string) string { return "https://coredns.io/plugins/" + name + "/" }

var meta = map[string]Meta{
	"root":         {"specifies the root of where to find zone files.", doc("root")},
	"metadata":     {"enables a metadata collector.", doc("metadata")},
	"geoip":        {"looks up .mmdb databases using the client IP and adds geoip data to the request context.", doc("geoip")},
	"cancel":       {"cancels a request's context after 5001 milliseconds.", doc("cancel")},
	"tls":          {"configures the server certificates for the TLS, gRPC and DoH servers.", doc("tls")},
	"timeouts":     {"configures server read, write and idle timeouts for TCP, TLS, DoH and DoQ servers.", doc("timeouts")},
	"multisocket":  {"starts multiple servers that listen on one port.", doc("multisocket")},
	"reload":       {"allows automatic reload of a changed Corefile.", doc("reload")},
	"nsid":         {"adds an identifier of this server to each reply.", doc("nsid")},
	"bufsize":      {"limits the EDNS0 buffer size to prevent IP fragmentation.", doc("bufsize")},
	"bind":         {"overrides the host to which the server should bind.", doc("bind")},
	"debug":        {"disables automatic crash recovery so you get a stack trace.", doc("debug")},
	"trace":        {"enables OpenTracing-based tracing of DNS requests through the plugin chain.", doc("trace")},
	"ready":        {"enables a readiness check HTTP endpoint.", doc("ready")},
	"health":       {"enables a health check endpoint.", doc("health")},
	"pprof":        {"publishes runtime profiling data at endpoints under /debug/pprof.", doc("pprof")},
	"prometheus":   {"enables Prometheus metrics.", doc("prometheus")},
	"errors":       {"enables error logging.", doc("errors")},
	"log":          {"enables query logging to standard output.", doc("log")},
	"dnstap":       {"enables logging to dnstap.", doc("dnstap")},
	"local":        {"responds to local names.", doc("local")},
	"dns64":        {"enables the DNS64 IPv6 transition mechanism.", doc("dns64")},
	"acl":          {"enforces access control policies on the source IP.", doc("acl")},
	"any":          {"gives a minimal response to ANY queries.", doc("any")},
	"chaos":        {"responds to TXT queries in the CH class.", doc("chaos")},
	"loadbalance":  {"randomizes the order of A, AAAA and MX records.", doc("loadbalance")},
	"tsig":         {"defines TSIG keys and validates/signs TSIG requests and responses.", doc("tsig")},
	"cache":        {"enables a frontend cache.", doc("cache")},
	"rewrite":      {"performs internal message rewriting.", doc("rewrite")},
	"header":       {"modifies the header for queries and responses.", doc("header")},
	"dnssec":       {"enables on-the-fly DNSSEC signing of served data.", doc("dnssec")},
	"autopath":     {"allows for server-side search path completion.", doc("autopath")},
	"minimal":      {"minimizes the size of the DNS response message when possible.", doc("minimal")},
	"template":     {"allows for dynamic responses based on the incoming query.", doc("template")},
	"transfer":     {"performs outgoing zone transfers for other plugins.", doc("transfer")},
	"hosts":        {"enables serving zone data from an /etc/hosts style file.", doc("hosts")},
	"route53":      {"enables serving zone data from AWS Route 53.", doc("route53")},
	"azure":        {"enables serving zone data from Microsoft Azure DNS.", doc("azure")},
	"clouddns":     {"enables serving zone data from GCP Cloud DNS.", doc("clouddns")},
	"k8s_external": {"resolves load balancer and external IPs from outside Kubernetes clusters.", doc("k8s_external")},
	"kubernetes":   {"enables reading zone data from a Kubernetes cluster.", doc("kubernetes")},
	"file":         {"enables serving zone data from an RFC 1035-style master file.", doc("file")},
	"auto":         {"serves zone data from RFC 1035-style master files picked up automatically from disk.", doc("auto")},
	"secondary":    {"enables serving a zone retrieved from a primary server.", doc("secondary")},
	"etcd":         {"enables SkyDNS service discovery from etcd.", doc("etcd")},
	"loop":         {"detects simple forwarding loops and halts the server.", doc("loop")},
	"forward":      {"facilitates proxying DNS messages to upstream resolvers.", doc("forward")},
	"grpc":         {"proxies DNS messages to upstream resolvers via gRPC.", doc("grpc")},
	"erratic":      {"a plugin useful for testing client behavior.", doc("erratic")},
	"whoami":       {"returns the resolver's local IP address, port and transport.", doc("whoami")},
	"on":           {"executes shell commands on startup, shutdown and restart events.", ""},
	"sign":         {"adds DNSSEC records to zone files.", doc("sign")},
	"view":         {"defines conditions a DNS request must meet to be routed to the server block.", doc("view")},
}

// Catalog returns the plugin reference metadata keyed by plugin name. Callers
// must not mutate the returned map.
func Catalog() map[string]Meta { return meta }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/plugins/`
Expected: PASS — the three new tests plus the existing ones. (If `TestCatalogCoversOrderExactly` fails on a count mismatch, the `meta` map is missing or has an extra name vs `Order` — reconcile to exactly the 53 `Order` names.)

- [ ] **Step 5: Commit**

```bash
git add internal/plugins/
git commit -m "feat: add plugin reference metadata and Catalog"
```

---

### Task 2: WASM pluginCatalog() export

**Files:**
- Modify: `cmd/wasm/main.go`

**Interfaces:**
- Consumes: `plugins.Catalog()` (Task 1).
- Produces: a JS global `pluginCatalog()` returning the catalog as a JSON string. `analyze` is unchanged.

- [ ] **Step 1: Add the export**

In `cmd/wasm/main.go`, add the `plugins` import and a `pluginCatalog` function, and register it in `main`. Add to the import block:

```go
	"github.com/gtadi/corefile-visualizer/internal/plugins"
```

Add the function (next to `analyze`):

```go
func pluginCatalog(_ js.Value, _ []js.Value) any {
	b, err := json.Marshal(plugins.Catalog())
	if err != nil {
		return "{}"
	}
	return string(b)
}
```

In `main`, register it alongside `analyze`:

```go
	js.Global().Set("analyze", js.FuncOf(analyze))
	js.Global().Set("pluginCatalog", js.FuncOf(pluginCatalog))
	select {}
```

- [ ] **Step 2: Verify the WASM target compiles**

Run: `GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/wasm/`
Expected: exits 0.

- [ ] **Step 3: Verify end-to-end with Node**

Run:
```bash
cd /Users/gtadi/workspace/corefile-visualizer
./scripts/build-wasm.sh
node -e '
const fs=require("fs"),path=require("path"),dir="web/public/wasm";
new Function(fs.readFileSync(path.join(dir,"wasm_exec.js"),"utf8"))();
const go=new globalThis.Go();
WebAssembly.instantiate(fs.readFileSync(path.join(dir,"main.wasm")),go.importObject).then(({instance})=>{
  go.run(instance);
  const c=JSON.parse(globalThis.pluginCatalog());
  console.log("entries:", Object.keys(c).length, "forward:", JSON.stringify(c.forward));
});
'
```
Expected: `entries: 53 forward: {"summary":"facilitates proxying DNS messages to upstream resolvers.","docUrl":"https://coredns.io/plugins/forward/"}`.

- [ ] **Step 4: Commit**

```bash
git add cmd/wasm/main.go
git commit -m "feat: expose plugin catalog via WASM"
```

---

### Task 3: Unknown-plugin validation rule

**Files:**
- Modify: `internal/validate/validate.go`
- Modify: `internal/validate/validate_test.go`

**Interfaces:**
- Consumes: `plugins.Rank` (existing), `model` types.
- Produces: `Validate` now also emits a `warning` Diagnostic per unknown top-level plugin (deduped per server block + name). Existing rules unchanged.

- [ ] **Step 1: Write the failing test**

Add to `internal/validate/validate_test.go`:

```go
func TestValidateUnknownPlugin(t *testing.T) {
	cf := &model.Corefile{ServerBlocks: []model.ServerBlock{{
		Keys: []string{"."}, Line: 1,
		Directives: []model.Directive{
			{Name: "forward", Line: 2},
			{Name: "bogusplugin", Line: 3},
		},
	}}}
	got := Validate(cf)
	var unknown []model.Diagnostic
	for _, d := range got {
		if d.Line == 3 {
			unknown = append(unknown, d)
		}
	}
	if len(unknown) != 1 || unknown[0].Severity != model.SeverityWarning {
		t.Fatalf("want one warning at line 3 for bogusplugin, got %+v", got)
	}
}

func TestValidateUnknownPluginDedupedPerBlock(t *testing.T) {
	cf := &model.Corefile{ServerBlocks: []model.ServerBlock{{
		Keys: []string{"."}, Line: 1,
		Directives: []model.Directive{
			{Name: "bogus", Line: 2},
			{Name: "bogus", Line: 3},
		},
	}}}
	count := 0
	for _, d := range Validate(cf) {
		if d.Severity == model.SeverityWarning && d.Line == 2 {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one deduped warning for repeated unknown, got %d", count)
	}
}

func TestValidateKnownPluginNoWarning(t *testing.T) {
	cf := &model.Corefile{ServerBlocks: []model.ServerBlock{{
		Keys: []string{"."}, Line: 1,
		Directives: []model.Directive{{Name: "whoami", Line: 2}},
	}}}
	for _, d := range Validate(cf) {
		if d.Line == 2 {
			t.Errorf("known plugin should not warn: %+v", d)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/validate/ -run TestValidateUnknownPlugin`
Expected: FAIL — no warning emitted for the unknown plugin.

- [ ] **Step 3: Implement the rule**

In `internal/validate/validate.go`, add the `plugins` import and the unknown-plugin loop inside the per-server-block iteration. Add to imports:

```go
	"github.com/gtadi/corefile-visualizer/internal/plugins"
```

Inside the `for _, sb := range cf.ServerBlocks {` loop (after the existing empty-block check, before/after the key loop — placement among existing checks does not matter), add:

```go
		seenUnknown := map[string]bool{}
		for _, d := range sb.Directives {
			if _, ok := plugins.Rank(d.Name); ok {
				continue
			}
			if seenUnknown[d.Name] {
				continue
			}
			seenUnknown[d.Name] = true
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityWarning,
				Message:  fmt.Sprintf("unknown plugin %q — not a recognized CoreDNS plugin", d.Name),
				Line:     d.Line,
			})
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/validate/`
Expected: PASS — the three new tests plus existing `TestValidateClean`, `TestValidateDuplicateZone`, `TestValidateEmptyBlock`. (`TestValidateClean` uses `whoami`, a known plugin, so it stays clean.)

- [ ] **Step 5: Commit**

```bash
git add internal/validate/
git commit -m "feat: warn on unknown plugins in validation"
```

---

### Task 4: Frontend catalog loader + selection store

**Files:**
- Modify: `web/src/lib/wasm.js`
- Create: `web/src/lib/selection.js`
- Create: `web/test/selection.test.js`

**Interfaces:**
- Consumes: the `pluginCatalog()` WASM global (Task 2).
- Produces:
  - `loadPluginCatalog(): Record<string, {summary, docUrl}>` in `wasm.js` (throws if engine not loaded).
  - `selection.js` exporting `selectedPlugin` — a `svelte/store` writable (string name or null).

- [ ] **Step 1: Add the catalog loader**

In `web/src/lib/wasm.js`, add (after `analyzeCorefile`):

```js
/**
 * Returns the static plugin catalog from the WASM engine.
 * @returns {Record<string, {summary: string, docUrl: string}>}
 */
export function loadPluginCatalog() {
  if (typeof globalThis.pluginCatalog !== 'function') {
    throw new Error('WASM engine not loaded; call loadWasm() first')
  }
  return JSON.parse(globalThis.pluginCatalog())
}
```

- [ ] **Step 2: Write the failing test for the store**

`web/test/selection.test.js`:

```js
import { describe, it, expect, beforeEach } from 'vitest'
import { get } from 'svelte/store'
import { selectedPlugin } from '../src/lib/selection.js'

describe('selectedPlugin store', () => {
  beforeEach(() => selectedPlugin.set(null))

  it('defaults to null', () => {
    expect(get(selectedPlugin)).toBe(null)
  })

  it('updates when set', () => {
    selectedPlugin.set('forward')
    expect(get(selectedPlugin)).toBe('forward')
  })
})
```

- [ ] **Step 3: Run test to verify it fails**

Run from `web/`: `npm run test -- selection`
Expected: FAIL — cannot resolve `../src/lib/selection.js`.

- [ ] **Step 4: Create the store**

`web/src/lib/selection.js`:

```js
import { writable } from 'svelte/store'

/** The plugin name currently selected for the reference panel, or null. */
export const selectedPlugin = writable(null)
```

- [ ] **Step 5: Run test to verify it passes**

Run from `web/`: `npm run test -- selection`
Expected: PASS (2 tests).

- [ ] **Step 6: Commit**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
git add web/src/lib/wasm.js web/src/lib/selection.js web/test/selection.test.js
git commit -m "feat: add plugin catalog loader and selection store"
```

---

### Task 5: Clickable plugins + PluginReference panel + wiring

**Files:**
- Create: `web/src/lib/PluginReference.svelte`
- Create: `web/test/PluginReference.test.js`
- Modify: `web/src/lib/Directives.svelte`
- Modify: `web/src/lib/RequestFlow.svelte`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: `selectedPlugin` store (Task 4); `loadPluginCatalog` (Task 4); the catalog shape `{ name: {summary, docUrl} }`.
- Produces: `PluginReference.svelte` (prop `catalog`); clickable plugin names in tree and flow that set `selectedPlugin`.

- [ ] **Step 1: Write the failing test**

`web/test/PluginReference.test.js`:

```js
import { render, screen } from '@testing-library/svelte'
import { describe, it, expect, beforeEach } from 'vitest'
import PluginReference from '../src/lib/PluginReference.svelte'
import { selectedPlugin } from '../src/lib/selection.js'

const catalog = {
  forward: { summary: 'facilitates proxying DNS messages to upstream resolvers.', docUrl: 'https://coredns.io/plugins/forward/' },
  on: { summary: 'executes shell commands on lifecycle events.', docUrl: '' },
}

describe('PluginReference', () => {
  beforeEach(() => selectedPlugin.set(null))

  it('shows empty state when nothing selected', () => {
    render(PluginReference, { catalog })
    expect(screen.getByTestId('reference-empty')).toBeInTheDocument()
  })

  it('shows summary and doc link for a known plugin', () => {
    render(PluginReference, { catalog })
    selectedPlugin.set('forward')
    expect(screen.getByText(/proxying DNS messages/)).toBeInTheDocument()
    const link = screen.getByRole('link')
    expect(link.getAttribute('href')).toBe('https://coredns.io/plugins/forward/')
  })

  it('omits the link when docUrl is empty', () => {
    render(PluginReference, { catalog })
    selectedPlugin.set('on')
    expect(screen.getByText(/lifecycle events/)).toBeInTheDocument()
    expect(screen.queryByRole('link')).toBeNull()
  })

  it('shows unrecognized note for a plugin not in the catalog', () => {
    render(PluginReference, { catalog })
    selectedPlugin.set('bogus')
    expect(screen.getByTestId('reference-unknown')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run from `web/`: `npm run test -- PluginReference`
Expected: FAIL — cannot resolve `../src/lib/PluginReference.svelte`.

- [ ] **Step 3: Write the PluginReference component**

`web/src/lib/PluginReference.svelte`:

```svelte
<script>
  import { selectedPlugin } from './selection.js'
  /** @type {Record<string, {summary: string, docUrl: string}>} */
  export let catalog = {}
</script>

<div class="reference">
  <h2>Reference</h2>
  {#if $selectedPlugin === null}
    <p data-testid="reference-empty">Select a plugin in the tree or flow to see its reference.</p>
  {:else if catalog[$selectedPlugin]}
    <h3 class="plugin-name">{$selectedPlugin}</h3>
    <p class="summary">{catalog[$selectedPlugin].summary}</p>
    {#if catalog[$selectedPlugin].docUrl}
      <a class="doclink" href={catalog[$selectedPlugin].docUrl} target="_blank" rel="noopener noreferrer">Documentation ↗</a>
    {/if}
  {:else}
    <h3 class="plugin-name">{$selectedPlugin}</h3>
    <p class="unknown" data-testid="reference-unknown">Unrecognized plugin — not a known CoreDNS plugin.</p>
  {/if}
</div>

<style>
  .plugin-name {
    font-family: ui-monospace, monospace;
    margin: 0.2rem 0;
  }
  .doclink {
    font-size: 0.9rem;
  }
</style>
```

- [ ] **Step 4: Run test to verify it passes**

Run from `web/`: `npm run test -- PluginReference`
Expected: PASS (4 tests).

- [ ] **Step 5: Make tree plugin names clickable**

In `web/src/lib/Directives.svelte`, import the store and turn the plugin name into a button. Update the `<script>` to add:

```js
  import { selectedPlugin } from './selection.js'
```

Replace the name span `<span class="name">{d.name}</span>` with a clickable button:

```svelte
      <button type="button" class="name" on:click={() => selectedPlugin.set(d.name)}>{d.name}</button>
```

Add to the component's `<style>` (or create one) a reset so the button reads like the prior text:

```svelte
<style>
  .name {
    background: none;
    border: none;
    padding: 0;
    font: inherit;
    color: inherit;
    cursor: pointer;
    text-decoration: underline dotted;
  }
</style>
```

(The existing `StructureTree.test.js` uses `getByText('forward')` etc., which still matches the button's text content.)

- [ ] **Step 6: Make flow steps clickable**

In `web/src/lib/RequestFlow.svelte`, import the store and turn each step into a button (keeping `data-testid="flow-step"` and the `known`/`unknown` classes so existing tests pass). Add to `<script>`:

```js
  import { selectedPlugin } from './selection.js'
```

Replace the step span:

```svelte
            <button type="button" class="step {step.known ? 'known' : 'unknown'}" data-testid="flow-step" on:click={() => selectedPlugin.set(step.name)}>{step.name}</button>
```

In the component `<style>`, ensure `.step` resets button chrome (add these properties to the existing `.step` rule): `background: none; font: inherit; cursor: pointer;` (keep the existing border/padding/border-radius).

- [ ] **Step 7: Wire catalog + PluginReference into App.svelte**

In `web/src/App.svelte`:

Add imports:

```js
  import { loadWasm, analyzeCorefile, loadPluginCatalog } from './lib/wasm.js'
  import PluginReference from './lib/PluginReference.svelte'
```

(Replace the existing `import { loadWasm, analyzeCorefile } from './lib/wasm.js'` line with the three-name version above.)

Add catalog state near the other `let` declarations:

```js
  /** @type {Record<string, {summary: string, docUrl: string}>} */
  let catalog = {}
```

In `onMount`, after `loaded` is set, load the catalog. Replace the existing `onMount` body with:

```js
  onMount(async () => {
    const wasmReady = loadWasm()
      .then(() => {
        loaded = true
        try { catalog = loadPluginCatalog() } catch { /* catalog optional */ }
      })
      .catch(() => { /* wasm failed; editor still renders, analysis unavailable */ })
    initialDoc = await loadInitialCorefile(SAMPLE)
    await wasmReady
    if (loaded) runAnalysis(initialDoc)
  })
```

In the right-pane `.views` div, add `PluginReference` as the fourth view (after `<RequestFlow .../>`):

```svelte
        <PluginReference {catalog} />
```

- [ ] **Step 8: Run the full frontend suite and build**

Run from `web/`:
```bash
npm run test
npm run build
```
Expected: all Vitest tests pass (StructureTree, ValidationPanel, RequestFlow, initialContent, selection, PluginReference); build succeeds.

- [ ] **Step 9: Commit**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
git add web/src/ web/test/
git commit -m "feat: add clickable plugins and plugin reference panel"
```

---

### Task 6: Kubernetes-blue + orange theme

**Files:**
- Modify: `web/src/app.css`

**Interfaces:**
- Consumes: existing component class names (`.pane h2`, `.keys`, `.step.known`/`.step.unknown`, `.diag-error`/`.diag-warning`, `.name`, `.doclink`).
- Produces: a theme via CSS custom properties; presentation-only (no markup or logic changes).

This task is visual; verification is a clean build and the existing test suite still passing (no functional regression). Apply the palette through CSS variables so it is consistent and easy to tweak.

- [ ] **Step 1: Replace `web/src/app.css` with the themed stylesheet**

```css
:root {
  /* Kubernetes blue + orange palette */
  --k8s-blue: #326ce5;
  --k8s-blue-dark: #2851b8;
  --k8s-blue-soft: #e8f0fe;
  --orange: #ed8b00;
  --orange-light: #f5a623;
  --ink: #1a2233;

  font-family: system-ui, -apple-system, sans-serif;
  line-height: 1.5;
  color-scheme: light dark;
  color: var(--ink);
}

body {
  margin: 0;
  min-width: 320px;
  min-height: 100vh;
  background: linear-gradient(180deg, var(--k8s-blue-soft), transparent 240px);
}

h1 {
  color: var(--k8s-blue-dark);
  border-bottom: 3px solid var(--orange);
  padding-bottom: 0.3rem;
}

/* Pane section labels */
.pane h2 {
  color: var(--k8s-blue);
}

/* Structure tree: server block keys */
.keys {
  color: var(--k8s-blue-dark);
}

/* Tree plugin name links */
.name {
  color: var(--k8s-blue);
}
.name:hover {
  color: var(--orange);
}

/* Request-flow steps */
.step.known {
  border-color: var(--k8s-blue) !important;
  color: var(--k8s-blue-dark);
  background: var(--k8s-blue-soft) !important;
}
.step.unknown {
  border-color: var(--orange) !important;
  color: var(--orange);
}
.endpoint {
  border-color: var(--k8s-blue-dark) !important;
}

/* Validation severities */
.diag-error .sev { color: #c0392b; }
.diag-warning .sev { color: var(--orange); }

/* Reference doc link */
.doclink {
  color: var(--k8s-blue);
}
.doclink:hover {
  color: var(--orange);
}
```

(`--orange-light` and `--k8s-blue-soft` are defined for use in the rules above and future tweaks; every declaration here is valid CSS.)

- [ ] **Step 2: Verify build and no regressions**

Run from `web/`:
```bash
npm run build
npm run test
```
Expected: build succeeds with no CSS errors; all Vitest tests still pass (the theme is presentation-only).

Manual check (recommended): `npm run dev` — confirm the header has a blue title with an orange underline, known flow steps are blue, unknown steps are orange, and plugin links turn orange on hover.

- [ ] **Step 3: Commit**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
git add web/src/app.css
git commit -m "style: apply Kubernetes-blue and orange theme"
```

---

### Task 7: Integration verification

**Files:** none (verification only).

- [ ] **Step 1: Full suites**

Run:
```bash
go test ./...
cd web && npm run test && npm run build
```
Expected: all Go tests pass (`plugins`, `validate`, `model`, `engine`, `analyzer`, `cliserver`, `webui`); all Vitest tests pass; build succeeds.

- [ ] **Step 2: End-to-end WASM check (analyze + catalog + unknown lint)**

Run:
```bash
cd /Users/gtadi/workspace/corefile-visualizer
./scripts/build-wasm.sh
node -e '
const fs=require("fs"),path=require("path"),dir="web/public/wasm";
new Function(fs.readFileSync(path.join(dir,"wasm_exec.js"),"utf8"))();
const go=new globalThis.Go();
WebAssembly.instantiate(fs.readFileSync(path.join(dir,"main.wasm")),go.importObject).then(({instance})=>{
  go.run(instance);
  const cat=JSON.parse(globalThis.pluginCatalog());
  console.log("catalog entries:", Object.keys(cat).length);
  const r=JSON.parse(globalThis.analyze(". {\n  forward . 8.8.8.8\n  bogusplugin\n}\n"));
  console.log("diagnostics:", JSON.stringify(r.diagnostics));
});
'
```
Expected: `catalog entries: 53`; diagnostics include a warning `unknown plugin "bogusplugin" — not a recognized CoreDNS plugin`.

- [ ] **Step 3: Confirm no build artifacts staged**

Run: `git status --short`
Expected: clean.

---

## Self-Review Notes

- **Spec coverage:** metadata + `Catalog` (Task 1); `pluginCatalog()` WASM (Task 2); unknown-plugin lint (Task 3); catalog loader + selection store (Task 4); clickable tree/flow + reference panel + App wiring (Task 5); Kubernetes-blue/orange theme (Task 6); integration (Task 7). Directive schemas remain out of scope.
- **Type/name consistency:** `plugins.Meta{Summary,DocURL}` / `Catalog()` / `pluginCatalog()` / `loadPluginCatalog()` / `selectedPlugin` / `PluginReference` `catalog` prop are used identically across tasks. JSON keys `summary`/`docUrl` match Go tags (Task 1) and frontend usage (Tasks 4–5).
- **Existing-test safety:** tree/flow plugins become `<button>`s but keep their text and `data-testid`/classes, so `StructureTree.test.js` and `RequestFlow.test.js` still pass. `TestValidateClean` uses `whoami` (known) so the new lint doesn't break it.
- **Catalog parity** (Task 1 test) prevents metadata/Order drift. The Task 6 stylesheet is clean, valid CSS (no placeholders).
