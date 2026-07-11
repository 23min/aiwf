// Package editbody implements the `aiwf edit-body` verb (per-verb
// subpackage of M-0115; cmd/aiwf/main.go's newRootCmd wires it via
// NewCmd). The package directory is `editbody` (no separator) per
// Go's package-name convention; the verb's external name on the CLI
// remains `edit-body` (hyphenated, the user-facing form).
package editbody

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/logger"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf edit-body <id> --body-file <path>` (and
// `--body-file -` for stdin) — the post-creation body-edit verb that
// closes the plain-git carve-out from G-052 / M-058. Frontmatter is
// untouched; only the markdown body below the frontmatter delimiter
// is replaced. One commit per invocation, standard provenance.
func NewCmd(correlationID string) *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
		bodyFile  string
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "edit-body <id>",
		Short: "Replace the entity's markdown body (frontmatter untouched)",
		Example: `  # Bless current working-copy edits to the entity body
  aiwf edit-body M-007

  # Replace the body from a file
  aiwf edit-body M-007 --body-file new-body.md --reason "refresh AC list"`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args[0], actor, principal, root, reason, bodyFile, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", `path to a file whose content becomes the entity's new body (use "-" to read from stdin); the file must contain body content only — leading "---" is refused. Omit to use bless mode: commit whatever the user edited in the working copy of the entity file`)
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	return cmd
}

// Run executes `aiwf edit-body`. Returns one of the cliutil.Exit* codes.
func Run(id, actor, principal, root, reason, bodyFile string, out cliutil.OutputFormat) (code int) {
	// Bless mode (M-060): when --body-file is absent, pass nil bytes
	// so the verb reads working-copy and HEAD itself and commits the
	// diff. Explicit mode (M-058): when --body-file is set, read the
	// file (or stdin for "-") and pass the bytes through.
	var body []byte
	if bodyFile != "" {
		var readErr error
		body, readErr = cliutil.ReadBodyFile(bodyFile)
		if readErr != nil {
			cliutil.Errorf("aiwf edit-body: %v\n", readErr)
			return cliutil.ExitUsage
		}
		if body == nil {
			body = []byte{}
		}
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		cliutil.Errorf("aiwf edit-body: %v\n", err)
		return cliutil.ExitUsage
	}
	actorStr, err := cliutil.ResolveActor(actor, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf edit-body: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()

	// M-0249: diagnostic-logging wiring, mirroring cancel.Run's own
	// M-0238/AC-5 pattern.
	diagLog, closeDiagLog := cliutil.ResolveLogger(rootDir, os.Getenv)
	defer func() { _ = closeDiagLog() }()
	if diagLog.Enabled(ctx, slog.LevelInfo) {
		runID := out.CorrelationID
		if runID == "" {
			runID = logger.NewRunID()
		}
		diagLog = logger.WithVerb(diagLog, "edit-body", id, actorStr, runID)
	}
	var sha string
	defer func() { cliutil.EmitVerbOutcome(diagLog, "verb", code, sha) }()

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf edit-body", out)
	if release == nil {
		return rc
	}
	defer release()

	// LoadTreeWithTrunk (not bare tree.Load) so the verb-time
	// body-prose-id scan sees TrunkIDs: a body referencing an entity
	// allocated on trunk but absent from this branch's tree must not
	// refuse the write (G-0241). Matches add/check/reallocate/rewidth.
	tr, _, err := cliutil.LoadTreeWithTrunk(ctx, rootDir)
	if err != nil {
		cliutil.Errorf("aiwf edit-body: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	result, vErr := verb.EditBody(ctx, tr, id, body, actorStr, reason)
	code, sha = cliutil.DecorateAndFinish(ctx, rootDir, "aiwf edit-body", tr, result, vErr, pctx, out)
	return code
}
