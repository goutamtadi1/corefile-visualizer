package plugins

import "github.com/gtadi/corefile-visualizer/internal/model"

// recommended lists best-practice plugins and the improvement tip shown when a
// server block does not use them. Order here is the order tips are presented.
var recommended = []struct{ name, tip string }{
	{"errors", "Add the 'errors' plugin to surface server errors in the log."},
	{"log", "Add the 'log' plugin to record queries for debugging and audit."},
	{"health", "Add the 'health' plugin for a liveness endpoint (recommended on Kubernetes)."},
	{"ready", "Add the 'ready' plugin for a readiness endpoint."},
	{"cache", "Add the 'cache' plugin to speed up repeated lookups and reduce upstream load."},
	{"loop", "Add the 'loop' plugin to detect forwarding loops at startup."},
	{"reload", "Add the 'reload' plugin to apply Corefile changes without a restart."},
	{"prometheus", "Add the 'prometheus' plugin to export metrics."},
}

// Suggestions returns best-practice improvement tips for a server block based on
// which recommended plugins are absent from its top-level directives. It returns
// nil when the block already uses all recommended plugins (nothing to suggest).
func Suggestions(directives []model.Directive) []string {
	present := make(map[string]bool, len(directives))
	for _, d := range directives {
		present[d.Name] = true
	}
	var tips []string
	for _, r := range recommended {
		if !present[r.name] {
			tips = append(tips, r.tip)
		}
	}
	return tips
}
