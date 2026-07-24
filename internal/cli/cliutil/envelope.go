package cliutil

import (
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/version"
)

// newEnvelope returns a render.Envelope with the two invariant identity
// fields filled: Tool ("aiwf") and Version (the running binary's
// version). It is the single source of truth for the aiwf/<version>
// pair every --format=json envelope carries — the constant that used to
// be hand-repeated at every emit* method and every read verb's JSON
// branch. Callers (the three typed helpers below, plus emitSuccess's
// hybrid findings+result shape) fill Status and the payload fields.
func newEnvelope() render.Envelope {
	return render.Envelope{
		Tool:    "aiwf",
		Version: version.Current().Version,
	}
}

// OKEnvelope builds a success envelope carrying `result` as the payload
// (status "ok", no findings) — the read-verb counterpart to the mutating
// side's emitSuccess. `metadata` is written through verbatim (nil is
// dropped by the Envelope's omitempty tag), matching each read verb's
// existing per-verb metadata map.
func OKEnvelope(result any, metadata map[string]any) render.Envelope {
	env := newEnvelope()
	env.Status = "ok"
	env.Result = result
	env.Metadata = metadata
	return env
}

// FindingsEnvelope builds a findings envelope — status is "ok" when the
// list is empty and "findings" otherwise (render.StatusFor), carrying the
// findings and `metadata`. It is the read-side counterpart to emitFindings
// and the shape `aiwf check` emits.
func FindingsEnvelope(findings []check.Finding, metadata map[string]any) render.Envelope {
	env := newEnvelope()
	env.Status = render.StatusFor(findings)
	env.Findings = findings
	env.Metadata = metadata
	return env
}

// ErrorEnvelope builds a terminal-error envelope (status "error") carrying
// a structured code (omitted when empty) and message, plus `metadata`. It
// is the counterpart to emitErrorEnvelope, usable by read verbs that
// report a not-found or validation error in JSON mode.
func ErrorEnvelope(code, message string, metadata map[string]any) render.Envelope {
	env := newEnvelope()
	env.Status = "error"
	env.Error = &render.EnvelopeError{Code: code, Message: message}
	env.Metadata = metadata
	return env
}
