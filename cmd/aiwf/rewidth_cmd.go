package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/verb"
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
		actor      string
		principal  string
		root       string
		apply      bool
		skipChecks bool
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
revert the resulting commit with git revert if needed.

Preflight (default): before --apply, the verb runs aiwf check and
refuses to proceed if any error-severity findings exist (broken refs,
duplicate ids, frontmatter-shape errors, id-path mismatch). Fix the
findings or re-run with --skip-checks to bypass. The verb also warns
when expected kind directories (work/epics, work/gaps, work/decisions,
work/contracts, docs/adr) are missing — advisory only; the verb still
runs.`,
		Example: `  # Preview what rewidth would do (dry-run)
  aiwf rewidth

  # Commit the canonicalization (preflight: aiwf check must be clean)
  aiwf rewidth --apply

  # Bypass preflight (consumer accepts the risk)
  aiwf rewidth --apply --skip-checks`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runRewidthCmd(actor, principal, root, apply, skipChecks))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&apply, "apply", false, "commit the canonicalization; without this flag the verb is dry-run")
	cmd.Flags().BoolVar(&skipChecks, "skip-checks", false, "skip the preflight aiwf check + layout-shape warnings (--apply only; dry-run is unaffected)")
	return cmd
}

func runRewidthCmd(actor, principal, root string, apply, skipChecks bool) int {
	rootDir, err := resolveRoot(root)
	if err != nil { //coverage:ignore resolveRoot only fails on missing aiwf.yaml + non-existent --root path; the test repo always provides one
		fmt.Fprintf(os.Stderr, "aiwf rewidth: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil { //coverage:ignore cliutil.ResolveActor only fails when actor can't be derived from any source; tests always pass --actor
		fmt.Fprintf(os.Stderr, "aiwf rewidth: %v\n", err)
		return cliutil.ExitUsage
	}

	// Provenance coherence check (mirrors `aiwf import`'s shape — bulk
	// sweep, no per-entity scope gating). A non-human actor needs a
	// principal; a human actor must not carry one.
	principalStr := strings.TrimSpace(principal)
	actorIsNonHuman := actorStr != "" && !strings.HasPrefix(actorStr, "human/")
	if actorIsNonHuman && principalStr == "" {
		fmt.Fprintf(os.Stderr, "aiwf rewidth: --principal human/<id> is required when --actor is non-human (got actor=%q)\n", actorStr)
		return cliutil.ExitUsage
	}
	if !actorIsNonHuman && principalStr != "" {
		fmt.Fprintln(os.Stderr, "aiwf rewidth: --principal is forbidden when --actor is human/ (humans act directly)")
		return cliutil.ExitUsage
	}

	// Dry-run is read-only; lock only when we'd write.
	if apply {
		release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf rewidth")
		if release == nil { //coverage:ignore cliutil.AcquireRepoLock only returns nil on lock-contention from a concurrent verb invocation; not reproducible in serial tests
			return rc
		}
		defer release()
	}

	ctx := context.Background()

	// Preflight (--apply only, unless --skip-checks is set):
	//   - Layout-shape warning: stat the expected kind directories;
	//     warn on each missing one (advisory; the verb still runs).
	//   - aiwf check error-severity gate: load the tree, run check.Run,
	//     refuse the migration if any error-severity finding fires.
	// Dry-run skips preflight: the dry-run output is itself a preflight
	// surface, and the operator may want to see the plan even on a
	// less-than-clean tree to understand the migration shape.
	if apply && !skipChecks {
		if rc := rewidthPreflight(ctx, rootDir); rc != cliutil.ExitOK {
			return rc
		}
	}

	result, err := verb.Rewidth(ctx, rootDir, actorStr)
	if err != nil { //coverage:ignore verb.Rewidth only errors on filesystem failures; tempdir-based tests can't reproduce
		fmt.Fprintf(os.Stderr, "aiwf rewidth: %v\n", err)
		return cliutil.ExitInternal
	}
	if result == nil { //coverage:ignore Rewidth always returns a non-nil Result on success path; defensive against future API drift
		fmt.Fprintln(os.Stderr, "aiwf rewidth: no result returned")
		return cliutil.ExitInternal
	}

	if result.NoOp {
		fmt.Println(result.NoOpMessage)
		return cliutil.ExitOK
	}
	if result.Plan == nil { //coverage:ignore non-NoOp result without a Plan is unreachable today; defensive against future API drift
		fmt.Fprintln(os.Stderr, "aiwf rewidth: validation passed but no plan produced")
		return cliutil.ExitInternal
	}

	if !apply {
		// Dry-run: print summary, no writes.
		printRewidthDryRun(result.Plan)
		return cliutil.ExitOK
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
		return cliutil.ExitInternal
	}
	if len(result.Findings) > 0 { //coverage:ignore Rewidth currently never populates Findings (validation is downstream — `aiwf check` post-commit); defensive surface for future projection-finding adoption
		_ = render.Text(os.Stderr, result.Findings)
	}
	fmt.Println(result.Plan.Subject)
	return cliutil.ExitOK
}

// rewidthExpectedDirs is the set of kind directories a typical
// aiwf-managed consumer carries. The preflight stats each and warns
// on misses — advisory, since a consumer might legitimately have no
// gaps or no contracts. Helps a fresh consumer notice if they're
// running rewidth in a directory that isn't actually aiwf-managed.
var rewidthExpectedDirs = []string{
	"work/epics",
	"work/gaps",
	"work/decisions",
	"work/contracts",
	"docs/adr",
}

// rewidthPreflight runs the default preflight before --apply: warn on
// missing expected kind directories, then run aiwf check and refuse
// the migration if any error-severity finding fires. Returns cliutil.ExitOK on
// pass, cliutil.ExitFindings on aiwf-check errors, or cliutil.ExitInternal on a load
// failure.
//
// Layout warnings print to stderr but never block. The verb still
// runs even if every expected dir is missing — the operator opted in
// by typing `aiwf rewidth --apply`.
//
// The check gate uses `check.Run(tr, loadErrs)` only — contracts and
// provenance audits are out of scope for "is the tree in a known-valid
// state for migration." A consumer who needs the full pipeline can
// run `aiwf check` themselves; rewidth's preflight is a pragmatic
// subset.
func rewidthPreflight(ctx context.Context, rootDir string) int {
	missing := []string{}
	for _, rel := range rewidthExpectedDirs {
		if _, statErr := os.Stat(filepath.Join(rootDir, rel)); os.IsNotExist(statErr) {
			missing = append(missing, rel)
		}
	}
	if len(missing) == len(rewidthExpectedDirs) {
		// All expected dirs missing — almost certainly not an aiwf
		// repo. Bail with a clear usage error rather than producing
		// a confusing empty-plan or an `aiwf check` torrent.
		fmt.Fprintf(os.Stderr, "aiwf rewidth: no aiwf-managed directories found under %q\n", rootDir)
		fmt.Fprintln(os.Stderr, "aiwf rewidth:   expected at least one of: "+strings.Join(rewidthExpectedDirs, ", "))
		fmt.Fprintln(os.Stderr, "aiwf rewidth:   if this is intentional, re-run with --skip-checks")
		return cliutil.ExitUsage
	}
	for _, rel := range missing {
		fmt.Fprintf(os.Stderr, "aiwf rewidth: warning: expected directory %q is missing (advisory; the verb continues)\n", rel)
	}

	tr, loadErrs, loadErr := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if loadErr != nil { //coverage:ignore cliutil.LoadTreeWithTrunk only fails on filesystem failures; tempdir tests can't reproduce
		fmt.Fprintf(os.Stderr, "aiwf rewidth: preflight: loading tree: %v\n", loadErr)
		return cliutil.ExitInternal
	}
	findings := check.Run(tr, loadErrs)
	var errs []check.Finding
	for i := range findings {
		if findings[i].Severity == check.SeverityError {
			errs = append(errs, findings[i])
		}
	}
	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "aiwf rewidth: preflight: aiwf check has error-severity findings; refusing to --apply")
		fmt.Fprintln(os.Stderr, "aiwf rewidth:   fix the findings or re-run with --skip-checks to override")
		fmt.Fprintln(os.Stderr)
		_ = render.Text(os.Stderr, errs)
		return cliutil.ExitFindings
	}
	return cliutil.ExitOK
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
