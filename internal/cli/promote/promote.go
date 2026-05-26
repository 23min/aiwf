// Package promote implements the `aiwf promote` verb (per-verb subpackage
// of M-0115; cmd/aiwf/main.go's newRootCmd wires it via NewCmd).
package promote

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf promote <id> <new-status>` and the I2 composite
// / --phase variants:
//
//	aiwf promote E-01 active                       (top-level entity)
//	aiwf promote M-007/AC-1 met                    (composite, status mode)
//	aiwf promote M-007/AC-1 --phase green          (composite, phase mode)
//
// --phase is mutex with the positional new-status: pass one or the
// other, never both. --phase is only valid for composite ids; using
// it on a top-level entity is a usage error.
func NewCmd() *cobra.Command {
	var (
		actor        string
		principal    string
		root         string
		reason       string
		phase        string
		tests        string
		by           string
		byCommit     string
		supersededBy string
		force        bool
		auditOnly    bool
		out          *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "promote <id> [new-status]",
		Short: "Advance an entity's status (or AC tdd_phase via --phase)",
		Example: `  # Move an epic from proposed to active
  aiwf promote E-01 active

  # Mark an acceptance criterion as met
  aiwf promote M-007/AC-1 met

  # Advance an AC's TDD phase
  aiwf promote M-007/AC-1 --phase green --tests "pass=12 fail=0 skip=0"`,
		Args:          cobra.RangeArgs(1, 2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args, actor, principal, root, reason,
				phase, tests, by, byCommit, supersededBy, force, auditOnly, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().StringVar(&phase, "phase", "", "advance an AC's tdd_phase (composite ids only; mutex with positional new-status)")
	cmd.Flags().StringVar(&tests, "tests", "", `optional test metrics for a phase promotion (composite + --phase only); format: "pass=N fail=N skip=N total=N" — keys must be one of pass/fail/skip/total, integers non-negative`)
	cmd.Flags().StringVar(&by, "by", "", "comma-separated entity ids to write into addressed_by (gap → addressed only); satisfies gap-addressed-has-resolver atomically with the status change")
	cmd.Flags().StringVar(&byCommit, "by-commit", "", "comma-separated commit SHAs to write into addressed_by_commit (gap → addressed only); use when the gap was closed by a specific commit rather than a milestone")
	cmd.Flags().StringVar(&supersededBy, "superseded-by", "", "ADR id to write into superseded_by (adr → superseded only); satisfies adr-supersession-mutual atomically with the status change")
	cmd.Flags().BoolVar(&force, "force", false, "skip the FSM transition rule (requires --reason); coherence checks still run")
	cmd.Flags().BoolVar(&auditOnly, "audit-only", false, "record an audit-trail commit without mutating files; entity must already be at <new-status> (requires --reason; mutex with --force; G24 recovery path)")
	out = cliutil.AddFormatFlags(cmd)
	cmd.ValidArgsFunction = func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			return cliutil.CompleteEntityIDs("")
		case 1:
			// args[0] is the entity id; the kind is determined by the
			// id prefix (E-/M-/ADR-/G-/D-/C-) without loading the tree.
			// Composite ids return nil — phase advancement uses --phase
			// rather than a positional new-status, so completion is a
			// no-op.
			if statuses := cliutil.StatusesForID(args[0]); len(statuses) > 0 {
				return statuses, cobra.ShellCompDirectiveNoFileComp
			}
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	_ = cmd.RegisterFlagCompletionFunc("phase", cobra.FixedCompletions(
		[]string{"red", "green", "refactor", "done"},
		cobra.ShellCompDirectiveNoFileComp,
	))
	_ = cmd.RegisterFlagCompletionFunc("by", cliutil.CompleteEntityIDFlag(""))
	_ = cmd.RegisterFlagCompletionFunc("superseded-by", cliutil.CompleteEntityIDFlag(entity.KindADR))
	return cmd
}

// Run executes `aiwf promote`. Returns one of the cliutil.Exit* codes.
func Run(args []string, actor, principal, root, reason,
	phase, tests, by, byCommit, supersededBy string, force, auditOnly bool, out cliutil.OutputFormat,
) int {
	id := args[0]

	phaseMode := phase != ""
	switch {
	case phaseMode && len(args) == 2:
		fmt.Fprintln(os.Stderr, "aiwf promote: --phase is mutex with the positional new-status; pass one or the other")
		return cliutil.ExitUsage
	case phaseMode && !entity.IsCompositeID(id):
		fmt.Fprintf(os.Stderr, "aiwf promote: --phase is only valid for composite ids (M-NNN/AC-N); got %q\n", id)
		return cliutil.ExitUsage
	case !phaseMode && len(args) != 2:
		fmt.Fprintln(os.Stderr, "aiwf promote: missing new-status. Usage: aiwf promote <id> <new-status>")
		return cliutil.ExitUsage
	}

	if force && auditOnly {
		fmt.Fprintln(os.Stderr, "aiwf promote: --force and --audit-only cannot coexist (force makes a transition; audit-only records one that already happened)")
		return cliutil.ExitUsage
	}
	if (force || auditOnly) && strings.TrimSpace(reason) == "" {
		gateFlag := "--force"
		if auditOnly {
			gateFlag = "--audit-only"
		}
		fmt.Fprintf(os.Stderr, "aiwf promote: --reason \"...\" is required when %s is set (non-empty after trim)\n", gateFlag)
		return cliutil.ExitUsage
	}

	resolverOpts := verb.PromoteOptions{
		AddressedBy:       cliutil.SplitCommaList(by),
		AddressedByCommit: cliutil.SplitCommaList(byCommit),
		SupersededBy:      strings.TrimSpace(supersededBy),
	}
	resolverSet := len(resolverOpts.AddressedBy) > 0 || len(resolverOpts.AddressedByCommit) > 0 || resolverOpts.SupersededBy != ""
	if resolverSet && auditOnly {
		fmt.Fprintln(os.Stderr, "aiwf promote: --by/--by-commit/--superseded-by are not allowed with --audit-only (audit-only records an existing transition; resolver-flag values would imply a mutation)")
		return cliutil.ExitUsage
	}
	if resolverSet && phase != "" {
		fmt.Fprintln(os.Stderr, "aiwf promote: --by/--by-commit/--superseded-by are not valid in phase mode (resolver fields apply to entity status, not AC phase)")
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf promote: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf promote: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf promote")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf promote: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}

	if phaseMode {
		metrics, mErr := cliutil.ParseTestsFlag(tests, "aiwf promote")
		if mErr != nil {
			return cliutil.ExitUsage
		}
		var result *verb.Result
		var vErr error
		if auditOnly {
			if metrics != nil {
				fmt.Fprintln(os.Stderr, "aiwf promote: --tests is not allowed with --audit-only (audit-only records an existing transition; no test cycle ran)")
				return cliutil.ExitUsage
			}
			result, vErr = verb.PromoteACPhaseAuditOnly(ctx, tr, id, phase, actorStr, reason)
		} else {
			result, vErr = verb.PromoteACPhase(ctx, tr, id, phase, actorStr, reason, force, metrics)
		}
		return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf promote", tr, result, vErr, pctx, out)
	}
	if strings.TrimSpace(tests) != "" {
		fmt.Fprintln(os.Stderr, "aiwf promote: --tests is only valid in phase mode (composite id with --phase <p>)")
		return cliutil.ExitUsage
	}
	newStatus := args[1]
	if !entity.IsCompositeID(id) {
		if e := tr.ByID(id); e != nil {
			pctx.IsTerminalPromote = cliutil.IsTerminalPromote(e.Kind, newStatus)
		}
	}
	if auditOnly {
		result, vErr := verb.PromoteAuditOnly(ctx, tr, id, newStatus, actorStr, reason)
		return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf promote", tr, result, vErr, pctx, out)
	}
	result, vErr := verb.Promote(ctx, tr, id, newStatus, actorStr, reason, force, resolverOpts)
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf promote", tr, result, vErr, pctx, out)
}
