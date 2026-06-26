// Package renamearea implements the `aiwf rename-area <old> <new>`
// verb (per-verb subpackage; internal/cli/root.go's NewRootCmd wires
// it via NewCmd). It renames a declared workstream area in aiwf.yaml
// and atomically rewrites every entity tagged with the old area, in
// one commit (E-0044, M-0177).
package renamearea

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf rename-area <old> <new>`: renames a declared
// area member and rewrites the `area:` frontmatter of every entity
// that references it, in one trailered commit.
func NewCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "rename-area <old> <new>",
		Short: "Rename a declared area; rewrite tagged entities",
		Long: `Rename a declared workstream area and rewrite every entity that references it.

aiwf rename-area renames the member in aiwf.yaml (areas.members) AND
rewrites the 'area:' frontmatter of every entity tagged with the old
name to the new name — in a single commit, the same referential-
integrity discipline 'aiwf reallocate' applies to ids.

Orphan trap: renaming an area by hand-editing aiwf.yaml (or removing
one) leaves every entity still carrying the old value orphaned — the
'area-unknown' check flags them and the grouping view silently buckets
them into the untagged complement. Using this verb instead of a hand
edit is what keeps referencing entities from being silently orphaned.

Refuses when <old> is not a declared member, or when <new> already
names one; the refusal names the declared set and writes nothing. The
rename reverses via the same verb with swapped args.`,
		Example: `  # Rename the 'platform' area to 'infra', carrying every tagged entity along
  aiwf rename-area platform infra

  # Reverse it
  aiwf rename-area infra platform`,
		Args:          cobra.ExactArgs(2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args[0], args[1], actor, principal, root, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	out = cliutil.AddFormatFlags(cmd)
	cmd.ValidArgsFunction = cliutil.CompleteAreaArg(0)
	return cmd
}

// Run executes `aiwf rename-area`. Returns one of the cliutil.Exit* codes.
func Run(oldName, newName, actor, principal, root string, out cliutil.OutputFormat) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename-area: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename-area: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf rename-area")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename-area: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	// The full member shape (label + paths) so the rename preserves each
	// member's paths (E-0044, M-0179); the default label still comes through
	// ConfiguredAreas (name-only). The handler has no *config.Config in scope,
	// so it reads through the cliutil helpers rather than config directly.
	members := cliutil.ConfiguredAreaMembersFull(rootDir)
	_, defaultLabel := cliutil.ConfiguredAreas(rootDir)
	doc, _, err := cliutil.LoadContractsDoc(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf rename-area: %v\n", err)
		return cliutil.ExitUsage
	}

	result, err := verb.RenameArea(ctx, tr, doc, members, defaultLabel, oldName, newName, actorStr)
	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
	}
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf rename-area", tr, result, err, pctx, out)
}
