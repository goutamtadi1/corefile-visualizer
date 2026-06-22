# CLI "pipe & visualize" Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a native `corefile-visualizer` CLI that reads a piped (or file-arg) Corefile, serves the existing Svelte/WASM app from an embedded local HTTP server with the content pre-loaded, opens the browser, and stays running until Ctrl-C.

**Architecture:** A new native Go binary `cmd/corefile-visualizer` embeds the built web app (`internal/webui`, `//go:embed`) and serves it locally (`internal/cliserver`): static app assets plus a `GET /corefile` route returning the piped text. The frontend gains one graceful change — on load it fetches `/corefile` and uses it as the initial editor content, falling back to the existing SAMPLE when absent.

**Tech Stack:** Go 1.26 (`embed`, `net/http`, `httptest`, `testing/fstest`), existing Svelte + Vite + Vitest frontend.

## Global Constraints

- Go module path: `github.com/gtadi/corefile-visualizer`.
- Go version floor: 1.26.
- The `GET /corefile` response MUST have `Content-Type: text/plain; charset=utf-8`.
- The frontend MUST only accept `/corefile` content when the response is OK **and** its content-type contains `text/plain` (a Vite/SPA host returns `index.html` with `text/html` for unknown paths — that must fall back to SAMPLE).
- Web app is embedded via `internal/webui` using `//go:embed all:dist`; `internal/webui/dist/` is gitignored except a committed `.gitkeep` (so the package always compiles, even before a real build).
- The standalone web app (dev server / GitHub Pages) behavior must be unchanged when `/corefile` is absent.
- `bin/` and `internal/webui/dist/*` (except `.gitkeep`) are gitignored.
- Conventional commit messages; no Claude co-author.
- Branch: `cli-pipe-visualize` (already created off the merged `main`).

## File Structure

```
cmd/corefile-visualizer/main.go      # native entrypoint: flags, wiring, browser open, Ctrl-C
internal/webui/embed.go              # //go:embed all:dist → fs.FS
internal/webui/embed_test.go
internal/webui/dist/.gitkeep         # committed; keeps //go:embed compilable
internal/cliserver/read.go           # ReadCorefile (stdin vs file)
internal/cliserver/read_test.go
internal/cliserver/server.go         # Handler + ListenLocal
internal/cliserver/server_test.go
scripts/build-cli.sh                 # build wasm → npm build → copy dist → go build
web/src/lib/initialContent.js        # loadInitialCorefile(sample, fetchFn)
web/test/initialContent.test.js
web/src/App.svelte                   # (modified) fetch initial content, then render editor
.gitignore                           # (modified) add bin/, internal/webui/dist rules
README.md                            # (modified) document the CLI
```

**Scope:** Single cohesive feature; one plan. Produces a working CLI plus the minimal frontend change.

---

### Task 1: Embed package for the web app

**Files:**
- Create: `internal/webui/embed.go`
- Create: `internal/webui/dist/.gitkeep`
- Test: `internal/webui/embed_test.go`
- Modify: `.gitignore`

**Interfaces:**
- Produces: `func webui.FS() (fs.FS, error)` — returns the embedded app filesystem rooted at the `dist` contents (so `index.html` is at the root of the returned FS).

**Background (verified):** `//go:embed all:dist` requires `dist/` to contain at least one file at compile time; a committed `.gitkeep` satisfies this. `fs.Sub(embedded, "dist")` returns the subtree. The real app is copied into `dist/` by `scripts/build-cli.sh` (Task 4) and is gitignored.

- [ ] **Step 1: Add .gitignore rules and the placeholder**

Append to `.gitignore`:

```gitignore
# Native CLI build output
bin/

# Embedded web app (populated by scripts/build-cli.sh; keep the dir compilable)
internal/webui/dist/*
!internal/webui/dist/.gitkeep
```

Create an empty file `internal/webui/dist/.gitkeep`:

```bash
mkdir -p internal/webui/dist
touch internal/webui/dist/.gitkeep
```

- [ ] **Step 2: Write the failing test**

