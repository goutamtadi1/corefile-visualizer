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

func TestRunPopulatesFlowInExecutionOrder(t *testing.T) {
	// Declaration order is log then errors; plugin.cfg ranks errors before log.
	res := Run(". {\n    log\n    errors\n}\n")
	if res.Corefile == nil || len(res.Corefile.ServerBlocks) != 1 {
		t.Fatalf("corefile = %+v", res.Corefile)
	}
	flow := res.Corefile.ServerBlocks[0].Flow
	if len(flow) != 2 {
		t.Fatalf("flow = %+v, want 2 steps", flow)
	}
	if flow[0].Name != "errors" || flow[1].Name != "log" {
		t.Errorf("flow order = [%s %s], want [errors log]", flow[0].Name, flow[1].Name)
	}
	if !flow[0].Known || !flow[1].Known {
		t.Errorf("both steps should be Known: %+v", flow)
	}
}
