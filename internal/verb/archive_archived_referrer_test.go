package verb_test

import (
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// archivedReferrerRunner builds a repo where an already-archived gap's
// committed body links to a still-active gap, and that active gap has
// since become terminal — so it is a live sweep candidate whose only
// referrer lives under archive/.
//
// The archived referrer keeps its link to the target's *active* path:
// the sweep that archived it had no reason to rewrite anything, because
// the target had not moved yet.
func archivedReferrerRunner(t *testing.T) (r *runner, targetPath, archivedLinkerPath string) {
	t.Helper()
	r = newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing from the fixture tree")
	}
	targetPath = filepath.ToSlash(target.Path)

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Linking gap", testActor,
		verb.AddOptions{BodyOverride: []byte(
			"## What's missing\n\nSee [the target](" + targetPath + ") for context.\n\n" +
				"## Why it matters\n\nFixture prose.\n",
		)}))
	// Sweep the linker into archive/ while the target is still active.
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0002", testActor, "fixture", false))
	r.must(verb.Archive(r.ctx, r.root, testActor, ""))

	linker := r.tree().ByID("G-0002")
	if linker == nil {
		t.Fatal("G-0002 missing from the fixture tree after the archive sweep")
	}
	archivedLinkerPath = filepath.ToSlash(linker.Path)
	if !entity.IsArchivedPath(archivedLinkerPath) {
		t.Fatalf("premise failed: %s is not under archive/ after the sweep; "+
			"this fixture cannot reproduce an archived referrer", archivedLinkerPath)
	}

	// Now the target itself becomes a sweep candidate.
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "fixture", false))
	return r, targetPath, archivedLinkerPath
}

// TestArchive_ArchivedReferrerMidEdit_DoesNotDeclineTheMove pins
// M-0286/AC-2.
//
// The rewrite pass skips archived entities — ADR-0004's forget-by-default
// rule — so an archived body is never rewritten when a sweep lands and
// can therefore lose no link. The decline pass applies no such filter, so
// it counts an entity the rewrite pass does not, and a draft under
// archive/ takes down a candidate that has nothing to do with it.
//
// That disagreement between the two predicates is the whole defect; this
// is the arrangement of the tree that exhibits it.
func TestArchive_ArchivedReferrerMidEdit_DoesNotDeclineTheMove(t *testing.T) {
	t.Parallel()
	r, targetPath, archivedLinkerPath := archivedReferrerRunner(t)

	// The operator is part-way through editing the archived gap's body.
	dirtyEntity(t, r, "G-0002", "Fixture prose.", "Draft rewording of an archived note.")

	res, err := verb.Archive(r.ctx, r.root, testActor, "")
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	var swept bool
	for _, src := range archiveMovesFor(res) {
		if src == targetPath {
			swept = true
		}
	}
	if !swept {
		t.Errorf("archive declined to sweep %s because the archived file %s is mid-edit; "+
			"planArchiveRewrites skips archived entities, so that file is never rewritten by a sweep "+
			"and can lose no link. The decline counts an entity the rewrite pass does not.\nReport:\n%s",
			targetPath, archivedLinkerPath, skipReport(res))
	}
}