`internal/webui/embed_test.go`:

```go
package webui

import (
	"io/fs"
	"testing"
)

func TestFSReturnsFilesystem(t *testing.T) {
	f, err := FS()
	if err != nil {
		t.Fatalf("FS() error: %v", err)
	}
	// The .gitkeep placeholder is always embedded; after a real build the
	// app's index.html is present too. We only assert the embed wiring works.
	if _, err := fs.Stat(f, "."); err != nil {
		t.Fatalf("root not statable: %v", err)
	}
	entries, err := fs.ReadDir(f, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("embedded filesystem is empty; expected at least .gitkeep")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/webui/`
Expected: FAIL — `undefined: FS` (package has no source yet).

- [ ] **Step 4: Write the implementation**

`internal/webui/embed.go`:

```go
// Package webui embeds the built Svelte/Vite web application so the native CLI
// can serve it from a single binary. The dist/ directory is populated by
// scripts/build-cli.sh and is gitignored except for a .gitkeep placeholder that
// keeps the //go:embed directive compilable before a build has run.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// FS returns the embedded web application's filesystem, rooted so that
// index.html sits at the root.
func FS() (fs.FS, error) {
	return fs.Sub(embedded, "dist")
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/webui/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/webui/ .gitignore
git commit -m "feat: add embedded web app package for the CLI"
```

---

### Task 2: Read the Corefile (stdin or file)

**Files:**
- Create: `internal/cliserver/read.go`
- Test: `internal/cliserver/read_test.go`

**Interfaces:**
- Produces: `func cliserver.ReadCorefile(stdin io.Reader, stdinIsPipe bool, fileArg string) (string, error)` — if `stdinIsPipe`, reads all of `stdin`; else if `fileArg != ""`, reads that file; else returns an error. Empty content is valid (not an error).

- [ ] **Step 1: Write the failing test**

`internal/cliserver/read_test.go`:

```go
package cliserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCorefileFromPipedStdin(t *testing.T) {
	got, err := ReadCorefile(strings.NewReader(". {\n  whoami\n}\n"), true, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != ". {\n  whoami\n}\n" {
		t.Errorf("got %q", got)
	}
}

func TestReadCorefilePipedStdinWinsOverFileArg(t *testing.T) {
	got, err := ReadCorefile(strings.NewReader("from-stdin"), true, "/should/not/be/read")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "from-stdin" {
		t.Errorf("expected stdin to win, got %q", got)
	}
}

func TestReadCorefileFromFileArg(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Corefile")
	if err := os.WriteFile(p, []byte("from-file"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadCorefile(strings.NewReader(""), false, p)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "from-file" {
		t.Errorf("got %q", got)
	}
}

func TestReadCorefileNoInputErrors(t *testing.T) {
	if _, err := ReadCorefile(strings.NewReader(""), false, ""); err == nil {
		t.Fatal("expected error when neither stdin pipe nor file arg provided")
	}
}

func TestReadCorefileUnreadableFileErrors(t *testing.T) {
	if _, err := ReadCorefile(strings.NewReader(""), false, "/no/such/file/here"); err == nil {
		t.Fatal("expected error for unreadable file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cliserver/`
Expected: FAIL — `undefined: ReadCorefile`.

- [ ] **Step 3: Write the implementation**

`internal/cliserver/read.go`:

```go
// Package cliserver provides the input-reading and HTTP-serving logic for the
// corefile-visualizer CLI, kept free of browser/process concerns so it is
// unit-testable.
package cliserver

import (
	"errors"
	"io"
	"os"
)

// ErrNoInput is returned when neither a piped stdin nor a file argument is given.
var ErrNoInput = errors.New("no Corefile provided: pipe one via stdin or pass a file path")

// ReadCorefile returns the Corefile text. A piped stdin takes precedence; if
// stdin is not piped, fileArg is read. Empty content is allowed.
func ReadCorefile(stdin io.Reader, stdinIsPipe bool, fileArg string) (string, error) {
	if stdinIsPipe {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	if fileArg != "" {
		b, err := os.ReadFile(fileArg)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", ErrNoInput
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cliserver/`
Expected: PASS (all five tests).

