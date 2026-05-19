package initrepo

import (
	"strings"
	"testing"
)

// TestHookScripts_UsePATHResolution pins the three hook templates
// (pre-push, pre-commit, post-commit) to PATH-relative `command -v
// aiwf` lookup at hook-fire time, not an absolute aiwf path baked at
// install time. The previous install-time bake (`exec '/path/to/aiwf'
// check`) broke across multi-context dev (host ↔ devcontainer ↔
// worktree) where GOPATH and the absolute install path differ — the
// G-0135 / M-0133 fix.
//
// Two assertions per hook:
//  1. Contains `command -v aiwf` — the PATH-lookup shape at hook-fire.
//  2. Contains a fail-loud not-found message — silent skip is wrong;
//     operators need to know if the hook can't find aiwf. (The
//     post-commit hook exits 0 on not-found by design — STATUS.md
//     regen is best-effort and must not disturb a successful commit —
//     but still logs the not-found message to stderr so the operator
//     sees the skip.)
func TestHookScripts_UsePATHResolution(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
	}{
		{"pre-push", preHookScript()},
		{"pre-commit", preCommitHookScript()},
		{"post-commit", postCommitHookScript()},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(tc.body, "command -v aiwf") {
				t.Errorf("hook %s lacks `command -v aiwf` lookup", tc.name)
			}
			if !strings.Contains(tc.body, "aiwf binary not found") {
				t.Errorf("hook %s lacks fail-loud not-found message", tc.name)
			}
		})
	}
}
