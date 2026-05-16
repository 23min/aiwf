// Package cliutil holds the verb-support kernel for aiwf's Cobra
// dispatchers: exit-code constants and the typed error shuttle, the
// post-verb apply path, identity resolution, flag re-ordering, repo
// locking, tree-loading with trunk-collision stamping, platform
// gating, and provenance gate-and-decorate plumbing.
//
// Every cmd/aiwf verb file calls into this package; nothing in
// internal/* depends on it (it lives under internal/cli/ so it stays
// invisible to external consumers per Go's internal-package rule).
package cliutil

import "fmt"

// Exit codes per CLAUDE.md § Go conventions § CLI conventions.
const (
	ExitOK       = 0 // no error-severity findings (warnings allowed)
	ExitFindings = 1 // at least one error-severity finding
	ExitUsage    = 2
	ExitInternal = 3
)

// ExitError carries a verb-handler return code through Cobra's
// Execute boundary. The CLI's run() loop unwraps it so the wrapped
// code becomes the process exit status. Without this typed shuttle,
// Cobra would collapse the 0/1/2/3 contract to "0 or non-zero"
// because it only knows about its own usage-error return path.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit %d", e.Code)
}

// WrapExitCode lifts a verb's int return code into the error channel
// Cobra's RunE expects. ExitOK collapses to nil (success); anything
// else becomes an *ExitError that the run() loop unwraps. Centralizing
// the translation keeps every RunE one-liner-shaped.
func WrapExitCode(code int) error {
	if code == ExitOK {
		return nil
	}
	return &ExitError{Code: code}
}
