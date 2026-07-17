package setpriority_test

import (
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/setpriority"
)

// TestRun_ErrorExits covers set-priority's guard exits, mirroring
// set-area's own TestRun_ErrorExits shape. Each deterministic guard
// must return a non-OK code:
//   - neither <level> nor --clear → ExitUsage;
//   - a malformed actor (no role/identifier slash) → ResolveActor rejects it;
//   - a non-existent root → repo-lock acquisition fails (before tree load).
//
// The ResolveRoot (broken-cwd) and LoadTreeWithTrunk (IO) arms are
// //coverage:ignore'd in setpriority.go as not deterministically
// reproducible.
func TestRun_ErrorExits(t *testing.T) {
	t.Parallel()
	var out cliutil.OutputFormat

	if rc := setpriority.Run([]string{"G-0001"}, "human/test", "", t.TempDir(), false, out); rc != cliutil.ExitUsage {
		t.Errorf("no level + no --clear: rc = %d, want ExitUsage", rc)
	}
	if rc := setpriority.Run([]string{"G-0001", "urgent"}, "notanactor", "", t.TempDir(), false, out); rc != cliutil.ExitUsage {
		t.Errorf("malformed actor: rc = %d, want ExitUsage", rc)
	}
	bad := filepath.Join(t.TempDir(), "does-not-exist")
	if rc := setpriority.Run([]string{"G-0001", "urgent"}, "human/test", "", bad, false, out); rc == cliutil.ExitOK {
		t.Errorf("non-existent root (lock should fail): rc = %d, want non-OK", rc)
	}
	if rc := setpriority.Run([]string{"G-0001", "urgent"}, "human/test", "", t.TempDir(), true, out); rc != cliutil.ExitUsage {
		t.Errorf("level + --clear mutex: rc = %d, want ExitUsage", rc)
	}
}
