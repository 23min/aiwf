package verb_test

// archive_linkrewrite_test.go — M-0246 AC-1/AC-2 real-tree integration
// tests for wiring the shared link-destination rewrite primitive
// (M-0245) into `aiwf archive`. Uses the verb_test.go runner harness
// (newRunner/r.must/r.tree) shared with the reallocate prose-rewrite
// tests, since this is the same class of "real git tree, drive the
// verb, inspect the resulting Plan and post-apply disk state" test.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestArchive_RewritesLinkToSweptEntity pins M-0246/AC-1: after
// archive sweeps a terminal gap, every OTHER entity-body link whose
// destination resolved to it now points at its archive path and
// resolves; a link to a non-swept entity and a bare-id prose mention
// of the swept entity are left untouched. The commit's Ops carry
// exactly the expected writes — no spurious rewrite of the untouched
// gap.
func TestArchive_RewritesLinkToSweptEntity(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Other gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))

	target := r.tree().ByID("G-0001")
	other := r.tree().ByID("G-0002")
	if target == nil || other == nil {
		t.Fatal("fixture gaps missing")
	}
	targetPath, otherPath := target.Path, other.Path

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Linking gap", testActor, verb.AddOptions{
		BodyOverride: []byte(
			"## What's missing\n\nSee [the target](" + targetPath + ") and a bare mention of G-0001 in prose, " +
				"plus [an untouched gap](" + otherPath + ").\n\n## Why it matters\n\nFixture.\n"),
	}))
	linking := r.tree().ByID("G-0003")
	if linking == nil {
		t.Fatal("G-0003 missing")
	}
	linkingPath := linking.Path

	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false))

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan")
	}

	var moveOps, writeOps []verb.FileOp
	for _, op := range res.Plan.Ops {
		switch op.Type {
		case verb.OpMove:
			moveOps = append(moveOps, op)
		case verb.OpWrite:
			writeOps = append(writeOps, op)
		}
	}
	if len(moveOps) != 1 || moveOps[0].Path != targetPath {
		t.Fatalf("moveOps = %+v, want exactly one move of %s", moveOps, targetPath)
	}
	archivedTargetPath := moveOps[0].NewPath
	if len(writeOps) != 1 || writeOps[0].Path != linkingPath {
		t.Fatalf("writeOps = %+v, want exactly one write to %s (the untouched gap must NOT be rewritten)", writeOps, linkingPath)
	}

	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatal(applyErr)
	}

	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(linkingPath)))
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if !strings.Contains(got, "("+archivedTargetPath+")") {
		t.Errorf("link to swept gap not rewritten to %s:\n%s", archivedTargetPath, got)
	}
	if !strings.Contains(got, "bare mention of G-0001 in prose") {
		t.Errorf("bare-id prose mention of G-0001 should be left untouched:\n%s", got)
	}
	if !strings.Contains(got, "("+otherPath+")") {
		t.Errorf("link to non-swept gap must remain unchanged (%s):\n%s", otherPath, got)
	}
}

