package cliutil

import (
	"context"
	"fmt"
	"os"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/verb"
)

// FinishVerb is the post-verb handler shared by every mutating
// subcommand: it surfaces a Go error as a usage error, renders any
// findings, applies the plan when present, and prints a one-line
// summary on success. NoOp results bypass the apply path entirely
// and print NoOpMessage on stdout.
func FinishVerb(ctx context.Context, root, label string, result *verb.Result, err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
		return ExitUsage
	}
	if result == nil {
		fmt.Fprintf(os.Stderr, "%s: no result returned\n", label)
		return ExitInternal
	}
	if check.HasErrors(result.Findings) {
		_ = render.Text(os.Stderr, result.Findings)
		return ExitFindings
	}
	if result.NoOp {
		fmt.Println(result.NoOpMessage)
		return ExitOK
	}
	if result.Plan == nil {
		fmt.Fprintf(os.Stderr, "%s: validation passed but no plan produced\n", label)
		return ExitInternal
	}
	if applyErr := verb.Apply(ctx, root, result.Plan); applyErr != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, applyErr)
		return ExitInternal
	}
	if len(result.Findings) > 0 {
		// Warning-level findings travel with a successful plan
		// (e.g., reallocate body-prose mentions). Surface them but
		// keep the exit code clean.
		_ = render.Text(os.Stderr, result.Findings)
	}
	fmt.Println(result.Plan.Subject)
	return ExitOK
}
