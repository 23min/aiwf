package cliutil

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/version"
)

// OutputFormat carries a mutating verb's chosen output format, plumbed
// from AddFormatFlags through the verb's Run into FinishVerb /
// DecorateAndFinish (M-0143 / D-0013). The zero value (Format "", Pretty
// false) renders as text, so any path that never set it behaves exactly
// as before the milestone. This is the sanctioned path for a mutating
// verb's terminal outcome (its JSON-vs-text-aware envelope). For
// operator-facing text that isn't a verb's terminal outcome — a
// pre-flight usage error, a prompt, informational output from a
// read-only verb — use textio.go's Errorf/Errorln/Printf/Println/Print
// instead; both are the forbidigo/logging_chokepoint-sanctioned
// alternatives to a bare fmt.Print* call (ADR-0017 AC-3).
type OutputFormat struct {
	Format string
	Pretty bool

	// CorrelationID is the invocation's shared id (M-0239/AC-1): the
	// same value threaded into logger.WithVerb as run_id, so an
	// envelope and its invocation's diagnostic log lines are cross-
	// referenceable by a single grep. Empty when the caller never
	// threaded one (e.g. a test constructing OutputFormat directly) —
	// every emit method omits metadata.correlation_id entirely in
	// that case rather than emitting an empty string.
	CorrelationID string

	// Trace is --trace (M-0239/AC-3): forces a debug-level diagnostic
	// logger on for this invocation alone, regardless of AIWF_LOG, so
	// FinishVerb can emit a phase.apply timing event without the
	// operator needing separate env configuration.
	Trace bool
}

// JSON reports whether a JSON envelope was requested (--format=json).
func (o OutputFormat) JSON() bool { return o.Format == "json" }

// Metadata builds the envelope's metadata map for this invocation:
// extra's keys (a verb's own per-verb facts, M-0239/AC-2) plus
// correlation_id when CorrelationID is set. Returns nil (never an
// empty non-nil map) when there is nothing to carry, so
// render.Envelope's Metadata field's omitempty keeps behaving exactly
// as it did before this field existed. Exported so a verb whose
// output doesn't route through emitSuccess/emitFindings/
// emitErrorEnvelope (e.g. worktree add, which builds its own
// render.Envelope directly) can still merge in the same
// correlation_id — this is the single source of truth for that merge.
func (o OutputFormat) Metadata(extra map[string]any) map[string]any {
	if o.CorrelationID == "" && len(extra) == 0 {
		return nil
	}
	md := make(map[string]any, len(extra)+1)
	for k, v := range extra {
		md[k] = v
	}
	if o.CorrelationID != "" {
		md["correlation_id"] = o.CorrelationID
	}
	return md
}

// AddFormatFlags registers --format and --pretty on a mutating verb's
// command and returns the bound OutputFormat. It mirrors the read-verb
// flag shape (check/show/status/...) so --format=json behaves uniformly
// across read and mutating verbs (D-0013, decision A2). The returned
// pointer is dereferenced inside the verb's RunE closure, after Cobra has
// parsed flags.
func AddFormatFlags(cmd *cobra.Command) *OutputFormat {
	out := &OutputFormat{Format: "text"}
	cmd.Flags().StringVar(&out.Format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&out.Pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().BoolVar(&out.Trace, "trace", false, "emit per-phase timing at debug level through the diagnostic logger, enabling it for this invocation even without AIWF_LOG set")
	RegisterFormatCompletion(cmd)
	return out
}

// emitErrorEnvelope reports a verb's terminal error. In text mode it
// writes the conventional "label: message" line to stderr (unchanged
// behavior). In JSON mode it writes a status:"error" envelope to stdout
// and nothing to stderr, so a CI consumer reading stdout gets a single
// clean envelope. code is the structured code of a Coded error ("" when
// the error is not Coded).
func (o OutputFormat) emitErrorEnvelope(label, code, message string) {
	if !o.JSON() {
		fmt.Fprintf(os.Stderr, "%s: %s\n", label, message)
		return
	}
	env := render.Envelope{
		Tool:     "aiwf",
		Version:  version.Current().Version,
		Status:   "error",
		Error:    &render.EnvelopeError{Code: code, Message: message},
		Metadata: o.Metadata(nil),
	}
	_ = render.JSON(os.Stdout, env, o.Pretty)
}

// emitFindings reports an error-severity findings outcome: the per-
// instance text rendering to stderr (text mode) or a findings envelope
// to stdout (JSON mode).
func (o OutputFormat) emitFindings(findings []check.Finding) {
	if !o.JSON() {
		_ = render.Text(os.Stderr, findings)
		return
	}
	env := render.Envelope{
		Tool:     "aiwf",
		Version:  version.Current().Version,
		Status:   render.StatusFor(findings),
		Findings: findings,
		Metadata: o.Metadata(nil),
	}
	_ = render.JSON(os.Stdout, env, o.Pretty)
}

// emitSuccess reports a successful verb outcome. In text mode it surfaces
// any warning-level findings to stderr (unchanged) and prints the subject
// to stdout. In JSON mode it writes an ok/findings envelope with
// result:{subject} to stdout. metadata carries the verb's own per-verb
// facts (M-0239/AC-2, e.g. entity_id/from/to, commit_sha) — nil for a
// verb that reports nothing beyond AC-1's correlation_id.
func (o OutputFormat) emitSuccess(subject string, findings []check.Finding, metadata map[string]any) {
	if !o.JSON() {
		if len(findings) > 0 {
			_ = render.Text(os.Stderr, findings)
		}
		fmt.Println(subject)
		return
	}
	env := render.Envelope{
		Tool:     "aiwf",
		Version:  version.Current().Version,
		Status:   render.StatusFor(findings),
		Findings: findings,
		Result:   map[string]any{"subject": subject},
		Metadata: o.Metadata(metadata),
	}
	_ = render.JSON(os.Stdout, env, o.Pretty)
}
