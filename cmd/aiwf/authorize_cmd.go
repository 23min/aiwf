package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/ai-workflow-v2/internal/tree"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// newAuthorizeCmd builds `aiwf authorize <id>` in its three modes:
//
//	aiwf authorize <id> --to <agent> [--reason "..."] [--force]
//	aiwf authorize <id> --pause "<reason>"
//	aiwf authorize <id> --resume "<reason>"
//
// Exactly one of --to / --pause / --resume must be set. The dispatcher
// resolves the actor (must be human/), locks the repo, loads the
// existing scopes for the entity from git log (so --pause / --resume
// can find their target), and hands off to verb.Authorize.
//
// Per docs/pocv3/design/provenance-model.md §"The aiwf authorize verb"
// the `--reason` argument has a different role per mode:
//   - --to: optional; the rationale the human writes when opening a scope.
//   - --pause / --resume: required; the argument to the flag is itself
//     the reason (e.g. `--pause "blocked by E-09"`). Passing both
//     `--pause "..."` and `--reason "..."` is a usage error.
func newAuthorizeCmd() *cobra.Command {
	var (
		actor  string
		root   string
		to     string
		pause  string
		resume string
		reason string
		force  bool
	)
	cmd := &cobra.Command{
		Use:   "authorize <id>",
		Short: "Open / pause / resume an autonomous-work scope on an entity",
		Example: `  # Delegate autonomous work on an epic to ai/claude
  aiwf authorize E-14 --to ai/claude --reason "delegated cobra+completion epic"

  # Pause the most-recently-opened active scope
  aiwf authorize E-14 --pause "blocked on review feedback"

  # Resume the most-recently-paused scope
  aiwf authorize E-14 --resume "review feedback addressed"`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runAuthorizeCmd(args[0], actor, root, to, pause, resume, reason, force))
		},
	}
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer (default: derived from git config user.email; must be human/...)")
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&to, "to", "", "agent the scope authorizes (e.g. ai/claude); opens a new scope")
	cmd.Flags().StringVar(&pause, "pause", "", "pause the most-recently-opened active scope on <id>; the argument is the reason")
	cmd.Flags().StringVar(&resume, "resume", "", "resume the most-recently-paused scope on <id>; the argument is the reason")
	cmd.Flags().StringVar(&reason, "reason", "", "rationale text for --to (optional) / --force (required); ignored by --pause and --resume (their argument is the reason)")
	cmd.Flags().BoolVar(&force, "force", false, "open a fresh scope on a terminal scope-entity (requires --reason)")
	cmd.ValidArgsFunction = completeEntityIDArg("", 0)
	return cmd
}

func runAuthorizeCmd(id, actor, root, to, pause, resume, reason string, force bool) int {
	modes := 0
	if to != "" {
		modes++
	}
	if pause != "" {
		modes++
	}
	if resume != "" {
		modes++
	}
	if modes != 1 {
		fmt.Fprintln(os.Stderr, "aiwf authorize: pick exactly one of --to <agent>, --pause \"<reason>\", or --resume \"<reason>\"")
		return exitUsage
	}
	// `--reason` is meaningful only with --to (and --to --force). For
	// --pause / --resume the flag value IS the reason; a separate
	// --reason would be ambiguous.
	if (pause != "" || resume != "") && reason != "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --reason is not used with --pause / --resume; the argument to --pause/--resume is itself the reason")
		return exitUsage
	}
	if force && to == "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --force is only meaningful with --to (overrides terminal-scope-entity refusal)")
		return exitUsage
	}
	if force && strings.TrimSpace(reason) == "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --force requires --reason \"...\" (non-empty after trim)")
		return exitUsage
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf authorize: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf authorize: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf authorize")
	if release == nil {
		return rc
	}
	defer release()

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf authorize: loading tree: %v\n", err)
		return exitInternal
	}

	opts := verb.AuthorizeOptions{}
	switch {
	case to != "":
		opts.Mode = verb.AuthorizeOpen
		opts.Agent = to
		opts.Reason = reason
		opts.Force = force
	case pause != "":
		opts.Mode = verb.AuthorizePause
		opts.Reason = pause
	case resume != "":
		opts.Mode = verb.AuthorizeResume
		opts.Reason = resume
	}
	if opts.Mode == verb.AuthorizePause || opts.Mode == verb.AuthorizeResume {
		scopes, scopesErr := loadEntityScopes(ctx, rootDir, id)
		if scopesErr != nil {
			fmt.Fprintf(os.Stderr, "aiwf authorize: %v\n", scopesErr)
			return exitInternal
		}
		opts.Scopes = scopes
	}

	result, vErr := verb.Authorize(ctx, tr, id, actorStr, opts)
	return finishVerb(ctx, rootDir, "aiwf authorize", result, vErr)
}
