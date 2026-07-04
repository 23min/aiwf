// Package render implements the `aiwf render` verb (per-verb subpackage of M-0116;
// includes the Resolver moved from render_resolver.go).
package render

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/htmlrender"
	"github.com/23min/aiwf/internal/pathutil"
	baserender "github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/roadmap"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/version"
)

// NewCmd builds `aiwf render`. Two surfaces:
//   - `aiwf render roadmap [--write]` → markdown roadmap.
//   - `aiwf render --format=html [...]` → static-site HTML render.
//
// Roadmap is a Cobra subcommand; html mode is the parent's RunE
// (matches the existing public CLI shape rather than introducing a new
// `render html` subverb that would break consumer scripts).
func NewCmd() *cobra.Command {
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
				return cliutil.WrapExitCode(cliutil.ExitUsage)
			}
			return cliutil.WrapExitCode(RunSite(root, format, out, scope, noHistory, pretty))
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
	cmd.AddCommand(newRoadmapCmd())
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
      default; with --write replaces ROADMAP.md on disk only — no
      commit. The caller commits it as part of whatever change
      they were already making.

  aiwf render --format=html [--out <dir>] [--scope <id>] [--no-history] [--pretty]
      Static-site governance render: index.html + one page per
      epic / milestone, plus a status.html page. Default output
      directory is 'site/' (override via --out or the
      aiwf.yaml.html.out_dir field). Read-only; no commit. The
      JSON envelope on stdout reports out_dir, files_written,
      and elapsed_ms.

See 'aiwf help' for the master verb catalog.`)
}

// newRoadmapCmd builds `aiwf render roadmap`: prints the markdown
// roadmap to stdout, or with --write replaces ROADMAP.md on disk.
// --write never commits — the caller stages and commits ROADMAP.md as
// part of whatever change they were already making (G-0350: this used
// to route through verb.Apply and commit on its own, which forced a
// separate gate and refused to run on a dirty tree). When the rendered
// output already matches the on-disk file, --write is a no-op so the
// verb is safely re-runnable in CI.
func newRoadmapCmd() *cobra.Command {
	var (
		root  string
		write bool
	)
	cmd := &cobra.Command{
		Use:   "roadmap",
		Short: "Print or write the markdown roadmap",
		Example: `  # Print the markdown roadmap to stdout
  aiwf render roadmap

  # Replace ROADMAP.md on disk (does not commit)
  aiwf render roadmap --write`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(RunRoadmap(root, write))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&write, "write", false, "write ROADMAP.md to disk, no commit (no-op when content is unchanged)")
	return cmd
}

// RunRoadmap executes `aiwf render roadmap`. Returns one of the cliutil.Exit* codes.
func RunRoadmap(root string, write bool) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()
	tr, _, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}

	// Group the roadmap by area when an areas block is declared (M-0175);
	// flat (today's output) otherwise. Single source of the declared set
	// is the same cliutil accessor status reads.
	areaMembers, areaDefault := cliutil.ConfiguredAreas(rootDir)
	var content []byte
	if len(areaMembers) == 0 {
		content = roadmap.Render(tr)
	} else {
		content = roadmap.RenderGrouped(tr, areaMembers, areaDefault)
	}

	// Reconcile the on-disk roadmap filename across case-sensitive and
	// case-insensitive filesystems (G-0185). A consumer that tracks a
	// lowercase `roadmap.md` would otherwise get a second, divergent
	// `ROADMAP.md` created on Linux/CI while the original stays stale.
	resolvedName := resolveRoadmapName(rootDir)

	// Preserve a hand-curated `## Candidates` (or `## Backlog`) block
	// from any existing roadmap file. The section is verbatim user
	// content — aiwf doesn't parse it — and survives regenerate
	// cycles. When --write is off we still merge so stdout matches
	// what --write would produce.
	dest := filepath.Join(rootDir, resolvedName)
	existing, readErr := os.ReadFile(dest)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", readErr)
		return cliutil.ExitInternal
	}
	content = roadmap.AppendCandidates(content, roadmap.ExtractCandidates(existing))

	if !write {
		if _, werr := os.Stdout.Write(content); werr != nil {
			fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", werr)
			return cliutil.ExitInternal
		}
		return cliutil.ExitOK
	}

	if bytes.Equal(existing, content) {
		fmt.Printf("aiwf render roadmap: %s is already up to date.\n", resolvedName)
		return cliutil.ExitOK
	}

	// Plain atomic write, no git involvement (G-0350): committing
	// ROADMAP.md is the caller's concern, folded into whatever change
	// they were already making. This also means the write runs
	// regardless of what else is staged or dirty in the tree.
	if err := pathutil.AtomicWriteFile(dest, content, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render roadmap: %v\n", err)
		return cliutil.ExitInternal
	}
	fmt.Printf("aiwf render roadmap: wrote %s\n", resolvedName)
	return cliutil.ExitOK
}

// canonicalRoadmapName is the casing aiwf writes by default and the
// name the README and rituals reference. The renderer reconciles to an
// existing case-variant when the consumer already tracks one.
const canonicalRoadmapName = "ROADMAP.md"

