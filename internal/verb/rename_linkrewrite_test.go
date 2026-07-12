package verb_test

// rename_linkrewrite_test.go — M-0247/AC-1 real-tree integration
// tests for wiring the shared link-destination rewrite primitive
// (M-0245) into `aiwf rename`.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestRename_RewritesLinkToRenamedEntity pins M-0247/AC-1: after
// rename swaps a slug, every OTHER entity-body link whose destination
// encoded the old slug now carries the new slug and resolves; a link
// to an unrelated gap and a bare-id prose mention of the renamed gap
// are left untouched.
func TestRename_RewritesLinkToRenamedEntity(t *testing.T) {
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

	// A second, independent linking gap: with two files needing a
	// rewrite from the same Rename call, planLinkRewriteWrites' final
	// sort.Slice actually has to compare two paths (rather than a
	// single-element no-op sort).
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Second linking gap", testActor, verb.AddOptions{
		BodyOverride: []byte("## What's missing\n\nAlso see [the target](" + targetPath + ").\n\n## Why it matters\n\nFixture.\n"),
	}))
	linking2 := r.tree().ByID("G-0004")
	if linking2 == nil {
		t.Fatal("G-0004 missing")
	}
	linking2Path := linking2.Path

	res, err := verb.Rename(r.ctx, r.tree(), "G-0001", "renamed-target", testActor, 0)
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
	renamedTargetPath := moveOps[0].NewPath
	if len(writeOps) != 2 {
		t.Fatalf("writeOps = %+v, want exactly two writes (%s and %s) — the untouched gap must NOT be rewritten", writeOps, linkingPath, linking2Path)
	}
	gotPaths := []string{writeOps[0].Path, writeOps[1].Path}
	wantPaths := []string{linkingPath, linking2Path}
	sort.Strings(wantPaths)
	if gotPaths[0] != wantPaths[0] || gotPaths[1] != wantPaths[1] {
		t.Fatalf("writeOps paths = %v, want %v (sorted)", gotPaths, wantPaths)
	}

	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatal(applyErr)
	}

	body, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(linkingPath)))
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if !strings.Contains(got, "("+renamedTargetPath+")") {
		t.Errorf("link to renamed gap not rewritten to %s:\n%s", renamedTargetPath, got)
	}
	if !strings.Contains(got, "bare mention of G-0001 in prose") {
		t.Errorf("bare-id prose mention of G-0001 should be left untouched:\n%s", got)
	}
	if !strings.Contains(got, "("+otherPath+")") {
		t.Errorf("link to non-renamed gap must remain unchanged (%s):\n%s", otherPath, got)
	}
}

// TestRename_DirectoryRename_RecomputesNestedSelfLinkAgainstFinalLayout
// exercises the directory-shaped rename case (epic/contract): the
// epic's own body links to a milestone nested inside it, and both
// move together as a single directory rename. The link must resolve
// against the FINAL post-rename layout — the co-moved-entities case
// M-0246/AC-2 covers for archive, exercised here for rename's own
// directory-expansion path (renameEntityMoves).
func TestRename_DirectoryRename_RecomputesNestedSelfLinkAgainstFinalLayout(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache layer", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	milestone := r.tree().ByID("M-0001")
	if milestone == nil {
		t.Fatal("M-0001 missing")
	}
	milestonePath := milestone.Path

	epic := r.tree().ByID("E-0001")
	if epic == nil {
		t.Fatal("E-0001 missing")
	}
	epicFull := filepath.Join(r.root, filepath.FromSlash(epic.Path))
	epicRaw, err := os.ReadFile(epicFull)
	if err != nil {
		t.Fatal(err)
	}
	epicUpdated := string(epicRaw) + "\nSee [the cache milestone](" + milestonePath + ") for detail.\n"
	if writeErr := os.WriteFile(epicFull, []byte(epicUpdated), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "renamed-platform", testActor, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan")
	}

	var writes int
	for _, op := range res.Plan.Ops {
		if op.Type == verb.OpWrite {
			writes++
		}
	}
	if writes != 1 {
		t.Errorf("writes = %d, want 1 (the epic's own body, rewriting its link to the co-moved milestone)", writes)
	}

	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatal(applyErr)
	}

	wantEpicPath := filepath.Join(r.root, "work", "epics", "E-0001-renamed-platform", "epic.md")
	newEpic, err := os.ReadFile(wantEpicPath)
	if err != nil {
		t.Fatalf("renamed epic missing: %v", err)
	}
	wantLink := "work/epics/E-0001-renamed-platform/M-0001-cache-layer.md"
	if !strings.Contains(string(newEpic), "("+wantLink+")") {
		t.Errorf("epic's link to its own co-moved milestone not recomputed against the final layout:\n%s", newEpic)
	}
}

// TestRename_SkipsAlreadyArchivedEntityAsLinkingFile pins the
// "forget-by-default" exclusion (ADR-0004, mirroring M-0246's
// identical exclusion for archive): an entity already under
// `archive/` is never considered a linking-file candidate by
// planLinkRewriteWrites, even when its on-disk body happens to
// contain a path that would otherwise match a rename.
func TestRename_SkipsAlreadyArchivedEntityAsLinkingFile(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Old note", testActor, verb.AddOptions{
		BodyOverride: bornCompleteFixtureBody(entity.KindGap),
	}))
	r.must(verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false))
	r.must(verb.Archive(r.ctx, r.root, testActor, ""))

	archivedPath := filepath.Join(r.root, "work", "gaps", "archive", "G-0001-old-note.md")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("G-0001 not archived by the sweep: %v", err)
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

	res, err := verb.Rename(r.ctx, r.tree(), "G-0002", "renamed-target", testActor, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range res.Plan.Ops {
		if op.Type == verb.OpWrite {
			t.Errorf("unexpected OpWrite %+v — an already-archived entity must never be treated as a linking-file candidate", op)
		}
	}
}
