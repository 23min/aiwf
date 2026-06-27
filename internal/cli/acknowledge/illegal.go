package acknowledge

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/verb"
)

// newIllegalCmd builds `aiwf acknowledge illegal <sha> --reason "..."`. The
// subcommand is the retroactive sovereign-override mechanism for the
// fsm-history-consistent rule's illegal-transition findings (M-0136 closes
// G-0150's request for a non-config-file exemption path).
//
// The verb requires a `human/...` actor (sovereign acts trace to a named
// human) and a non-empty `--reason`. Per M-0136 design, the acknowledgment is
// an empty commit carrying aiwf-force-for: <sha> alongside the standard
// aiwf-verb / aiwf-actor / aiwf-reason trailers — no aiwf.yaml entry, no
// history rewrite. The emitted `aiwf-verb: acknowledge-illegal` trailer is
// unchanged by the M-0181/AC-5 regroup.
//
// G-0231 item 3: an optional `--for-entity <id>` flag binds the ack to a
// specific (SHA, entity) pair. The verb verifies at write time that <sha>'s
// diff actually touches <id>'s file; if not, the ack is refused. Required when
// acking against provenance-untrailered-entity-commit.
func newIllegalCmd() *cobra.Command {
	var (
		actor     string
		root      string
		reason    string
		forEntity string
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "illegal <sha>",
		Short: "Acknowledge a historical commit so kernel audit rules silence its findings",
		Long: `Records an acknowledgment commit for a historical commit that one of the
kernel's audit rules would otherwise flag. Every rule that consumes the
acknowledged-SHA set (via the M-0159/AC-3 lift) is silenced through the same
aiwf-force-for trailer:

    - fsm-history-consistent / illegal-transition     (M-0136/AC-2)
    - fsm-history-consistent / forced-untrailered     (M-0159/AC-4)
    - isolation-escape                                (M-0159/AC-4 via AC-3 lift)
    - isolation-escape-orphaned-ai-commit             (M-0161/AC-5; G-0236)
    - promote-on-wrong-branch                         (M-0161/AC-8)
    - id-rename-untrailered                           (M-0160/AC-4)
    - trailer-verb-unknown                            (G-0150 lift)
    - provenance-untrailered-entity-commit            (G-0231 item 3; --for-entity required)

The acknowledgment is a separate, current-day empty commit carrying:

    aiwf-verb: acknowledge-illegal
    aiwf-force-for: <historical-sha>
    aiwf-actor: human/<name>
    aiwf-reason: <text>
    aiwf-entity: <id>           (only when --for-entity is supplied)

The CLI gather layer at internal/cli/check/check.go walks HEAD's reachable
history for aiwf-force-for trailers once per check invocation (the M-0159/AC-3
lift) and threads the resulting SHA set to every rule above; each rule
exempts findings whose offending commit appears in the set. The acknowledgment
lives in git (queryable via aiwf history); it does NOT pollute aiwf.yaml and
does NOT rewrite the offending commit's history — the original author,
trailers, and SHA are preserved per M-0136's no-history-rewrite principle.

Target-SHA validity (M-0136/AC-4 + G-0236): the target must either be
reachable from HEAD (the primary case — covers FSM-history rules and
isolation-escape proper) OR present in the local object database as an
orphan (the G-0236 fallback — covers isolation-escape-orphaned-ai-commit,
whose offending SHAs are by construction unreachable since the reflog
walker surfaces force-pushed-away tips). Typos and SHAs from unrelated
repos fail both checks and are refused.

--for-entity verification (G-0231 item 3): when --for-entity <id> is supplied,
the verb runs git diff-tree against <sha> and refuses the ack unless one of
the diff's paths resolves to <id>. This is what makes the per-(SHA, entity)
ack tamper-resistant against operator-attested bindings (LLM or human writing
the wrong entity id with a real SHA): the kernel walks the actual git diff
and refuses if <sha> doesn't touch <id>. Required for
provenance-untrailered-entity-commit acks; optional for the other seven
rules (which use the per-SHA blanket shape).

Per-SHA closed-set scoping: an acknowledgment for one SHA exempts only that
SHA. There is no "exempt everything" knob.

Both --reason (non-empty after trim) and a human/... actor are required
— sovereign acts trace to a named human with written rationale.`,
		Example: `  # Acknowledge a squash-merge commit whose intermediate FSM steps were lost
  aiwf acknowledge illegal f4ea7329 \
    --reason "pre-AC-2 era squash; legal feature-branch progression existed but was collapsed"

  # Acknowledge an untrailered entity-edit commit (per-(SHA, entity) ack)
  aiwf acknowledge illegal 6a1e70cc --for-entity ADR-0007 \
    --reason "post-E-0038 terminology refresh landed inline; should have used aiwf edit-body"`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(runIllegal(args[0], actor, root, reason, forEntity, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer (must be human/...; derived from git config if unset)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining the acknowledgment; required, non-empty after trim")
	cmd.Flags().StringVar(&forEntity, "for-entity", "", "bind the ack to a specific entity id; the verb verifies the SHA's diff touches that entity (required for provenance-untrailered-entity-commit acks)")
	// Dynamic entity-id completion: any kind is valid (the verb checks
	// that the SHA's diff touches the named entity regardless of kind).
	_ = cmd.RegisterFlagCompletionFunc("for-entity", cliutil.CompleteEntityIDFlag(""))
	out = cliutil.AddFormatFlags(cmd)
	return cmd
}

// runIllegal executes `aiwf acknowledge illegal`. Returns one of the
// cliutil.Exit* codes; the caller (RunE) wraps the int in cliutil.WrapExitCode
// so Cobra's RunE channel preserves the exit code through the run() dispatcher.
func runIllegal(sha, actor, root, reason, forEntity string, out cliutil.OutputFormat) int {
	if strings.TrimSpace(reason) == "" {
		fmt.Fprintln(os.Stderr, "aiwf acknowledge illegal: --reason \"...\" is required (non-empty after trim)")
		return cliutil.ExitUsage
	}
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		//coverage:ignore ResolveRoot errors only on a broken cwd — filepath.Abs failure (explicit --root) or os.Getwd failure (empty --root); neither is deterministically reproducible.
		fmt.Fprintf(os.Stderr, "aiwf acknowledge illegal: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf acknowledge illegal: %v\n", err)
		return cliutil.ExitUsage
	}
	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf acknowledge illegal")
	if release == nil {
		return rc
	}
	defer release()
	ctx := context.Background()
	result, vErr := verb.AcknowledgeIllegal(ctx, rootDir, sha, forEntity, actorStr, reason)
	return cliutil.FinishVerb(ctx, rootDir, "aiwf acknowledge illegal", result, vErr, out)
}
