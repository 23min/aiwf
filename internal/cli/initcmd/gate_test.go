package initcmd

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
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

	rc := gateAndPersistHookDecisions(rootDir, hooks, nil)
	if rc != cliutil.ExitInternal {
		t.Errorf("gateAndPersistHookDecisions() = %d, want ExitInternal", rc)
	}
}
