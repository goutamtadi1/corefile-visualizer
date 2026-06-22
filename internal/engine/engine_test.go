package engine

import (
	"testing"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

func TestRunValid(t *testing.T) {
	res := Run(". {\n    whoami\n}\n")
	if res.Corefile == nil || len(res.Corefile.ServerBlocks) != 1 {
		t.Fatalf("corefile = %+v", res.Corefile)
	}
	if len(res.Diagnostics) != 0 {
		t.Errorf("diagnostics = %+v, want none", res.Diagnostics)
	}
}

func TestRunParseErrorBecomesDiagnostic(t *testing.T) {
	res := Run(". {\n    whoami\n") // missing closing brace
	if res.Corefile != nil {
		t.Errorf("corefile = %+v, want nil on parse error", res.Corefile)
	}
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Severity != model.SeverityError {
		t.Fatalf("diagnostics = %+v, want one error", res.Diagnostics)
	}
}
