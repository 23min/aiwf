package check

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/contract"
	"github.com/23min/aiwf/internal/config"
	baserender "github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/version"
)

// NewCmd builds `aiwf check`: validate the consumer repo's planning
// state. Read-only; produces no commit. The pre-push git hook runs
// this verb — its findings + exit code are the framework's
// authoritative correctness gate.
func NewCmd() *cobra.Command {
	var (
		root      string
		format    string
		pretty    bool
		since     string
		shapeOnly bool
		verbose   bool
		commitMsg string
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate the consumer repo's planning state",
		Example: `  # Default: errors per-instance, warnings collapsed to a per-code summary
  aiwf check

  # Restore the full per-instance shape (one line per finding) for warnings too
  aiwf check --verbose

  # Emit a JSON envelope for CI scripts (always per-instance regardless of --verbose)
  aiwf check --format=json --pretty`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			verbs := enumerateRegisteredVerbs(c.Root())
			if commitMsg != "" {
				return cliutil.WrapExitCode(runCommitMsg(commitMsg, verbs, c.ErrOrStderr()))
			}
			return cliutil.WrapExitCode(Run(root, format, pretty, since, shapeOnly, verbose, verbs))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: discover via aiwf.yaml)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	cmd.Flags().StringVar(&since, "since", "", "explicit base ref for the provenance untrailered-entity audit (default: @{u} when set, else skipped)")
	cmd.Flags().BoolVar(&shapeOnly, "shape-only", false, "run only the tree-discipline rule (skips trunk read, provenance audit, contract validation); used by the pre-commit hook for a fast LLM-loop check")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "print one line per warning instance instead of the per-code summary; errors are always per-instance regardless")
	cmd.Flags().StringVar(&commitMsg, "commit-msg", "", "validate aiwf-verb trailers in the named commit-message file and exit; refuses values outside the Cobra verb tree ∪ ritualVerbs (used by the .git/hooks/commit-msg hook installed by aiwf init/update — G-0218)")
	cliutil.RegisterFormatCompletion(cmd)
	return cmd
}

