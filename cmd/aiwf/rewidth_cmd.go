package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/render"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// newRewidthCmd builds `aiwf rewidth [--apply] [--root <path>]`.
//
// Default invocation (no `--apply`) is dry-run: the verb computes a
// Plan and the dispatcher prints a per-kind summary of what would
// happen. `--apply` runs verb.Apply against the same Plan, producing
// exactly one git commit per kernel principle #7 with trailer
// `aiwf-verb: rewidth` (no `aiwf-entity:` trailer — multi-entity
// sweep, same shape as `aiwf archive`).
//
// Per ADR-0008, this is a one-shot ritual: the canonical case for the
// "no skill when --help suffices" branch of ADR-0006. Every consumer
// runs it at most once after upgrading past the kernel version that
// declares the canonical-width policy. Idempotent re-runs are
// no-ops.
func newRewidthCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		apply     bool
	)
	cmd := &cobra.Command{
		Use:   "rewidth [--apply]",
		Short: "Sweep narrow-width entity ids to canonical 4-digit form (one-shot per consumer; per ADR-0008)",
		Long: `Sweep narrow-width (E-NN, M-NNN, G-NNN, D-NNN, C-NNN) entity ids
in the active planning tree to canonical 4-digit form (E-NNNN, etc.),
per ADR-0008. Files under <kind>/archive/ are preserved. Default is
dry-run; --apply commits the canonicalization as a single commit with
trailer aiwf-verb: rewidth.

Idempotent: an already-canonical or empty tree is a no-op. The
reverse path (canonical -> narrow) is intentionally not implemented;
revert the resulting commit with git revert if needed.`,
		Example: `  # Preview what rewidth would do (dry-run)
  aiwf rewidth

  # Commit the canonicalization
  aiwf rewidth --apply`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runRewidthCmd(actor, principal, root, apply))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&apply, "apply", false, "commit the canonicalization; without this flag the verb is dry-run")
	return cmd
}

func runRewidthCmd(actor, principal, root string, apply bool) int {
	rootDir, err := resolveRoot(root)
	if err != nil { //coverage:ignore resolveRoot only fails on missing aiwf.yaml + non-existent --root path; the test repo always provides one
		fmt.Fprintf(os.Stderr, "aiwf rewidth: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
	if err != nil { //coverage:ignore resolveActor only fails when actor can't be derived from any source; tests always pass --actor
		fmt.Fprintf(os.Stderr, "aiwf rewidth: %v\n", err)
		return exitUsage
	}

	// Provenance coherence check (mirrors `aiwf import`'s shape — bulk
	// sweep, no per-entity scope gating). A non-human actor needs a
	// principal; a human actor must not carry one.
	principalStr := strings.TrimSpace(principal)
	actorIsNonHuman := actorStr != "" && !strings.HasPrefix(actorStr, "human/")
	if actorIsNonHuman && principalStr == "" {
		fmt.Fprintf(os.Stderr, "aiwf rewidth: --principal human/<id> is required when --actor is non-human (got actor=%q)\n", actorStr)
		return exitUsage
	}
	if !actorIsNonHuman && principalStr != "" {
		fmt.Fprintln(os.Stderr, "aiwf rewidth: --principal is forbidden when --actor is human/ (humans act directly)")
		return exitUsage
	}

	// Dry-run is read-only; lock only when we'd write.
	if apply {
		release, rc := acquireRepoLock(rootDir, "aiwf rewidth")
		if release == nil { //coverage:ignore acquireRepoLock only returns nil on lock-contention from a concurrent verb invocation; not reproducible in serial tests
			return rc
		}
		defer release()
	}

	ctx := context.Background()
	result, err := verb.Rewidth(ctx, rootDir, actorStr)
	if err != nil { //coverage:ignore verb.Rewidth only errors on filesystem failures; tempdir-based tests can't reproduce
		fmt.Fprintf(os.Stderr, "aiwf rewidth: %v\n", err)
		return exitInternal
	}
	if result == nil { //coverage:ignore Rewidth always returns a non-nil Result on success path; defensive against future API drift
		fmt.Fprintln(os.Stderr, "aiwf rewidth: no result returned")
		return exitInternal
	}

	if result.NoOp {
		fmt.Println(result.NoOpMessage)
		return exitOK
	}
	if result.Plan == nil { //coverage:ignore non-NoOp result without a Plan is unreachable today; defensive against future API drift
		fmt.Fprintln(os.Stderr, "aiwf rewidth: validation passed but no plan produced")
		return exitInternal
	}

	if !apply {
		// Dry-run: print summary, no writes.
		printRewidthDryRun(result.Plan)
		return exitOK
	}

	// Stamp principal trailer when the operator is non-human, mirroring
	// import's bulk-sweep shape.
	if actorIsNonHuman {
		result.Plan.Trailers = append(result.Plan.Trailers, gitops.Trailer{
			Key:   gitops.TrailerPrincipal,
			Value: principalStr,
		})
	}

	if applyErr := verb.Apply(ctx, rootDir, result.Plan); applyErr != nil { //coverage:ignore Apply only errors on git mv/commit failures; tests don't reproduce a corrupted repo state
		fmt.Fprintf(os.Stderr, "aiwf rewidth: %v\n", applyErr)
		return exitInternal
	}
	if len(result.Findings) > 0 { //coverage:ignore Rewidth currently never populates Findings (validation is downstream — `aiwf check` post-commit); defensive surface for future projection-finding adoption
		_ = render.Text(os.Stderr, result.Findings)
	}
	fmt.Println(result.Plan.Subject)
	return exitOK
}

// printRewidthDryRun prints a human-readable summary of the planned
// renames + body rewrites. Stdout, not stderr — the user reads this
// to decide whether to re-run with --apply.
func printRewidthDryRun(p *verb.Plan) {
	fmt.Println(p.Subject + " (dry-run; re-run with --apply to commit)")
	if p.Body != "" {
		fmt.Println()
		fmt.Print(p.Body)
	}
	// Per-op detail keeps the verb self-documenting in CI logs and
	// shell sessions.
	moves := 0
	writes := 0
	for _, op := range p.Ops {
		switch op.Type {
		case verb.OpMove:
			moves++
		case verb.OpWrite:
			writes++
		}
	}
	fmt.Println()
	fmt.Println("Operations:")
	for _, op := range p.Ops {
		switch op.Type {
		case verb.OpMove:
			fmt.Printf("  rename  %s -> %s\n", op.Path, op.NewPath)
		case verb.OpWrite:
			fmt.Printf("  rewrite %s (%d bytes)\n", op.Path, len(op.Content))
		}
	}
}
