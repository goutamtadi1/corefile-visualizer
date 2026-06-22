# Web frontend

Svelte + Vite frontend for the CoreDNS Corefile Visualizer. It loads the Go→WASM engine from `public/wasm/` and renders the parsed Corefile.

## Develop

```bash
../scripts/build-wasm.sh   # build the WASM engine into public/wasm/
npm install
npm run dev
```

## Build

```bash
npm run build
```
