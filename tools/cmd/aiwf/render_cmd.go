package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/roadmap"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// runRender is the dispatcher for `aiwf render <subcommand>`. The
// only subcommand today is `roadmap`; new derived views can be added
// alongside it without disturbing the rest of the CLI.
func runRender(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "aiwf render: missing subcommand. Try 'aiwf render roadmap'.")
		return exitUsage
	}
	switch args[0] {
	case "roadmap":
		return runRenderRoadmap(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "aiwf render: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

// runRenderRoadmap prints the markdown roadmap to stdout, or with
// --write replaces ROADMAP.md and creates a single commit. When the
// rendered output already matches the on-disk file, --write is a
// no-op (no commit) so the verb is safely re-runnable in CI.
func runRenderRoadmap(args []string) int {
	fs := flag.NewFlagSet("render roadmap", flag.ContinueOnError)
	root := fs.String("root", "", "consumer repo root")
	write := fs.Bool("write", false, "write ROADMAP.md and commit (no-op when content is unchanged)")
	actor := fs.String("actor", "", "actor for the commit trailer (only with --write)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	rootDir, err := resolveRoot(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", err)
		return exitUsage
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: loading tree: %v\n", err)
		return exitInternal
	}

	content := roadmap.Render(tr)

	// Preserve a hand-curated `## Candidates` (or `## Backlog`) block
	// from any existing ROADMAP.md. The section is verbatim user
	// content — aiwf doesn't parse it — and survives regenerate
	// cycles. When --write is off we still merge so stdout matches
	// what --write would produce.
	dest := filepath.Join(rootDir, "ROADMAP.md")
	existing, readErr := os.ReadFile(dest)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", readErr)
		return exitInternal
	}
	content = roadmap.AppendCandidates(content, roadmap.ExtractCandidates(existing))

	if !*write {
		if _, werr := os.Stdout.Write(content); werr != nil {
			fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", werr)
			return exitInternal
		}
		return exitOK
	}

	if bytes.Equal(existing, content) {
		fmt.Println("aiwf render roadmap: ROADMAP.md is already up to date.")
		return exitOK
	}

	actorStr, err := resolveActor(*actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", err)
		return exitUsage
	}

	if err := os.WriteFile(dest, content, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", err)
		return exitInternal
	}
	if err := gitops.Add(ctx, rootDir, "ROADMAP.md"); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", err)
		return exitInternal
	}
	subject := "aiwf render roadmap"
	trailers := []gitops.Trailer{
		{Key: "aiwf-verb", Value: "render-roadmap"},
		{Key: "aiwf-actor", Value: actorStr},
	}
	if err := gitops.Commit(ctx, rootDir, subject, trailers); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", err)
		return exitInternal
	}
	fmt.Println(subject)
	return exitOK
}
