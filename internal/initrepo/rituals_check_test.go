package initrepo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/tree"
)

// TestInit_RitualSkillsMaterializedAndCheckClean covers M-0149 AC-3: a
// repo set up by `aiwf init` (which materializes verb + ritual skills)
// loads and validates with zero error-severity findings. Materializing
// the rituals into .claude/skills/ must not introduce any aiwf check
// regression.
func TestInit_RitualSkillsMaterializedAndCheckClean(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Ritual skills landed via init's materialize step.
	for _, name := range []string{"aiwfx-plan-epic", "wf-tdd-cycle"} {
		if _, err := os.Stat(filepath.Join(root, ".claude", "skills", name, "SKILL.md")); err != nil {
			t.Errorf("ritual skill %s not materialized by init: %v", name, err)
		}
	}

	tr, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	for _, f := range check.Run(tr, loadErrs) {
		if f.Severity == check.SeverityError {
			t.Errorf("aiwf check error on materialized repo: %s — %s", f.Code, f.Message)
		}
	}
}
