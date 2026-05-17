// Package rename implements the `aiwf rename` verb (per-verb subpackage
// of M-0115; cmd/aiwf/main.go's newRootCmd wires it via NewCmd).
package rename

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf rename <id> <new-slug>`. The verb is slug-only:
// the entity id is preserved across the rename. Title and frontmatter
// are untouched.
func NewCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
	)
	cmd := &cobra.Command{
		Use:   "rename <id> <new-slug>",
		Short: "Rename the file/dir slug; id preserved",
		Example: `  # Rename M-007's slug to a clearer phrase
  aiwf rename M-007 cobra-and-completion`,
		Args:          cobra.ExactArgs(2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args[0], args[1], actor, principal, root))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	return cmd
}

// Run executes `aiwf rename`. Returns one of the cliutil.Exit* codes.
func Run(id, newSlug, actor, principal, root string) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf rename")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	result, err := verb.Rename(ctx, tr, id, newSlug, actorStr, cliutil.ConfiguredTitleMaxLength(rootDir))
	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf rename", tr, result, err, pctx)
}