- [ ] **Step 5: Commit**

```bash
git add internal/cliserver/read.go internal/cliserver/read_test.go
git commit -m "feat: add Corefile input reading for the CLI"
```

---

### Task 3: HTTP handler and local listener

**Files:**
- Create: `internal/cliserver/server.go`
- Test: `internal/cliserver/server_test.go`

**Interfaces:**
- Consumes: nothing from other tasks (tests inject an `fs.FS` via `testing/fstest`).
- Produces:
  - `func cliserver.Handler(app fs.FS, corefile string) http.Handler` — serves `GET /corefile` (the text, `text/plain; charset=utf-8`) and all other paths from `app` via a static file server.
  - `func cliserver.ListenLocal(port int) (net.Listener, error)` — binds `127.0.0.1:<port>` (port 0 = a random free port); returns the listener so the chosen port is knowable.

- [ ] **Step 1: Write the failing test**

`internal/cliserver/server_test.go`:

```go
package cliserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testApp() fstest.MapFS {
	return fstest.MapFS{
		"index.html":     {Data: []byte("<!doctype html><title>app</title>")},
		"assets/app.css": {Data: []byte("body{}")},
	}
}

func TestHandlerServesCorefile(t *testing.T) {
	srv := httptest.NewServer(Handler(testApp(), ". {\n  whoami\n}\n"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/corefile")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("content-type = %q, want text/plain", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != ". {\n  whoami\n}\n" {
		t.Errorf("body = %q", string(body))
	}
}

func TestHandlerServesIndex(t *testing.T) {
	srv := httptest.NewServer(Handler(testApp(), "x"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<title>app</title>") {
		t.Errorf("index not served, body = %q", string(body))
	}
}

func TestHandlerServesAsset(t *testing.T) {
	srv := httptest.NewServer(Handler(testApp(), "x"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/assets/app.css")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestListenLocalRandomPort(t *testing.T) {
	ln, err := ListenLocal(0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)
	if addr.Port == 0 {
		t.Fatal("expected a non-zero assigned port")
	}
}
```

This test file imports: `"io"`, `"net"`, `"net/http"`, `"net/http/httptest"`, `"strings"`, `"testing"`, `"testing/fstest"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cliserver/`
Expected: FAIL — `undefined: Handler`, `undefined: ListenLocal`.

- [ ] **Step 3: Write the implementation**

`internal/cliserver/server.go`:

```go
package cliserver

import (
	"io/fs"
	"net"
	"net/http"
	"strconv"
)

// Handler serves the embedded web app and a GET /corefile route returning the
// provided Corefile text as text/plain.
func Handler(app fs.FS, corefile string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/corefile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(corefile))
	})
	mux.Handle("/", http.FileServer(http.FS(app)))
	return mux
}

// ListenLocal binds 127.0.0.1 on the given port (0 = a random free port) and
// returns the listener so the caller can read the assigned address.
func ListenLocal(port int) (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cliserver/`
Expected: PASS (all tests, including ReadCorefile tests from Task 2).

- [ ] **Step 5: Commit**

```bash
git add internal/cliserver/server.go internal/cliserver/server_test.go
git commit -m "feat: add CLI HTTP handler and local listener"
```

---

### Task 4: CLI entrypoint + build script + README

**Files:**
- Create: `cmd/corefile-visualizer/main.go`
- Create: `scripts/build-cli.sh`
- Modify: `README.md`

**Interfaces:**
- Consumes: `webui.FS()` (Task 1); `cliserver.ReadCorefile`, `cliserver.Handler`, `cliserver.ListenLocal` (Tasks 2–3).
- Produces: the `corefile-visualizer` binary. No exported API for later tasks.

- [ ] **Step 1: Write the entrypoint**

`cmd/corefile-visualizer/main.go`:

