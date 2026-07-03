// Package update implements the `aiwf update ` verb (per-verb subpackage of M-0116;
// cmd/aiwf/main.go's newRootCmd wires it via NewCmd).
package update

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/doctor"
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
		remove       bool
		force        bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Refresh marker-managed framework artifacts (skills, hooks)",
		Example: `  # Refresh skills + hooks against the current binary version
  aiwf update

  # Refresh as above plus scaffold the aiwf-aware statusline
  # (refreshed to the embedded copy each run; user scope by default)
  aiwf update --statusline
  aiwf update --statusline --scope project

  # Remove a scope's statusline script + settings wiring (G-0354)
  aiwf update --scope project --remove
  aiwf update --scope project --remove --force`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(root, statusline, scope, wireSettings, remove, force))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&statusline, "statusline", false, "also scaffold the aiwf-aware Claude Code statusline script (refreshed to the embedded copy on each --statusline run)")
	cmd.Flags().StringVar(&scope, "scope", string(skills.StatuslineScopeUser), "where --statusline writes the script (or --remove deletes it): user (~/.claude, default — resolves in any worktree) or project (<repo>/.claude, opt-in)")
	_ = cmd.RegisterFlagCompletionFunc("scope", cobra.FixedCompletions(
		[]string{string(skills.StatuslineScopeProject), string(skills.StatuslineScopeUser)},
		cobra.ShellCompDirectiveNoFileComp,
	))
	cmd.Flags().BoolVar(&wireSettings, "wire-settings", false, "write statusLine to the settings file without interactive confirmation (non-TTY consent per ADR-0015)")
	cmd.Flags().BoolVar(&remove, "remove", false, "remove the --scope statusline's script + statusLine settings key (mutually exclusive with --statusline)")
	cmd.Flags().BoolVar(&force, "force", false, "with --remove, delete the script/settings key even if it does not look aiwf-authored")
	return cmd
}

// Run executes `aiwf update`. Returns one of the cliutil.Exit* codes.
// When `statusline` is true, also runs the shared statusline scaffold
// (scope-appropriate destination, byte-refreshed on every update). When
// `remove` is true, instead removes the --scope statusline's script +
// settings key (G-0354); `statusline` and `remove` are mutually
// exclusive. The statusline action runs after the artifact refresh
// succeeds.
func Run(root string, statusline bool, scope string, wireSettings, remove, force bool) int {
	if statusline && remove {
		fmt.Fprintln(os.Stderr, "aiwf update: --statusline and --remove are mutually exclusive")
		return cliutil.ExitUsage
	}

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
		WireClaudeMd:       cfg.WireClaudeMd(),
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

	statuslineRC := cliutil.ExitOK
	switch {
	case statusline:
		// A settings-wiring finding (e.g. a pre-existing, differing
		// statusLine key) is orthogonal to the artifact refresh: the
		// statusline script itself is written and version-marked first, so
		// the health file below must still reflect it. Capture the rc and
		// return it after WriteHealth rather than early-returning here — an
		// early return would skip the health refresh and leave the stoplight
		// rendering a stale pre-mark warning about the marker just written
		// (G-0347).
		statuslineRC = cliutil.RunStatuslineScaffold(cliutil.StatuslineOpts{
			RootDir:      rootDir,
			Scope:        scope,
			WireSettings: wireSettings,
		})
	case remove:
		// Explicit removal (G-0354): mutually exclusive with the
		// upgrade-only auto-refresh below -- the operator asked to
		// tear this scope's wiring down, not refresh it.
		statuslineRC = cliutil.RunStatuslineRemove(cliutil.StatuslineRemoveOpts{
			RootDir: rootDir,
			Scope:   scope,
			Force:   force,
		})
	default:
		// Plain `aiwf update`: upgrade-only auto-refresh of an
		// already-installed statusline (G-0344). Refreshes only an
		// aiwf-marked copy, never below its installed version, and never
		// creates a copy or touches any settings file — initial install
		// and settings wiring stay behind the explicit `--statusline`
		// opt-in (ADR-0015 consent unchanged).
		refreshStatuslineInPlace(rootDir)
	}

	// Refresh the installation-health file so the statusline stoplight reflects
	// the just-updated setup — written last, after every artifact (including the
	// statusline itself) has been refreshed. This runs a full doctor pass
	// (LookPath, tree load, a filesystem-case probe); acceptable at update
	// cadence. Best-effort: a write failure only logs, never fails update.
	if err := doctor.WriteHealth(context.Background(), rootDir, time.Now().UTC().Format(time.RFC3339), doctor.DoctorOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: could not refresh health.aiwf.json: %v\n", err) //coverage:ignore best-effort refresh; post-materialization git is reachable, so WriteHealth fails only on a filesystem fault (mirrors doctor.go runWriteHealth)
	}

	return statuslineRC
}

// refreshStatuslineInPlace runs the upgrade-only statusline
// auto-refresh (G-0344) and reports any action taken. Best-effort: a
// filesystem fault only logs to stderr, never fails the update — the
// statusline is an advisory convenience, not a correctness artifact.
// An already-current copy is silent (no ledger noise on the common
// path); only a real change or a skip-worth-knowing is printed.
func refreshStatuslineInPlace(rootDir string) {
	outcomes, err := skills.AutoRefreshStatusline(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf update: could not auto-refresh statusline: %v\n", err) //coverage:ignore best-effort; AutoRefreshStatusline errors only on a filesystem fault reading an existing script, unreachable from tempdir tests
		return
	}
	for _, o := range outcomes {
		if line, show := o.LedgerLine(); show {
			fmt.Println(line)
		}
	}
}
