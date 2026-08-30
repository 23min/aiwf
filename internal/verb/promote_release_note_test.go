package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// promote_release_note_test.go pins the second surface of
// milestone-done-empty-release-note: because Promote runs the projection
// findings as preconditions and gates on error severity, the rule that reports
// an unwritten `## Release note` in `aiwf check` is the same rule that refuses
// the `done` promote which would create that state.
//
// The rule's severity is load-bearing for this and nothing else pins it: at
// warning severity check.HasErrors does not match, the promote lands, and the
// only signal arrives afterwards — by which point aiwfx-wrap-milestone has
// already pushed, since its push gate precedes its promote-done step.

// setupDoneReadyMilestone returns a runner whose M-0001 is in_progress on the
// epic branch with its single AC met, so the only thing standing between it and
// `done` is the release note.
func setupDoneReadyMilestone(t *testing.T) *runner {
	t.Helper()
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
	return r
}

// appendReleaseNote writes a `## Release note` section carrying content onto
// M-0001 and commits it, standing in for `aiwf edit-body`.
func appendReleaseNote(t *testing.T, r *runner, content string) {
	t.Helper()
	m := r.tree().ByID("M-0001")
	if m == nil {
		t.Fatal("M-0001 missing from the tree")
	}
	path := filepath.Join(r.root, m.Path)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(string(raw)+"\n## Release note\n\n"+content+"\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	commitFixture(t, r.root, "fixture: add a release note")
}

// TestPromote_MilestoneDoneRefusedWithoutAReleaseNote is the headline case: the
// section is absent entirely, which is the shape every spec written before the
// section existed carries — so absence has to count, or deleting the heading
// would be the escape.
func TestPromote_MilestoneDoneRefusedWithoutAReleaseNote(t *testing.T) {
	t.Parallel()
	r := setupDoneReadyMilestone(t)

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001", "done", testActor, "", false, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("the refusal arrives as findings, not a Go error: %v", err)
	}
	if res.Plan != nil {
		t.Fatalf("expected nil Plan when the projection blocks; got %+v", res.Plan)
	}
	if !check.HasErrors(res.Findings) {
		t.Fatalf("expected an error-severity finding; got %+v", res.Findings)
	}
	var found bool
	for _, f := range res.Findings {
		if f.Code == check.CodeMilestoneDoneEmptyReleaseNote {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s among the blocking findings; got %+v",
			check.CodeMilestoneDoneEmptyReleaseNote, res.Findings)
	}
	if m := r.tree().ByID("M-0001"); m == nil || m.Status != entity.StatusInProgress {
		t.Errorf("refused promote must not mutate status; M-0001 = %+v", m)
	}
}

// TestPromote_MilestoneDoneRefusedWithAScaffoldOnlyReleaseNote pins the other
// half: a section carrying only the template's guidance comment is the state a
// freshly scaffolded spec is in, and it is not a written note.
func TestPromote_MilestoneDoneRefusedWithAScaffoldOnlyReleaseNote(t *testing.T) {
	t.Parallel()
	r := setupDoneReadyMilestone(t)
	appendReleaseNote(t, r, "<!-- The user-visible delta of this milestone. -->")

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001", "done", testActor, "", false, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if res.Plan != nil || !check.HasErrors(res.Findings) {
		t.Fatalf("a scaffold-only note must block the promote; Plan=%+v Findings=%+v", res.Plan, res.Findings)
	}
}

// TestPromote_MilestoneDoneLandsWithAWrittenReleaseNote is the regression
// companion — the rule must not block a milestone that did the thing it asks.
func TestPromote_MilestoneDoneLandsWithAWrittenReleaseNote(t *testing.T) {
	t.Parallel()
	r := setupDoneReadyMilestone(t)
	appendReleaseNote(t, r, "The verb now accepts a flag.")

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001", "done", testActor, "", false, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("promote should land: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf("expected a Plan; findings were %+v", res.Findings)
	}
}

// TestPromote_MilestoneDoneAcceptsNoUserVisibleChange pins the escape the rule
// offers instead of a scope that lets an unwritten note through: a milestone
// with nothing user-facing says so in four words and is not blocked.
func TestPromote_MilestoneDoneAcceptsNoUserVisibleChange(t *testing.T) {
	t.Parallel()
	r := setupDoneReadyMilestone(t)
	appendReleaseNote(t, r, "No user-visible change.")

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001", "done", testActor, "", false, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("promote should land: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf(`"no user-visible change" must be a valid note; findings were %+v`, res.Findings)
	}
	if strings.Contains(strings.Join(findingCodes(res.Findings), ","), check.CodeMilestoneDoneEmptyReleaseNote) {
		t.Errorf("rule fired on a written note: %+v", res.Findings)
	}
}

func findingCodes(fs []check.Finding) []string {
	out := make([]string, 0, len(fs))
	for i := range fs {
		out = append(out, fs[i].Code)
	}
	return out
}