// resolveRoadmapName picks the basename `aiwf render roadmap` writes to.
//
// The roadmap is a generated root artifact, and consumers legitimately
// track it under different casing (`roadmap.md` is a common convention).
// On a case-insensitive filesystem (macOS APFS, Windows NTFS) a
// hardcoded `ROADMAP.md` resolves to whatever variant exists; on a
// case-sensitive filesystem (Linux, CI) it does not — so the same repo
// plus the same command would target a different file by filesystem,
// silently creating a second divergent file and losing the consumer's
// hand-curated `## Candidates` block (G-0185).
//
// To make the behavior filesystem-independent, scan the repo root for a
// case-insensitive match of the canonical name:
//   - exactly one match  → return its actual on-disk name (preserve the
//     consumer's casing);
//   - zero matches       → return the canonical `ROADMAP.md`;
//   - more than one match → return the canonical `ROADMAP.md`. This
//     genuinely-broken state is only physically possible on a
//     case-sensitive filesystem; reconciliation cannot silently pick
//     one, so the renderer defaults to canonical and the new
//     `roadmap-case-collision` check finding flags the divergence.
//
// On any directory-read error, fall back to the canonical name.
func resolveRoadmapName(rootDir string) string {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return canonicalRoadmapName
	}
	var matches []string
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		if strings.EqualFold(ent.Name(), canonicalRoadmapName) {
			matches = append(matches, ent.Name())
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return canonicalRoadmapName
}

// RunSite handles `aiwf render --format=html [--out <dir>]
// [--scope <id>] [--no-history] [--pretty]`. Read-only — produces a
// directory of HTML files. No commit. Always emits the standard JSON
// envelope on stdout per I3 plan §5; --pretty toggles indent.
//
// Result payload:
//
//	{ "result": { "out_dir": "<abs>", "files_written": N, "elapsed_ms": M } }
//
// RunSite executes `aiwf render --format=html`. Returns one of the cliutil.Exit* codes.
func RunSite(root, format, out, scope string, noHistory, pretty bool) int {
	if format != "html" {
		fmt.Fprintf(os.Stderr, "aiwf render: --format must be 'html'; got %q\n", format)
		return cliutil.ExitUsage
	}
	_ = scope     // step-4 placeholder: reserved for §3 incremental render
	_ = noHistory // step-4 placeholder: reserved for the no-history flag

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render: %v\n", err)
		return cliutil.ExitUsage
	}

	ctx := context.Background()
	tr, loadErrs, err := tree.Load(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render: loading tree: %v\n", err)
		return cliutil.ExitInternal
	}
	cfg, _ := config.Load(rootDir)
	findings := check.Run(tr, loadErrs)

	// One shared HEAD-history walk feeds every per-entity history row and
	// scope view (E-0054 / M-0221), replacing render's ~N-per-milestone
	// git-log fan-out. Fail loud on a walk error: degrading here would
	// silently blank the history / provenance section of *every* page —
	// strictly worse than the old per-entity best-effort that dropped one
	// tab. A healthy tree never triggers this; a corrupt/partial repo
	// should stop the render, not emit a misleadingly-empty site.
	head, err := check.WalkHeadCommits(ctx, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render: reading history: %v\n", err)
		return cliutil.ExitInternal
	}
	resolver := NewRenderResolver(ctx, rootDir, tr, cfg, findings, head)

	outDir := resolveHTMLOutDir(rootDir, out)
	res, err := htmlrender.Render(htmlrender.Options{
		OutDir: outDir,
		Tree:   tr,
		Root:   rootDir,
		Data:   resolver,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf render: %v\n", err)
		return cliutil.ExitInternal
	}
	emitGitignoreWarning(rootDir, outDir, cfg)

	env := baserender.Envelope{
		Tool:    "aiwf",
		Version: version.Current().Version,
		Status:  "ok",
		Result: map[string]any{
			"out_dir":       outDir,
			"files_written": res.FilesWritten,
			"elapsed_ms":    res.ElapsedMs,
		},
		Metadata: map[string]any{"root": rootDir},
	}
	if werr := baserender.JSON(os.Stdout, env, pretty); werr != nil {
		fmt.Fprintf(os.Stderr, "aiwf render: %v\n", werr)
		return cliutil.ExitInternal
	}
	return cliutil.ExitOK
}

// emitGitignoreWarning probes whether outDir is covered by the
// consumer's .gitignore and prints a one-line stderr warning when it
// isn't. Defense-in-depth for G-056: catches the cases the
// init/update reconciliation cannot — operator passed an ad-hoc
// --out, the consumer hasn't run aiwf update since changing
// html.out_dir, or a custom gitignore workflow stripped the marker
// block. Silent when html.commit_output: true (operator opted in to
// tracking the rendered files), when outDir is outside the repo
// root (gitignore semantics don't apply), or when `git
// check-ignore` is unavailable (fail-soft).
func emitGitignoreWarning(root, outDir string, cfg *config.Config) {
	if cfg != nil && cfg.HTML.CommitOutput {
		return
	}
	rel, err := filepath.Rel(root, outDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return
	}
	target := filepath.ToSlash(rel) + "/"
	cmd := exec.Command("git", "-C", root, "check-ignore", "-q", target)
	err = cmd.Run()
	if err == nil {
		return
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		fmt.Fprintf(os.Stderr,
			"aiwf render: warning: %s is not gitignored; rendered files will appear in `git status`.\n"+
				"             Run `aiwf update` to reconcile, or set `html.commit_output: true` to track them.\n",
			target)
	}
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
