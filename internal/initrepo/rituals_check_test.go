package initrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// TestInit_RitualAgentsAndTemplatesMaterialized covers M-0150 AC-1 at the
// init seam: `aiwf init` materializes the ritual agents into
// .claude/agents/ and the templates into .claude/templates/ (D-0015),
// alongside the skills.
func TestInit_RitualAgentsAndTemplatesMaterialized(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	for _, name := range []string{"planner.md", "builder.md", "reviewer.md", "deployer.md"} {
		if _, err := os.Stat(filepath.Join(root, ".claude", "agents", name)); err != nil {
			t.Errorf("ritual agent %s not materialized by init: %v", name, err)
		}
	}
	for _, name := range []string{"adr.md", "decision.md", "epic-spec.md", "milestone-spec.md"} {
		if _, err := os.Stat(filepath.Join(root, ".claude", "templates", name)); err != nil {
			t.Errorf("ritual template %s not materialized by init: %v", name, err)
		}
	}
}

// TestInit_GitignoreCoversAgentsAndTemplates covers M-0150 AC-2 at the
// ensureGitignore seam: a fresh `aiwf init` writes the enumerated
// agent/template lines (and their manifests) into the consumer's
// .gitignore, so the materialized flat artifacts never land in a commit.
func TestInit_GitignoreCoversAgentsAndTemplates(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	have := map[string]bool{}
	for _, l := range strings.Split(string(raw), "\n") {
		have[strings.TrimSpace(l)] = true
	}
	for _, want := range []string{
		".claude/agents/planner.md",
		".claude/agents/.aiwf-owned",
		".claude/templates/adr.md",
		".claude/templates/.aiwf-owned",
	} {
		if !have[want] {
			t.Errorf(".gitignore missing %q after init", want)
		}
	}
}
