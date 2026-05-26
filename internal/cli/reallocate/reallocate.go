// Package reallocate implements the `aiwf reallocate` verb (per-verb
// subpackage of M-0115; cmd/aiwf/main.go's newRootCmd wires it via NewCmd).
package reallocate

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf reallocate <id-or-path>`: renumbers an entity
// (id rewritten) and rewrites references to it across the tree.
// Standard resolution path for an `ids-unique` finding from `aiwf check`.
func NewCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "reallocate <id-or-path>",
		Short: "Renumber the entity; rewrite refs in others",
		Example: `  # Resolve an id collision detected by aiwf check
  aiwf reallocate M-007`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args[0], actor, principal, root, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	out = cliutil.AddFormatFlags(cmd)
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	return cmd
}

// Run executes `aiwf reallocate`. Returns one of the cliutil.Exit* codes.
func Run(target, actor, principal, root string, out cliutil.OutputFormat) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf reallocate: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf reallocate: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf reallocate")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf reallocate: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	result, err := verb.Reallocate(ctx, tr, target, actorStr)
	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  target,
	}
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf reallocate", tr, result, err, pctx, out)
}
