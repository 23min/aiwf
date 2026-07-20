// Package rewidth implements the `aiwf rewidth` verb (per-verb subpackage of M-0116;
// cmd/aiwf/main.go newRootCmd wires it via NewCmd).
package rewidth

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf rewidth [--apply] [--root <path>]`.
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
func NewCmd(correlationID string) *cobra.Command {
	var (
		actor      string
		principal  string
		root       string
		apply      bool
		skipChecks bool
		out        *cliutil.OutputFormat
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
			return cliutil.WrapExitCode(Run(actor, principal, root, apply, skipChecks, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&apply, "apply", false, "commit the canonicalization; without this flag the verb is dry-run")
	cmd.Flags().BoolVar(&skipChecks, "skip-checks", false, "skip the preflight aiwf check + layout-shape warnings (--apply only; dry-run is unaffected)")
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	return cmd
}

// Run executes `aiwf rewidth`. Returns one of the cliutil.Exit* codes.
func Run(actor, principal, root string, apply, skipChecks bool, out cliutil.OutputFormat) (code int) {
	ctx := context.Background()

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore cliutil.ResolveRoot only fails on missing aiwf.yaml + non-existent --root path; the test repo always provides one
		code, _ = cliutil.FinishVerbOutcome(ctx, root, "aiwf rewidth", nil, err, out)
		return code
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil { //coverage:ignore cliutil.ResolveActor only fails when actor can't be derived from any source; tests always pass --actor
		code, _ = cliutil.FinishVerbOutcome(ctx, rootDir, "aiwf rewidth", nil, err, out)
		return code
	}

	// M-0249: diagnostic-logging wiring, mirroring cancel.Run's own
	// M-0238/AC-5 pattern. rewidth is a multi-entity sweep (no single
	// TargetID), so entity stays empty, matching archive/import.
	diagLog, closeDiagLog := cliutil.ResolveLogger(rootDir, os.Getenv)
	defer func() { _ = closeDiagLog() }()
	if diagLog.Enabled(ctx, slog.LevelInfo) {
		runID := out.CorrelationID
		if runID == "" {
			runID = logger.NewRunID()
		}
		diagLog = logger.WithVerb(diagLog, "rewidth", "", actorStr, runID)
	}
	var sha string
	defer func() { cliutil.EmitVerbOutcome(diagLog, "verb", code, sha) }()

	// Provenance coherence check (mirrors `aiwf import`'s shape — bulk
	// sweep, no per-entity scope gating). A non-human actor needs a
	// principal; a human actor must not carry one.
	principalStr := strings.TrimSpace(principal)
	actorIsNonHuman := actorStr != "" && !strings.HasPrefix(actorStr, "human/")
	if actorIsNonHuman && principalStr == "" {
		code, _ = cliutil.FinishVerbOutcome(ctx, rootDir, "aiwf rewidth", nil, fmt.Errorf("--principal human/<id> is required when --actor is non-human (got actor=%q)", actorStr), out)
		return code
	}
	if !actorIsNonHuman && principalStr != "" {
		code, _ = cliutil.FinishVerbOutcome(ctx, rootDir, "aiwf rewidth", nil, fmt.Errorf("--principal is forbidden when --actor is human/ (humans act directly)"), out)
		return code
	}

	// Dry-run is read-only; lock only when we'd write.
	if apply {
		release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf rewidth", out)
		if release == nil { //coverage:ignore cliutil.AcquireRepoLock only returns nil on lock-contention from a concurrent verb invocation; not reproducible in serial tests
			return rc
		}
		defer release()
	}

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
		code, _ = cliutil.FinishVerbOutcome(ctx, rootDir, "aiwf rewidth", nil, cliutil.ErrInternal(err), out)
		return code
	}

	outcome := &cliutil.Outcome{}
	if result != nil {
		outcome.NoOp = result.NoOp
		outcome.NoOpMessage = result.NoOpMessage
		if result.Plan != nil {
			// Stamp principal trailer when the operator is non-human,
			// mirroring import's bulk-sweep shape. Harmless on the
			// dry-run branch below — FinishVerbOutcome never applies a
			// Plan whose DryRun is set.
			if apply && actorIsNonHuman {
				result.Plan.Trailers = append(result.Plan.Trailers, gitops.Trailer{
					Key:   gitops.TrailerPrincipal,
					Value: principalStr,
				})
			}
			outcome.Plans = []*verb.Plan{result.Plan}
			outcome.Findings = result.Findings
			outcome.Metadata = result.Metadata
			if !apply {
				outcome.DryRun = true
				outcome.Subject = result.Plan.Subject + " (dry-run; re-run with --apply to commit)"
				outcome.TextDetail = func() { printRewidthDryRun(outcome.Subject, result.Plan) }
			}
		}
	} else {
		outcome = nil //coverage:ignore Rewidth always returns a non-nil Result on success path; defensive against future API drift
	}

	code, sha = cliutil.FinishVerbOutcome(ctx, rootDir, "aiwf rewidth", outcome, nil, out)
	return code
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
		cliutil.Errorf("aiwf rewidth: no aiwf-managed directories found under %q\n", rootDir)
		cliutil.Errorln("aiwf rewidth:   expected at least one of: " + strings.Join(rewidthExpectedDirs, ", "))
		cliutil.Errorln("aiwf rewidth:   if this is intentional, re-run with --skip-checks")
		return cliutil.ExitUsage
	}
	for _, rel := range missing {
		cliutil.Errorf("aiwf rewidth: warning: expected directory %q is missing (advisory; the verb continues)\n", rel)
	}

	tr, loadErrs, loadErr := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if loadErr != nil { //coverage:ignore cliutil.LoadTreeWithTrunk only fails on filesystem failures; tempdir tests can't reproduce
		cliutil.Errorf("aiwf rewidth: preflight: loading tree: %v\n", loadErr)
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
		cliutil.Errorln("aiwf rewidth: preflight: aiwf check has error-severity findings; refusing to --apply")
		cliutil.Errorln("aiwf rewidth:   fix the findings or re-run with --skip-checks to override")
		cliutil.Errorln()
		_ = render.Text(os.Stderr, errs)
		return cliutil.ExitFindings
	}
	return cliutil.ExitOK
}

// printRewidthDryRun prints a human-readable summary of the planned
// renames + body rewrites. Stdout, not stderr — the user reads this
// to decide whether to re-run with --apply. subject is the caller's
// already-computed dry-run subject (the same string riding in
// Outcome.Subject) so the two never drift independently.
func printRewidthDryRun(subject string, p *verb.Plan) {
	cliutil.Println(subject)
	if p.Body != "" {
		cliutil.Println()
		cliutil.Print(p.Body)
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
	cliutil.Println()
	cliutil.Println("Operations:")
	for _, op := range p.Ops {
		switch op.Type {
		case verb.OpMove:
			cliutil.Printf("  rename  %s -> %s\n", op.Path, op.NewPath)
		case verb.OpWrite:
			cliutil.Printf("  rewrite %s (%d bytes)\n", op.Path, len(op.Content))
		}
	}
}
