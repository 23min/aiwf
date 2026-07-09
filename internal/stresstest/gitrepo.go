package stresstest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// gitInitAndConfig git-inits dir and sets a deterministic commit
// identity. No `aiwf init` is needed — `aiwf add`/`promote` work
// against a bare git repo with no aiwf.yaml. Shared by every scenario
// in this package whose Setup needs a fresh disposable repo.
func gitInitAndConfig(dir string) error {
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "stresstest@example.com"},
		{"config", "user.name", "stresstest"},
	} {
		if err := runGit(dir, args...); err != nil { //coverage:ignore defensive: git init/config on a fresh os.MkdirTemp dir has no realistic failure mode short of filesystem sabotage
			return err
		}
	}
	return nil
}

// runGit runs one git subcommand in dir, returning combined output
// wrapped into the error on failure.
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil { //coverage:ignore defensive: exercised only through call sites whose own git operations (init/config/worktree add/merge) on a scenario-managed disposable repo have no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("git %v: %w\n%s", args, err, out)
	}
	return nil
}

// newSiblingWorktreesFixture creates a main repo with a seed commit
// under dir/main, then adds two sibling worktrees (actor-a, actor-b)
// off it — dir/wt-a, dir/wt-b. Shared by every scenario whose Setup
// needs two independent working copies of one repo (M-0241/AC-3,
// AC-5).
func newSiblingWorktreesFixture(dir string) error {
	mainDir := filepath.Join(dir, "main")
	if err := os.MkdirAll(mainDir, 0o755); err != nil { //coverage:ignore defensive: mainDir is a fresh subdirectory of RunScenario's own os.MkdirTemp result, no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("creating main repo dir: %w", err)
	}
	if err := gitInitAndConfig(mainDir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}
	if err := runGit(mainDir, "commit", "-q", "--allow-empty", "-m", "seed"); err != nil { //coverage:ignore defensive: an empty commit in a freshly-initialized repo has no realistic failure mode
		return err
	}
	if err := runGit(mainDir, "worktree", "add", "-q", "-b", "actor-a", filepath.Join(dir, "wt-a")); err != nil { //coverage:ignore defensive: adding a worktree at a fresh, never-before-used path has no realistic failure mode
		return err
	}
	if err := runGit(mainDir, "worktree", "add", "-q", "-b", "actor-b", filepath.Join(dir, "wt-b")); err != nil { //coverage:ignore defensive: see the actor-a worktree add above
		return err
	}
	return nil
}
