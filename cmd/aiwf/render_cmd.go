package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/config"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/htmlrender"
	"github.com/23min/ai-workflow-v2/internal/render"
	"github.com/23min/ai-workflow-v2/internal/roadmap"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// newRenderCmd builds `aiwf render`. Two surfaces:
//   - `aiwf render roadmap [--write]` → markdown roadmap.
//   - `aiwf render --format=html [...]` → static-site HTML render.
//
// Roadmap is a Cobra subcommand; html mode is the parent's RunE
// (matches the existing public CLI shape rather than introducing a new
// `render html` subverb that would break consumer scripts).
func newRenderCmd() *cobra.Command {
	var (
		root      string
		format    string
		out       string
		scope     string
		noHistory bool
		pretty    bool
	)
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Produce derived views of the planning tree",
		Example: `  # Render the static-site governance pages under ./site
  aiwf render --format=html

  # Render to a custom output directory
  aiwf render --format=html --out /tmp/aiwf-site --pretty`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			if format == "" {
				fmt.Fprintln(os.Stderr, "aiwf render: missing subcommand or --format. Try 'aiwf render roadmap' or 'aiwf render --format=html'.")
				return &exitError{code: exitUsage}
			}
			return wrapExitCode(runRenderSiteCmd(root, format, out, scope, noHistory, pretty))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().StringVar(&format, "format", "", "output format (required: html)")
	cmd.Flags().StringVar(&out, "out", "", "output directory (overrides aiwf.yaml.html.out_dir; default 'site')")
	cmd.Flags().StringVar(&scope, "scope", "", "render only this entity and its referenced children (reserved; not yet implemented)")
	cmd.Flags().BoolVar(&noHistory, "no-history", false, "skip git-log walks per page (reserved; not yet implemented)")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent the JSON envelope on stdout")
	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"html"},
		cobra.ShellCompDirectiveNoFileComp,
	))
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		if c == cmd {
			printRenderHelp()
			return
		}
		// Non-render-parent descendants render Cobra's standard usage
		// block directly. SetHelpFunc on render is inherited by every
		// descendant, and c.Help() would re-enter this function and
		// recurse to stack overflow — same shape as the bug-fix on the
		// root SetHelpFunc in newRootCmd. M-061 AC-5 pins this.
		out := c.OutOrStderr()
		switch {
		case c.Long != "":
			_, _ = fmt.Fprintln(out, c.Long)
			_, _ = fmt.Fprintln(out)
		case c.Short != "":
			_, _ = fmt.Fprintln(out, c.Short)
			_, _ = fmt.Fprintln(out)
		}
		_, _ = fmt.Fprint(out, c.UsageString())
	})
	cmd.AddCommand(newRenderRoadmapCmd())
	// `aiwf render help` is a positional alias for `aiwf render --help`,
	// matching the pre-Cobra dispatcher's accepted shapes. Hidden so it
	// does not appear in the auto-generated subcommand list.
	cmd.AddCommand(&cobra.Command{
		Use:    "help",
		Short:  "Show help for aiwf render",
		Hidden: true,
		Run: func(_ *cobra.Command, _ []string) {
			printRenderHelp()
		},
	})
	return cmd
}

// printRenderHelp emits the verb's catalog of surfaces. Two
// surfaces today (roadmap + html); the catalog is colocated with
// the dispatcher so adding a third later only requires one edit.
// The master verb catalog lives in `aiwf help`.
func printRenderHelp() {
	fmt.Println(`aiwf render — produce derived views of the planning tree.

Surfaces:
  aiwf render roadmap [--write]
      Markdown roadmap (epics + milestones). Prints to stdout by
      default; with --write replaces ROADMAP.md and creates a
      single commit.

  aiwf render --format=html [--out <dir>] [--scope <id>] [--no-history] [--pretty]
      Static-site governance render: index.html + one page per
      epic / milestone, plus a status.html page. Default output
      directory is 'site/' (override via --out or the
      aiwf.yaml.html.out_dir field). Read-only; no commit. The
      JSON envelope on stdout reports out_dir, files_written,
      and elapsed_ms.

See 'aiwf help' for the master verb catalog.`)
}

// newRenderRoadmapCmd builds `aiwf render roadmap`: prints the markdown
// roadmap to stdout, or with --write replaces ROADMAP.md and creates a
// single commit. When the rendered output already matches the on-disk
// file, --write is a no-op (no commit) so the verb is safely re-runnable
// in CI.
func newRenderRoadmapCmd() *cobra.Command {
	var (
		root  string
		write bool
		actor string
	)
	cmd := &cobra.Command{
		Use:   "roadmap",
		Short: "Print or write the markdown roadmap",
		Example: `  # Print the markdown roadmap to stdout
  aiwf render roadmap

  # Replace ROADMAP.md and create a single commit
  aiwf render roadmap --write`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runRenderRoadmapCmd(root, write, actor))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&write, "write", false, "write ROADMAP.md and commit (no-op when content is unchanged)")
	cmd.Flags().StringVar(&actor, "actor", "", "actor for the commit trailer (only with --write)")
	return cmd
}

