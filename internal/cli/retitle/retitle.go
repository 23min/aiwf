// Package retitle implements the `aiwf retitle ` verb (per-verb subpackage of M-0116;
// cmd/aiwf/main.go's newRootCmd wires it via NewCmd).
package retitle

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// NewCmd builds `aiwf retitle <id|composite-id> <new-title>
// [--reason "..."]`. Title mutation: updates the entity's frontmatter
// `title:`; for top-level entities whose slug still tracks the title it
// also re-derives the on-disk slug (G-0108), leaving a slug set with
// `aiwf rename` alone; and syncs a canonical `# <ID> — <title>` body H1
// if one is present (G-0083); for composite ids regenerates the matching
// `### AC-N — <title>` body heading inside the parent milestone.
// Closes G-065 — the asymmetry where `aiwf rename` exists for slugs
// but no verb exists for titles.
//
// Two positional arguments matching `aiwf rename`'s shape:
// id (or M-NNN/AC-N), new-title. The optional `--reason` flag lands
// in the commit body and surfaces in `aiwf history`, matching the
// pattern from `aiwf promote`/`cancel`/`authorize`/`edit-body`.
func NewCmd(correlationID string) *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "retitle <id> <new-title>",
		Short: "Update an entity's or AC's frontmatter title",
		Long: `Update an entity's or AC's frontmatter title.

A title is a single line: it is written into a one-line YAML scalar, a
` + "`# <id> — <title>`" + ` body H1, and a commit subject, so a title containing a
line break is refused rather than truncated. Put multi-line detail in the
entity body with aiwf edit-body.

Retitling to the title already stored has nothing to change, so a re-run reports
that at exit 0 and commits nothing — unless the body H1 has drifted from the
title, which retitle still repairs. The slug is re-derived only while it still
tracks the title; one set deliberately with aiwf rename is preserved.`,
		Example: `  # Refocus an epic's title after scope shifts
  aiwf retitle E-22 "Planning toolchain hardening" --reason "scope absorbed E-21"

  # Retitle an AC (updates frontmatter and body heading atomically)
  aiwf retitle M-077/AC-1 "retitle works for all top-level kinds"`,
		Args:          cobra.ExactArgs(2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(args[0], args[1], actor, principal, root, reason, *out))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	out = cliutil.AddFormatFlags(cmd)
	out.CorrelationID = correlationID
	cmd.ValidArgsFunction = cliutil.CompleteEntityIDArg("", 0)
	return cmd
}

// Run executes `aiwf retitle`. Returns one of the cliutil.Exit* codes.
func Run(id, newTitle, actor, principal, root, reason string, out cliutil.OutputFormat) (code int) {
	rootDir, actorStr, code, ok := cliutil.ResolvePrelude("aiwf retitle", root, actor)
	if !ok {
		return code
	}

	ctx := context.Background()

	finish := cliutil.BeginVerbDiag(rootDir, "retitle", id, actorStr, out.CorrelationID)
	var sha string
	defer finish(&code, &sha)

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf retitle", out)
	if release == nil {
		return rc
	}
	defer release()

	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil { //coverage:ignore tree.Load errors only on filesystem IO failure (e.g. a permission fault) or context cancellation; malformed entities surface as load findings, not an error here.
		cliutil.Errorf("aiwf retitle: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	result, vErr := verb.Retitle(ctx, tr, id, newTitle, actorStr, reason, cliutil.ConfiguredTitleMaxLength(rootDir))
	pctx := cliutil.ProvenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	code, sha = cliutil.DecorateAndFinish(ctx, rootDir, "aiwf retitle", tr, result, vErr, pctx, out)
	return code
}
