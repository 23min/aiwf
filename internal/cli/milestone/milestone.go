// Package milestone implements the `aiwf milestone` verb namespace.
// It carries two children today — depends-on (sets or clears a
// milestone's depends_on list) and tdd (sets a milestone's TDD policy
// after creation). The parent itself is non-Runnable (the kind-scoped
// namespace is forward-compatible with G-073's eventual cross-kind
// generalisation).
package milestone

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds the `aiwf milestone` parent command. One child today
// (depends-on). The parent itself is non-Runnable.
func NewCmd(correlationID string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "milestone",
		Short:         "Milestone-scoped verbs",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newDependsOnCmd(correlationID))
	cmd.AddCommand(newTDDCmd(correlationID))
	return cmd
}

// newTDDCmd builds `aiwf milestone tdd <M-id> --policy
// none|advisory|required [--reason "..."]`. The post-creation mutator
// for a milestone's TDD policy, mirroring the depends-on subverb's
// shape and closing the `tdd:` slice of G-0168. Gating is
// uniform-ordinary (D-0048): any actor may flip the policy in either
// direction with no `--force`.
func newTDDCmd(correlationID string) *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
		policy    string
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "tdd <milestone-id>",
		Short: "Set a milestone's TDD policy after creation",
		Long: `Set a milestone's TDD policy after creation.

A milestone already carrying the named policy has nothing to change, so a re-run
reports that at exit 0 and commits nothing.`,
		Example: `  # Downgrade a milestone's TDD policy
  aiwf milestone tdd M-003 --policy advisory

  # Re-require TDD discipline
  aiwf milestone tdd M-003 --policy required --reason "AC list stabilized"`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runTDD(args[0], actor, principal, root, reason, policy, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().StringVar(&policy, "policy", "", "the TDD policy to set: none | advisory | required")
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	_ = cmd.RegisterFlagCompletionFunc("policy", cobra.FixedCompletions(
		entity.AllowedTDDPolicies(),
		cobra.ShellCompDirectiveNoFileComp,
	))
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindMilestone, 0)
	return cmd
}

func runTDD(id, actor, principal, root, reason, policy string, out cliutil.OutputFormat) int {
	if policy == "" {
		cliutil.Errorln("aiwf milestone tdd: pass --policy <none|advisory|required>")
		return cliutil.ExitUsage
	}

	rootDir, actorStr, code, ok := cliutil.ResolvePrelude("aiwf milestone tdd", root, actor)
	if !ok {
		return code
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf milestone tdd", out)
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil { //coverage:ignore tree.Load errors only on filesystem IO failure (e.g. a permission fault) or context cancellation; malformed entities surface as load findings, not an error here.
		cliutil.Errorf("aiwf milestone tdd: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	result, vErr := verb.MilestoneTDD(ctx, tr, id, policy, actorStr, reason)
	code, _ = cliutil.DecorateAndFinish(ctx, rootDir, "aiwf milestone tdd", tr, result, vErr, pctx, out)
	return code
}

// newDependsOnCmd builds `aiwf milestone depends-on M-NNN --on
// M-PPP[,M-QQQ] [--clear]`. Closes the post-allocation half of
// G-072 (the create-time half is the --depends-on flag on
// `aiwf add milestone`). Replace-not-append semantics; --on and
// --clear are mutually exclusive.
func newDependsOnCmd(correlationID string) *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
		on        string
		clearList bool
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "depends-on <milestone-id>",
		Short: "Set or clear a milestone's depends_on list",
		Long: `Set or clear a milestone's depends_on list.

--on replaces the list rather than appending to it. A list that already reads
exactly as requested has nothing to change, so a re-run reports that at exit 0
and commits nothing — as does --clear against a milestone with no edges. Order
counts, so a reordered list is a real change and still commits. Ids compare at
canonical width, so a narrow --on spelling names the stored entity. Every --on
id must resolve: an unknown one is refused, never converged.`,
		Example: `  # Declare M-003 depends on M-001 and M-002
  aiwf milestone depends-on M-003 --on M-001,M-002

  # Empty the depends_on list
  aiwf milestone depends-on M-003 --clear`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runDependsOn(args[0], actor, principal, root, reason, on, clearList, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().StringVar(&on, "on", "", "comma-separated milestone ids the target depends on; replace-not-append semantics")
	cmd.Flags().BoolVar(&clearList, "clear", false, "empty the depends_on list (mutually exclusive with --on)")
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	_ = cmd.RegisterFlagCompletionFunc("on", cliutil.CompleteEntityIDFlag(entity.KindMilestone))
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindMilestone, 0)
	return cmd
}

func runDependsOn(id, actor, principal, root, reason, on string, clearList bool, out cliutil.OutputFormat) int {
	if on != "" && clearList {
		cliutil.Errorln("aiwf milestone depends-on: --on and --clear are mutually exclusive")
		return cliutil.ExitUsage
	}
	if on == "" && !clearList {
		cliutil.Errorln("aiwf milestone depends-on: pass --on <id,id,...> to set the list, or --clear to empty it")
		return cliutil.ExitUsage
	}

	rootDir, actorStr, code, ok := cliutil.ResolvePrelude("aiwf milestone depends-on", root, actor)
	if !ok {
		return code
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf milestone depends-on", out)
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil { //coverage:ignore tree.Load errors only on filesystem IO failure (e.g. a permission fault) or context cancellation; malformed entities surface as load findings, not an error here.
		cliutil.Errorf("aiwf milestone depends-on: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	deps := cliutil.SplitCommaList(on)
	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	result, vErr := verb.MilestoneDependsOn(ctx, tr, id, deps, clearList, actorStr, reason)
	code, _ = cliutil.DecorateAndFinish(ctx, rootDir, "aiwf milestone depends-on", tr, result, vErr, pctx, out)
	return code
}
