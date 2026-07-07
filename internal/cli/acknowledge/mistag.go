package acknowledge

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/verb"
)

// newMistagCmd builds `aiwf acknowledge mistag <id> --reason "..."` (M-0181/AC-6).
// It records a sovereign acknowledgement that <id>'s area tag and its commits'
// landing zone legitimately disagree, suppressing the area-mistag warning for
// that entity. Like `acknowledge illegal` it is a human-sovereign empty-commit
// act, but keyed per-entity rather than per-SHA.
func newMistagCmd() *cobra.Command {
	var (
		actor  string
		root   string
		reason string
		out    *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "mistag <id>",
		Short: "Acknowledge an entity's area-mistag as intentional cross-cutting work",
		Long: `Records a sovereign acknowledgement that <id>'s area tag and its commits'
landing zone legitimately disagree, suppressing the area-mistag warning for
that entity. Use it when an entity's work genuinely spans areas (e.g. moving
code into a shared area) rather than being mis-filed.

The acknowledgement is a separate, current-day empty commit carrying:

    aiwf-verb: acknowledge-mistag
    aiwf-entity: <id>
    aiwf-actor: human/<name>
    aiwf-reason: <text>

The check's area-mistag rule walks HEAD for these commits and exempts the
named entities. The acknowledgement lives in git (queryable via aiwf history);
it does not pollute aiwf.yaml. If you later realize the tag was simply wrong,
re-tag with aiwf set-area instead — the mistag then no longer fires.

Both --reason (non-empty after trim) and a human/... actor are required —
sovereign acts trace to a named human with written rationale.`,
		Example: `  aiwf acknowledge mistag G-0301 \
    --reason "moving billing's auth into the shared platform lib; cross-cutting by design"`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runMistag(args[0], actor, root, reason, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer (must be human/...; derived from git config if unset)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why the cross-cutting work is intentional; required, non-empty after trim")
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	out = cliutil.AddFormatFlags(cmd)
	return cmd
}

// runMistag executes `aiwf acknowledge mistag`. Returns one of the
// cliutil.Exit* codes; the caller wraps it via cliutil.WrapExitCode.
func runMistag(id, actor, root, reason string, out cliutil.OutputFormat) int {
	if strings.TrimSpace(reason) == "" {
		cliutil.Errorln("aiwf acknowledge mistag: --reason \"...\" is required (non-empty after trim)")
		return cliutil.ExitUsage
	}
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		//coverage:ignore ResolveRoot errors only on a broken cwd (filepath.Abs / os.Getwd); not deterministically reproducible.
		cliutil.Errorf("aiwf acknowledge mistag: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf acknowledge mistag: %v\n", err)
		return cliutil.ExitUsage
	}
	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf acknowledge mistag")
	if release == nil {
		return rc
	}
	defer release()
	ctx := context.Background()
	tr, _, err := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		//coverage:ignore LoadTreeWithTrunk errors only on a filesystem/git IO failure; malformed entities surface as load findings, not an error here.
		cliutil.Errorf("aiwf acknowledge mistag: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	result, vErr := verb.AcknowledgeMistag(ctx, tr, id, actorStr, reason)
	return cliutil.FinishVerb(ctx, rootDir, "aiwf acknowledge mistag", result, vErr, out)
}