```go
// Command corefile-visualizer reads a CoreDNS Corefile (from a piped stdin or a
// file argument), serves the embedded web visualization from a local HTTP
// server with that content pre-loaded, opens the browser, and runs until Ctrl-C.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/gtadi/corefile-visualizer/internal/cliserver"
	"github.com/gtadi/corefile-visualizer/internal/webui"
)

// openBrowser is a variable so tests can stub it.
var openBrowser = openBrowserDefault

func main() {
	noOpen := flag.Bool("no-open", false, "start the server but do not open a browser")
	port := flag.Int("port", 0, "port to listen on (0 = a random free port)")
	flag.Parse()

	fi, _ := os.Stdin.Stat()
	stdinIsPipe := (fi.Mode() & os.ModeCharDevice) == 0

	content, err := cliserver.ReadCorefile(os.Stdin, stdinIsPipe, flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintln(os.Stderr, "usage: corefile-visualizer [--no-open] [--port N] [FILE]  (or pipe a Corefile via stdin)")
		os.Exit(2)
	}

	app, err := webui.FS()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: embedded web app unavailable:", err)
		os.Exit(1)
	}

	ln, err := cliserver.ListenLocal(*port)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot listen:", err)
		os.Exit(1)
	}
	url := fmt.Sprintf("http://%s/", ln.Addr().String())

	srv := &http.Server{Handler: cliserver.Handler(app, content)}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "server error:", err)
		}
	}()

	fmt.Println("CoreDNS Corefile Visualizer serving at", url)
	if *noOpen {
		fmt.Println("(--no-open) open the URL above in your browser")
	} else if err := openBrowser(url); err != nil {
		fmt.Fprintln(os.Stderr, "could not open browser automatically:", err)
	}
	fmt.Println("Press Ctrl-C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\nshutting down")
	_ = srv.Close()
}

func openBrowserDefault(url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{url}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		name, args = "xdg-open", []string{url}
	}
	return exec.Command(name, args...).Start()
}
```

- [ ] **Step 2: Verify the binary compiles**

Run: `go build -o /dev/null ./cmd/corefile-visualizer/`
Expected: exits 0, no output.

- [ ] **Step 3: Write the build script**

`scripts/build-cli.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Builds the self-contained corefile-visualizer CLI:
#   WASM engine -> web app build -> copy into embed dir -> go build.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# 1. WASM engine into web/public/wasm/
"$ROOT/scripts/build-wasm.sh"

# 2. Build the web app
cd "$ROOT/web"
npm install
npm run build

# 3. Copy the built app into the gitignored embed directory (keep .gitkeep)
DEST="$ROOT/internal/webui/dist"
mkdir -p "$DEST"
find "$DEST" -mindepth 1 ! -name '.gitkeep' -delete
cp -R "$ROOT/web/dist/." "$DEST/"

# 4. Build the native CLI
cd "$ROOT"
mkdir -p bin
go build -o bin/corefile-visualizer ./cmd/corefile-visualizer
echo "Built $ROOT/bin/corefile-visualizer"
```

Then make it executable:

```bash
chmod +x scripts/build-cli.sh
```

- [ ] **Step 4: Build and verify end-to-end**

Run:
```bash
./scripts/build-cli.sh
echo '. {
  whoami
}' | ./bin/corefile-visualizer --no-open --port 8123 &
sleep 1
curl -s -i http://127.0.0.1:8123/corefile | head -n 5
curl -s -o /dev/null -w "index:%{http_code} ct:%{content_type}\n" http://127.0.0.1:8123/
curl -s -o /dev/null -w "wasm:%{http_code} ct:%{content_type}\n" http://127.0.0.1:8123/wasm/main.wasm
kill %1
```
Expected: `/corefile` returns `200` with `Content-Type: text/plain; charset=utf-8` and the piped body; `index:200 ct:text/html...`; `wasm:200 ct:application/wasm`.

- [ ] **Step 5: Document the CLI in the README**

In `README.md`, add a `## CLI` section after the "Usage" section:

````markdown
## CLI (pipe & visualize)

Build a self-contained binary that opens any Corefile in the browser visualizer:

```bash
./scripts/build-cli.sh            # builds WASM + web app + embeds them into the binary
```