// TestArchive_MultiEntitySweep_RecomputesAgainstFinalLayout pins
// M-0246/AC-2: a sweep moving several entities at once — including an
// epic subtree whose milestone moves via the epic-dir rename —
// recomputes each affected link against the FINAL post-move layout.
// A gap links into the nested milestone; the epic's own body links
// to the gap. Both the epic (dir-shaped) and the gap (flat-file) move
// in the same sweep, so this exercises the case the M-0245 reviewer
// flagged as having zero coverage: a link between two entities that
// both moved in the same sweep.
func TestArchive_MultiEntitySweep_RecomputesAgainstFinalLayout(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache layer", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	milestone := r.tree().ByID("M-0001")
	if milestone == nil {
		t.Fatal("M-0001 missing")
	}
	milestonePath := milestone.Path

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Companion gap", testActor, verb.AddOptions{
		BodyOverride: []byte("## What's missing\n\nSee [the milestone](" + milestonePath + ") for context.\n\n## Why it matters\n\nFixture.\n"),
	}))
	gap := r.tree().ByID("G-0001")
	if gap == nil {
		t.Fatal("G-0001 missing")
	}
	gapPath := gap.Path

	epic := r.tree().ByID("E-0001")
	if epic == nil {
		t.Fatal("E-0001 missing")
	}
	epicFull := filepath.Join(r.root, filepath.FromSlash(epic.Path))
	epicRaw, err := os.ReadFile(epicFull)
	if err != nil {
		t.Fatal(err)
	}
	// The epic's own body links to the gap — this link must be
	// rewritten even though the epic dir itself moves in this same
	// sweep, so the final destination must reflect the gap's archive
	// path, not its pre-sweep path.
	epicUpdated := string(epicRaw) + "\nSee [companion gap](" + gapPath + ") for the tracking issue.\n"
	if writeErr := os.WriteFile(epicFull, []byte(epicUpdated), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Cascade order: terminal-ize the milestone before the epic
	// (D-0003 guard refuses an epic cancel with non-terminal children).
	r.must(verb.Cancel(r.ctx, r.tree(), "M-0001", testActor, "", false))
	r.must(verb.Cancel(r.ctx, r.tree(), "E-0001", testActor, "", false))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false))

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan")
	}

	var moves, writes int
	for _, op := range res.Plan.Ops {
		switch op.Type {
		case verb.OpMove:
			moves++
		case verb.OpWrite:
			writes++
		}
	}
	if moves != 2 {
		t.Errorf("moves = %d, want 2 (epic dir, gap file)", moves)
	}
	if writes != 2 {
		t.Errorf("writes = %d, want 2 (gap body referencing the milestone, epic body referencing the gap)", writes)
	}

	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatal(applyErr)
	}

	archivedGap, err := os.ReadFile(filepath.Join(r.root, "work", "gaps", "archive", filepath.Base(gapPath)))
	if err != nil {
		t.Fatalf("archived gap missing: %v", err)
	}
	wantMilestoneLink := "work/epics/archive/E-0001-platform/M-0001-cache-layer.md"
	if !strings.Contains(string(archivedGap), "("+wantMilestoneLink+")") {
		t.Errorf("archived gap's link to the nested milestone not recomputed against the final layout:\n%s", archivedGap)
	}

	archivedEpic, err := os.ReadFile(filepath.Join(r.root, "work", "epics", "archive", "E-0001-platform", "epic.md"))
	if err != nil {
		t.Fatalf("archived epic missing: %v", err)
	}
	wantGapLink := "work/gaps/archive/" + filepath.Base(gapPath)
	if !strings.Contains(string(archivedEpic), "("+wantGapLink+")") {
		t.Errorf("archived epic's link to the co-swept gap not recomputed against the final layout:\n%s", archivedEpic)
	}
}

// TestArchive_SkipsAlreadyArchivedEntityAsLinkingFile pins the
// "forget-by-default" exclusion (ADR-0004, mirroring rewidth's own
// `archive/` skip): an entity already under `archive/` is never
// considered a linking-file candidate, even when its on-disk body
// happens to contain a path that would otherwise match an upcoming
// move.
func TestArchive_SkipsAlreadyArchivedEntityAsLinkingFile(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Old note", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false))
	r.must(verb.Archive(r.ctx, r.root, testActor, ""))

	archivedPath := filepath.Join(r.root, "work", "gaps", "archive", "G-0001-old-note.md")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("G-0001 not archived by the first sweep: %v", err)
	}

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	target := r.tree().ByID("G-0002")
	if target == nil {
		t.Fatal("G-0002 missing")
	}
	targetPath := target.Path

	raw, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(raw) + "\nSee [target](" + targetPath + ") too.\n"
	if writeErr := os.WriteFile(archivedPath, []byte(updated), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	r.must(verb.Cancel(r.ctx, r.tree(), "G-0002", testActor, "", false))

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range res.Plan.Ops {
		if op.Type == verb.OpWrite {
			t.Errorf("unexpected OpWrite %+v — an already-archived entity must never be treated as a linking-file candidate", op)
		}
	}
}
