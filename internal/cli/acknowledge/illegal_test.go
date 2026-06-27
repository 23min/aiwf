package acknowledge

import (
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestRunIllegal_ErrorExits covers runIllegal's guard exits (M-0181/AC-5). The
// happy path is exercised by the trailer-shape integration test (in-process
// cli.Execute); these are the error branches the old top-level package never
// unit-tested. Each must return a non-OK exit code:
//   - empty --reason → ExitUsage (sovereign acts require a written rationale);
//   - a malformed actor (no role/identifier slash) → ResolveActor rejects it;
//   - a non-existent root → repo-lock acquisition fails.
func TestRunIllegal_ErrorExits(t *testing.T) {
	t.Parallel()
	var out cliutil.OutputFormat

	if rc := runIllegal("deadbeef", "human/test", t.TempDir(), "   ", "", out); rc != cliutil.ExitUsage {
		t.Errorf("empty reason: rc = %d, want ExitUsage", rc)
	}
	if rc := runIllegal("deadbeef", "notanactor", t.TempDir(), "real reason", "", out); rc != cliutil.ExitUsage {
		t.Errorf("malformed actor: rc = %d, want ExitUsage", rc)
	}
	bad := filepath.Join(t.TempDir(), "does-not-exist")
	if rc := runIllegal("deadbeef", "human/test", bad, "real reason", "", out); rc == cliutil.ExitOK {
		t.Errorf("non-existent root (lock should fail): rc = %d, want non-OK", rc)
	}
}
