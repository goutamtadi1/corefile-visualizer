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
