package skills

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMaterializeTo_ClaudeTarget covers M-0151 AC-1: the seam writes the
// Claude target's locations exactly as M-0149/M-0150 did — skills as
// dir-per-skill under SkillsDir, agents and templates flat under their
// dirs.
func TestMaterializeTo_ClaudeTarget(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := MaterializeTo(root, ClaudeTarget); err != nil {
		t.Fatalf("MaterializeTo: %v", err)
	}
	checks := []string{
		filepath.Join(root, ".claude", "skills", "aiwf-check", "SKILL.md"),
		filepath.Join(root, ".claude", "skills", "aiwfx-plan-epic", "SKILL.md"),
		filepath.Join(root, ".claude", "agents", "planner.md"),
		filepath.Join(root, ".claude", "templates", "adr.md"),
	}
	for _, p := range checks {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("Claude target did not materialize %s: %v", p, err)
		}
	}
}

// TestMaterialize_DefaultsToClaude covers AC-1: the back-compat
// Materialize(root) wrapper is exactly MaterializeTo(root, ClaudeTarget),
// so existing callers (init/update) and M2/M3 tests see no change.
func TestMaterialize_DefaultsToClaude(t *testing.T) {
	t.Parallel()
	rootA := t.TempDir()
	rootB := t.TempDir()
	if err := Materialize(rootA); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if err := MaterializeTo(rootB, ClaudeTarget); err != nil {
		t.Fatalf("MaterializeTo: %v", err)
	}
	// Same set of relative paths must exist under both roots.
	rels := []string{
		filepath.Join(".claude", "skills", "aiwfx-start-milestone", "SKILL.md"),
		filepath.Join(".claude", "agents", "reviewer.md"),
		filepath.Join(".claude", "templates", "milestone-spec.md"),
	}
	for _, rel := range rels {
		if _, err := os.Stat(filepath.Join(rootA, rel)); err != nil {
			t.Errorf("Materialize missing %s: %v", rel, err)
		}
		if _, err := os.Stat(filepath.Join(rootB, rel)); err != nil {
			t.Errorf("MaterializeTo missing %s: %v", rel, err)
		}
	}
}

// TestMaterializeTo_SecondTarget covers M-0151 AC-2: a second, non-Claude
// target routes every artifact kind to that target's locations and writes
// nothing under .claude/ — proving a new target is a new value, not a
// rewrite. Shaped after the Codex layout (ADR-0014 §4: verbatim SKILL.md
// to .agents/skills/). This is a test fixture, not a shipped writer.
func TestMaterializeTo_SecondTarget(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	codex := Target{
		Name:         "codex",
		SkillsDir:    ".agents/skills",
		AgentsDir:    ".agents/agents",
		TemplatesDir: ".agents/templates",
	}
	if err := MaterializeTo(root, codex); err != nil {
		t.Fatalf("MaterializeTo(codex): %v", err)
	}
	for _, p := range []string{
		filepath.Join(root, ".agents", "skills", "aiwfx-plan-epic", "SKILL.md"),
		filepath.Join(root, ".agents", "agents", "planner.md"),
		filepath.Join(root, ".agents", "templates", "adr.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("second target did not materialize %s: %v", p, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Errorf("second target wrongly wrote under .claude/ (stat err = %v)", err)
	}
}

// TestMaterializeTo_NoAgentTarget covers AC-2's ADR-0014 §4 case: a
// target with no subagent concept (empty AgentsDir) materializes skills
// and templates but no agents — the agent writer is a no-op for it.
func TestMaterializeTo_NoAgentTarget(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	noAgents := Target{
		Name:         "flat-rules",
		SkillsDir:    ".rules/skills",
		AgentsDir:    "",
		TemplatesDir: ".rules/templates",
	}
	if err := MaterializeTo(root, noAgents); err != nil {
		t.Fatalf("MaterializeTo(noAgents): %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".rules", "skills", "aiwf-check", "SKILL.md")); err != nil {
		t.Errorf("no-agent target did not materialize skills: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".rules", "templates", "adr.md")); err != nil {
		t.Errorf("no-agent target did not materialize templates: %v", err)
	}
	// No agents dir should have been created at all.
	entries, _ := os.ReadDir(filepath.Join(root, ".rules"))
	for _, e := range entries {
		if e.Name() == "agents" {
			t.Errorf("no-agent target created an agents dir despite empty AgentsDir")
		}
	}
}
