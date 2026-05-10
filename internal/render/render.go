// Package render formats check findings for stdout — either as
// human-readable text (default) or as a structured JSON envelope
// (--format=json).
//
// # JSON envelope contract
//
// Every aiwf invocation that emits JSON writes a single object with
// these slots:
//
//	tool      // always "aiwf"
//	version   // the binary's reported version
//	status    // "ok" | "findings" | "error" — overall outcome
//	findings  // []Finding — validation outcomes, cross-cutting; may
//	          // be present on any verb that runs the validators.
//	          // Empty when the run produced none.
//	result    // verb-specific payload. Different shape per verb:
//	          //   - check    → omitted (findings is the result)
//	          //   - history  → { id, events: [...] }
//	          //   - future verbs → their own shape
//	metadata  // counts, timing, root path, correlation_id when
//	          // present — auxiliary data, not load-bearing
//
// findings vs result is the load-bearing distinction: findings is
// always the same shape across verbs (so a CI script can grep one
// thing), result is whatever the verb returns (so each verb can
// model its own output without compromise). Downstream tooling that
// touches both reads findings the same way everywhere and switches
// on the verb name to interpret result.
package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/23min/aiwf/internal/check"
)

// Envelope is the JSON shape every aiwf invocation writes when invoked
// with --format=json. Status is one of "ok", "findings", "error".
// Result carries the verb's payload (nil for `aiwf check`).
// Metadata carries timing, counts, correlation_id, etc.
type Envelope struct {
	Tool     string          `json:"tool"`
	Version  string          `json:"version"`
	Status   string          `json:"status"`
	Findings []check.Finding `json:"findings"`
	Result   any             `json:"result,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}

// StatusFor returns the canonical envelope status string for a given
// findings list. "ok" if empty, "findings" if anything was reported.
// Internal-error envelopes are constructed directly by callers.
func StatusFor(findings []check.Finding) string {
	if len(findings) == 0 {
		return "ok"
	}
	return "findings"
}

// Text writes one finding per line in linter-style format:
//
//	{path}:{line}: {severity} {code}[/{subcode}]: {message} — hint: {hint}
//
// followed by a one-line summary. The `:line` is omitted when the
// finding has no line (load-errors that fail before parsing). The
// `— hint: ...` suffix is omitted when the finding has no hint.
// Findings without a path are still rendered (the path:line prefix
// is dropped but the rest of the line is unchanged).
func Text(w io.Writer, findings []check.Finding) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(w, "ok — no findings")
		return err
	}
	errCount, warnCount := 0, 0
	for i := range findings {
		f := &findings[i]
		switch f.Severity {
		case check.SeverityError:
			errCount++
		case check.SeverityWarning:
			warnCount++
		}
		code := f.Code
		if f.Subcode != "" {
			code = code + "/" + f.Subcode
		}
		hint := ""
		if f.Hint != "" {
			hint = " — hint: " + f.Hint
		}
		switch {
		case f.Path != "" && f.Line > 0:
			if _, err := fmt.Fprintf(w, "%s:%d: %s %s: %s%s\n", f.Path, f.Line, f.Severity, code, f.Message, hint); err != nil {
				return err
			}
		case f.Path != "":
			if _, err := fmt.Fprintf(w, "%s: %s %s: %s%s\n", f.Path, f.Severity, code, f.Message, hint); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprintf(w, "%s %s: %s%s\n", f.Severity, code, f.Message, hint); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintf(w, "\n%d findings (%d errors, %d warnings)\n", len(findings), errCount, warnCount)
	return err
}

// JSON writes the envelope to w as a single JSON object. Pretty enables
// indented output for human reading; the default (non-indented) is what
// CI and downstream tooling consume.
func JSON(w io.Writer, env Envelope, pretty bool) error {
	enc := json.NewEncoder(w)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if env.Findings == nil {
		env.Findings = []check.Finding{}
	}
	return enc.Encode(env)
}
