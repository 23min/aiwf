package acknowledge

import (
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestMistagCmd_HasPositionalCompletion pins M-0181/AC-7: the `mistag <id>`
// subcommand wires positional entity-id completion (the id is a closed set,
// unlike illegal's free-form <sha>). Complements the completion-drift gate.
func TestMistagCmd_HasPositionalCompletion(t *testing.T) {
	t.Parallel()
	if newMistagCmd().ValidArgsFunction == nil {
		t.Error("mistag command ValidArgsFunction not wired (positional entity-id completion expected)")
	}
}

// TestRunMistag_ErrorExits covers runMistag's guard exits (M-0181/AC-6). The
// happy path is exercised end-to-end by the acknowledge-suppresses integration
// test (in-process cli.Execute); these pin the error branches:
//   - empty --reason → ExitUsage;
//   - a malformed actor (no role/identifier slash) → ResolveActor rejects it;
//   - a non-existent root → repo-lock acquisition fails (before tree load).
func TestRunMistag_ErrorExits(t *testing.T) {
	t.Parallel()
	var out cliutil.OutputFormat

	if rc := runMistag("G-0001", "human/test", t.TempDir(), "   ", out); rc != cliutil.ExitUsage {
		t.Errorf("empty reason: rc = %d, want ExitUsage", rc)
	}
	if rc := runMistag("G-0001", "notanactor", t.TempDir(), "real reason", out); rc != cliutil.ExitUsage {
		t.Errorf("malformed actor: rc = %d, want ExitUsage", rc)
	}
	bad := filepath.Join(t.TempDir(), "does-not-exist")
	if rc := runMistag("G-0001", "human/test", bad, "real reason", out); rc == cliutil.ExitOK {
		t.Errorf("non-existent root (lock should fail): rc = %d, want non-OK", rc)
	}
}
