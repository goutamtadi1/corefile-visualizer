# CoreDNS Corefile Visualizer — Phase 1 (MVP) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the end-to-end pipeline — paste a CoreDNS Corefile, parse it in-browser via a Go→WASM engine, and render a structure tree plus a validation panel in a Svelte app.

**Architecture:** A Go engine (compiled to WebAssembly) wraps the real CoreDNS parser (`github.com/coredns/caddy`) at the token level, builds an ordered structured model, runs basic validation, and exposes a single `analyze(text) → JSON` function to JavaScript. A Svelte/Vite app loads the WASM module, calls `analyze` on debounced editor input, and renders the returned model. Fully client-side; no backend.

**Tech Stack:** Go 1.26 (`GOOS=js GOARCH=wasm`), `github.com/coredns/caddy/caddyfile`, Svelte + Vite, CodeMirror 6, Vitest + @testing-library/svelte.

## Global Constraints

- Go module path: `github.com/gtadi/corefile-visualizer`.
- Go version floor: 1.26 (uses `lib/wasm/wasm_exec.js`).
- Parser dependency pinned: `github.com/coredns/caddy v1.1.4`.
- The engine MUST run fully client-side; no network calls, no backend, no telemetry.
- License: Apache-2.0 (file added in this phase).
- The JSON contract between WASM and the frontend is the `model.Result` shape; all views derive from it.
- WASM build output goes to `web/public/wasm/{main.wasm,wasm_exec.js}`.
- Commit after every task. Conventional commit messages. Do NOT add Claude as co-author.

## File Structure

```
go.mod, go.sum
LICENSE                        # Apache-2.0
.gitignore
cmd/wasm/main.go               # syscall/js glue → engine.Run → JSON
internal/
  model/model.go               # Corefile, ServerBlock, Directive, Diagnostic, Result
  model/model_test.go
  analyzer/analyzer.go         # token walker → model.Corefile
  analyzer/analyzer_test.go
  validate/validate.go         # basic semantic lint → []model.Diagnostic
  validate/validate_test.go
  engine/engine.go             # Run(text) → model.Result (analyze + validate compose)
  engine/engine_test.go
scripts/build-wasm.sh          # builds main.wasm + copies wasm_exec.js
web/                           # Svelte/Vite app
  package.json, vite.config.js, svelte.config.js
  index.html
  public/wasm/                 # build artifacts (gitignored except .gitkeep)
  src/
    lib/wasm.js                # WASM loader + analyze() wrapper
    lib/types.js               # JSDoc typedefs mirroring model.Result
    lib/StructureTree.svelte
    lib/ValidationPanel.svelte
    lib/Editor.svelte
    App.svelte
    main.js
  test/
    StructureTree.test.js
    ValidationPanel.test.js
```

**Phase boundaries:** This plan delivers Phase 1 only (engine + structure tree + validation panel). The request-flow diagram, plugin reference panel, and full plugin metadata registry are Phase 2; samples, shareable URLs, and GitHub Pages deploy are Phase 3. Each gets its own plan.

---

### Task 1: Project scaffold + model types

**Files:**
- Create: `go.mod`, `LICENSE`, `.gitignore`
- Create: `internal/model/model.go`
- Test: `internal/model/model_test.go`

**Interfaces:**
- Produces: the `model` package — types `Corefile`, `ServerBlock`, `Directive`, `Severity` (+ constants `SeverityError`, `SeverityWarning`, `SeverityInfo`), `Diagnostic`, `Result`. JSON field names are the contract used by every later task and the frontend.

- [ ] **Step 1: Initialize the Go module and supporting files**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
go mod init github.com/gtadi/corefile-visualizer
go get github.com/coredns/caddy@v1.1.4
```

Create `.gitignore`:

```gitignore
# Go build cache (worktree-local, if used)
.gocache/
.gomodcache/

# WASM build artifacts
web/public/wasm/main.wasm
web/public/wasm/wasm_exec.js

# Node
web/node_modules/
web/dist/
```

Create `LICENSE` with the standard Apache License 2.0 text (full text from https://www.apache.org/licenses/LICENSE-2.0.txt; copyright line: `Copyright 2026 corefile-visualizer contributors`).

- [ ] **Step 2: Write the failing test**

`internal/model/model_test.go`:

```go
package model

import (
	"encoding/json"
	"testing"
)

