# CoreDNS Corefile Visualizer

A fully client-side, open-source tool for visualizing [CoreDNS](https://coredns.io)
**Corefiles** in the browser. Paste or upload a Corefile and see its structure and
validation results instantly.

The parsing engine is written in Go and compiled to **WebAssembly**, so everything
runs in your browser — **your Corefile never leaves your machine** and there is no
backend to host.

> **Note:** "Corefile" here means a [CoreDNS configuration file](https://coredns.io/manual/configuration/),
> not a process core dump.

## Features

**Available now (Phase 1):**

- **Structure tree** — server blocks → zones/ports → plugins, rendered in
  declaration order (including nested plugin blocks).
- **Validation panel** — flags parse errors, duplicate zones, and empty server
  blocks, with line references.
- **Editor** — a CodeMirror editor with file upload; analysis runs as you type.

**Planned:**

- _Phase 2:_ request-flow diagram (plugin execution order), a plugin reference
  panel, and a curated metadata registry for the official in-tree plugins (for
  richer, plugin-aware validation).
- _Phase 3:_ bundled sample Corefiles, shareable URLs, a GitHub Pages deploy
  workflow, and visual styling.

## How it works

```
Corefile text ──► [WASM engine] ──► JSON model ──► [Svelte UI] ──► views
   (editor)        (Go, in-browser)                               ├─ Structure tree
                                                                  └─ Validation panel
```

The engine wraps the **real CoreDNS parser** (`github.com/coredns/caddy`) at the
token level, so parsing matches CoreDNS itself — and directive order and repeated
directives are preserved (which the high-level map-based parse API would lose).

## Project layout

```
cmd/wasm/            # WebAssembly entrypoint (syscall/js glue)
internal/
  model/             # structured types + JSON contract (engine ⇄ frontend)
  analyzer/          # token walker: Corefile text → model
  validate/          # semantic lint rules
  engine/            # composes analyze + validate into one Result
scripts/build-wasm.sh # builds the WASM engine into web/public/wasm/
web/                 # Svelte + Vite frontend
docs/superpowers/    # design spec and implementation plan
```

## Install & run

There is no hosted version yet (a GitHub Pages deploy is planned for Phase 3),
so for now you run it locally.

**Prerequisites:** Go 1.26+ and Node.js.

```bash
# Clone
git clone https://github.com/goutamtadi1/corefile-visualizer.git
cd corefile-visualizer

# 1. Build the WASM engine into web/public/wasm/
./scripts/build-wasm.sh

# 2. Install frontend dependencies and start the dev server
cd web
npm install
npm run dev
```

Then open the local URL it prints (e.g. `http://localhost:5173`).

To produce a static production build instead, run `npm run build` from `web/`
(output in `web/dist/`), and serve that directory with any static file server.

## Usage

1. Open the app in your browser (see above).
2. **Paste** a Corefile into the editor, or click **Upload Corefile** to load one
   from disk. Analysis runs automatically as you type.
3. Read the results:
   - The **structure tree** shows each server block, its zones/ports, and its
     plugins in order (nested blocks are expanded inline).
   - The **validation panel** lists any problems — parse errors, duplicate zones,
     or empty server blocks — with line numbers.

Try it with this example:

```corefile
example.org:53 {
    log
    errors
    forward . 8.8.8.8 9.9.9.9
    cache {
        success 5000
    }
}

. {
    whoami
}
```

You should see two server blocks — `example.org:53` with `log → errors → forward
→ cache` (and `success` nested under `cache`), and `.` with `whoami` — and an
empty validation panel ("No issues found").

## Testing

```bash
# Go engine
go test ./...

# Frontend component tests
cd web && npm run test
```

## Contributing

This project is built in phases, each with a design spec and implementation plan
under `docs/superpowers/`. Issues and pull requests are welcome.

## License

[Apache-2.0](LICENSE) — matching CoreDNS and its parser.
