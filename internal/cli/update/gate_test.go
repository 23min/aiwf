package update

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/skills"
)

// TestGateAndSyncHookDecisions_MissingAiwfYamlReturnsInternal exercises the
// defensive error path: gateAndSyncHookDecisions assumes its caller (Run)
// only invokes it once config.Load has already confirmed aiwf.yaml exists,
// but the function itself makes no such guarantee — a rootDir with no
// aiwf.yaml must fail loudly (ExitInternal), not panic.
func TestGateAndSyncHookDecisions_MissingAiwfYamlReturnsInternal(t *testing.T) {
	t.Parallel()
	rootDir := t.TempDir() // deliberately no aiwf.yaml written here
	hooks := []skills.HookDef{{Name: "test-hook", Description: "does a thing"}}

	rc := gateAndSyncHookDecisions(rootDir, hooks, nil)
	if rc != cliutil.ExitInternal {
		t.Errorf("gateAndSyncHookDecisions() = %d, want ExitInternal", rc)
	}
}

// TestGateAndSyncHookDecisions_UnknownFieldInExistingHooksBlockReturnsInternal
// covers doc.Hooks()'s own reachable decode error: a hand-edited hooks:
// block carrying an unrecognized key inside one hook's entry fails the
// strict KnownFields decode, and that error must propagate as
// ExitInternal rather than being swallowed.
func TestGateAndSyncHookDecisions_UnknownFieldInExistingHooksBlockReturnsInternal(t *testing.T) {
	t.Parallel()
	rootDir := t.TempDir()
	raw := "hooks:\n  bad-hook:\n    unknown_field: true\n"
	if err := os.WriteFile(filepath.Join(rootDir, config.FileName), []byte(raw), 0o644); err != nil {
		t.Fatalf("writing aiwf.yaml fixture: %v", err)
	}
	hooks := []skills.HookDef{{Name: "test-hook", Description: "does a thing"}}

	rc := gateAndSyncHookDecisions(rootDir, hooks, nil)
	if rc != cliutil.ExitInternal {
		t.Errorf("gateAndSyncHookDecisions() = %d, want ExitInternal", rc)
	}
}
