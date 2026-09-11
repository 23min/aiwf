package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// promote_release_note_test.go pins that milestone-done-empty-release-note does
// NOT block the `done` promote.
//
// The rule reports an unwritten `## Release note` at warning severity. Promote
// gates its projection findings on `check.HasErrors`, which matches error
// severity alone, so the transition lands and the note is a standing report
// rather than a gate. Escalating the rule to error reverses that, and this test
// is what says so — nothing else observes the severity from the verb side.
//
// The severity is deliberate. At error the rule would demand a section the
// kernel's own scaffold does not write: `entity.RequiredSections` for a
// milestone is Goal and Acceptance criteria, so a milestone created through
// `aiwf add` could not reach `done` at all, and `--force` does not relax a
// projection finding.
func TestPromote_MilestoneDoneIsNotBlockedByAnUnwrittenReleaseNote(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Ship the thing", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"Does the thing"},
		[][]byte{[]byte("Real prose describing the criterion.")}, testActor))
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-platform")
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "met", testActor, "", false, verb.PromoteOptions{}))

	// The milestone carries no `## Release note` at all — the shape every spec
	// written before the section existed has, and the one the rule reports on.
	res := r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "done", testActor, "", false, verb.PromoteOptions{}))
	if res.Plan == nil {
		t.Fatalf("expected a Plan; the promote was blocked by findings %+v", res.Findings)
	}
	if m := r.tree().ByID("M-0001"); m == nil || m.Status != entity.StatusDone {
		t.Errorf("milestone should have reached done; M-0001 = %+v", m)
	}
}
