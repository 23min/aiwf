package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestMove_ToCurrentParent_ReturnsNoOp pins M-0281/AC-3: moving a milestone to
// the epic it is already under converges to a NoOp instead of the "already
// under epic" error, so a re-run (interactive or scripted) is a clean exit 0.
func TestMove_ToCurrentParent_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	res, err := verb.Move(r.ctx, r.tree(), "M-0001", "E-0001", testActor)
	if err != nil {
		t.Fatalf("move to the current parent returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (move to the current parent is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}

// TestMove_ToCurrentParentAtLegacyWidth_ReturnsNoOp pins that AC-3's guard
// compares ids, not spellings. Parsers accept narrower legacy widths on input
// (ByID canonicalizes both sides before matching), so `--epic E-01` names the
// very epic the milestone already sits under. Comparing the raw argument
// against the stored canonical value missed that, and the miss cost more than a
// NoOp: the move then rewrote an already-canonical `parent:` to the operator's
// narrower spelling. Converging is what removes that write.
func TestMove_ToCurrentParentAtLegacyWidth_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	res, err := verb.Move(r.ctx, r.tree(), "M-0001", "E-01", testActor)
	if err != nil {
		t.Fatalf("move to the current parent at legacy width returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true — E-01 and E-0001 are the same epic")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil — applying it would rewrite parent: for no change", res.Plan)
	}
}

// TestMove_ParentMatchesButFileMisplaced_StillMoves pins the second half of
// AC-3's guard. A move writes `parent:` AND relocates the file, so a milestone
// whose frontmatter already names the target epic while its file sits under
// another still has work to do — and `move` is the verb that would repair it.
// Comparing the field alone reported "nothing to move" over exactly that state.
func TestMove_ParentMatchesButFileMisplaced_StillMoves(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Second", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	// Point the frontmatter at E-0002 while the file stays under E-0001.
	writeEntityParent(t, r, "M-0001", "E-0002")

	res, err := verb.Move(r.ctx, r.tree(), "M-0001", "E-0002", testActor)
	if err != nil {
		t.Fatalf("move of a misplaced milestone: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false — the file is under the wrong epic, so there is a move to make")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan relocating the file")
	}
}

// writeEntityParent rewrites an entity's `parent:` frontmatter directly,
// producing the field-vs-location drift only a hand edit can create.
func writeEntityParent(t *testing.T, r *runner, id, parent string) {
	t.Helper()
	e := r.tree().ByID(id)
	if e == nil {
		t.Fatalf("%s missing from the fixture tree", id)
	}
	path := filepath.Join(r.root, e.Path)
	raw, err := os.ReadFile(path) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", id, err)
	}
	patched := strings.Replace(string(raw), "parent: "+e.Parent+"\n", "parent: "+parent+"\n", 1)
	if patched == string(raw) {
		t.Fatalf("fixture had no parent: %s to rewrite:\n%s", e.Parent, raw)
	}
	if writeErr := os.WriteFile(path, []byte(patched), 0o600); writeErr != nil {
		t.Fatalf("writing %s: %v", id, writeErr)
	}
}
