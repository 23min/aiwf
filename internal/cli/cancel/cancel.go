// Package cancel implements the `aiwf cancel` verb. It is the per-verb
// subpackage that cmd/aiwf/main.go's newRootCmd wires via NewCmd();
// per the M-0115 pattern, every cmd/aiwf verb lives under
// internal/cli/<verb>/.
package cancel

import (
	"context"
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
func NewCmd(correlationID string) *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
		force     bool
		auditOnly bool
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "cancel <id>",
		Short: "Promote to the kind's terminal-cancel status",
		Long: `Set an entity to its kind's terminal-cancel status.

An entity already at a terminal status is already disposed, so a re-run reports
that status at exit 0 and commits nothing. Cancel's target is a terminal
end-state rather than one specific status, so an entity that reached a terminal
by another path — a done epic, an addressed gap — converges too. Convergence
holds even under --force: there is no diff for a sovereign override to re-apply.

--audit-only is the exception, by design. It records a transition that already
happened, so it commits precisely in the state the other paths converge on, and
requires the entity to be at the terminal-cancel target already.`,
		Example: `  # Cancel an in-flight epic with a rationale
  aiwf cancel E-01 --reason "scope absorbed into E-02"`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(Options{
				ID:        args[0],
				Actor:     actor,
				Principal: principal,
				Root:      root,
				Reason:    reason,
				Force:     force,
				AuditOnly: auditOnly,
				Out:       *out,
			}))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().BoolVar(&force, "force", false, "record an audit trailer even when the verb's existing checks would normally allow it (requires --reason); sovereign, so the actor must be human/... — a force trailer from a non-human actor is refused before anything is written")
	cmd.Flags().BoolVar(&auditOnly, "audit-only", false, "record an audit-trail commit without mutating files; entity must already be at the kind's terminal-cancel target (requires --reason; mutex with --force; G24 recovery path)")
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	return cmd
}

// Options carries the `aiwf cancel` invocation inputs to Run.
// Collapsing the previously-positional signature into a named struct
// removes the transpose-two-strings hazard the long positional form
// carried, mirroring internal/verb's PromoteOptions / AddOptions
// convention at the CLI adapter boundary (G-0227).
type Options struct {
	ID        string
	Actor     string
	Principal string
	Root      string
	Reason    string
	Force     bool
	AuditOnly bool
	Out       cliutil.OutputFormat
}

// Run executes `aiwf cancel`. Returns one of the cliutil.Exit* codes;
// the caller (RunE in NewCmd) wraps the int in cliutil.WrapExitCode
// so Cobra's RunE channel preserves the exit code through the run()
// dispatcher.
func Run(opts Options) (code int) {
	if opts.Force && opts.AuditOnly {
		cliutil.Errorln("aiwf cancel: --force and --audit-only cannot coexist (force makes a transition; audit-only records one that already happened)")
		return cliutil.ExitUsage
	}
	if (opts.Force || opts.AuditOnly) && strings.TrimSpace(opts.Reason) == "" {
		gateFlag := "--force"
		if opts.AuditOnly {
			gateFlag = "--audit-only"
		}
		cliutil.Errorf("aiwf cancel: --reason \"...\" is required when %s is set (non-empty after trim)\n", gateFlag)
		return cliutil.ExitUsage
	}

	rootDir, actorStr, code, ok := cliutil.ResolvePrelude("aiwf cancel", opts.Root, opts.Actor)
	if !ok { //coverage:ignore prelude resolution failure is covered by the shared helper's own tests; this per-verb short-circuit is not separately reproducible
		return code
	}

	if forceCode, forceOK := cliutil.RefuseNonHumanSovereignForce("aiwf cancel", actorStr, opts.Force); !forceOK {
		return forceCode
	}

	ctx := context.Background()

	finish := cliutil.BeginVerbDiag(rootDir, "cancel", opts.ID, actorStr, opts.Out.CorrelationID)
	var sha string
	defer finish(&code, &sha)

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf cancel", opts.Out)
	if release == nil {
		return rc
	}
	defer release()

	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf cancel: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	pctx := cliutil.ProvenanceContext{
		Actor:             actorStr,
		Principal:         strings.TrimSpace(opts.Principal),
		VerbKind:          verb.VerbAct,
		TargetID:          opts.ID,
		IsTerminalPromote: !entity.IsCompositeID(opts.ID),
	}
	if opts.AuditOnly {
		result, vErr := verb.CancelAuditOnly(ctx, tr, opts.ID, actorStr, opts.Reason)
		code, sha = cliutil.DecorateAndFinish(ctx, rootDir, "aiwf cancel", tr, result, vErr, pctx, opts.Out)
	} else {
		result, vErr := verb.Cancel(ctx, tr, opts.ID, actorStr, opts.Reason, opts.Force)
		code, sha = cliutil.DecorateAndFinish(ctx, rootDir, "aiwf cancel", tr, result, vErr, pctx, opts.Out)
	}
	return code
}
