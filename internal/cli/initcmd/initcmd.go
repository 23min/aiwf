// Package initcmd implements the `aiwf init` verb (per-verb subpackage of M-0116;
// directory and package are `initcmd` because `init` is a special Go function name).
package initcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/skills"
)

// NewCmd builds `aiwf init`: writes aiwf.yaml, scaffolds entity
// directories, materializes skills, appends to .gitignore, writes a
// CLAUDE.md template, and installs the pre-push hook. No commit.
//
// --dry-run reports the would-be ledger without touching disk.
// --skip-hook performs every other step but omits hook installation.
func NewCmd() *cobra.Command {
	var (
		root          string
		actor         string
		dryRun        bool
		skipHook      bool
		statusline    bool
		scope         string
		wireSettings  bool
		allowUntagged bool
		enableHooks   []string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "One-time setup: aiwf.yaml, scaffolding, skills, pre-push hook",
		Long: `One-time setup: writes aiwf.yaml, scaffolds entity directories, materializes skills, appends to .gitignore, writes a CLAUDE.md template, and installs the pre-push hook.

Safe to re-run: init is idempotent. A second run never overwrites an existing aiwf.yaml, .claude/settings.json, or user-authored git hooks — only derived artifacts (skills, aiwf.example.yaml, the hooks aiwf manages, STATUS.md wiring) refresh.`,
		Example: `  # Scaffold a fresh consumer repo (run once)
  aiwf init

  # Preview what init would do without writing
  aiwf init --dry-run

  # Same scaffolding plus the aiwf-aware Claude Code statusline
  # (user scope by default; add --scope project to keep it in-repo)
  aiwf init --statusline
  aiwf init --statusline --scope project

  # Consent to a specific registry hook without an interactive prompt
  aiwf init --enable-hook <hook-name>`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(root, actor, dryRun, skipHook, statusline, scope, wireSettings, allowUntagged, enableHooks, skills.ShippedHooks))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (default: cwd)")
	cmd.Flags().StringVar(&actor, "actor", "", "default actor for the commit trailer (overrides git config derivation)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what init would do without writing anything")
	cmd.Flags().BoolVar(&skipHook, "skip-hook", false, "skip installing the pre-push hook (every other step still runs)")
	cmd.Flags().BoolVar(&statusline, "statusline", false, "also scaffold the aiwf-aware Claude Code statusline script (refreshed to the embedded copy on each --statusline run)")
	cmd.Flags().StringVar(&scope, "scope", string(skills.StatuslineScopeUser), "where --statusline writes the script: user (~/.claude, default — resolves in any worktree) or project (<repo>/.claude, opt-in)")
	_ = cmd.RegisterFlagCompletionFunc("scope", cobra.FixedCompletions(
		[]string{string(skills.StatuslineScopeProject), string(skills.StatuslineScopeUser)},
		cobra.ShellCompDirectiveNoFileComp,
	))
	cmd.Flags().BoolVar(&wireSettings, "wire-settings", false, "write statusLine to the settings file without interactive confirmation (non-TTY consent per ADR-0015)")
	cmd.Flags().BoolVar(&allowUntagged, "allow-untagged-statusline", false, "write the statusline script even when this binary's version is untagged (a dev/worktree build), without interactive confirmation (G-0367)")
	cmd.Flags().StringArrayVar(&enableHooks, "enable-hook", nil, "consent to enabling the named registry hook without an interactive prompt (repeatable; non-TTY consent per ADR-0032)")
	_ = cmd.RegisterFlagCompletionFunc("enable-hook", completeHookNames)
	return cmd
}

// completeHookNames offers the shipped hook registry's names for
// `--enable-hook <TAB>`. Empty (no completions) until a milestone registers
// the first concrete hook (M-0236) — mirrors completeDeclaredValidators'
// shape for a runtime-derived, possibly-empty set.
func completeHookNames(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return skills.HookNamesFrom(skills.ShippedHooks), cobra.ShellCompDirectiveNoFileComp
}

