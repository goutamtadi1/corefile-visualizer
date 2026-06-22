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