// Run is the check verb's body. Loads the tree, runs every rule
// (pure-tree + provenance + tests-metrics + contracts + tree
// discipline), applies aiwf.yaml-driven severity bumps, renders the
// findings in the chosen format, and returns the exit code.
func Run(root, format string, pretty bool, since string, shapeOnly, verbose bool, registeredVerbs map[string]struct{}) int {
	if format != "text" && format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf check: --format must be 'text' or 'json', got %q\n", format)
		return cliutil.ExitUsage
	}
	if pretty && format != "json" {
		fmt.Fprintln(os.Stderr, "aiwf check: --pretty has no effect without --format=json")
	}

	resolved, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()
	if shapeOnly {
		return runShapeOnly(ctx, resolved, format, pretty)
	}

	tr, loadErrs, err := cliutil.LoadTreeWithTrunk(ctx, resolved)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	findings := check.Run(tr, loadErrs)

	contracts, contractErr := cliutil.LoadContractsBlock(resolved)
	if contractErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", contractErr)
		return cliutil.ExitInternal
	}
	contractFindings := contract.RunValidation(ctx, tr, resolved, contracts)
	findings = append(findings, contractFindings...)

	// M-0159/AC-3: compute the retroactive-acknowledgment SHA set
	// once per check invocation, then pass it to every rule that
	// consumes it. The single-compute invariant is policed by
	// internal/policies/acks_helper_lift.go; rule-internal recompute
	// is forbidden (violation class 3c). The four consumers below
	// are RunProvenanceCheck (which forwards to RunIsolationEscape,
	// RunTrailerVerbUnknown, and RunIDRenameUntrailered — the
	// fourth added at M-0160/AC-4) and FSMHistoryConsistent.
	ackedSHAs := check.WalkAcknowledgedSHAs(ctx, resolved)

	// G-0231 item 3: per-(SHA, entity) ack set, consumed only by
	// RunUntrailedAudit (the seventh ack consumer, added in this
	// item). Distinct from the per-SHA blanket set above because
	// the rule's findings are per-(commit, entity) — requiring a
	// per-pair ack rather than a SHA-only blanket. Same
	// single-compute / cascading-pass-through pattern.
	ackedSHAEntities := check.WalkAcknowledgedSHAEntities(ctx, resolved)

	// G-0218 Patch 2: compute the post-cutoff SHA set once per check
	// invocation, then pass it to RunProvenanceCheck (which forwards
	// to RunTrailerVerbUnknown). Mirrors the ackedSHAs single-compute
	// / cascading-pass-through pattern. nil-fallback for unreachable
	// HookInstallSHA (shallow clone, fork divergence) preserves the
	// G-0150 baseline.
	postCutoffSHAs := check.WalkPostCutoffSHAs(ctx, resolved)

	provenanceFindings, pErr := RunProvenanceCheck(ctx, resolved, tr, since, registeredVerbs, ackedSHAs, ackedSHAEntities, postCutoffSHAs)
	if pErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", pErr)
		return cliutil.ExitInternal
	}
	findings = append(findings, provenanceFindings...)

	// FSMHistoryConsistent walks per-entity git history in DAG order
	// (per-parent comparison, not linearization adjacency) and emits
	// findings for status transitions that violate the per-kind FSM.
	// Lives in the CLI layer rather than check.Run because the per-
	// entity git walk is too expensive for the pre-commit hook's
	// shape-only policy path. M-0130 lands the predicates
	// incrementally; AC-1 wires the walker without emitting any
	// findings yet.
	findings = append(findings, check.FSMHistoryConsistent(ctx, resolved, tr, ackedSHAs)...)

	requireMetrics := false
	var treeAllow []string
	treeStrict := false
	tddStrict := false
	archiveThreshold := 0
	archiveThresholdSet := false
	var areaMembers []string
	var areaPaths []check.AreaPaths
	areaRequired := false
	if cfg, cfgErr := config.Load(resolved); cfgErr == nil && cfg != nil {
		requireMetrics = cfg.TDD.RequireTestMetrics
		treeAllow = cfg.Tree.AllowPaths
		treeStrict = cfg.Tree.Strict
		tddStrict = cfg.TDD.Strict
		archiveThreshold, archiveThresholdSet = cfg.ArchiveSweepThreshold()
		areaMembers = cfg.Areas.MemberNames()
		areaRequired = cfg.Areas.Required
		// Project the declared members to the check package's
		// config-agnostic AreaPaths so the path-axis rules (dead-glob)
		// stay free of any aiwf.yaml type, the M-0171/AC-4 boundary.
		for _, m := range cfg.Areas.Members {
			areaPaths = append(areaPaths, check.AreaPaths{Name: m.Name, Paths: m.Paths})
		}
	}
	metricsFindings, mErr := RunTestsMetricsCheck(ctx, resolved, tr, requireMetrics)
	if mErr != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: %v\n", mErr)
		return cliutil.ExitInternal
	}
	findings = append(findings, metricsFindings...)

	// G-0155: detect misset `core.worktree` before any verb can be
	// confused by it. The chokepoint catches the silent failure mode
	// where git operations get redirected against the wrong worktree.
	findings = append(findings, RunGitConfigCheck(ctx, resolved)...)

	findings = append(findings, check.TreeDiscipline(tr, treeAllow, treeStrict)...)

	// M-0172: area-unknown is a config-dependent tree rule composed here
	// (not in the pure check.Run) with the declared set from aiwf.yaml:
	// areas.members — the same CLI-layer seam TreeDiscipline uses. Inert
	// when no areas block is declared (empty member set).
	findings = append(findings, check.AreaUnknown(tr, areaMembers)...)

	// M-0178: area-required is the present-at-all chokepoint for the 1:1
	// monorepo — a config-dependent tree rule composed here (not in the
	// pure check.Run) with the declared set and the `areas.required` bool
	// from aiwf.yaml. Inert (emits nothing) when required is false or no
	// areas block is declared.
	findings = append(findings, check.AreaRequired(tr, areaMembers, areaRequired)...)

	// M-0180: area-dead-glob is the path-claim half of the area matrix —
	// a config-dependent tree rule composed here (not in the pure
	// check.Run) with the declared areas' path globs from aiwf.yaml. It
	// reads the filesystem read-only and fires when a declared glob locates
	// nothing. Inert when no member declares a `paths:` glob.
	findings = append(findings, check.AreaDeadGlob(tr, areaPaths)...)

	// M-066/AC-2: aiwf.yaml: tdd.strict bumps entity-body-empty
	// (and any future TDD-strict-covered finding) from warning to
	// error so the pre-push hook blocks the push.
	check.ApplyTDDStrict(findings, tddStrict)

	// M-0178/AC-7: aiwf.yaml: areas.required bumps area-unknown from
	// warning to error so the pre-push hook blocks a present-but-
	// undeclared area too. Composed here (not in the pure check.Run)
	// where areaRequired is in scope — the same seam ApplyTDDStrict
	// uses. With required off, area-unknown stays a warning.
	check.ApplyAreaRequiredStrict(findings, areaRequired)

	// M-0088/AC-2: aiwf.yaml: archive.sweep_threshold bumps the
	// aggregate `archive-sweep-pending` finding from warning to
	// error when the pending-sweep count exceeds the consumer's
	// declared ceiling. The count is the same value the rule's
	// Message already names — computed once via CountPendingSweep
	// so the bumper does not re-iterate the tree.
	check.ApplyArchiveSweepThreshold(findings, archiveThreshold, archiveThresholdSet, check.CountPendingSweep(tr))

	contract.ApplyHintsLikeRun(findings)
	check.SortFindings(findings)

	switch format {
	case "text":
		// M-0089 AC-1/AC-2/AC-3: default text mode collapses warnings
		// into a per-code summary while keeping errors per-instance;
		// --verbose restores the full per-instance shape (byte-for-byte
		// identical to the pre-M-0089 output). JSON is never affected
		// (AC-4).
		writeText := baserender.TextSummary
		if verbose {
			writeText = baserender.Text
		}
		if err := writeText(os.Stdout, findings); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	case "json":
		env := baserender.Envelope{
			Tool:     "aiwf",
			Version:  version.Current().Version,
			Status:   baserender.StatusFor(findings),
			Findings: findings,
			Metadata: map[string]any{
				"root":     resolved,
				"entities": len(tr.Entities),
				"bindings": contract.BindingCount(contracts),
				"findings": len(findings),
			},
		}
		if err := baserender.JSON(os.Stdout, env, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	}

	if check.HasErrors(findings) {
		return cliutil.ExitFindings
	}
	return cliutil.ExitOK
}

