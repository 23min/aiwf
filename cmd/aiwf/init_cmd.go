package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/initrepo"
)

// newInitCmd builds `aiwf init`: writes aiwf.yaml, scaffolds entity
// directories, materializes skills, appends to .gitignore, writes a
// CLAUDE.md template, and installs the pre-push hook. No commit.
//
// --dry-run reports the would-be ledger without touching disk.
// --skip-hook performs every other step but omits hook installation.
func newInitCmd() *cobra.Command {
	var (
		root     string
		actor    string
		dryRun   bool
		skipHook bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "One-time setup: aiwf.yaml, scaffolding, skills, pre-push hook",
		Example: `  # Scaffold a fresh consumer repo (run once)
  aiwf init

  # Preview what init would do without writing
  aiwf init --dry-run`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runInitCmd(root, actor, dryRun, skipHook))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: cwd)")
	cmd.Flags().StringVar(&actor, "actor", "", "default actor for the commit trailer (overrides git config derivation)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what init would do without writing anything")
	cmd.Flags().BoolVar(&skipHook, "skip-hook", false, "skip installing the pre-push hook (every other step still runs)")
	return cmd
}

func runInitCmd(root, actor string, dryRun, skipHook bool) int {
	rootDir, err := resolveInitRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf init: %v\n", err)
		return exitUsage
	}

	if !dryRun {
		release, rc := acquireRepoLock(rootDir, "aiwf init")
		if release == nil {
			return rc
		}
		defer release()
	}

	res, err := initrepo.Init(context.Background(), rootDir, initrepo.Options{
		ActorOverride: actor,
		DryRun:        dryRun,
		SkipHook:      skipHook,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf init: %v\n", err)
		return exitInternal
	}

	if res.DryRun {
		fmt.Println("aiwf init: dry-run — nothing was written.")
	}
	for _, s := range res.Steps {
		if s.Detail != "" {
			fmt.Printf("  %-9s  %s  (%s)\n", s.Action, s.What, s.Detail)
		} else {
			fmt.Printf("  %-9s  %s\n", s.Action, s.What)
		}
	}

	if res.HookConflict {
		fmt.Println()
		fmt.Println("aiwf init: hook chain collision (G45).")
		fmt.Println("aiwf wanted to migrate a pre-existing non-aiwf hook to its `.local`")
		fmt.Println("sibling, but a `.local` file already exists. To preserve your work,")
		fmt.Println("aiwf left both files untouched.")
		fmt.Println()
		fmt.Println("Resolve manually:")
		fmt.Println("  1. Open the existing hook (.git/hooks/pre-push and/or pre-commit) and")
		fmt.Println("     the `.local` sibling that's blocking the migration.")
		fmt.Println("  2. Merge the content into one file at the `.local` path.")
		fmt.Println("  3. Delete the original (non-`.local`) hook.")
		fmt.Println("  4. Re-run `aiwf init`.")
		fmt.Println()
		fmt.Println("Until then, `aiwf check` does not run automatically on `git push`/`git commit`.")
		fmt.Println("You can still validate manually with `aiwf check`.")
		return exitFindings
	}

	switch {
	case res.DryRun:
		fmt.Println("\naiwf init: dry-run complete. Re-run without --dry-run to apply.")
	case skipHook:
		fmt.Println("\naiwf init: done (pre-push hook skipped). Commit aiwf.yaml when you're ready.")
		fmt.Println("Run `aiwf init` again later to install the hook, or wire `aiwf check` into your push flow manually.")
		if !ritualsPluginInstalled(rootDir) {
			printRitualsSuggestion()
		}
	default:
		fmt.Println("\naiwf init: done. Commit aiwf.yaml when you're ready.")
		if !ritualsPluginInstalled(rootDir) {
			printRitualsSuggestion()
		}
	}
	return exitOK
}

// resolveInitRoot picks the root directory for `aiwf init`. Unlike
// resolveRoot, it does not error when aiwf.yaml is missing — that's
// the normal case for init.
func resolveInitRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolving --root: %w", err)
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting cwd: %w", err)
	}
	return cwd, nil
}