Then pipe or pass a Corefile:

```bash
cat Corefile | ./bin/corefile-visualizer    # read from stdin
./bin/corefile-visualizer ./Corefile          # or a file argument
./bin/corefile-visualizer --no-open ./Corefile # don't auto-open a browser (headless)
./bin/corefile-visualizer --port 8080 ...       # pin the port (default: random)
```

It starts a local server, opens your browser with the Corefile pre-loaded, prints
the URL, and stays running until you press Ctrl-C.
````

- [ ] **Step 6: Commit**

```bash
git add cmd/corefile-visualizer/ scripts/build-cli.sh README.md
git commit -m "feat: add corefile-visualizer CLI and build script"
```

---

### Task 5: Frontend — load piped content, fall back to SAMPLE

**Files:**
- Create: `web/src/lib/initialContent.js`
- Create: `web/test/initialContent.test.js`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: nothing from Go tasks at build time (runtime contract: `GET /corefile` returns `text/plain` when served by the CLI).
- Produces: `loadInitialCorefile(sample, fetchFn = fetch): Promise<string>` — returns the `/corefile` body when the fetch is OK **and** content-type contains `text/plain` and the body is non-empty; otherwise returns `sample`.

- [ ] **Step 1: Write the failing test**

`web/test/initialContent.test.js`:

```js
import { describe, it, expect } from 'vitest'
import { loadInitialCorefile } from '../src/lib/initialContent.js'

const SAMPLE = '. {\n  whoami\n}\n'

function resp({ ok = true, ct = 'text/plain; charset=utf-8', body = '' }) {
  return Promise.resolve({
    ok,
    headers: { get: (h) => (h.toLowerCase() === 'content-type' ? ct : null) },
    text: () => Promise.resolve(body),
  })
}

describe('loadInitialCorefile', () => {
  it('uses /corefile text when ok and content-type is text/plain', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => resp({ body: 'piped-corefile' }))
    expect(got).toBe('piped-corefile')
  })

  it('falls back to sample when content-type is html (SPA fallback)', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => resp({ ct: 'text/html', body: '<html>' }))
    expect(got).toBe(SAMPLE)
  })

  it('falls back to sample on non-ok response', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => resp({ ok: false, body: 'nope' }))
    expect(got).toBe(SAMPLE)
  })

  it('falls back to sample on empty body', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => resp({ body: '' }))
    expect(got).toBe(SAMPLE)
  })

  it('falls back to sample when fetch throws', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => Promise.reject(new Error('network')))
    expect(got).toBe(SAMPLE)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run from `web/`: `npm run test -- initialContent`
Expected: FAIL — cannot resolve `../src/lib/initialContent.js`.

- [ ] **Step 3: Write the implementation**

`web/src/lib/initialContent.js`:

```js
/**
 * Resolve the editor's initial content. When served by the CLI, GET /corefile
 * returns the piped Corefile as text/plain; otherwise (standalone web app, or a
 * SPA host that returns index.html for unknown paths) we fall back to `sample`.
 *
 * @param {string} sample - fallback content
 * @param {typeof fetch} [fetchFn] - injectable fetch (for testing)
 * @returns {Promise<string>}
 */
export async function loadInitialCorefile(sample, fetchFn = fetch) {
  try {
    const resp = await fetchFn(`${import.meta.env.BASE_URL}corefile`)
    const ct = resp.headers.get('content-type') || ''
    if (resp.ok && ct.includes('text/plain')) {
      const text = await resp.text()
      if (text.length > 0) return text
    }
  } catch {
    // ignore — fall through to sample
  }
  return sample
}
```

- [ ] **Step 4: Run test to verify it passes**

Run from `web/`: `npm run test -- initialContent`
Expected: PASS (5 tests).

- [ ] **Step 5: Wire it into App.svelte**

In `web/src/App.svelte`, update the `<script>` so the editor renders only after the initial content resolves, and analysis runs on that content. Add the import:

```js
  import { loadInitialCorefile } from './lib/initialContent.js'
