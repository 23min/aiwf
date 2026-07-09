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
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil { //coverage:ignore defensive: git init/config on a fresh os.MkdirTemp dir has no realistic failure mode short of filesystem sabotage
			return fmt.Errorf("git %v: %w\n%s", args, err, out)
		}
	}
	return nil
}
