// Package acknowledgeillegal implements the `aiwf acknowledge-illegal`
// verb. It is the per-verb subpackage that internal/cli/root.go's
// NewRootCmd wires via NewCmd(); per the M-0115 pattern, every cmd/aiwf
// verb lives under internal/cli/<verb>/.
package acknowledgeillegal

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf acknowledge-illegal <sha> --reason "..."`. The
// verb is the retroactive sovereign-override mechanism for the
// fsm-history-consistent rule's illegal-transition findings (M-0136
// closes G-0150's request for a non-config-file exemption path).
//
// The verb requires a `human/...` actor (sovereign acts trace to a
// named human) and a non-empty `--reason`. Per M-0136 design, the
// acknowledgment is an empty commit carrying aiwf-force-for: <sha>
// alongside the standard aiwf-verb / aiwf-actor / aiwf-reason
// trailers — no aiwf.yaml entry, no history rewrite.
func NewCmd() *cobra.Command {
	var (
		actor  string
		root   string
		reason string
	)
	cmd := &cobra.Command{
		Use:   "acknowledge-illegal <sha>",
		Short: "Record a sovereign override for a historical FSM-illegal commit",
		Long: `Records an acknowledgment commit for a historical FSM-illegal commit that the
fsm-history-consistent rule's illegal-transition predicate would otherwise flag.

The acknowledgment is a separate, current-day empty commit carrying:

    aiwf-verb: acknowledge-illegal
    aiwf-force-for: <historical-sha>
    aiwf-actor: human/<name>
    aiwf-reason: <text>

The fsm-history-consistent rule walks HEAD's reachable history for
aiwf-force-for trailers and exempts any illegal-transition finding
whose offending commit appears as a target. The acknowledgment lives
in git (queryable via aiwf history); it does NOT pollute aiwf.yaml and
does NOT rewrite the offending commit's history.

Both --reason (non-empty after trim) and a human/... actor are required
— sovereign acts trace to a named human with written rationale.`,
		Example: `  # Acknowledge a squash-merge commit whose intermediate FSM steps were lost
  aiwf acknowledge-illegal f4ea7329 \
    --reason "pre-AC-2 era squash; legal feature-branch progression existed but was collapsed"`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args[0], actor, root, reason))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer (must be human/...; derived from git config if unset)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining the acknowledgment; required, non-empty after trim")
	return cmd
}

// Run executes `aiwf acknowledge-illegal`. Returns one of the
// cliutil.Exit* codes; the caller (RunE in NewCmd) wraps the int in
// cliutil.WrapExitCode so Cobra's RunE channel preserves the exit code
// through the run() dispatcher.
func Run(sha, actor, root, reason string) int {
	if strings.TrimSpace(reason) == "" {
		fmt.Fprintln(os.Stderr, "aiwf acknowledge-illegal: --reason \"...\" is required (non-empty after trim)")
		return cliutil.ExitUsage
	}
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf acknowledge-illegal: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf acknowledge-illegal: %v\n", err)
		return cliutil.ExitUsage
	}
	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf acknowledge-illegal")
	if release == nil {
		return rc
	}
	defer release()
	ctx := context.Background()
	result, vErr := verb.AcknowledgeIllegal(ctx, rootDir, sha, actorStr, reason)
	return cliutil.FinishVerb(ctx, rootDir, "aiwf acknowledge-illegal", result, vErr)
}
