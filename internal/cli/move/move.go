// Package move implements the `aiwf move` verb (per-verb subpackage of
// M-0115; cmd/aiwf/main.go's newRootCmd wires it via NewCmd).
package move

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf move <M-id> --epic <E-id>`: relocates a
// milestone to a different epic in one commit.
func NewCmd(correlationID string) *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		epic      string
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "move <M-id> --epic <E-id>",
		Short: "Move a milestone to a different epic; id preserved",
		Long: `Reparent a milestone under a different epic. The milestone id is preserved.

A milestone already under the named epic has nothing to relocate, so a re-run
reports that at exit 0 and commits nothing. The move spans two surfaces — the
` + "`parent:`" + ` field and the file's location under the epic's directory — and both must
already hold to converge. Ids compare at canonical width, so a narrow --epic
spelling names the stored epic. The target epic must exist either way: an
unknown --epic is refused, never converged.

Markdown links in other entity bodies that point at the moved milestone are
rewritten to its new path in the same commit. Those bodies are part of the
move's write set, so an uncommitted edit to any of them refuses the move,
naming the file. The moved milestone's own outbound links are not rewritten.`,
		Example: `  # Reparent M-007 under epic E-04
  aiwf move M-007 --epic E-04`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			if epic == "" {
				cliutil.Errorln("aiwf move: --epic <E-id> is required")
				return cliutil.WrapExitCode(cliutil.ExitUsage)
			}
			return cliutil.WrapExitCode(Run(args[0], epic, actor, principal, root, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&epic, "epic", "", "target epic id (e.g., E-04)")
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg(entity.KindMilestone, 0)
	_ = cmd.RegisterFlagCompletionFunc("epic", cliutil.CompleteEntityIDFlag(entity.KindEpic))
	return cmd
}

// Run executes `aiwf move`. Returns one of the cliutil.Exit* codes.
func Run(id, epic, actor, principal, root string, out cliutil.OutputFormat) (code int) {
	rootDir, actorStr, code, ok := cliutil.ResolvePrelude("aiwf move", root, actor)
	if !ok { //coverage:ignore prelude resolution failure is covered by the shared helper's own tests; this per-verb short-circuit is not separately reproducible
		return code
	}

	ctx := context.Background()

	// entity is id — the milestone being moved, not epic (the
	// destination): pctx.TargetID below is epic because that is whose
	// authorization scope governs the move, a different question ("who
	// may do this") from what the diagnostic entity field records
	// ("what this verb acted on").
	finish := cliutil.BeginVerbDiag(rootDir, "move", id, actorStr, out.CorrelationID)
	var sha string
	defer finish(&code, &sha)

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf move", out)
	if release == nil {
		return rc
	}
	defer release()

	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf move: loading tree: %v\n", err)
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
	code, sha = cliutil.DecorateAndFinish(ctx, rootDir, "aiwf move", tr, result, err, pctx, out)
	return code
}
