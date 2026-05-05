package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/tree"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// runAuthorize handles `aiwf authorize <id>` in its three modes:
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
func runAuthorize(args []string) int {
	fs := flag.NewFlagSet("authorize", flag.ContinueOnError)
	actor := fs.String("actor", "", "actor for the commit trailer (default: derived from git config user.email; must be human/...)")
	root := fs.String("root", "", "consumer repo root")
	to := fs.String("to", "", "agent the scope authorizes (e.g. ai/claude); opens a new scope")
	pause := fs.String("pause", "", "pause the most-recently-opened active scope on <id>; the argument is the reason")
	resume := fs.String("resume", "", "resume the most-recently-paused scope on <id>; the argument is the reason")
	reason := fs.String("reason", "", "rationale text for --to (optional) / --force (required); ignored by --pause and --resume (their argument is the reason)")
	force := fs.Bool("force", false, "open a fresh scope on a terminal scope-entity (requires --reason)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(reorderFlagsFirst(args, []string{"actor", "root", "to", "pause", "resume", "reason"}, []string{"force"})); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "aiwf authorize: usage: aiwf authorize <id> --to <agent> [--reason \"...\"] | --pause \"<reason>\" | --resume \"<reason>\"")
		return exitUsage
	}
	id := rest[0]

	modes := 0
	if *to != "" {
		modes++
	}
	if *pause != "" {
		modes++
	}
	if *resume != "" {
		modes++
	}
	if modes != 1 {
		fmt.Fprintln(os.Stderr, "aiwf authorize: pick exactly one of --to <agent>, --pause \"<reason>\", or --resume \"<reason>\"")
		return exitUsage
	}
	// `--reason` is meaningful only with --to (and --to --force). For
	// --pause / --resume the flag value IS the reason; a separate
	// --reason would be ambiguous.
	if (*pause != "" || *resume != "") && *reason != "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --reason is not used with --pause / --resume; the argument to --pause/--resume is itself the reason")
		return exitUsage
	}
	if *force && *to == "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --force is only meaningful with --to (overrides terminal-scope-entity refusal)")
		return exitUsage
	}
	if *force && strings.TrimSpace(*reason) == "" {
		fmt.Fprintln(os.Stderr, "aiwf authorize: --force requires --reason \"...\" (non-empty after trim)")
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf authorize: %v\n", err)
		return exitUsage
	}
	actorStr, err := resolveActor(*actor, rootDir)
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
	case *to != "":
		opts.Mode = verb.AuthorizeOpen
		opts.Agent = *to
		opts.Reason = *reason
		opts.Force = *force
	case *pause != "":
		opts.Mode = verb.AuthorizePause
		opts.Reason = *pause
	case *resume != "":
		opts.Mode = verb.AuthorizeResume
		opts.Reason = *resume
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
