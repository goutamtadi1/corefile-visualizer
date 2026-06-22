// Package cliserver provides the input-reading and HTTP-serving logic for the
// corefile-visualizer CLI, kept free of browser/process concerns so it is
// unit-testable.
package cliserver

import (
	"errors"
	"io"
	"os"
)

// ErrNoInput is returned when neither a piped stdin nor a file argument is given.
var ErrNoInput = errors.New("no Corefile provided: pipe one via stdin or pass a file path")

// ReadCorefile returns the Corefile text. A piped stdin takes precedence; if
// stdin is not piped, fileArg is read. Empty content is allowed.
func ReadCorefile(stdin io.Reader, stdinIsPipe bool, fileArg string) (string, error) {
	if stdinIsPipe {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	if fileArg != "" {
		b, err := os.ReadFile(fileArg)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", ErrNoInput
}
