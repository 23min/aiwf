package initcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/skills"
)

// TestGateAndPersistHookDecisions_MissingAiwfYamlReturnsInternal exercises
// the defensive error path: gateAndPersistHookDecisions assumes its caller
// (Run) only invokes it right after initrepo.Init has successfully written
// aiwf.yaml, but the function itself makes no such guarantee — a rootDir
// with no aiwf.yaml (e.g. a future call site that skips that precondition)
// must fail loudly (ExitInternal), not panic or silently do nothing.
func TestGateAndPersistHookDecisions_MissingAiwfYamlReturnsInternal(t *testing.T) {
	t.Parallel()
	rootDir := t.TempDir() // deliberately no aiwf.yaml written here
	hooks := []skills.HookDef{{Name: "test-hook", Description: "does a thing"}}

	rc := gateAndPersistHookDecisions(rootDir, hooks, nil, false)
	if rc != cliutil.ExitInternal {
		t.Errorf("gateAndPersistHookDecisions() = %d, want ExitInternal", rc)
	}
}

// TestGateAndPersistHookDecisions_UnknownFieldInExistingHooksBlockReturnsInternal
// covers the doc.Hooks() decode-error branch introduced by G-0446's
// honor-existing read: a hand-edited hooks: block with an unrecognized key
// fails the strict KnownFields decode, and that error must propagate as
// ExitInternal rather than being swallowed.
func TestGateAndPersistHookDecisions_UnknownFieldInExistingHooksBlockReturnsInternal(t *testing.T) {
	t.Parallel()
	rootDir := t.TempDir()
	raw := "hooks:\n  bad-hook:\n    unknown_field: true\n"
	if err := os.WriteFile(filepath.Join(rootDir, config.FileName), []byte(raw), 0o644); err != nil {
		t.Fatalf("writing aiwf.yaml fixture: %v", err)
	}
	hooks := []skills.HookDef{{Name: "test-hook", Description: "does a thing"}}

	rc := gateAndPersistHookDecisions(rootDir, hooks, nil, false)
	if rc != cliutil.ExitInternal {
		t.Errorf("gateAndPersistHookDecisions() = %d, want ExitInternal", rc)
	}
}
