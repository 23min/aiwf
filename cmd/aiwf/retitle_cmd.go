package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
)

// newRetitleCmd builds `aiwf retitle <id|composite-id> <new-title>
// [--reason "..."]`. Title mutation: updates the entity's frontmatter
// `title:`; for top-level entities also re-derives the on-disk slug
// (G-0108) and syncs a canonical `# <ID> — <title>` body H1 if one is
// present (G-0083); for composite ids regenerates the matching
// `### AC-N — <title>` body heading inside the parent milestone.
// Closes G-065 — the asymmetry where `aiwf rename` exists for slugs
// but no verb exists for titles.
//
// Two positional arguments matching `aiwf rename`'s shape:
// id (or M-NNN/AC-N), new-title. The optional `--reason` flag lands
// in the commit body and surfaces in `aiwf history`, matching the
// pattern from `aiwf promote`/`cancel`/`authorize`/`edit-body`.
func newRetitleCmd() *cobra.Command {
	var (
		actor     string
		principal string
		root      string
		reason    string
	)
	cmd := &cobra.Command{
		Use:   "retitle <id> <new-title>",
		Short: "Update an entity's or AC's frontmatter title",
		Example: `  # Refocus an epic's title after scope shifts
  aiwf retitle E-22 "Planning toolchain hardening" --reason "scope absorbed E-21"

  # Retitle an AC (updates frontmatter and body heading atomically)
  aiwf retitle M-077/AC-1 "retitle works for all top-level kinds"`,
		Args:          cobra.ExactArgs(2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runRetitleCmd(args[0], args[1], actor, principal, root, reason))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer")
	cmd.Flags().StringVar(&principal, "principal", "", "the human/<id> the actor is acting on behalf of (required when --actor is non-human; gates the verb through the I2.5 allow-rule)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&reason, "reason", "", "free-form prose explaining why; lands in the commit body, surfaces in `aiwf history`")
	cmd.ValidArgsFunction = completeEntityIDArg("", 0)
	return cmd
}

func runRetitleCmd(id, newTitle, actor, principal, root, reason string) int {
	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf retitle: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf retitle: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf retitle")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf retitle: loading tree: %v\n", err)
		return exitInternal
	}
	result, vErr := verb.Retitle(ctx, tr, id, newTitle, actorStr, reason, configuredTitleMaxLength(rootDir))
	pctx := provenanceContext{
		Actor:     actorStr,
		Principal: strings.TrimSpace(principal),
		VerbKind:  verb.VerbAct,
		TargetID:  id,
	}
	return decorateAndFinish(ctx, rootDir, "aiwf retitle", tr, result, vErr, pctx)
}
