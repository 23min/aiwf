package skills

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMaterializedRituals_AllPresentAfterMaterialize covers M-0152 AC-1:
// after a full Materialize, every ritual artifact is reported present
// and none missing.
func TestMaterializedRituals_AllPresentAfterMaterialize(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	present, missing, err := MaterializedRituals(root, ClaudeTarget)
	if err != nil {
		t.Fatalf("MaterializedRituals: %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("expected no missing artifacts, got %v", missing)
	}
	if len(present) == 0 {
		t.Fatal("expected present artifacts, got none")
	}
	// Sanity: at least one of each kind.
	var sawSkill, sawAgent, sawTemplate bool
	for _, p := range present {
		switch {
		case len(p) > 7 && p[:7] == "skills/":
			sawSkill = true
		case len(p) > 7 && p[:7] == "agents/":
			sawAgent = true
		case len(p) > 10 && p[:10] == "templates/":
			sawTemplate = true
		}
	}
	if !sawSkill || !sawAgent || !sawTemplate {
		t.Errorf("present set missing a kind: skill=%v agent=%v template=%v (%v)", sawSkill, sawAgent, sawTemplate, present)
	}
}

// TestMaterializedRituals_ReportsMissing covers M-0152 AC-1: when the
// artifacts are absent (e.g. a fresh clone before `aiwf update`),
// every ritual artifact is reported missing.
func TestMaterializedRituals_ReportsMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir() // nothing materialized
	present, missing, err := MaterializedRituals(root, ClaudeTarget)
	if err != nil {
		t.Fatalf("MaterializedRituals: %v", err)
	}
	if len(present) != 0 {
		t.Errorf("expected nothing present in empty root, got %v", present)
	}
	if len(missing) == 0 {
		t.Fatal("expected missing artifacts in empty root, got none")
	}
}

// TestMaterializedRituals_PartialMissing covers AC-1's partial case:
// remove one materialized agent and confirm it (and only it among
// agents) is reported missing.
func TestMaterializedRituals_PartialMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := Materialize(root); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if err := os.Remove(filepath.Join(root, AgentsDir, "planner.md")); err != nil {
		t.Fatal(err)
	}
	_, missing, err := MaterializedRituals(root, ClaudeTarget)
	if err != nil {
		t.Fatalf("MaterializedRituals: %v", err)
	}
	want := "agents/planner.md"
	found := false
	for _, m := range missing {
		if m == want {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %q in missing, got %v", want, missing)
	}
}

// TestMaterializedRituals_NoAgentTargetSkipsAgents covers the seam
// interaction: a target with an empty AgentsDir does not count agents
// as artifacts (they are never materialized for it).
func TestMaterializedRituals_NoAgentTargetSkipsAgents(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tgt := Target{Name: "noagents", SkillsDir: ".x/skills", AgentsDir: "", TemplatesDir: ".x/templates"}
	if err := MaterializeTo(root, tgt); err != nil {
		t.Fatalf("MaterializeTo: %v", err)
	}
	_, missing, err := MaterializedRituals(root, tgt)
	if err != nil {
		t.Fatalf("MaterializedRituals: %v", err)
	}
	for _, m := range missing {
		if len(m) >= 7 && m[:7] == "agents/" {
			t.Errorf("no-agent target should not report agent artifacts, got %q", m)
		}
	}
}
