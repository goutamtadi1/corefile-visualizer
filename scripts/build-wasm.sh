#!/usr/bin/env bash
set -euo pipefail

# Builds the WASM engine and copies the Go JS support file into the web app.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/web/public/wasm"
mkdir -p "$OUT"

GOOS=js GOARCH=wasm go build -o "$OUT/main.wasm" ./cmd/wasm/
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$OUT/wasm_exec.js"

echo "Built $OUT/main.wasm and copied wasm_exec.js"
