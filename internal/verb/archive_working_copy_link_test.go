package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// workingCopyOnlyLinkRunner builds the divergence the decline and the
// rewrite pass read from opposite sides: G-0002's committed body carries
// no link, and its working copy has since gained one into G-0001's move.
func workingCopyOnlyLinkRunner(t *testing.T) (r *runner, targetPath, linkerPath string) {
	t.Helper()
	r = newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	targetPath = filepath.ToSlash(target.Path)

	// Committed with no link to anything.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Linking gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))

	// The link exists only in the working copy, added since the commit.
	linkerPath = dirtyEntity(t, r, "G-0002",
		"## Why it matters\n\nFixture prose for test setup; not the subject under test.",
		"## Why it matters\n\nSee [the target]("+targetPath+") for context.")
	return r, targetPath, linkerPath
}

// TestArchive_WorkingCopyOnlyLink_DeclinesThatCandidate pins M-0286/AC-4.
//
// The decline asks HEAD whether a referrer would lose a link and finds
// none, so the move survives. The rewrite pass asks the working copy and
// finds one, so it emits a write against a mid-edit file. The plan
// therefore carries a move nothing declined and a write the guard must
// refuse — and the refusal names the file, not the candidate.
//
// The two predicates are reading opposite sides of the same question.
// A link in either copy is a link the sweep's verdict rests on.
func TestArchive_WorkingCopyOnlyLink_DeclinesThatCandidate(t *testing.T) {
	t.Parallel()
	r, targetPath, linkerPath := workingCopyOnlyLinkRunner(t)

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			t.Errorf("archive planned to sweep %s while its referrer %s carries the link only in "+
				"the working copy; the rewrite pass emits a write against that mid-edit file, "+
				"which the commit-side guard then refuses for the whole verb", targetPath, linkerPath)
		}
	}
	report := skipReport(res)
	if !strings.Contains(report, linkerPath) && !strings.Contains(report, "G-0001") {
		t.Errorf("the declined move is not reported; the operator is left to discover the "+
			"working-copy link themselves:\n%s", report)
	}
}

// TestArchive_DeletedNonReferrer_DoesNotDeclineTheMove is the negative
// case of reading both copies, and the bound on enumerating candidates
// from the record.
//
// A deleted entity file differs from the record maximally, so it is in
// the candidate set by construction. Being mid-edit is not what makes a
// file a blocker, though — carrying a link into the move is. Neither copy
// of this one does: the record's has no link, and there is no working
// copy left to read. The move it has nothing to do with proceeds.
func TestArchive_DeletedNonReferrer_DoesNotDeclineTheMove(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	// Committed, and linking to nothing.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Unrelated gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))

	target := r.tree().ByID("G-0001")
	unrelated := r.tree().ByID("G-0002")
	if target == nil || unrelated == nil {
		t.Fatal("fixture entities missing from the tree")
	}
	targetPath := filepath.ToSlash(target.Path)
	unrelatedPath := filepath.ToSlash(unrelated.Path)

	if err := os.Remove(filepath.Join(r.root, unrelatedPath)); err != nil {
		t.Fatalf("removing %s: %v", unrelatedPath, err)
	}

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	var planned bool
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			planned = true
		}
	}
	if !planned {
		t.Errorf("archive declined to sweep %s because the unrelated file %s is missing from disk; "+
			"neither copy of that file links into the move, so it decides nothing about it. "+
			"Enumerating a candidate is not the same as blocking on it.\nReport:\n%s",
			targetPath, unrelatedPath, skipReport(res))
	}
}
