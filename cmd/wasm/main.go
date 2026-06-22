//go:build js && wasm

// Command wasm is the browser entrypoint. It registers a global JS function
// analyze(text) that returns the JSON-encoded engine result.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/gtadi/corefile-visualizer/internal/engine"
	"github.com/gtadi/corefile-visualizer/internal/model"
	"github.com/gtadi/corefile-visualizer/internal/plugins"
)

func analyze(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorJSON("analyze: missing input argument")
	}
	res := engine.Run(args[0].String())
	b, err := json.Marshal(res)
	if err != nil {
		return errorJSON("analyze: " + err.Error())
	}
	return string(b)
}

func pluginCatalog(_ js.Value, _ []js.Value) any {
	b, err := json.Marshal(plugins.Catalog())
	if err != nil {
		return "{}"
	}
	return string(b)
}

func errorJSON(msg string) string {
	b, _ := json.Marshal(model.Result{
		Diagnostics: []model.Diagnostic{{Severity: model.SeverityError, Message: msg}},
	})
	return string(b)
}

func main() {
	js.Global().Set("analyze", js.FuncOf(analyze))
	js.Global().Set("pluginCatalog", js.FuncOf(pluginCatalog))
	select {} // keep the Go runtime alive for JS calls
}
