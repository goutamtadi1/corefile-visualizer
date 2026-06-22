package plugins

import (
	"testing"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

func TestRankKnownAndUnknown(t *testing.T) {
	ie, ok := Rank("errors")
	il, _ := Rank("log")
	if !ok || ie >= il {
		t.Fatalf("expected errors known and ranked before log: errors=%d ok=%v log=%d", ie, ok, il)
	}
	if _, ok := Rank("definitely-not-a-plugin"); ok {
		t.Error("expected unknown plugin to report ok=false")
	}
}

func dirs(names ...string) []model.Directive {
	out := make([]model.Directive, len(names))
	for i, n := range names {
		out[i] = model.Directive{Name: n}
	}
	return out
}

func names(flow []model.FlowStep) []string {
	out := make([]string, len(flow))
	for i, f := range flow {
		out[i] = f.Name
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildFlowOrdersByPluginCfg(t *testing.T) {
	// Declaration order: log, errors, forward, cache.
	// plugin.cfg execution order: errors < log < cache < forward.
	flow := BuildFlow(dirs("log", "errors", "forward", "cache"))
	got := names(flow)
	want := []string{"errors", "log", "cache", "forward"}
	if !equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for _, f := range flow {
		if !f.Known {
			t.Errorf("expected %q known", f.Name)
		}
	}
}

func TestBuildFlowCollapsesRepeats(t *testing.T) {
	flow := BuildFlow(dirs("file", "file"))
	if got := names(flow); !equal(got, []string{"file"}) {
		t.Fatalf("repeats not collapsed: %v", got)
	}
}

func TestBuildFlowAppendsUnknownFlagged(t *testing.T) {
	flow := BuildFlow(dirs("whoami", "customplugin"))
	got := names(flow)
	if !equal(got, []string{"whoami", "customplugin"}) {
		t.Fatalf("order = %v, want [whoami customplugin]", got)
	}
	if !flow[0].Known {
		t.Error("whoami should be Known")
	}
	if flow[1].Known {
		t.Error("customplugin should be Known=false")
	}
}

func TestBuildFlowEmptyIsNonNil(t *testing.T) {
	flow := BuildFlow(nil)
	if flow == nil {
		t.Fatal("BuildFlow(nil) returned nil; want empty non-nil slice")
	}
	if len(flow) != 0 {
		t.Fatalf("want empty, got %v", names(flow))
	}
}

func TestBuildFlowCollapsesUnknownRepeats(t *testing.T) {
	flow := BuildFlow(dirs("customx", "customx"))
	if got := names(flow); !equal(got, []string{"customx"}) {
		t.Fatalf("unknown repeats not collapsed: %v", got)
	}
	if flow[0].Known {
		t.Error("customx should be Known=false")
	}
}
