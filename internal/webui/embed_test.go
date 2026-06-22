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
