// Package worktree implements the `aiwf worktree` verb namespace.
// One child today (`add`): create a git worktree and materialize
// aiwf's marker-managed ritual artifacts (skills, agents, templates,
// guidance) into it in one atomic step, so a fresh worktree never
// starts without them (G-0374).
package worktree

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/version"
)

// NewCmd builds the `aiwf worktree` parent command. Non-Runnable;
// dispatches to `add`.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "worktree",
		Short:         "Worktree-scoped verbs",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newAddCmd())
	return cmd
}

func newAddCmd() *cobra.Command {
	var (
		root      string
		base      string
		printPath bool
		out       *cliutil.OutputFormat
	)
	cmd := &cobra.Command{
		Use:   "add <branch> [path]",
		Short: "Create a git worktree and materialize aiwf rituals into it atomically",
		Example: `  # Create an in-repo worktree for a new branch off main
  aiwf worktree add epic/E-0099-my-epic --base main

  # Create a worktree at an explicit sibling directory
  aiwf worktree add milestone/M-0300-my-milestone ../aiwf-milestone

  # Compose with cd (only the path is printed on success)
  cd "$(aiwf worktree add patch/G-0100-fix --print-path)"`,
		Args:          cobra.RangeArgs(1, 2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			path := ""
			if len(args) == 2 {
				path = args[1]
			}
			return cliutil.WrapExitCode(Run(args[0], path, base, root, printPath, *out))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root (the source checkout 'git worktree add' runs from)")
	cmd.Flags().StringVar(&base, "base", "", "commit-ish the new branch starts from; only valid when <branch> does not already exist locally (default: HEAD)")
	cmd.Flags().BoolVar(&printPath, "print-path", false, "print only the resulting absolute path to stdout, for shell cd composition; nothing else on success, nothing on failure")
	out = cliutil.AddFormatFlags(cmd)
	return cmd
}

// Run executes `aiwf worktree add`. Returns one of the cliutil.Exit*
// codes. Every error path writes to stderr only — never stdout —
// which is what makes --print-path's "nothing on stdout on failure"
// contract hold without a separate code path per output mode.
func Run(branch, path, base, root string, printPath bool, out cliutil.OutputFormat) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore cliutil.ResolveRoot only fails on a broken cwd (filepath.Abs / os.Getwd); not deterministically reproducible.
		return fail("aiwf worktree add", err, cliutil.ExitUsage)
	}

	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf worktree add")
	if release == nil { //coverage:ignore cliutil.AcquireRepoLock only returns nil on lock contention from a concurrent verb invocation; not reproducible in serial tests.
		return rc
	}
	defer release()

	ctx := context.Background()

	exists, err := gitops.BranchExists(ctx, rootDir, branch)
	if err != nil { //coverage:ignore gitops.BranchExists only errors on a non-exit-1 git failure (missing git binary, repo corruption); not deterministically reproducible.
		return fail("aiwf worktree add", err, cliutil.ExitInternal)
	}
	if exists && base != "" {
		return fail("aiwf worktree add", fmt.Errorf("--base is only valid when <branch> does not already exist locally (%q already exists)", branch), cliutil.ExitUsage)
	}

	targetPath := path
	if targetPath == "" {
		// Default-path resolution routes through config.WorktreeDir(),
		// which owns the repo-escape rejection (M-0190/AC-4). An
		// explicit path (the branch above) never reaches WorktreeDir,
		// so it is never subject to that rejection (M-0233/AC-3).
		cfg, _ := config.Load(rootDir)
		targetPath = filepath.Join(cfg.WorktreeDir(), branch)
	}

	if exists {
		err = gitops.WorktreeAdd(ctx, rootDir, targetPath, branch)
	} else {
		err = gitops.WorktreeAddNewBranch(ctx, rootDir, targetPath, branch, base)
	}
	if err != nil {
		return fail("aiwf worktree add", err, cliutil.ExitInternal)
	}

	absPath, err := resolveCreatedPath(rootDir, targetPath)
	if err != nil { //coverage:ignore resolveCreatedPath only errors on filepath.Abs failure (broken cwd); not deterministically reproducible.
		return fail("aiwf worktree add", err, cliutil.ExitInternal)
	}

	wtCfg, err := config.Load(absPath)
	if err != nil {
		return fail("aiwf worktree add", fmt.Errorf("reading aiwf.yaml in the new worktree: %w", err), cliutil.ExitInternal)
	}

	steps, conflict, err := initrepo.RefreshArtifacts(ctx, absPath, initrepo.RefreshOptions{
		StatusMdAutoUpdate: wtCfg.StatusMdAutoUpdate(),
		WireClaudeMd:       wtCfg.WireClaudeMd(),
	})
	if err != nil { //coverage:ignore RefreshArtifacts fails only on a filesystem fault (permission denied, disk full) writing marker-managed artifacts; not deterministically reproducible.
		return fail("aiwf worktree add", err, cliutil.ExitInternal)
	}

	if conflict {
		for _, s := range steps {
			printStep(s)
		}
		fmt.Fprintln(os.Stderr, "aiwf worktree add: hook chain collision (G45) in the new worktree; "+
			"resolve manually (merge the existing hook into its `.local` sibling, delete the original) "+
			"and re-run `aiwf update` there.")
		return cliutil.ExitFindings
	}

	if printPath {
		fmt.Println(absPath)
		return cliutil.ExitOK
	}

	if out.JSON() {
		// D-0013: JSON output is a single clean envelope on stdout — the
		// materialization ledger (text-mode only, below) must not precede it.
		env := render.Envelope{
			Tool:    "aiwf",
			Version: version.Current().Version,
			Status:  "ok",
			Result:  map[string]any{"path": absPath},
		}
		if werr := render.JSON(os.Stdout, env, out.Pretty); werr != nil { //coverage:ignore render.JSON to os.Stdout fails only on a write fault (broken pipe, closed fd); not deterministically reproducible.
			return fail("aiwf worktree add", werr, cliutil.ExitInternal)
		}
		return cliutil.ExitOK
	}

	for _, s := range steps {
		printStep(s)
	}
	fmt.Println(absPath)
	return cliutil.ExitOK
}

