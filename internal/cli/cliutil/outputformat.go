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
// as before the milestone.
type OutputFormat struct {
	Format string
	Pretty bool
}

// JSON reports whether a JSON envelope was requested (--format=json).
func (o OutputFormat) JSON() bool { return o.Format == "json" }

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
		Tool:    "aiwf",
		Version: version.Current().Version,
		Status:  "error",
		Error:   &render.EnvelopeError{Code: code, Message: message},
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
	}
	_ = render.JSON(os.Stdout, env, o.Pretty)
}

// emitSuccess reports a successful verb outcome. In text mode it surfaces
// any warning-level findings to stderr (unchanged) and prints the subject
// to stdout. In JSON mode it writes an ok/findings envelope with
// result:{subject} to stdout.
func (o OutputFormat) emitSuccess(subject string, findings []check.Finding) {
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
	}
	_ = render.JSON(os.Stdout, env, o.Pretty)
}
