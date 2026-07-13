package cancel_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cancel"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestRun_AuditOnlyBranch_EntityNotFound covers M-0253/AC-1's sole
// cancel.go flagged branch: the --audit-only dispatch arm (`if
// auditOnly { ... }`), which no existing test reached — the one
// integration test exercising `aiwf cancel --audit-only`
// (internal/cli/integration/auditonly_cmd_test.go) drives a separate
// compiled binary as a subprocess, invisible to this package's
// coverage instrumentation.
//
// A nonexistent id is enough to prove the branch's two statements
// (the verb.CancelAuditOnly call and the DecorateAndFinish call)
// execute; verb.CancelAuditOnly's own not-found handling is out of
// this milestone's scope (internal/verb has its own coverage).
func TestRun_AuditOnlyBranch_EntityNotFound(t *testing.T) {
	t.Parallel()
	var out cliutil.OutputFormat
	root := t.TempDir()
	rc := cancel.Run("G-0001", "human/test", "", root, "manual flip from earlier", false, true, out)
	if rc == cliutil.ExitOK {
		t.Errorf("audit-only cancel of a nonexistent entity: rc = ExitOK, want a non-OK exit code")
	}
}
