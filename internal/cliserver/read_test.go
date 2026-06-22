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
