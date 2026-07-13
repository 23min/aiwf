package contract

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// M-0254/AC-1 backfill: runUnbind's ResolveRoot guard is
// `//coverage:ignore`d in unbind.go itself. The two remaining flagged
// branches — actor resolution and LoadContractsDoc — get real tests
// below. White-box (package contract) so this file can call the
// unexported runUnbind directly.

// TestRunUnbind_ResolveActorFailure covers runUnbind's cliutil.ResolveActor
// guard using M-0252's BrokenGitIdentity fixture. Serial: BrokenGitIdentity
// uses t.Setenv, which panics under t.Parallel.
func TestRunUnbind_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	rc := runUnbind("C-0001", root, "", cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunUnbind_LoadContractsDocFailure covers runUnbind's
// cliutil.LoadContractsDoc guard, reusing the malformed-contracts-block
// trigger already proven at internal/cli/add/add_error_paths_test.go.
func TestRunUnbind_LoadContractsDocFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, contractsMalformedYAML)
	rc := runUnbind("C-0001", root, "human/test", cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
