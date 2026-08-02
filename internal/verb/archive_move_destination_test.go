package verb_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// terminalGap cancels a freshly-added gap and returns its path.
func terminalGap(t *testing.T, r *runner, title, id string) string {
	t.Helper()
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, title, testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Cancel(r.ctx, r.tree(), id, testActor, "fixture", false))
	e := r.tree().ByID(id)
	if e == nil {
		t.Fatalf("%s missing from the fixture tree", id)
	}
	return filepath.ToSlash(e.Path)
}

// archiveDestinationOf returns the path an active entity path sweeps to.
func archiveDestinationOf(t *testing.T, active string) string {
	t.Helper()
	dir, base := filepath.Split(active)
	to := filepath.ToSlash(filepath.Join(dir, "archive", base))
	if to == active {
		t.Fatalf("fixture assumption broken: cannot derive an archive destination from %q", active)
	}
	return to
}

// occupyPath plants an untracked file at rel. It is called after every
// entity the fixture needs is in place: the planted content is not a
// parseable entity, so the loader records a load error for it, and the
// runner's tree() helper fatals on those.
func occupyPath(t *testing.T, r *runner, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(r.root, rel)), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.root, rel), []byte("leftover from a half-finished sweep\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestArchive_OccupiedDestination_DeclinesThatCandidate pins M-0286/AC-3.
//
// The decline enumerates a move's source and stops there, while the
// commit-side guard enumerates both ends (planCarriedPaths walks
// op.Path and op.NewPath alike). So a file sitting at the destination is
// invisible to the decline and live at the commit, where the guard
// refuses the whole verb for one participant — the behaviour the
// per-candidate decline exists to replace. Reaching it through a path the
// decline never enumerated is a defect in the decline, not in the guard.
func TestArchive_OccupiedDestination_DeclinesThatCandidate(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	from := terminalGap(t, r, "Target gap", "G-0001")
	to := archiveDestinationOf(t, from)
	occupyPath(t, r, to)

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == from {
			t.Errorf("archive planned to sweep %s onto %s, which is already occupied; "+
				"the move would replace content nobody named, and the decline never looked "+
				"at the destination it lands on", from, to)
		}
	}
	report := skipReport(res)
	if !strings.Contains(report, to) && !strings.Contains(report, "G-0001") {
		t.Errorf("the declined move is not reported, so the operator cannot find the occupied "+
			"destination:\n%s", report)
	}
}

// TestArchive_OccupiedDestination_LeavesOtherCandidatesSweeping is the
// half that makes this a decline rather than a refusal. One blocked
// destination costs exactly one candidate.
func TestArchive_OccupiedDestination_LeavesOtherCandidatesSweeping(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	blocked := terminalGap(t, r, "Target gap", "G-0001")
	independentPath := terminalGap(t, r, "Independent gap", "G-0002")
	occupyPath(t, r, archiveDestinationOf(t, blocked))

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	moves := archiveMovesFor(res)
	var sweptIndependent bool
	for _, src := range moves {
		if src == blocked {
			t.Errorf("archive planned to sweep %s despite its occupied destination", blocked)
		}
		if src == independentPath {
			sweptIndependent = true
		}
	}
	if !sweptIndependent {
		t.Errorf("%s has a free destination and nothing referring to it, yet the sweep skipped it. "+
			"Planned moves: %v", independentPath, moves)
	}
}

// TestArchive_OccupiedDestination_PlanApplies is the end-to-end
// consequence. With the destination enumerated, the surviving plan
// carries only decidable moves, so the commit-side guard has nothing to
// refuse. Without it, this plan reaches Apply and the whole verb fails —
// naming the destination path, but no candidate.
func TestArchive_OccupiedDestination_PlanApplies(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	blocked := terminalGap(t, r, "Target gap", "G-0001")
	terminalGap(t, r, "Independent gap", "G-0002")
	occupyPath(t, r, archiveDestinationOf(t, blocked))

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if res.Plan == nil {
		t.Fatalf("no plan produced; the independent gap should still sweep. Report:\n%s", skipReport(res))
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("Apply refused the whole sweep because %s has an occupied destination; "+
			"declining that one candidate is what keeps the rest moving: %v", blocked, applyErr)
	}
}

// TestArchive_RecordedDivergentAtDirectoryDestination_DeclinesThatCandidate
// covers the criterion's other named shape — a destination file the record
// carries and the working copy has since changed — on the move shape where
// a destination is a whole subtree rather than one file.
//
// A directory move lands on everything beneath its destination, so a
// divergent file there is content the commit would carry without anyone
// naming it.
func TestArchive_RecordedDivergentAtDirectoryDestination_DeclinesThatCandidate(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Some epic", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindEpic)}))
	r.must(verb.Cancel(r.ctx, r.tree(), "E-0001", testActor, "fixture", false))
	e := r.tree().ByID("E-0001")
	if e == nil {
		t.Fatal("E-0001 missing from the fixture tree")
	}
	epicDir := filepath.ToSlash(filepath.Dir(e.Path))
	destDir := archiveDestinationOf(t, epicDir)

	// A stray already committed inside the destination subtree. It is not
	// an entity file, so the loader records it as a stray rather than a
	// load error and the fixture stays usable.
	strayRel := filepath.ToSlash(filepath.Join(destDir, "notes.txt"))
	if err := os.MkdirAll(filepath.Join(r.root, destDir), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.root, strayRel), []byte("committed note\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", r.root, "add", strayRel).Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", r.root, "commit", "-m", "seed a stray at the archive destination").Run(); err != nil {
		t.Fatal(err)
	}
	// ...and since edited, so the record and the working copy disagree.
	if err := os.WriteFile(filepath.Join(r.root, strayRel), []byte("edited note\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == epicDir {
			t.Errorf("archive planned to sweep %s onto %s, whose subtree holds the divergent file %s; "+
				"the move carries everything under its destination, so that file rides into the "+
				"commit unnamed", epicDir, destDir, strayRel)
		}
	}
}