// resolveCreatedPath turns the (possibly relative) path passed to
// `git worktree add` into the canonical absolute path of the
// resulting worktree. Relative paths resolve against rootDir, the
// same directory git itself used as cmd.Dir when creating it.
// Symlinks are resolved when possible so the reported path matches
// what a subsequent `cd` would land on (e.g. macOS's /var vs
// /private/var), falling back to the plain absolute path when
// EvalSymlinks fails (e.g. permissions).
func resolveCreatedPath(rootDir, targetPath string) (string, error) {
	abs := targetPath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(rootDir, targetPath)
	}
	abs, err := filepath.Abs(abs)
	if err != nil { //coverage:ignore filepath.Abs only fails on a broken cwd; not deterministically reproducible.
		return "", fmt.Errorf("resolving worktree path: %w", err)
	}
	if resolved, evalErr := filepath.EvalSymlinks(abs); evalErr == nil {
		return resolved, nil
	}
	return abs, nil //coverage:ignore EvalSymlinks fails only when the just-created worktree path doesn't resolve (permission fault, filesystem race); not deterministically reproducible.
}

// printStep renders one initrepo.StepResult in the same ledger shape
// `aiwf update` prints, so the two verbs' output stays visually
// consistent for an operator running either.
func printStep(s initrepo.StepResult) {
	if s.Detail != "" {
		fmt.Printf("  %-9s  %s  (%s)\n", s.Action, s.What, s.Detail)
	} else {
		fmt.Printf("  %-9s  %s\n", s.Action, s.What)
	}
}

// fail writes a text error to stderr and returns code. Centralized so
// every error path in Run stays stdout-clean by construction — the
// property --print-path's contract depends on.
func fail(label string, err error, code int) int {
	fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
	return code
}
