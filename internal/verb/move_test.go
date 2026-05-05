package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// TestMove_RoundTrip is the golden path: a milestone moves between two
// epics, the file lands at the new path, frontmatter parent is
// rewritten, and the commit carries the expected trailers.
func TestMove_RoundTrip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "First half", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Second half", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Travelling", testActor, verb.AddOptions{EpicID: "E-01"}))

	r.must(verb.Move(r.ctx, r.tree(), "M-001", "E-02", testActor))

	oldPath := filepath.Join(r.root, "work", "epics", "E-01-first-half", "M-001-travelling.md")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old path still exists: stat err = %v", err)
	}
	newPath := filepath.Join(r.root, "work", "epics", "E-02-second-half", "M-001-travelling.md")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("milestone missing at new path: %v", err)
	}

	m := r.tree().ByID("M-001")
	if m == nil || m.Parent != "E-02" {
		t.Errorf("M-001 = %+v, want Parent=E-02", m)
	}

	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-move tree has errors: %+v", findings)
	}

	subj, err := gitops.HeadSubject(r.ctx, r.root)
	if err != nil || subj != "aiwf move M-001 E-01 -> E-02" {
		t.Errorf("subject = %q (err %v)", subj, err)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-verb", "move")
	mustHaveTrailer(t, tr, "aiwf-entity", "M-001")
	mustHaveTrailer(t, tr, "aiwf-prior-parent", "E-01")
	mustHaveTrailer(t, tr, "aiwf-actor", testActor)
}

// TestMove_PreservesReferencingGap: a gap that references the moved
// milestone via discovered_in still resolves after the move. This is
// the load-bearing property of stable ids.
func TestMove_PreservesReferencingGap(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "First", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Second", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Travelling", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Found something", testActor, verb.AddOptions{DiscoveredIn: "M-001"}))

	r.must(verb.Move(r.ctx, r.tree(), "M-001", "E-02", testActor))

	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("references broke: %+v", findings)
	}
	g := r.tree().ByID("G-001")
	if g == nil || g.DiscoveredIn != "M-001" {
		t.Errorf("G-001 = %+v, want DiscoveredIn=M-001", g)
	}
}

func TestMove_RejectsNonMilestone(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Bar", testActor, verb.AddOptions{}))

	_, err := verb.Move(r.ctx, r.tree(), "E-01", "E-02", testActor)
	if err == nil || !strings.Contains(err.Error(), "only milestones") {
		t.Errorf("expected non-milestone error, got %v", err)
	}
}

func TestMove_RejectsUnknownID(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))

	_, err := verb.Move(r.ctx, r.tree(), "M-999", "E-01", testActor)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestMove_RejectsUnknownEpic(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "M", testActor, verb.AddOptions{EpicID: "E-01"}))

	_, err := verb.Move(r.ctx, r.tree(), "M-001", "E-99", testActor)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected target-not-found error, got %v", err)
	}
}

func TestMove_RejectsTargetWrongKind(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "M", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "G", testActor, verb.AddOptions{}))

	_, err := verb.Move(r.ctx, r.tree(), "M-001", "G-001", testActor)
	if err == nil || !strings.Contains(err.Error(), "is not an epic") {
		t.Errorf("expected wrong-kind error, got %v", err)
	}
}

func TestMove_RejectsSameEpic(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "M", testActor, verb.AddOptions{EpicID: "E-01"}))

	_, err := verb.Move(r.ctx, r.tree(), "M-001", "E-01", testActor)
	if err == nil || !strings.Contains(err.Error(), "already under") {
		t.Errorf("expected same-epic error, got %v", err)
	}
}

func TestMove_RequiresEpicFlag(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "M", testActor, verb.AddOptions{EpicID: "E-01"}))

	_, err := verb.Move(r.ctx, r.tree(), "M-001", "", testActor)
	if err == nil || !strings.Contains(err.Error(), "--epic") {
		t.Errorf("expected --epic-required error, got %v", err)
	}
}