func TestResultJSONRoundTrip(t *testing.T) {
	in := Result{
		Corefile: &Corefile{
			ServerBlocks: []ServerBlock{{
				Keys: []string{"example.org:53"},
				Line: 1,
				Directives: []Directive{
					{Name: "forward", Args: []string{".", "8.8.8.8"}, Line: 2},
					{Name: "cache", Line: 3, Block: []Directive{
						{Name: "success", Args: []string{"5000"}, Line: 4},
					}},
				},
			}},
		},
		Diagnostics: []Diagnostic{
			{Severity: SeverityWarning, Message: "duplicate zone", Line: 1},
		},
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Result
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Corefile.ServerBlocks[0].Keys[0] != "example.org:53" {
		t.Errorf("key = %q", out.Corefile.ServerBlocks[0].Keys[0])
	}
	if out.Corefile.ServerBlocks[0].Directives[1].Block[0].Name != "success" {
		t.Errorf("nested directive lost: %+v", out.Corefile.ServerBlocks[0].Directives[1])
	}
	if out.Diagnostics[0].Severity != SeverityWarning {
		t.Errorf("severity = %q", out.Diagnostics[0].Severity)
	}
}

func TestResultJSONFieldNames(t *testing.T) {
	b, _ := json.Marshal(Result{Corefile: &Corefile{}, Diagnostics: []Diagnostic{}})
	got := string(b)
	want := `{"corefile":{"serverBlocks":null},"diagnostics":[]}`
	if got != want {
		t.Errorf("contract changed:\n got=%s\nwant=%s", got, want)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/model/`
Expected: FAIL — `undefined: Result` (package has no types yet).

- [ ] **Step 4: Write the implementation**

`internal/model/model.go`:

```go
// Package model defines the structured, JSON-serializable representation of a
// parsed CoreDNS Corefile. It is the contract between the WASM engine and the
// frontend; JSON field names must not change without updating the frontend.
package model

// Corefile is the top-level parsed document.
type Corefile struct {
	ServerBlocks []ServerBlock `json:"serverBlocks"`
}

// ServerBlock associates one or more keys (zones/addresses) with an ordered
// list of directives (plugins).
type ServerBlock struct {
	Keys       []string    `json:"keys"`
	Line       int         `json:"line"`
	Directives []Directive `json:"directives"`
}

// Directive is a single plugin invocation, preserving declaration order, its
// arguments, and any nested block.
type Directive struct {
	Name  string      `json:"name"`
	Args  []string    `json:"args,omitempty"`
	Line  int         `json:"line"`
	Block []Directive `json:"block,omitempty"`
}

// Severity classifies a diagnostic.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Diagnostic is a single validation finding.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Line     int      `json:"line"`
}

// Result is the full output of the engine for one Corefile input.
type Result struct {
	Corefile    *Corefile    `json:"corefile"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/model/`
Expected: PASS (both tests).

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum LICENSE .gitignore internal/model/
git commit -m "feat: add Go module, license, and core model types"
```

---

### Task 2: Analyzer — token walker

**Files:**
- Create: `internal/analyzer/analyzer.go`
- Test: `internal/analyzer/analyzer_test.go`

**Interfaces:**
- Consumes: `model.Corefile`, `model.ServerBlock`, `model.Directive` from Task 1; `caddyfile.NewDispenser` from `github.com/coredns/caddy/caddyfile`.
- Produces: `func Analyze(input string) (*model.Corefile, error)`. Preserves directive order and repeated directives; captures line numbers. Returns a sentinel error on a server block missing its closing `}`.

**Background (verified against caddy v1.1.4):** `caddyfile.NewDispenser("Corefile", reader)` yields a flat, ordered token stream via `Next()/Val()/Line()`, where `{` and `}` are individual tokens carrying their source line. The walker reconstructs structure from that stream. (The higher-level `caddyfile.Parse` is NOT used: it returns directives in a `map[string][]Token`, losing order and merging repeated directives.)

- [ ] **Step 1: Write the failing test**

`internal/analyzer/analyzer_test.go`:

```go
package analyzer

import (
	"testing"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

func TestAnalyzeOrderAndNesting(t *testing.T) {
	in := `example.org:53 {
    log
    errors
    file db.example.org
    cache {
        success 5000
    }
}

. {
    forward . 8.8.8.8
}
`
	cf, err := Analyze(in)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(cf.ServerBlocks) != 2 {
		t.Fatalf("server blocks = %d, want 2", len(cf.ServerBlocks))
	}

	b0 := cf.ServerBlocks[0]
	if b0.Keys[0] != "example.org:53" || b0.Line != 1 {
		t.Errorf("block0 keys/line = %v / %d", b0.Keys, b0.Line)
	}
	gotOrder := []string{}
	for _, d := range b0.Directives {
		gotOrder = append(gotOrder, d.Name)
	}
	wantOrder := []string{"log", "errors", "file", "cache"}
	if len(gotOrder) != len(wantOrder) {
		t.Fatalf("directive order = %v, want %v", gotOrder, wantOrder)
	}
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("directive order = %v, want %v", gotOrder, wantOrder)
		}
	}

	file := b0.Directives[2]
	if len(file.Args) != 1 || file.Args[0] != "db.example.org" || file.Line != 4 {
		t.Errorf("file directive = %+v", file)
	}

	cache := b0.Directives[3]
	if len(cache.Block) != 1 || cache.Block[0].Name != "success" || cache.Block[0].Args[0] != "5000" {
		t.Errorf("cache block = %+v", cache.Block)
	}

	fwd := cf.ServerBlocks[1].Directives[0]
	if fwd.Name != "forward" || len(fwd.Args) != 2 || fwd.Args[1] != "8.8.8.8" {
		t.Errorf("forward = %+v", fwd)
	}
}

func TestAnalyzeRepeatedDirectives(t *testing.T) {
	in := `. {
    file a.db
    file b.db
}
`
	cf, err := Analyze(in)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	dirs := cf.ServerBlocks[0].Directives
	if len(dirs) != 2 {
		t.Fatalf("directives = %d, want 2 (repeats preserved)", len(dirs))
	}
	if dirs[0].Args[0] != "a.db" || dirs[1].Args[0] != "b.db" {
		t.Errorf("repeated directives = %+v", dirs)
	}
}

func TestAnalyzeMissingCloseBrace(t *testing.T) {
	_, err := Analyze(". {\n    forward . 8.8.8.8\n")
	if err == nil {
		t.Fatal("expected error for missing closing brace")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/analyzer/`
Expected: FAIL — `undefined: Analyze`.

- [ ] **Step 3: Write the implementation**

`internal/analyzer/analyzer.go`:

```go
// Package analyzer parses a CoreDNS Corefile into the structured model. It
// walks the caddy tokenizer's flat, ordered token stream so that directive
// order and repeated directives are preserved (unlike caddyfile.Parse, which
// groups directives into an order-losing map).
package analyzer

import (
	"errors"
	"strings"

	"github.com/coredns/caddy/caddyfile"
	"github.com/gtadi/corefile-visualizer/internal/model"
)

// ErrMissingCloseBrace is returned when a server block or nested block is not
// closed before end of input.
var ErrMissingCloseBrace = errors.New("missing closing brace '}'")

// ErrMissingOpenBrace is returned when server-block keys are not followed by '{'.
var ErrMissingOpenBrace = errors.New("missing opening brace '{'")

type token struct {
	text string
	line int
}

type parser struct {
	tokens []token
	pos    int
}

// Analyze parses the Corefile text into a *model.Corefile.
func Analyze(input string) (*model.Corefile, error) {
	d := caddyfile.NewDispenser("Corefile", strings.NewReader(input))
	var ts []token
	for d.Next() {
		ts = append(ts, token{text: d.Val(), line: d.Line()})
	}

	p := &parser{tokens: ts}
	cf := &model.Corefile{}
	for p.pos < len(p.tokens) {
		sb, err := p.parseServerBlock()
		if err != nil {
			return nil, err
		}
		cf.ServerBlocks = append(cf.ServerBlocks, *sb)
	}
	return cf, nil
}

func (p *parser) parseServerBlock() (*model.ServerBlock, error) {
	sb := &model.ServerBlock{Line: p.tokens[p.pos].line}
	for p.pos < len(p.tokens) && p.tokens[p.pos].text != "{" {
		sb.Keys = append(sb.Keys, p.tokens[p.pos].text)
		p.pos++
	}
	if p.pos >= len(p.tokens) {
		return nil, ErrMissingOpenBrace
	}
	p.pos++ // consume "{"

	dirs, err := p.parseDirectives()
	if err != nil {
		return nil, err
	}
	sb.Directives = dirs
	return sb, nil
}

// parseDirectives consumes directives until a matching "}" (which it consumes).
func (p *parser) parseDirectives() ([]model.Directive, error) {
	dirs := []model.Directive{}
	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		if t.text == "}" {
			p.pos++
			return dirs, nil
		}

		d := model.Directive{Name: t.text, Line: t.line}
		p.pos++

		for p.pos < len(p.tokens) {
			n := p.tokens[p.pos]
			if n.text == "}" {
				break
			}
			if n.text == "{" {
				p.pos++ // consume "{"
				blk, err := p.parseDirectives()
				if err != nil {
					return nil, err
				}
				d.Block = blk
				break
			}
			if n.line != t.line {
				break // next directive begins on a new line
			}
			d.Args = append(d.Args, n.text)
			p.pos++
		}
		dirs = append(dirs, d)
	}
	return nil, ErrMissingCloseBrace
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/analyzer/`
Expected: PASS (all three tests).

- [ ] **Step 5: Commit**

```bash
git add internal/analyzer/
git commit -m "feat: add Corefile token-walking analyzer"
```

---

### Task 3: Basic validation

**Files:**
- Create: `internal/validate/validate.go`
- Test: `internal/validate/validate_test.go`

**Interfaces:**
- Consumes: `model.Corefile`, `model.Diagnostic`, `model.Severity` constants from Task 1.
- Produces: `func Validate(cf *model.Corefile) []model.Diagnostic`. Always returns non-nil (empty slice when clean). Phase 1 rules: duplicate zone key across server blocks (warning), empty server block (warning). Plugin-aware lint is Phase 2.

- [ ] **Step 1: Write the failing test**

`internal/validate/validate_test.go`:

```go
package validate

import (
	"testing"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

func TestValidateClean(t *testing.T) {
	cf := &model.Corefile{ServerBlocks: []model.ServerBlock{{
		Keys: []string{"example.org:53"}, Line: 1,
		Directives: []model.Directive{{Name: "whoami", Line: 2}},
	}}}
	got := Validate(cf)
	if got == nil {
		t.Fatal("Validate returned nil; want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Errorf("diagnostics = %+v, want none", got)
	}
}

func TestValidateDuplicateZone(t *testing.T) {
	cf := &model.Corefile{ServerBlocks: []model.ServerBlock{
		{Keys: []string{"example.org:53"}, Line: 1, Directives: []model.Directive{{Name: "whoami", Line: 2}}},
		{Keys: []string{"example.org:53"}, Line: 5, Directives: []model.Directive{{Name: "whoami", Line: 6}}},
	}}
	got := Validate(cf)
	if len(got) != 1 || got[0].Severity != model.SeverityWarning || got[0].Line != 5 {
		t.Fatalf("diagnostics = %+v, want one warning at line 5", got)
	}
}

func TestValidateEmptyBlock(t *testing.T) {
	cf := &model.Corefile{ServerBlocks: []model.ServerBlock{
		{Keys: []string{"."}, Line: 1, Directives: []model.Directive{}},
	}}
	got := Validate(cf)
	if len(got) != 1 || got[0].Severity != model.SeverityWarning {
		t.Fatalf("diagnostics = %+v, want one empty-block warning", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/validate/`
Expected: FAIL — `undefined: Validate`.

- [ ] **Step 3: Write the implementation**

`internal/validate/validate.go`:

```go
// Package validate runs basic semantic lint over a parsed Corefile. Phase 1
// covers structural checks (duplicate zones, empty blocks); plugin-aware rules
// arrive in Phase 2.
package validate

import (
	"fmt"
	"strings"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

// Validate returns diagnostics for the given Corefile. The result is always
// non-nil (empty when there are no findings).
func Validate(cf *model.Corefile) []model.Diagnostic {
	diags := []model.Diagnostic{}
	if cf == nil {
		return diags
	}

	seen := map[string]bool{}
	for _, sb := range cf.ServerBlocks {
		if len(sb.Directives) == 0 {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityWarning,
				Message:  fmt.Sprintf("server block %q has no plugins", strings.Join(sb.Keys, " ")),
				Line:     sb.Line,
			})
		}
		for _, key := range sb.Keys {
			if seen[key] {
				diags = append(diags, model.Diagnostic{
					Severity: model.SeverityWarning,
					Message:  fmt.Sprintf("duplicate zone %q declared in more than one server block", key),
					Line:     sb.Line,
				})
			}
			seen[key] = true
		}
	}
	return diags
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/validate/`
Expected: PASS (all three tests).

- [ ] **Step 5: Commit**

```bash
git add internal/validate/
git commit -m "feat: add basic Corefile validation rules"
```

---

### Task 4: Engine composition + WASM entrypoint + build script

**Files:**
- Create: `internal/engine/engine.go`
- Test: `internal/engine/engine_test.go`
- Create: `cmd/wasm/main.go`
- Create: `scripts/build-wasm.sh`
- Create: `web/public/wasm/.gitkeep`

**Interfaces:**
- Consumes: `analyzer.Analyze`, `analyzer.ErrMissingCloseBrace`/`ErrMissingOpenBrace` (Task 2); `validate.Validate` (Task 3); `model.Result`/`model.Diagnostic` (Task 1).
- Produces: `func engine.Run(input string) model.Result` — composes parse + validate; on parse error returns a `Result` with `Corefile: nil` and a single error `Diagnostic`. The WASM glue exposes JS global `analyze(text) → JSON string of model.Result`.

- [ ] **Step 1: Write the failing test**

`internal/engine/engine_test.go`:

```go
package engine

import (
	"testing"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

func TestRunValid(t *testing.T) {
	res := Run(". {\n    whoami\n}\n")
	if res.Corefile == nil || len(res.Corefile.ServerBlocks) != 1 {
		t.Fatalf("corefile = %+v", res.Corefile)
	}
	if len(res.Diagnostics) != 0 {
		t.Errorf("diagnostics = %+v, want none", res.Diagnostics)
	}
}

func TestRunParseErrorBecomesDiagnostic(t *testing.T) {
	res := Run(". {\n    whoami\n") // missing closing brace
	if res.Corefile != nil {
		t.Errorf("corefile = %+v, want nil on parse error", res.Corefile)
	}
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Severity != model.SeverityError {
		t.Fatalf("diagnostics = %+v, want one error", res.Diagnostics)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/`
Expected: FAIL — `undefined: Run`.

- [ ] **Step 3: Write the engine implementation**

`internal/engine/engine.go`:

```go
// Package engine composes parsing and validation into a single Result. It is
// the platform-independent core called by the WASM entrypoint, kept separate
// so it can be unit-tested without a js/wasm build.
package engine

import (
	"github.com/gtadi/corefile-visualizer/internal/analyzer"
	"github.com/gtadi/corefile-visualizer/internal/model"
	"github.com/gtadi/corefile-visualizer/internal/validate"
)

// Run parses and validates the Corefile text. On a parse error, Corefile is nil
// and Diagnostics holds a single error describing the failure.
func Run(input string) model.Result {
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
	return model.Result{Corefile: cf, Diagnostics: validate.Validate(cf)}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/`
Expected: PASS (both tests).

- [ ] **Step 5: Write the WASM glue**

`cmd/wasm/main.go`:

```go
//go:build js && wasm

// Command wasm is the browser entrypoint. It registers a global JS function
// analyze(text) that returns the JSON-encoded engine result.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/gtadi/corefile-visualizer/internal/engine"
	"github.com/gtadi/corefile-visualizer/internal/model"
)

func analyze(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON("analyze: missing input argument")
	}
	res := engine.Run(args[0].String())
	b, err := json.Marshal(res)
	if err != nil {
		return errorJSON("analyze: " + err.Error())
	}
	return string(b)
}

func errorJSON(msg string) string {
	b, _ := json.Marshal(model.Result{
		Diagnostics: []model.Diagnostic{{Severity: model.SeverityError, Message: msg}},
	})
	return string(b)
}

func main() {
	js.Global().Set("analyze", js.FuncOf(analyze))
	select {} // keep the Go runtime alive for JS calls
}
```

- [ ] **Step 6: Verify the WASM target compiles**

Run: `GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/wasm/`
Expected: exits 0, no output.

- [ ] **Step 7: Write the build script**

`scripts/build-wasm.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Builds the WASM engine and copies the Go JS support file into the web app.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/web/public/wasm"
mkdir -p "$OUT"

GOOS=js GOARCH=wasm go build -o "$OUT/main.wasm" ./cmd/wasm/
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$OUT/wasm_exec.js"

echo "Built $OUT/main.wasm and copied wasm_exec.js"
```

Then:

```bash
chmod +x scripts/build-wasm.sh
touch web/public/wasm/.gitkeep
./scripts/build-wasm.sh
```

Expected: prints the "Built ..." line; `web/public/wasm/main.wasm` and `wasm_exec.js` exist (both gitignored).

- [ ] **Step 8: Commit**

```bash
git add internal/engine/ cmd/wasm/ scripts/build-wasm.sh web/public/wasm/.gitkeep
git commit -m "feat: add engine composition, WASM entrypoint, and build script"
```

---

### Task 5: Svelte/Vite scaffold + WASM loader

**Files:**
- Create: `web/package.json`, `web/vite.config.js`, `web/svelte.config.js`, `web/index.html`, `web/src/main.js`, `web/src/App.svelte`
- Create: `web/src/lib/types.js`, `web/src/lib/wasm.js`

**Interfaces:**
- Consumes: the `analyze` JS global registered by `web/public/wasm/main.wasm` (Task 4); the `model.Result` JSON shape (Task 1).
- Produces: `web/src/lib/wasm.js` exporting `async function loadWasm(): Promise<void>` and `function analyzeCorefile(text: string): Result` (throws if WASM not loaded). `web/src/lib/types.js` provides JSDoc typedefs (`Result`, `Corefile`, `ServerBlock`, `Directive`, `Diagnostic`) used by component tests and editor wiring.

- [ ] **Step 1: Scaffold the Vite + Svelte app**

```bash
cd /Users/gtadi/workspace/corefile-visualizer/web
npm create vite@latest . -- --template svelte
npm install
npm install --save-dev vitest @testing-library/svelte @testing-library/jest-dom jsdom
npm install codemirror @codemirror/view @codemirror/state
```

If `npm create` refuses because the directory is non-empty (the `public/wasm` dir exists), scaffold in a temp dir and copy: `npm create vite@latest ../.vite-tmp -- --template svelte && cp -r ../.vite-tmp/* . && cp ../.vite-tmp/.gitignore . 2>/dev/null; rm -rf ../.vite-tmp`. Keep the existing `public/wasm/` contents.

- [ ] **Step 2: Configure Vitest**

Overwrite `web/vite.config.js`:

```js
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  base: './',
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./test/setup.js'],
  },
})
```

Create `web/test/setup.js`:

```js
import '@testing-library/jest-dom'
```

Add a `test` script to `web/package.json` `"scripts"`: `"test": "vitest run"`.

- [ ] **Step 3: Write the JSDoc typedefs**

`web/src/lib/types.js`:

```js
/**
 * @typedef {Object} Directive
 * @property {string} name
 * @property {string[]} [args]
 * @property {number} line
 * @property {Directive[]} [block]
 */

/**
 * @typedef {Object} ServerBlock
 * @property {string[]} keys
 * @property {number} line
 * @property {Directive[]} directives
 */

/**
 * @typedef {Object} Corefile
 * @property {ServerBlock[]} serverBlocks
 */

/**
 * @typedef {Object} Diagnostic
 * @property {"error"|"warning"|"info"} severity
 * @property {string} message
 * @property {number} line
 */

/**
 * @typedef {Object} Result
 * @property {Corefile|null} corefile
 * @property {Diagnostic[]} diagnostics
 */

export {}
```

- [ ] **Step 4: Write the WASM loader**

`web/src/lib/wasm.js`:

```js
/** @typedef {import('./types.js').Result} Result */

let ready = false

/** Loads wasm_exec.js and instantiates the engine, registering global analyze(). */
export async function loadWasm() {
  if (ready) return
  // wasm_exec.js defines globalThis.Go and is plain (non-module) script text.
  const execSrc = await (await fetch(`${import.meta.env.BASE_URL}wasm/wasm_exec.js`)).text()
  // eslint-disable-next-line no-new-func
  new Function(execSrc)()
  const go = new globalThis.Go()
  const resp = await fetch(`${import.meta.env.BASE_URL}wasm/main.wasm`)
  const { instance } = await WebAssembly.instantiateStreaming(resp, go.importObject)
  go.run(instance) // runs forever (select{}); registers globalThis.analyze
  ready = true
}

/**
 * Analyzes Corefile text via the WASM engine.
 * @param {string} text
 * @returns {Result}
 */
export function analyzeCorefile(text) {
  if (typeof globalThis.analyze !== 'function') {
    throw new Error('WASM engine not loaded; call loadWasm() first')
  }
  return JSON.parse(globalThis.analyze(text))
}
```

- [ ] **Step 5: Minimal App shell to verify load**

Overwrite `web/src/App.svelte`:

```svelte
<script>
  import { onMount } from 'svelte'
  import { loadWasm, analyzeCorefile } from './lib/wasm.js'

  let status = 'loading…'
  onMount(async () => {
    try {
      await loadWasm()
      const r = analyzeCorefile('. {\n    whoami\n}\n')
      status = `engine ready — ${r.corefile.serverBlocks.length} block(s)`
    } catch (e) {
      status = `error: ${e.message}`
    }
  })
</script>

<main>
  <h1>CoreDNS Corefile Visualizer</h1>
  <p data-testid="status">{status}</p>
</main>
```

- [ ] **Step 6: Verify build and dev server**

Run from `web/`:
```bash
../scripts/build-wasm.sh   # ensure wasm artifacts are present
npm run build
```
Expected: Vite build succeeds, `web/dist/` produced.

Manual check (optional but recommended): `npm run dev`, open the URL, confirm the page shows `engine ready — 1 block(s)`.

- [ ] **Step 7: Commit**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
git add web/
git commit -m "feat: scaffold Svelte/Vite app with WASM loader"
```

---

### Task 6: Editor component + file upload, wired to the engine

**Files:**
- Create: `web/src/lib/Editor.svelte`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: `analyzeCorefile`/`loadWasm` from `web/src/lib/wasm.js` (Task 5).
- Produces: `Editor.svelte` — a CodeMirror-backed editor with a `value` prop (two-way via `bind:value` is not used; it dispatches a `change` event with `{ detail: string }`) and a file-upload input that loads a `.Corefile`/text file into the editor. `App.svelte` owns the `result` state (the `Result` object) by calling `analyzeCorefile` on debounced changes; later tasks read `result` to render views.

- [ ] **Step 1: Write the Editor component**

`web/src/lib/Editor.svelte`:

```svelte
<script>
  import { onMount, createEventDispatcher } from 'svelte'
  import { EditorView, keymap } from '@codemirror/view'
  import { defaultKeymap } from '@codemirror/commands'
  import { EditorState } from '@codemirror/state'

  export let value = ''
  const dispatch = createEventDispatcher()

  let host
  let view

  onMount(() => {
    view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: value,
        extensions: [
          keymap.of(defaultKeymap),
          EditorView.updateListener.of((u) => {
            if (u.docChanged) dispatch('change', u.state.doc.toString())
          }),
        ],
      }),
    })
    return () => view?.destroy()
  })

  function setDoc(text) {
    view?.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } })
  }

  async function onFile(e) {
    const file = e.target.files?.[0]
    if (!file) return
    const text = await file.text()
    setDoc(text)
    dispatch('change', text)
  }
</script>

<div class="editor">
  <label class="upload">
    Upload Corefile
    <input type="file" accept=".Corefile,.conf,text/plain" on:change={onFile} />
  </label>
  <div class="cm" bind:this={host} data-testid="editor"></div>
</div>
```

`@codemirror/commands` is a transitive dep of the `codemirror` meta-package; if the import fails, run `npm install @codemirror/commands` in `web/`.

- [ ] **Step 2: Wire Editor into App with debounced analysis**

Overwrite `web/src/App.svelte`:

```svelte
<script>
  import { onMount } from 'svelte'
  import { loadWasm, analyzeCorefile } from './lib/wasm.js'
  import Editor from './lib/Editor.svelte'

  const SAMPLE = `example.org:53 {
    log
    errors
    forward . 8.8.8.8
    cache {
        success 5000
    }
}

. {
    whoami
}
`

  /** @type {import('./lib/types.js').Result|null} */
  let result = null
  let loaded = false
  let timer

  onMount(async () => {
    await loadWasm()
    loaded = true
    runAnalysis(SAMPLE)
  })

  function runAnalysis(text) {
    if (!loaded) return
    result = analyzeCorefile(text)
  }

  function onChange(e) {
    clearTimeout(timer)
    timer = setTimeout(() => runAnalysis(e.detail), 250)
  }
</script>

<main>
  <h1>CoreDNS Corefile Visualizer</h1>
  <div class="layout">
    <Editor value={SAMPLE} on:change={onChange} />
    <section data-testid="result-summary">
      {#if result?.corefile}
        {result.corefile.serverBlocks.length} server block(s),
        {result.diagnostics.length} diagnostic(s)
      {:else if result}
        parse error
      {:else}
        analyzing…
      {/if}
    </section>
  </div>
</main>
```

- [ ] **Step 3: Verify the build still succeeds**

Run from `web/`: `npm run build`
Expected: build succeeds.

Manual check (recommended): `npm run dev` shows the editor with the sample and "2 server block(s), 0 diagnostic(s)".

- [ ] **Step 4: Commit**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
git add web/src/
git commit -m "feat: add CodeMirror editor with file upload wired to engine"
```

---

### Task 7: Structure tree view

**Files:**
- Create: `web/src/lib/StructureTree.svelte`
- Create: `web/test/StructureTree.test.js`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: the `Corefile` shape (`serverBlocks → keys/directives → name/args/block`) from Task 1's JSON contract.
- Produces: `StructureTree.svelte` with a `corefile` prop (`Corefile|null`); renders each server block's keys and its directives in order, recursing into nested `block` directives.

- [ ] **Step 1: Write the failing test**

`web/test/StructureTree.test.js`:

```js
import { render, screen } from '@testing-library/svelte'
import { describe, it, expect } from 'vitest'
import StructureTree from '../src/lib/StructureTree.svelte'

const corefile = {
  serverBlocks: [
    {
      keys: ['example.org:53'],
      line: 1,
      directives: [
        { name: 'forward', args: ['.', '8.8.8.8'], line: 2 },
        { name: 'cache', line: 3, block: [{ name: 'success', args: ['5000'], line: 4 }] },
      ],
    },
  ],
}

describe('StructureTree', () => {
  it('renders server block keys', () => {
    render(StructureTree, { corefile })
    expect(screen.getByText('example.org:53')).toBeInTheDocument()
  })

  it('renders directives in order with args', () => {
    render(StructureTree, { corefile })
    expect(screen.getByText('forward')).toBeInTheDocument()
    expect(screen.getByText('. 8.8.8.8')).toBeInTheDocument()
  })

  it('renders nested block directives', () => {
    render(StructureTree, { corefile })
    expect(screen.getByText('success')).toBeInTheDocument()
  })

  it('shows an empty state when corefile is null', () => {
    render(StructureTree, { corefile: null })
    expect(screen.getByTestId('tree-empty')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run from `web/`: `npm run test -- StructureTree`
Expected: FAIL — cannot resolve `../src/lib/StructureTree.svelte`.

- [ ] **Step 3: Write the component**

`web/src/lib/StructureTree.svelte`:

```svelte
<script>
  /** @type {import('./types.js').Corefile|null} */
  export let corefile = null
</script>

{#if !corefile}
  <p data-testid="tree-empty">No Corefile parsed.</p>
{:else}
  <ul class="tree">
    {#each corefile.serverBlocks as block}
      <li class="block">
        <span class="keys">{block.keys.join(' ')}</span>
        <svelte:self-directives directives={block.directives} />
      </li>
    {/each}
  </ul>
{/if}

<!-- Recursive directive list rendered via a snippet-like inner component -->
<script context="module">
  // no module state needed
</script>
```

Note: Svelte has no `svelte:self-directives`. Implement recursion with a dedicated `Directives.svelte` instead. Replace the component body above with:

`web/src/lib/StructureTree.svelte` (final):

```svelte
<script>
  import Directives from './Directives.svelte'
  /** @type {import('./types.js').Corefile|null} */
  export let corefile = null
</script>

{#if !corefile}
  <p data-testid="tree-empty">No Corefile parsed.</p>
{:else}
  <ul class="tree">
    {#each corefile.serverBlocks as block}
      <li class="block">
        <span class="keys">{block.keys.join(' ')}</span>
        <Directives directives={block.directives} />
      </li>
    {/each}
  </ul>
{/if}
```

`web/src/lib/Directives.svelte`:

```svelte
<script>
  import Self from './Directives.svelte'
  /** @type {import('./types.js').Directive[]} */
  export let directives = []
</script>

<ul class="directives">
  {#each directives as d}
    <li class="directive">
      <span class="name">{d.name}</span>
      {#if d.args?.length}
        <span class="args">{d.args.join(' ')}</span>
      {/if}
      {#if d.block?.length}
        <Self directives={d.block} />
      {/if}
    </li>
  {/each}
</ul>
```

- [ ] **Step 4: Run test to verify it passes**

Run from `web/`: `npm run test -- StructureTree`
Expected: PASS (4 tests).

- [ ] **Step 5: Render the tree in App**

In `web/src/App.svelte`, add the import and replace the `result-summary` section's tree rendering. Add to `<script>`:

```js
  import StructureTree from './lib/StructureTree.svelte'
```

Replace the `<section data-testid="result-summary">…</section>` with:

```svelte
    <section class="views">
      <StructureTree corefile={result?.corefile ?? null} />
    </section>
```

- [ ] **Step 6: Verify build**

Run from `web/`: `npm run build`
Expected: build succeeds.

- [ ] **Step 7: Commit**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
git add web/src/ web/test/
git commit -m "feat: add structure tree view"
```

---

### Task 8: Validation panel view

**Files:**
- Create: `web/src/lib/ValidationPanel.svelte`
- Create: `web/test/ValidationPanel.test.js`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: the `Diagnostic[]` shape (`severity`, `message`, `line`) from Task 1's JSON contract.
- Produces: `ValidationPanel.svelte` with a `diagnostics` prop (`Diagnostic[]`); renders a "no issues" state when empty, otherwise one row per diagnostic showing severity, line, and message.

- [ ] **Step 1: Write the failing test**

`web/test/ValidationPanel.test.js`:

```js
import { render, screen } from '@testing-library/svelte'
import { describe, it, expect } from 'vitest'
import ValidationPanel from '../src/lib/ValidationPanel.svelte'

describe('ValidationPanel', () => {
  it('shows a clean state when there are no diagnostics', () => {
    render(ValidationPanel, { diagnostics: [] })
    expect(screen.getByTestId('validation-clean')).toBeInTheDocument()
  })

  it('renders a row per diagnostic with severity, line, and message', () => {
    render(ValidationPanel, {
      diagnostics: [
        { severity: 'warning', message: 'duplicate zone "."', line: 5 },
        { severity: 'error', message: 'missing closing brace', line: 0 },
      ],
    })
    expect(screen.getAllByTestId('diagnostic-row')).toHaveLength(2)
    expect(screen.getByText('duplicate zone "."')).toBeInTheDocument()
    expect(screen.getByText(/line 5/i)).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run from `web/`: `npm run test -- ValidationPanel`
Expected: FAIL — cannot resolve `../src/lib/ValidationPanel.svelte`.

- [ ] **Step 3: Write the component**

`web/src/lib/ValidationPanel.svelte`:

```svelte
<script>
  /** @type {import('./types.js').Diagnostic[]} */
  export let diagnostics = []
</script>

<div class="validation">
  {#if diagnostics.length === 0}
    <p data-testid="validation-clean">No issues found.</p>
  {:else}
    <ul>
      {#each diagnostics as d}
        <li class="diag diag-{d.severity}" data-testid="diagnostic-row">
          <span class="sev">{d.severity}</span>
          {#if d.line > 0}<span class="loc">line {d.line}</span>{/if}
          <span class="msg">{d.message}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>
```

- [ ] **Step 4: Run test to verify it passes**

Run from `web/`: `npm run test -- ValidationPanel`
Expected: PASS (2 tests).

- [ ] **Step 5: Render the panel in App and run the full suite**

In `web/src/App.svelte` add to `<script>`:

```js
  import ValidationPanel from './lib/ValidationPanel.svelte'
```

Inside the `<section class="views">`, below `<StructureTree …/>`, add:

```svelte
      <ValidationPanel diagnostics={result?.diagnostics ?? []} />
```

Run the full suites to confirm nothing regressed:
```bash
cd /Users/gtadi/workspace/corefile-visualizer
go test ./...
cd web && npm run test && npm run build
```
Expected: all Go tests pass; all Vitest tests pass; Vite build succeeds.

- [ ] **Step 6: Commit**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
git add web/src/ web/test/
git commit -m "feat: add validation panel view"
```

---

## Self-Review Notes

- **Spec coverage (Phase 1 scope):** structure tree (Task 7), validation panel (Tasks 3+8), engine/parser reuse of `coredns/caddy` (Task 2), Go→WASM client-side (Task 4), Svelte/Vite (Tasks 5–8), Apache-2.0 license (Task 1). Request-flow diagram, plugin reference, full plugin registry, samples/shareable-URL/Pages deploy are explicitly Phase 2/3 and out of scope here.
- **Parser-API note:** verified against `github.com/coredns/caddy v1.1.4` — `Parse` returns an order-losing `map`, so the analyzer walks `NewDispenser` tokens directly. Confirmed braces arrive as standalone tokens with line numbers.
- **Type consistency:** JSON contract (`serverBlocks`, `keys`, `directives`, `name`, `args`, `line`, `block`, `corefile`, `diagnostics`, `severity`, `message`) is defined once in Task 1 and consumed unchanged by Tasks 4–8 and `types.js`.
