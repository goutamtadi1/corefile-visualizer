// Package engine composes parsing and validation into a single Result. It is
// the platform-independent core called by the WASM entrypoint, kept separate
// so it can be unit-tested without a js/wasm build.
package engine

import (
	"github.com/gtadi/corefile-visualizer/internal/analyzer"
	"github.com/gtadi/corefile-visualizer/internal/model"
	"github.com/gtadi/corefile-visualizer/internal/validate"
)

// Run parses and validates the Corefile text. On a parse error, Corefile is nil
// and Diagnostics holds a single error describing the failure.
func Run(input string) model.Result {
	cf, err := analyzer.Analyze(input)
	if err != nil {
		return model.Result{
			Corefile: nil,
			Diagnostics: []model.Diagnostic{{
				Severity: model.SeverityError,
				Message:  err.Error(),
				Line:     0,
			}},
		}
	}
	return model.Result{Corefile: cf, Diagnostics: validate.Validate(cf)}
}