func runRenderRoadmapCmd(root string, write bool, actor string) int {
	rootDir, err := resolveRoot(root)
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

	if !write {
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

	actorStr, err := resolveActor(actor, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", err)
		return exitUsage
	}

	release, rc := acquireRepoLock(rootDir, "aiwf render roadmap")
	if release == nil {
		return rc
	}
	defer release()

	// G34: isolate the user's pre-existing staged changes from the
	// render-roadmap commit. If the user has staged ROADMAP.md
	// themselves (manual edit), refuse — we can't pick between their
	// content and the regenerated content. Other staged paths are
	// pushed onto the stash for the duration of the commit and
	// popped after.
	staged, err := gitops.StagedPaths(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: checking pre-staged changes: %v\n", err)
		return exitInternal
	}
	for _, p := range staged {
		if p == "ROADMAP.md" {
			fmt.Fprintf(os.Stderr,
				"aiwf render roadmap: ROADMAP.md is already staged with your own edits.\n"+
					"  run `git restore --staged ROADMAP.md` (or `git stash`) and re-run.\n")
			return exitUsage
		}
	}
	stashed := false
	if len(staged) > 0 {
		if err := gitops.StashStaged(ctx, rootDir, "aiwf pre-verb stash: render roadmap"); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf render roadmap: stashing pre-staged changes: %v\n", err)
			return exitInternal
		}
		stashed = true
	}
	defer func() {
		if stashed {
			if popErr := gitops.StashPop(ctx, rootDir); popErr != nil {
				fmt.Fprintf(os.Stderr,
					"aiwf render roadmap: restoring your pre-staged changes failed: %v\n"+
						"  your work is safe in `git stash list`; run `git stash pop` to restore it\n",
					popErr)
			}
		}
	}()

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
	if err := gitops.Commit(ctx, rootDir, subject, "", trailers); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", err)
		return exitInternal
	}
	fmt.Println(subject)
	return exitOK
}

// runRenderSiteCmd handles `aiwf render --format=html [--out <dir>]
// [--scope <id>] [--no-history] [--pretty]`. Read-only — produces a
// directory of HTML files. No commit. Always emits the standard JSON
// envelope on stdout per I3 plan §5; --pretty toggles indent.
//
// Result payload:
//
//	{ "result": { "out_dir": "<abs>", "files_written": N, "elapsed_ms": M } }
func runRenderSiteCmd(root, format, out, scope string, noHistory, pretty bool) int {
	if format != "html" {
		fmt.Fprintf(os.Stderr, "aiwf render: --format must be 'html'; got %q\n", format)
		return exitUsage
	}
	_ = scope     // step-4 placeholder: reserved for §3 incremental render
	_ = noHistory // step-4 placeholder: reserved for the no-history flag

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render: %v\n", err)
		return exitUsage
	}

	ctx := context.Background()
	tr, loadErrs, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render: loading tree: %v\n", err)
		return exitInternal
	}
	cfg, _ := config.Load(rootDir)
	findings := check.Run(tr, loadErrs)
	resolver := newRenderResolver(ctx, rootDir, tr, cfg, findings)

	outDir := resolveHTMLOutDir(rootDir, out)
	res, err := htmlrender.Render(htmlrender.Options{
		OutDir: outDir,
		Tree:   tr,
		Root:   rootDir,
		Data:   resolver,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render: %v\n", err)
		return exitInternal
	}

	env := render.Envelope{
		Tool:    "aiwf",
		Version: Version,
		Status:  "ok",
		Result: map[string]any{
			"out_dir":       outDir,
			"files_written": res.FilesWritten,
			"elapsed_ms":    res.ElapsedMs,
		},
		Metadata: map[string]any{"root": rootDir},
	}
	if werr := render.JSON(os.Stdout, env, pretty); werr != nil {
		fmt.Fprintf(os.Stderr, "aiwf render: %v\n", werr)
		return exitInternal
	}
	return exitOK
}

// resolveHTMLOutDir picks the absolute output path. Precedence:
//  1. --out flag (if non-empty).
//  2. aiwf.yaml.html.out_dir (if non-empty).
//  3. config.DefaultHTMLOutDir.
//
// Relative paths resolve against the consumer repo root so the
// behavior is identical regardless of cwd.
func resolveHTMLOutDir(root, flagOut string) string {
	out := flagOut
	if out == "" {
		if cfg, err := config.Load(root); err == nil && cfg != nil {
			out = cfg.HTMLOutDir()
		} else {
			out = config.DefaultHTMLOutDir
		}
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(root, out)
	}
	return out
}
