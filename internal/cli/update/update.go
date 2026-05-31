// Package update implements the `aiwf update ` verb (per-verb subpackage of M-0116;
// cmd/aiwf/main.go's newRootCmd wires it via NewCmd).
package update

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/skills"
)

// NewCmd builds `aiwf update`: refreshes every marker-managed
// framework artifact the consumer is opted into. The pipeline is the
// same one `aiwf init` runs after first-time scaffolding —
// `initrepo.RefreshArtifacts` — so init and update converge to the
// same state for a given binary version + aiwf.yaml.
//
// Concretely the verb refreshes:
//   - the embedded skills under .claude/skills/aiwf-*
//   - the .gitignore patterns covering them
//   - the marker-managed pre-push hook
//   - the marker-managed pre-commit hook (gated by
//     aiwf.yaml's status_md.auto_update; default-on)
//
// Hook conflicts (a non-marker hook already in place) are reported
// in the per-step ledger and surface a remediation block, mirroring
// `aiwf init`'s conflict path.
func NewCmd() *cobra.Command {
	var (
		root         string
		statusline   bool
		scope        string
		wireSettings bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Refresh marker-managed framework artifacts (skills, hooks)",
		Example: `  # Refresh skills + hooks against the current binary version
  aiwf update

  # Refresh as above plus scaffold the aiwf-aware statusline if absent
  # (never clobbers an existing copy)
  aiwf update --statusline
  aiwf update --statusline --scope user`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(root, statusline, scope, wireSettings))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&statusline, "statusline", false, "also scaffold the aiwf-aware Claude Code statusline script (writes only if absent; never clobbers an existing copy)")
	cmd.Flags().StringVar(&scope, "scope", string(skills.StatuslineScopeProject), "where --statusline writes the script: project (<repo>/.claude) or user (~/.claude)")
	_ = cmd.RegisterFlagCompletionFunc("scope", cobra.FixedCompletions(
		[]string{string(skills.StatuslineScopeProject), string(skills.StatuslineScopeUser)},
		cobra.ShellCompDirectiveNoFileComp,
	))
	cmd.Flags().BoolVar(&wireSettings, "wire-settings", false, "write statusLine to the settings file without interactive confirmation (non-TTY consent per ADR-0015)")
	return cmd
}

// Run executes `aiwf update`. Returns one of the cliutil.Exit* codes.
// When `statusline` is true, also runs the shared statusline scaffold
// (scope-appropriate destination, scaffold-if-absent — never clobbers
// a pre-existing copy). The scaffold action runs after the artifact
// refresh succeeds.
func Run(root string, statusline bool, scope string, wireSettings bool) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return cliutil.ExitUsage
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf update")
	if release == nil {
		return rc
	}
	defer release()

	cfg, err := config.Load(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return cliutil.ExitInternal
	}

	steps, conflict, err := initrepo.RefreshArtifacts(context.Background(), rootDir, initrepo.RefreshOptions{
		StatusMdAutoUpdate: cfg.StatusMdAutoUpdate(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: %v\n", err)
		return cliutil.ExitInternal
	}

	for _, s := range steps {
		if s.Detail != "" {
			fmt.Printf("  %-9s  %s  (%s)\n", s.Action, s.What, s.Detail)
		} else {
			fmt.Printf("  %-9s  %s\n", s.Action, s.What)
		}
	}

	// G-0136 / M-0133 / AC-2: when invoked from a linked worktree,
	// the hook writes land in the shared `<main>/.git/hooks/` (which
	// git actually fires). Flag the affects-all-worktrees scope so
	// the operator isn't surprised that an update from worktree A
	// changes the hook chain used by worktree B and the main checkout.
	if inWT, err := gitops.InWorktree(context.Background(), rootDir); err == nil && inWT {
		fmt.Println("\nNote: running from a linked worktree. Hook writes go to the shared")
		fmt.Println("`.git/hooks/` directory; this update affects all worktrees of the repo.")
	}

	if conflict {
		fmt.Println()
		fmt.Println("aiwf update: hook chain collision (G45).")
		fmt.Println("A non-aiwf hook would auto-migrate to its `.local` sibling, but a `.local`")
		fmt.Println("file already exists at .git/hooks/pre-push.local or .git/hooks/pre-commit.local.")
		fmt.Println("Resolve manually: merge the existing hook's content into the `.local` file,")
		fmt.Println("delete the original (non-`.local`) hook, and re-run `aiwf update`.")
		return cliutil.ExitFindings
	}

	fmt.Println("\naiwf update: done.")

	if statusline {
		if rc := cliutil.RunStatuslineScaffold(cliutil.StatuslineOpts{
			RootDir:      rootDir,
			Scope:        scope,
			WireSettings: wireSettings,
		}); rc != cliutil.ExitOK {
			return rc
		}
	}
	return cliutil.ExitOK
}
