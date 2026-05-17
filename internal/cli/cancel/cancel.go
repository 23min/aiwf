// Package cancel implements the `aiwf cancel` verb. It is the per-verb
// subpackage that cmd/aiwf/main.go's newRootCmd wires via NewCmd();
// per the M-0115 pattern, every cmd/aiwf verb lives under
// internal/cli/<verb>/.
package cancel

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

// NewCmd builds `aiwf cancel <id> [--reason "..."]`. The verb is the
// kind-aware terminal-cancel transition: an epic cancels to "cancelled",
// a gap to "wontfix", an ADR to "rejected", etc. — the per-kind FSM
// target lives in entity.AllowedTransitions and the verb layer.
func NewCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
		force     bool
		auditOnly bool
	)
	cmd := &cobra.Command{
		Use:   "cancel <id>",
		Short: "Promote to the kind's terminal-cancel status",
		Example: `  # Cancel an in-flight epic with a rationale
  aiwf cancel E-01 --reason "scope absorbed into E-02"`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args[0], actor, principal, root, reason, force, auditOnly))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().BoolVar(&force, "force", false, "record an audit trailer even when the verb's existing checks would normally allow it (requires --reason)")
	cmd.Flags().BoolVar(&auditOnly, "audit-only", false, "record an audit-trail commit without mutating files; entity must already be at the kind's terminal-cancel target (requires --reason; mutex with --force; G24 recovery path)")
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	return cmd
}

// Run executes `aiwf cancel`. Returns one of the cliutil.Exit* codes;
// the caller (RunE in NewCmd) wraps the int in cliutil.WrapExitCode
// so Cobra's RunE channel preserves the exit code through the run()
// dispatcher.
func Run(id, actor, principal, root, reason string, force, auditOnly bool) int {
	if force && auditOnly {
		fmt.Fprintln(os.Stderr, "aiwf cancel: --force and --audit-only cannot coexist (force makes a transition; audit-only records one that already happened)")
		return cliutil.ExitUsage
	}
	if (force || auditOnly) && strings.TrimSpace(reason) == "" {
		gateFlag := "--force"
		if auditOnly {
			gateFlag = "--audit-only"
		}
		fmt.Fprintf(os.Stderr, "aiwf cancel: --reason \"...\" is required when %s is set (non-empty after trim)\n", gateFlag)
		return cliutil.ExitUsage
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf cancel: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf cancel: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf cancel")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf cancel: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	pctx := cliutil.ProvenanceContext{
		Actor:             actorStr,
		Principal:         strings.TrimSpace(principal),
		VerbKind:          verb.VerbAct,
		TargetID:          id,
		IsTerminalPromote: !entity.IsCompositeID(id),
	}
	if auditOnly {
		result, vErr := verb.CancelAuditOnly(ctx, tr, id, actorStr, reason)
		return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf cancel", tr, result, vErr, pctx)
	}
	result, vErr := verb.Cancel(ctx, tr, id, actorStr, reason, force)
	return cliutil.DecorateAndFinish(ctx, rootDir, "aiwf cancel", tr, result, vErr, pctx)
}
