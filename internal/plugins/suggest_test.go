package plugins

import (
	"strings"
	"testing"

	"github.com/gtadi/corefile-visualizer/internal/model"
)

func TestSuggestionsFlagsMissingRecommended(t *testing.T) {
	// A bare block missing everything recommended.
	tips := Suggestions(dirs("whoami"))
	if len(tips) == 0 {
		t.Fatal("expected suggestions for a bare block")
	}
	joined := strings.Join(tips, "\n")
	for _, want := range []string{"errors", "cache", "health"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected a suggestion mentioning %q, got: %s", want, joined)
		}
	}
}

func TestSuggestionsOmitsPresentPlugins(t *testing.T) {
	tips := Suggestions(dirs("errors", "cache"))
	joined := strings.Join(tips, "\n")
	if strings.Contains(joined, "'errors'") || strings.Contains(joined, "'cache'") {
		t.Errorf("present plugins should not be suggested, got: %s", joined)
	}
}

func TestSuggestionsNilWhenAllPresent(t *testing.T) {
	all := []model.Directive{}
	for _, r := range recommended {
		all = append(all, model.Directive{Name: r.name})
	}
	if got := Suggestions(all); got != nil {
		t.Errorf("expected nil suggestions when all recommended plugins present, got %v", got)
	}
}
