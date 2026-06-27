// Package setarea implements the `aiwf set-area <id> <member>` verb
// (per-verb subpackage; internal/cli/root.go's NewRootCmd wires it via
// NewCmd). It points ONE entity at an existing declared area member —
// or clears its area tag via --clear — in one trailered commit (E-0044,
// M-0183). It is the membership-change sibling of `rename-area`.
package setarea

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf set-area <id> <member>` (and `aiwf set-area <id>
// --clear` to untag): a verb that owns one entity's `area:` frontmatter,
// setting it to an existing declared member or clearing it, in one
// trailered commit.
func NewCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		clearTag  bool
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "set-area <id> <member>",
		Short: "Tag one entity to a declared area, or clear its tag",
		Long: `Point one entity at an existing declared area member, or clear its tag.

aiwf set-area rewrites the 'area:' frontmatter of a single entity to
<member> (which must already be declared in aiwf.yaml: areas.members),
or — with --clear — empties the tag back to the untagged state. One
trailered commit either way.

Orphan trap: hand-editing an entity's 'area:' frontmatter trips the
'provenance-untrailered-entity-commit' audit. set-area is the clean
path — its 'aiwf-verb: set-area' trailer suppresses that audit, so
tag, retag, and untag all flow through a discoverable commit rather
than a hand-edit. It is the one-command remediation when 'areas.required'
flags an untagged entity (run 'aiwf set-area <id> <member>'), and the
clean correction back to untagged (run 'aiwf set-area <id> --clear').

Refuses a milestone or composite/AC-id target — those derive their area
from the parent epic, and the refusal names the epic and the command to
run there. Also refuses an unknown id, an undeclared <member> (naming
the declared set), <member> together with --clear, and a no-op. The
change reverses totally via the same verb: a tag reverses with --clear,
a retag with the prior member.`,
		Example: `  # Tag an untagged entity (the areas.required remediation)
  aiwf set-area E-0001 platform

  # Move it to another declared area
  aiwf set-area E-0001 billing

  # Untag it back to the untagged state
  aiwf set-area E-0001 --clear`,
		Args:          cobra.RangeArgs(1, 2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args, actor, principal, root, clearTag, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&clearTag, "clear", false, "clear the entity's area tag (mutually exclusive with <member>)")
	out = cliutil.AddFormatFlags(cmd)
	// Composed positional completion: neither CompleteEntityIDArg nor
	// CompleteAreaValueArg composes two positions, so dispatch on len(args)
	// — position 0 offers entity ids, position 1 offers settable area
	// values (declared members PLUS the reserved `global` sentinel, since
	// set-area accepts global — M-0184).
	cmd.ValidArgsFunction = func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return cliutil.CompleteEntityIDArg("", 0)(c, args, toComplete)
		}
		return cliutil.CompleteAreaValueArg(1)(c, args, toComplete)
	}
	return cmd
}

// Run executes `aiwf set-area`. Returns one of the cliutil.Exit* codes.
func Run(args []string, actor, principal, root string, clearTag bool, out cliutil.OutputFormat) int {
	id := args[0]
	member := ""
	if len(args) == 2 {
		member = args[1]
	}

	// Arity-vs-clear: <member> and --clear are mutually exclusive, and
	// exactly one of them must be supplied.
	if member != "" && clearTag {
		fmt.Fprintln(os.Stderr, "aiwf set-area: <member> and --clear are mutually exclusive")
		return cliutil.ExitUsage
	}
	if member == "" && !clearTag {
		fmt.Fprintln(os.Stderr, "aiwf set-area: pass <member> to tag, or --clear to untag")
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf set-area: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf set-area: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf set-area")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf set-area: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	members, _ := cliutil.ConfiguredAreas(rootDir)
	result, vErr := verb.SetArea(ctx, tr, members, id, member, clearTag, actorStr)
	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		// Non-empty TargetID is what makes set-area authorized-AI-eligible
		// (the inverse of rename-area's empty-target, human-only posture):
		// a scoped ai/<id> agent whose scope reaches this entity may run it.
		TargetID: entity.Canonicalize(id),
	}
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf set-area", tr, result, vErr, pctx, out)
}
