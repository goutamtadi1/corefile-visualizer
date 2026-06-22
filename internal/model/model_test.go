package model

import (
	"encoding/json"
	"testing"
)

func TestResultJSONRoundTrip(t *testing.T) {
	in := Result{
		Corefile: &Corefile{
			ServerBlocks: []ServerBlock{{
				Keys: []string{"example.org:53"},
				Line: 1,
				Directives: []Directive{
					{Name: "forward", Args: []string{".", "8.8.8.8"}, Line: 2},
					{Name: "cache", Line: 3, Block: []Directive{
						{Name: "success", Args: []string{"5000"}, Line: 4},
					}},
				},
			}},
		},
		Diagnostics: []Diagnostic{
			{Severity: SeverityWarning, Message: "duplicate zone", Line: 1},
		},
	}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Result
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Corefile.ServerBlocks[0].Keys[0] != "example.org:53" {
		t.Errorf("key = %q", out.Corefile.ServerBlocks[0].Keys[0])
	}
	if out.Corefile.ServerBlocks[0].Directives[1].Block[0].Name != "success" {
		t.Errorf("nested directive lost: %+v", out.Corefile.ServerBlocks[0].Directives[1])
	}
	if out.Diagnostics[0].Severity != SeverityWarning {
		t.Errorf("severity = %q", out.Diagnostics[0].Severity)
	}
}

func TestResultJSONFieldNames(t *testing.T) {
	b, _ := json.Marshal(Result{Corefile: &Corefile{}, Diagnostics: []Diagnostic{}})
	got := string(b)
	want := `{"corefile":{"serverBlocks":null},"diagnostics":[]}`
	if got != want {
		t.Errorf("contract changed:\n got=%s\nwant=%s", got, want)
	}
}