// Run executes `aiwf init`. Returns one of the cliutil.Exit* codes.
// When `statusline` is true, also scaffolds the aiwf-aware Claude Code
// statusline (scope-appropriate destination; scaffold-if-absent, never
// clobbers a pre-existing copy). The scaffold action runs after the
// main init pipeline succeeds; a `--dry-run` init reports without
// scaffolding.
//
// hooks is the shipped hook registry (ADR-0032); the production call site
// passes skills.ShippedHooks, and tests inject a synthetic registry to
// exercise the consent-gating step independent of the real registry's
// current contents. Every
// registry hook is gated (enableHooks bypasses the interactive prompt for
// the named ones) and the resulting decisions are baked into the
// freshly-written aiwf.yaml, after the main init pipeline succeeds — a
// `--dry-run` init skips gating entirely, same as the statusline scaffold.
func Run(root, actor string, dryRun, skipHook, statusline bool, scope string, wireSettings, allowUntagged bool, enableHooks []string, hooks []skills.HookDef) int {
	rootDir, err := resolveInitRoot(root)
	if err != nil {
		cliutil.Errorf("aiwf init: %v\n", err)
		return cliutil.ExitUsage
	}

	if !dryRun {
		release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf init", cliutil.OutputFormat{})
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
		cliutil.Errorf("aiwf init: %v\n", err)
		return cliutil.ExitInternal
	}

	if res.DryRun {
		cliutil.Println("aiwf init: dry-run — nothing was written.")
	}
	for _, s := range res.Steps {
		if s.Detail != "" {
			cliutil.Printf("  %-9s  %s  (%s)\n", s.Action, s.What, s.Detail)
		} else {
			cliutil.Printf("  %-9s  %s\n", s.Action, s.What)
		}
	}

	if res.HookConflict {
		cliutil.Println()
		cliutil.Println("aiwf init: hook chain collision (G45).")
		cliutil.Println("aiwf wanted to migrate a pre-existing non-aiwf hook to its `.local`")
		cliutil.Println("sibling, but a `.local` file already exists. To preserve your work,")
		cliutil.Println("aiwf left both files untouched.")
		cliutil.Println()
		cliutil.Println("Resolve manually:")
		cliutil.Println("  1. Open the existing hook (.git/hooks/pre-push and/or pre-commit) and")
		cliutil.Println("     the `.local` sibling that's blocking the migration.")
		cliutil.Println("  2. Merge the content into one file at the `.local` path.")
		cliutil.Println("  3. Delete the original (non-`.local`) hook.")
		cliutil.Println("  4. Re-run `aiwf init`.")
		cliutil.Println()
		cliutil.Println("Until then, `aiwf check` does not run automatically on `git push`/`git commit`.")
		cliutil.Println("You can still validate manually with `aiwf check`.")
		return cliutil.ExitFindings
	}

	switch {
	case res.DryRun:
		cliutil.Println("\naiwf init: dry-run complete. Re-run without --dry-run to apply.")
	case skipHook:
		cliutil.Println("\naiwf init: done (pre-push hook skipped). Commit aiwf.yaml when you're ready.")
		cliutil.Println("Run `aiwf init` again later to install the hook, or wire `aiwf check` into your push flow manually.")
		cliutil.Println("Skills, ritual skills, agents, and templates were materialized into .claude/ (no plugin install needed; see CLAUDE.md \"Operator setup\").")
	default:
		cliutil.Println("\naiwf init: done. Commit aiwf.yaml when you're ready.")
		cliutil.Println("Skills, ritual skills, agents, and templates were materialized into .claude/ (no plugin install needed; see CLAUDE.md \"Operator setup\").")
	}

	if len(hooks) > 0 {
		if !dryRun {
			if rc := gateAndPersistHookDecisions(rootDir, hooks, enableHooks); rc != cliutil.ExitOK { //coverage:ignore gateAndPersistHookDecisions's own failure paths are unit-tested directly (TestGateAndPersistHookDecisions_MissingAiwfYamlReturnsInternal); triggering one from here would require initrepo.Init to report success while leaving no readable aiwf.yaml, which its own contract precludes
				return rc
			}
			if rc := cliutil.SyncHookMaterialization(rootDir, skills.ClaudeTarget, hooks); rc != cliutil.ExitOK { //coverage:ignore SyncHookMaterialization's own failure paths are unit-tested directly against the function itself; triggering one from here would require the aiwf.yaml gateAndPersistHookDecisions just wrote successfully to become unreadable before this call, which its own contract precludes
				return rc
			}
		} else {
			cliutil.Println("aiwf init --enable-hook: dry-run — hook consent gating skipped.")
		}
	}

	if statusline && !dryRun {
		if rc := cliutil.RunStatuslineScaffold(cliutil.StatuslineOpts{
			RootDir:       rootDir,
			Scope:         scope,
			WireSettings:  wireSettings,
			AllowUntagged: allowUntagged,
		}); rc != cliutil.ExitOK {
			return rc
		}
	} else if statusline && dryRun {
		cliutil.Println("aiwf init --statusline: dry-run — statusline scaffold skipped.")
	}
	return cliutil.ExitOK
}

// gateAndPersistHookDecisions runs the consent gate (ADR-0032) over every
// registry hook and splices the resulting decisions into the just-written
// aiwf.yaml's hooks: block, via aiwfyaml's surgical splice so every other
// byte of the file — including the full commented schema reference
// initrepo.Init already wrote — survives untouched.
func gateAndPersistHookDecisions(rootDir string, hooks []skills.HookDef, enableHooks []string) int {
	decisions := cliutil.GateHookDecisions(hooks, enableHooks, false)

	configPath := filepath.Join(rootDir, config.FileName)
	doc, _, err := aiwfyaml.Read(configPath)
	if err != nil {
		cliutil.Errorf("aiwf init: %v\n", err)
		return cliutil.ExitInternal
	}
	doc.SetHooks(decisions)
	if err := doc.Write(configPath); err != nil { //coverage:ignore the preceding Read already succeeded against the same path; only external interference (disk failure, permission change between the two calls) reaches this, not any code path this binary's own control flow produces
		cliutil.Errorf("aiwf init: %v\n", err)
		return cliutil.ExitInternal
	}

	for _, h := range hooks {
		state := "declined"
		if decisions[h.Name] {
			state = "enabled"
		}
		cliutil.Printf("aiwf init: hook %q — %s\n", h.Name, state)
	}
	return cliutil.ExitOK
}

// resolveInitRoot picks the root directory for `aiwf init`. Unlike
// cliutil.ResolveRoot, it does not error when aiwf.yaml is missing — that's
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
