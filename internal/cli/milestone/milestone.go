// Package milestone implements the `aiwf milestone` verb namespace.
// Currently it carries one child — depends-on — that sets or clears
// a milestone's depends_on list. The parent itself is non-Runnable
// (the kind-scoped namespace is forward-compatible with G-073's
// eventual cross-kind generalisation).
package milestone

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

// NewCmd builds the `aiwf milestone` parent command. One child today
// (depends-on). The parent itself is non-Runnable.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "milestone",
		Short:         "Milestone-scoped verbs",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newDependsOnCmd())
	return cmd
}

// newDependsOnCmd builds `aiwf milestone depends-on M-NNN --on
// M-PPP[,M-QQQ] [--clear]`. Closes the post-allocation half of
// G-072 (the create-time half is the --depends-on flag on
// `aiwf add milestone`). Replace-not-append semantics; --on and
// --clear are mutually exclusive.
func newDependsOnCmd() *cobra.Command {
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
	_ = cmd.RegisterFlagCompletionFunc("on", cliutil.CompleteEntityIDFlag(entity.KindMilestone))
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindMilestone, 0)
	return cmd
}

func runDependsOn(id, actor, principal, root, reason, on string, clearList bool, out cliutil.OutputFormat) int {
	if on != "" && clearList {
		fmt.Fprintln(os.Stderr, "aiwf milestone depends-on: --on and --clear are mutually exclusive")
		return cliutil.ExitUsage
	}
	if on == "" && !clearList {
		fmt.Fprintln(os.Stderr, "aiwf milestone depends-on: pass --on <id,id,...> to set the list, or --clear to empty it")
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf milestone depends-on: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf milestone depends-on: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf milestone depends-on")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf milestone depends-on: loading tree: %v\n", err)
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
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf milestone depends-on", tr, result, vErr, pctx, out)
}
