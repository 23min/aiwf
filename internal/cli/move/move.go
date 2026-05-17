// Package move implements the `aiwf move` verb (per-verb subpackage of
// M-0115; cmd/aiwf/main.go's newRootCmd wires it via NewCmd).
package move

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

// NewCmd builds `aiwf move <M-id> --epic <E-id>`: relocates a
// milestone to a different epic in one commit.
func NewCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		epic      string
	)
	cmd := &cobra.Command{
		Use:   "move <M-id> --epic <E-id>",
		Short: "Move a milestone to a different epic; id preserved",
		Example: `  # Reparent M-007 under epic E-04
  aiwf move M-007 --epic E-04`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			if epic == "" {
				fmt.Fprintln(os.Stderr, "aiwf move: --epic <E-id> is required")
				return cliutil.WrapExitCode(cliutil.ExitUsage)
			}
			return cliutil.WrapExitCode(Run(args[0], epic, actor, principal, root))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&epic, "epic", "", "target epic id (e.g., E-04)")
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindMilestone, 0)
	_ = cmd.RegisterFlagCompletionFunc("epic", cliutil.CompleteEntityIDFlag(entity.KindEpic))
	return cmd
}

// Run executes `aiwf move`. Returns one of the cliutil.Exit* codes.
func Run(id, epic, actor, principal, root string) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf move: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf move: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf move")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf move: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	// Move endpoints for the allow-rule are the source epic (the
	// milestone's current parent) and the destination epic (--epic).
	// Both must reach the scope-entity per the strict-move rule.
	var moveSource string
	if e := tr.ByID(id); e != nil {
		moveSource = e.Parent
	}
	result, err := verb.Move(ctx, tr, id, epic, actorStr)
	pctx := cliutil.ProvenanceContext{
		Actor:      actorStr,
		Principal:  strings.TrimSpace(principal),
		VerbKind:   verb.VerbMove,
		TargetID:   epic,
		MoveSource: moveSource,
	}
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf move", tr, result, err, pctx)
}
