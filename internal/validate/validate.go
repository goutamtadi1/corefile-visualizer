// Package validate runs basic semantic lint over a parsed Corefile. Phase 1
// covers structural checks (duplicate zones, empty blocks); plugin-aware rules
// arrive in Phase 2.
package validate

import (
	"fmt"
	"strings"

	"github.com/gtadi/corefile-visualizer/internal/model"
	"github.com/gtadi/corefile-visualizer/internal/plugins"
)

// Validate returns diagnostics for the given Corefile. The result is always
// non-nil (empty when there are no findings).
func Validate(cf *model.Corefile) []model.Diagnostic {
	diags := []model.Diagnostic{}
	if cf == nil {
		return diags
	}

	seen := map[string]bool{}
	for _, sb := range cf.ServerBlocks {
		if len(sb.Directives) == 0 {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityWarning,
				Message:  fmt.Sprintf("server block %q has no plugins", strings.Join(sb.Keys, " ")),
				Line:     sb.Line,
			})
		}
		seenUnknown := map[string]bool{}
		for _, d := range sb.Directives {
			if _, ok := plugins.Rank(d.Name); ok {
				continue
			}
			if seenUnknown[d.Name] {
				continue
			}
			seenUnknown[d.Name] = true
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityWarning,
				Message:  fmt.Sprintf("unknown plugin %q — not a recognized CoreDNS plugin", d.Name),
				Line:     d.Line,
			})
		}
		for _, key := range sb.Keys {
			if seen[key] {
				diags = append(diags, model.Diagnostic{
					Severity: model.SeverityWarning,
					Message:  fmt.Sprintf("duplicate zone %q declared in more than one server block", key),
					Line:     sb.Line,
				})
			}
			seen[key] = true
		}
	}
	return diags
}