// runShapeOnly runs the tree-discipline rule and nothing else.
// Used by the pre-commit hook to give the LLM a fast, in-loop signal
// when a stray file lands under work/ — the full check.Run pipeline
// (trunk read, provenance walk, contract validation) is too slow and
// too noisy to fire on every commit, but the tree-discipline rule is
// cheap and exact. Honors `aiwf.yaml: tree.{allow_paths,strict}` the
// same way the full check does.
//
// Exit codes match `aiwf check`'s contract: 0 ok, 1 findings (errors
// present — only fires when tree.strict: true), 3 internal.
func runShapeOnly(ctx context.Context, root, format string, pretty bool) int {
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf check: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	var allow []string
	strict := false
	if cfg, cfgErr := config.Load(root); cfgErr == nil && cfg != nil {
		allow = cfg.Tree.AllowPaths
		strict = cfg.Tree.Strict
	}
	findings := check.TreeDiscipline(tr, allow, strict)
	contract.ApplyHintsLikeRun(findings)
	check.SortFindings(findings)

	switch format {
	case "text":
		if err := baserender.Text(os.Stdout, findings); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	case "json":
		env := baserender.Envelope{
			Tool:     "aiwf",
			Version:  version.Current().Version,
			Status:   baserender.StatusFor(findings),
			Findings: findings,
			Metadata: map[string]any{
				"root":       root,
				"entities":   len(tr.Entities),
				"shape_only": true,
				"findings":   len(findings),
			},
		}
		if err := baserender.JSON(os.Stdout, env, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf check: writing output: %v\n", err)
			return cliutil.ExitInternal
		}
	}

	if check.HasErrors(findings) {
		return cliutil.ExitFindings
	}
	return cliutil.ExitOK
}
