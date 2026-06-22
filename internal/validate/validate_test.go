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
