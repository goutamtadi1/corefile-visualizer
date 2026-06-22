package analyzer

import (
	"errors"
	"testing"
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
	if !errors.Is(err, ErrMissingCloseBrace) {
		t.Fatalf("expected ErrMissingCloseBrace, got %v", err)
	}
}

func TestAnalyzeMissingOpenBrace(t *testing.T) {
	_, err := Analyze("example.org:53\n")
	if !errors.Is(err, ErrMissingOpenBrace) {
		t.Fatalf("expected ErrMissingOpenBrace, got %v", err)
	}
}
