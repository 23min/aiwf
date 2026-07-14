package contract

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// M-0254/AC-1 backfill: runBind's ResolveRoot and tree.Load guards are
// `//coverage:ignore`d in bind.go itself, mirroring the established
// internal/cli/archive and internal/cli/promote precedent. The two
// remaining flagged branches — the actor-resolution guard and the
// LoadContractsDoc guard — get real tests below. White-box (package
// contract, not contract_test) so this file can call the unexported
// runBind directly, matching diag_fallback_internal_test.go's
// existing precedent in this same package.

// TestRunBind_ResolveActorFailure covers runBind's cliutil.ResolveActor
// guard using M-0252's BrokenGitIdentity fixture. Serial: BrokenGitIdentity
// uses t.Setenv, which panics under t.Parallel.
func TestRunBind_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	rc := runBind("C-0001", root, "", "render", "schema.cue", "fixtures", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunBind_LoadContractsDocFailure covers runBind's
// cliutil.LoadContractsDoc guard, reusing the malformed-contracts-block
// trigger already proven at internal/cli/add/add_error_paths_test.go
// (an unrecognized "bindings" key is a hard error under the contracts:
// block's strict-unknown-fields parsing rule).
func TestRunBind_LoadContractsDocFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAiwfYAML(t, root, contractsMalformedYAML)
	rc := runBind("C-0001", root, "human/test", "render", "schema.cue", "fixtures", false, cliutil.OutputFormat{})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
