// Package render formats check findings for stdout — either as
// human-readable text (default) or as a structured JSON envelope
// (--format=json).
package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
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
//	{path}: {severity} {code}[/{subcode}]: {message}
//
// followed by a one-line summary. Findings without a path are still
// rendered (the path field is omitted but the rest is shown).
func Text(w io.Writer, findings []check.Finding) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(w, "ok — no findings")
		return err
	}
	errCount, warnCount := 0, 0
	for _, f := range findings {
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
		if f.Path != "" {
			if _, err := fmt.Fprintf(w, "%s: %s %s: %s\n", f.Path, f.Severity, code, f.Message); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "%s %s: %s\n", f.Severity, code, f.Message); err != nil {
			return err
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
