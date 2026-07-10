package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/manifest"
	"github.com/23min/aiwf/internal/verb"
)

// epic_terminal_children_accidental_guard_test.go pins G-0398's own
// framing: neither Add nor Import has a dedicated precondition against
// creating a milestone under an already-terminal epic, but the attempt
// is refused TODAY anyway — as a side effect of check.CodeEpicTerminal-
// NonTerminalChildren tripping the generic before/after projection-
// findings gate every mutating verb runs (see epic_terminal_children.go's
// "Second, accidental role" doc comment). These tests exist so a future
// change to that generic gate (or to Add's/Import's own validation
// pipeline) can't silently drop this protection — if it did, these
// tests would go red instead of the gap surfacing only much later via
// a routine `aiwf check` run.

// TestAdd_MilestoneUnderTerminalEpic_RefusedViaFindings pins the
// accidental refusal for `aiwf add milestone`.
func TestAdd_MilestoneUnderTerminalEpic_RefusedViaFindings(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Orphan", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"})
	if err != nil {
		t.Fatalf("Add: unexpected Go error: %v", err)
	}
	if res.Plan != nil {
		t.Error("expected no plan (refused via findings), got a plan")
	}
	if !check.HasErrors(res.Findings) {
		t.Fatalf("expected error-severity findings; got %+v", res.Findings)
	}
	found := false
	for _, f := range res.Findings {
		if f.Code == check.CodeEpicTerminalNonTerminalChildren {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a %s finding; got %+v", check.CodeEpicTerminalNonTerminalChildren, res.Findings)
	}
}

// TestImport_MilestoneUnderTerminalEpic_RefusedViaFindings pins the
// same accidental refusal for `aiwf import`'s equivalent path.
func TestImport_MilestoneUnderTerminalEpic_RefusedViaFindings(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "", false, verb.PromoteOptions{}))

	src := `version: 1
entities:
  - kind: milestone
    id: M-0001
    frontmatter:
      title: "Orphan"
      status: draft
      parent: E-0001
      tdd: none
    body: "## Goal\nOrphaned.\n"
`
	m, err := manifest.Parse([]byte(src), "yaml")
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	res, err := verb.Import(r.ctx, r.tree(), m, testActor, verb.ImportOptions{})
	if err != nil {
		t.Fatalf("Import: unexpected Go error: %v", err)
	}
	if len(res.Plans) != 0 {
		t.Errorf("expected no plans (refused via findings), got %d", len(res.Plans))
	}
	if !check.HasErrors(res.Findings) {
		t.Fatalf("expected error-severity findings; got %+v", res.Findings)
	}
	found := false
	for _, f := range res.Findings {
		if f.Code == check.CodeEpicTerminalNonTerminalChildren {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a %s finding; got %+v", check.CodeEpicTerminalNonTerminalChildren, res.Findings)
	}
}
