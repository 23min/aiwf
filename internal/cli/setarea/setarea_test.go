package setarea_test

import (
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/setarea"
)

// TestRun_ErrorExits covers set-area's guard exits — inherited coverage debt
// (the package shipped with no unit test; only a happy-path integration
// dispatcher test). Surfaced by M-0181's epic-relative coverage gate and fixed
// here. Each deterministic guard must return a non-OK code:
//   - neither <member> nor --clear → ExitUsage;
//   - a malformed actor (no role/identifier slash) → ResolveActor rejects it;
//   - a non-existent root → repo-lock acquisition fails (before tree load).
//
// The ResolveRoot (broken-cwd) and LoadTreeWithTrunk (IO) arms are
// //coverage:ignore'd in setarea.go as not deterministically reproducible.
func TestRun_ErrorExits(t *testing.T) {
	t.Parallel()
	var out cliutil.OutputFormat

	if rc := setarea.Run([]string{"G-0001"}, "human/test", "", t.TempDir(), false, out); rc != cliutil.ExitUsage {
		t.Errorf("no member + no --clear: rc = %d, want ExitUsage", rc)
	}
	if rc := setarea.Run([]string{"G-0001", "app-a"}, "notanactor", "", t.TempDir(), false, out); rc != cliutil.ExitUsage {
		t.Errorf("malformed actor: rc = %d, want ExitUsage", rc)
	}
	bad := filepath.Join(t.TempDir(), "does-not-exist")
	if rc := setarea.Run([]string{"G-0001", "app-a"}, "human/test", "", bad, false, out); rc == cliutil.ExitOK {
		t.Errorf("non-existent root (lock should fail): rc = %d, want non-OK", rc)
	}
}
