package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// newMilestoneCmd builds the `aiwf milestone` parent command — a
// kind-prefixed verb namespace whose first child is `depends-on`. The
// shape is forward-compatible with G-073's eventual cross-kind
// generalisation: `aiwf <kind> depends-on <id> --on <ids>` extends to
// other kinds without renaming the verb. The parent is non-Runnable —
// `aiwf milestone` with no subcommand prints help.
func newMilestoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "milestone",
		Short:         "Milestone-scoped verbs",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newMilestoneDependsOnCmd())
	return cmd
}

// newMilestoneDependsOnCmd builds `aiwf milestone depends-on M-NNN
// --on M-PPP[,M-QQQ] [--clear]`. Closes the post-allocation half of
// G-072 (the create-time half is the --depends-on flag on `aiwf add
// milestone`). Replace-not-append semantics; --on and --clear are
// mutually exclusive.
func newMilestoneDependsOnCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
		on        string
		clearList bool
	)
	cmd := &cobra.Command{
		Use:   "depends-on <milestone-id>",
		Short: "Set or clear a milestone's depends_on list",
		Example: `  # Declare M-003 depends on M-001 and M-002
  aiwf milestone depends-on M-003 --on M-001,M-002

  # Empty the depends_on list
  aiwf milestone depends-on M-003 --clear`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runMilestoneDependsOnCmd(args[0], actor, principal, root, reason, on, clearList))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().StringVar(&on, "on", "", "comma-separated milestone ids the target depends on; replace-not-append semantics")
	cmd.Flags().BoolVar(&clearList, "clear", false, "empty the depends_on list (mutually exclusive with --on)")
	_ = cmd.RegisterFlagCompletionFunc("on", completeEntityIDFlag(entity.KindMilestone))
	cmd.ValidArgsFunction = completeEntityIDArg(entity.KindMilestone, 0)
	return cmd
}

func runMilestoneDependsOnCmd(id, actor, principal, root, reason, on string, clearList bool) int {
	if on != "" && clearList {
		fmt.Fprintln(os.Stderr, "aiwf milestone depends-on: --on and --clear are mutually exclusive")
		return exitUsage
	}
	if on == "" && !clearList {
		fmt.Fprintln(os.Stderr, "aiwf milestone depends-on: pass --on <id,id,...> to set the list, or --clear to empty it")
		return exitUsage
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf milestone depends-on: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf milestone depends-on: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf milestone depends-on")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf milestone depends-on: loading tree: %v\n", err)
		return exitInternal
	}

	deps := splitCommaList(on)
	pctx := provenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	result, vErr := verb.MilestoneDependsOn(ctx, tr, id, deps, clearList, actorStr, reason)
	return decorateAndFinish(ctx, rootDir, "aiwf milestone depends-on", tr, result, vErr, pctx)
}
