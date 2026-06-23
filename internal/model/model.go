// Package model defines the structured, JSON-serializable representation of a
// parsed CoreDNS Corefile. It is the contract between the WASM engine and the
// frontend; JSON field names must not change without updating the frontend.
package model

// Corefile is the top-level parsed document.
type Corefile struct {
	ServerBlocks []ServerBlock `json:"serverBlocks"`
}

// ServerBlock associates one or more keys (zones/addresses) with an ordered
// list of directives (plugins).
type ServerBlock struct {
	Keys       []string    `json:"keys"`
	Line       int         `json:"line"`
	Directives []Directive `json:"directives"`
	Flow       []FlowStep  `json:"flow,omitempty"`
	// Suggestions are best-practice improvement tips for this server block
	// (e.g. "add the cache plugin"). Empty when nothing applies.
	Suggestions []string `json:"suggestions,omitempty"`
}

// Directive is a single plugin invocation, preserving declaration order, its
// arguments, and any nested block.
type Directive struct {
	Name  string      `json:"name"`
	Args  []string    `json:"args,omitempty"`
	Line  int         `json:"line"`
	Block []Directive `json:"block,omitempty"`
}

// FlowStep is one plugin in a server block's request-execution chain. Steps are
// ordered by CoreDNS plugin.cfg execution order; Known is false for plugins not
// present in plugin.cfg.
type FlowStep struct {
	Name  string `json:"name"`
	Known bool   `json:"known"`
}

// Severity classifies a diagnostic.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Diagnostic is a single validation finding.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Line     int      `json:"line"`
}

// Result is the full output of the engine for one Corefile input.
type Result struct {
	Corefile    *Corefile    `json:"corefile"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
