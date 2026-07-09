package stresstest

import (
	"fmt"
	"os/exec"
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
