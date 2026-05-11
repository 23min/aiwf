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
	"sort"

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
		switch findings[i].Severity {
		case check.SeverityError:
			errCount++
		case check.SeverityWarning:
			warnCount++
		}
	}
	if err := renderPerInstance(w, findings); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "\n%d findings (%d errors, %d warnings)\n", len(findings), errCount, warnCount)
	return err
}

// TextSummary writes the default-mode text rendering of findings used
// by `aiwf check` (without --verbose). Errors are rendered per
// instance — identical to the full per-instance shape produced by
// Text — because each error is per-instance-actionable. Warnings are
// collapsed into a per-code summary:
//
//	<code> (warning) × N — <representative message>
//
// where N is the count of findings sharing that Code and the
// representative message is the Message of the first finding in the
// input slice with that code (per M-0089 *Constraints*: "the first
// finding's Message field, verbatim").
//
// Summary lines are sorted by count descending, with ties broken
// alphabetically by code (also pinned in *Constraints* — pinned here
// so the golden files in the test suite don't drift).
//
// The footer line ("N findings (E errors, W warnings)") is unchanged
// and reflects raw instance counts, not summary-line counts.
func TextSummary(w io.Writer, findings []check.Finding) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(w, "ok — no findings")
		return err
	}

	// Partition: errors flow through the existing per-instance path;
	// warnings collect into summary buckets keyed by Code.
	var errors []check.Finding
	type bucket struct {
		code   string
		count  int
		sample string // first finding's Message (verbatim, per Constraints)
	}
	buckets := make(map[string]*bucket)
	var bucketOrder []string // codes in first-seen order, used for stable iteration
	errCount, warnCount := 0, 0

	for i := range findings {
		f := &findings[i]
		switch f.Severity {
		case check.SeverityError:
			errCount++
			errors = append(errors, *f)
		case check.SeverityWarning:
			warnCount++
			b, ok := buckets[f.Code]
			if !ok {
				b = &bucket{code: f.Code, sample: f.Message}
				buckets[f.Code] = b
				bucketOrder = append(bucketOrder, f.Code)
			}
			b.count++
		}
	}

	// Render errors per-instance using the same formatting Text uses.
	if err := renderPerInstance(w, errors); err != nil {
		return err
	}

	// Sort buckets: count desc, alphabetic tie-break.
	sort.SliceStable(bucketOrder, func(i, j int) bool {
		bi, bj := buckets[bucketOrder[i]], buckets[bucketOrder[j]]
		if bi.count != bj.count {
			return bi.count > bj.count
		}
		return bi.code < bj.code
	})

	for _, code := range bucketOrder {
		b := buckets[code]
		if _, err := fmt.Fprintf(w, "%s (warning) × %d — %s\n", b.code, b.count, b.sample); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, "\n%d findings (%d errors, %d warnings)\n", len(findings), errCount, warnCount)
	return err
}

// renderPerInstance writes the per-finding text rendering used by
// both Text (verbose mode) and TextSummary (default mode, for the
// errors slice). Extracted so the two callers stay byte-identical on
// the per-instance path — AC-3 of M-0089 requires --verbose output
// reproduce the pre-milestone behavior byte-for-byte.
func renderPerInstance(w io.Writer, findings []check.Finding) error {
	for i := range findings {
		f := &findings[i]
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
	return nil
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
