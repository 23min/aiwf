package check

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/check"
)

// RunGitConfigCheck verifies that the local repo's `core.worktree`
// configuration, if set, resolves to the directory `aiwf` is operating
// in (the resolved root). A mismatch means every subsequent git
// operation from this directory is silently redirected against a
// different working tree — git reports no error, `git status` shows
// the wrong worktree's files, and `aiwf check` itself loads the wrong
// tree. The failure is invisible without inspection; this rule is the
// chokepoint that catches it. G-0155.
//
// Returns no findings in three healthy cases:
//
//   - core.worktree unset (the normal state — `git config --get` exits 1)
//   - core.worktree set but resolves to the directory aiwf was invoked
//     from (linked worktrees typically have this set to their own
//     path; that's legitimate)
//   - The root or configured path can't be made absolute (don't fire
//     a noisy finding on path-resolution edge cases)
//
// Fires SeverityError otherwise — the misdirection corrupts every
// subsequent aiwf verb's view of the tree, so the push must block.
func RunGitConfigCheck(ctx context.Context, root string) []check.Finding {
	cmd := exec.CommandContext(ctx, "git", "config", "--local", "--get", "core.worktree")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// `git config --get` exits 1 when the key is not set — the
		// healthy default. Also covers "not a git repo" (other checks
		// surface that separately).
		return nil
	}
	configured := strings.TrimSpace(string(out))
	if configured == "" {
		return nil
	}
	cfgAbs, cerr := filepath.Abs(configured)
	rootAbs, rerr := filepath.Abs(root)
	if cerr != nil || rerr != nil {
		return nil
	}
	if cfgAbs == rootAbs {
		// Configured to ourselves — legitimate. Linked worktrees
		// typically carry this in their per-worktree config.
		return nil
	}
	return []check.Finding{{
		Code:     "git-config-core-worktree-misset",
		Severity: check.SeverityError,
		Message: "core.worktree=" + configured + " does not resolve to " + rootAbs +
			" (the directory aiwf is operating in); every git operation from here is silently redirected elsewhere",
		Path: ".git/config",
	}}
}