```

Replace the existing `onMount(...)` block and add an `initialDoc` state. The script's state declarations become:

```js
  /** @type {import('./lib/types.js').Result|null} */
  let result = null
  let loaded = false
  /** @type {string|null} */
  let initialDoc = null
  let timer

  onMount(async () => {
    const [doc] = await Promise.all([
      loadInitialCorefile(SAMPLE),
      loadWasm().then(() => { loaded = true }),
    ])
    initialDoc = doc
    runAnalysis(doc)
  })
```

(Keep the existing `runAnalysis` and `onChange` functions unchanged.)

Then gate the `<Editor>` on `initialDoc` so CodeMirror initializes once with the resolved content. Replace the existing `<Editor value={SAMPLE} on:change={onChange} />` with:

```svelte
    {#if initialDoc !== null}
      <Editor value={initialDoc} on:change={onChange} />
    {/if}
```

- [ ] **Step 6: Run the full frontend suite and build**

Run from `web/`:
```bash
npm run test
npm run build
```
Expected: all Vitest tests pass (StructureTree + ValidationPanel + initialContent); Vite build succeeds.

- [ ] **Step 7: Commit**

```bash
cd /Users/gtadi/workspace/corefile-visualizer
git add web/src/ web/test/
git commit -m "feat: load piped Corefile from /corefile with SAMPLE fallback"
```

---

### Task 6: Full integration verification

**Files:** none (verification only).

**Interfaces:** Consumes the complete feature (Tasks 1–5).

- [ ] **Step 1: Rebuild the CLI with the frontend change embedded**

Run: `./scripts/build-cli.sh`
Expected: prints `Built .../bin/corefile-visualizer`.

- [ ] **Step 2: Verify the full suite passes**

Run:
```bash
go test ./...
cd web && npm run test && npm run build
```
Expected: all Go tests pass (`webui`, `cliserver`, plus the existing `model`/`analyzer`/`validate`/`engine`); all Vitest tests pass; Vite build succeeds.

- [ ] **Step 3: Verify piped content reaches the served endpoint**

Run:
```bash
printf 'example.org:53 {\n  forward . 8.8.8.8\n}\n' | ./bin/corefile-visualizer --no-open --port 8124 &
sleep 1
echo "--- /corefile body ---"
curl -s http://127.0.0.1:8124/corefile
curl -s -o /dev/null -w "corefile ct: %{content_type}\n" http://127.0.0.1:8124/corefile
kill %1
```
Expected: the body is exactly the piped Corefile; content-type is `text/plain; charset=utf-8`. (The browser would then load this into the editor and render the tree/validation; the served `/corefile` is the integration seam being verified here.)

- [ ] **Step 4: Confirm no build artifacts are staged**

Run: `git status --short`
Expected: clean (no `bin/`, no `internal/webui/dist/` app files, no `web/dist`, no `node_modules`).

---

## Self-Review Notes

- **Spec coverage:** browser output via embedded local server (Tasks 1,3,4); stdin + file-arg inputs (Task 2); `--no-open`/`--port` + Ctrl-C lifecycle (Task 4); `GET /corefile` text/plain contract (Task 3); single graceful frontend change with content-type guard + SAMPLE fallback (Task 5); `go:embed` build coupling via `internal/webui` + `build-cli.sh` (Tasks 1,4); testing across `cliserver`/`webui`/frontend (Tasks 2,3,5); README (Task 4); end-to-end check (Task 6).
- **Content-type guard** (global constraint) is implemented in `loadInitialCorefile` and asserted by the `text/html` fallback test — this is the SPA-fallback safeguard surfaced during planning.
- **Type/name consistency:** `ReadCorefile`, `Handler`, `ListenLocal`, `webui.FS`, `loadInitialCorefile(sample, fetchFn)` are used identically across tasks. The `/corefile` route and its `text/plain` content-type match between Task 3 (server) and Task 5 (frontend).
- **Embed compilability:** the committed `.gitkeep` keeps `//go:embed all:dist` compiling before any build; the `cliserver` tests use `testing/fstest` and never depend on a real build.
