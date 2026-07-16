// Package setpriority implements the `aiwf set-priority <id> <level>`
// verb (per-verb subpackage; internal/cli/root.go's NewRootCmd wires it
// via NewCmd). It points ONE gap or decision at a closed-set priority
// level — or clears its priority tag via --clear — in one trailered
// commit (G-0078, E-0066, M-0262). It is the write-surface sibling of
// `set-area`, minus the config-declared member lookup: the priority
// level set is fixed in Go (entity.AllowedPriorityLevels), not
// project-configured.
package setpriority

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf set-priority <id> <level>` (and `aiwf set-priority
// <id> --clear` to unset): a verb that owns one gap's or decision's
// `priority:` frontmatter, setting it to a closed-set level or clearing
// it, in one trailered commit.
func NewCmd(correlationID string) *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		clearTag  bool
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "set-priority <id> <level>",
		Short: "Set a gap or decision's priority, or clear it",
		Long: `Set one gap or decision's priority to a closed-set level, or clear it.

aiwf set-priority rewrites the 'priority:' frontmatter of a single gap
or decision to <level> (one of: urgent, high, medium, low), or — with
--clear — empties the tag back to the unset state. One trailered commit
either way.

Orphan trap: hand-editing an entity's 'priority:' frontmatter trips the
'provenance-untrailered-entity-commit' audit. set-priority is the clean
path — its 'aiwf-verb: set-priority' trailer suppresses that audit, so
set, reset, and clear all flow through a discoverable commit rather
than a hand-edit.

Refuses an unknown id, a target whose kind does not carry a priority
(only gap and decision do), an out-of-range <level>, <level> together
with --clear, and a no-op. The change reverses totally via the same
verb: a set reverses with --clear, a reset with the prior level.`,
		Example: `  # Set a gap's priority
  aiwf set-priority G-0001 urgent

  # Change it to another level
  aiwf set-priority G-0001 medium

  # Clear it back to unset
  aiwf set-priority G-0001 --clear`,
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
	cmd.Flags().BoolVar(&clearTag, "clear", false, "clear the entity's priority tag (mutually exclusive with <level>)")
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	// Composed positional completion: position 0 offers entity ids,
	// position 1 offers the fixed priority-level set — mirrors
	// set-area's composed ValidArgsFunction, minus the config lookup
	// (the level set is Go-hardcoded, not project-configured).
	cmd.ValidArgsFunction = func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return cliutil.CompleteEntityIDArg("", 0)(c, args, toComplete)
		}
		if len(args) == 1 {
			return entity.AllowedPriorityLevels(), cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return cmd
}

// Run executes `aiwf set-priority`. Returns one of the cliutil.Exit* codes.
func Run(args []string, actor, principal, root string, clearTag bool, out cliutil.OutputFormat) (code int) {
	id := args[0]
	level := ""
	if len(args) == 2 {
		level = args[1]
	}

	// Arity-vs-clear: <level> and --clear are mutually exclusive, and
	// exactly one of them must be supplied.
	if level != "" && clearTag {
		cliutil.Errorln("aiwf set-priority: <level> and --clear are mutually exclusive")
		return cliutil.ExitUsage
	}
	if level == "" && !clearTag {
		cliutil.Errorln("aiwf set-priority: pass <level> to set, or --clear to unset")
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		//coverage:ignore ResolveRoot errors only on a broken cwd (filepath.Abs / os.Getwd); not deterministically reproducible.
		cliutil.Errorf("aiwf set-priority: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf set-priority: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()

	diagLog, closeDiagLog := cliutil.ResolveLogger(rootDir, os.Getenv)
	defer func() { _ = closeDiagLog() }()
	if diagLog.Enabled(ctx, slog.LevelInfo) {
		runID := out.CorrelationID
		if runID == "" {
			runID = logger.NewRunID()
		}
		diagLog = logger.WithVerb(diagLog, "set-priority", id, actorStr, runID)
	}
	var sha string
	defer func() { cliutil.EmitVerbOutcome(diagLog, "verb", code, sha) }()

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf set-priority", out)
	if release == nil {
		return rc
	}
	defer release()

	tr, _, err := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		//coverage:ignore LoadTreeWithTrunk errors only on filesystem/git IO failure; malformed entities surface as load findings, not an error here.
		cliutil.Errorf("aiwf set-priority: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	result, vErr := verb.SetPriority(ctx, tr, id, level, clearTag, actorStr)
	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		// Non-empty TargetID mirrors set-area's authorized-AI-eligible
		// posture: a scoped ai/<id> agent whose scope reaches this
		// entity may run set-priority.
		TargetID: entity.Canonicalize(id),
	}
	code, sha = cliutil.DecorateAndFinish(ctx, rootDir, "aiwf set-priority", tr, result, vErr, pctx, out)
	return code
}
