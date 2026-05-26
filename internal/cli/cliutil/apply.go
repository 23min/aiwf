package cliutil

import (
	"context"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// FinishVerb is the post-verb handler shared by every mutating
// subcommand: it surfaces the verb's outcome in the chosen output format
// (text by default, a JSON envelope under --format=json per D-0013),
// applies the plan when present, and reports the exit code.
//
// Exit-code contract (format-independent):
//   - a Coded error (entity.Code resolves) → ExitFindings (1): a
//     legality refusal, unified with the check-time exit for the same
//     violation class (D-0013, decision C2);
//   - any other verb error → ExitUsage (2);
//   - nil result / no plan / apply failure → ExitInternal (3);
//   - error-severity findings → ExitFindings (1);
//   - success (incl. NoOp, warnings) → ExitOK (0).
func FinishVerb(ctx context.Context, root, label string, result *verb.Result, err error, out OutputFormat) int {
	if err != nil {
		code, isCoded := entity.Code(err)
		out.emitErrorEnvelope(label, code, err.Error())
		if isCoded {
			return ExitFindings
		}
		return ExitUsage
	}
	if result == nil {
		out.emitErrorEnvelope(label, "", "no result returned")
		return ExitInternal
	}
	if check.HasErrors(result.Findings) {
		out.emitFindings(result.Findings)
		return ExitFindings
	}
	if result.NoOp {
		out.emitSuccess(result.NoOpMessage, nil)
		return ExitOK
	}
	if result.Plan == nil {
		out.emitErrorEnvelope(label, "", "validation passed but no plan produced")
		return ExitInternal
	}
	if applyErr := verb.Apply(ctx, root, result.Plan); applyErr != nil {
		out.emitErrorEnvelope(label, "", applyErr.Error())
		return ExitInternal
	}
	// Warning-level findings may travel with a successful plan (e.g.,
	// reallocate body-prose mentions); emitSuccess surfaces them but the
	// exit code stays clean.
	out.emitSuccess(result.Plan.Subject, result.Findings)
	return ExitOK
}
