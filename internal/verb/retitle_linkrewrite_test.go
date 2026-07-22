package verb_test

// retitle_linkrewrite_test.go — M-0247/AC-2 real-tree integration
// tests for wiring the shared link-destination rewrite primitive
// (M-0245) into `aiwf retitle`.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestRetitle_SlugChange_RewritesLinkAndSyncsH1 pins the slug-changing
// half of M-0247/AC-2: a retitle that changes the slug rewrites other
// entities' link destinations the same way `aiwf rename` does, in
// addition to its existing H1-sync behavior — both land on the same
// commit, not two competing writes to the same path.
func TestRetitle_SlugChange_RewritesLinkAndSyncsH1(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Target gap", testActor, verb.AddOptions{
		BodyOverride: []byte("# G-0001 — Target gap\n\n## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n"),
	}))
	target := r.tree().ByID("G-0001")
	if target == nil {
		t.Fatal("G-0001 missing")
	}
	targetPath := target.Path

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Linking gap", testActor, verb.AddOptions{
		BodyOverride: []byte("## What's missing\n\nSee [the target](" + targetPath + ") for context.\n\n## Why it matters\n\nFixture.\n"),
	}))
	linking := r.tree().ByID("G-0002")
	if linking == nil {
		t.Fatal("G-0002 missing")
	}
	linkingPath := linking.Path

	res, err := verb.Retitle(r.ctx, r.tree(), "G-0001", "Renamed Target", testActor, "", 0)
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
	newTargetPath := moveOps[0].NewPath
	if len(writeOps) != 2 {
		t.Fatalf("writeOps = %+v, want exactly two writes (the retitled gap's own H1-synced body, and the linking gap's rewritten link)", writeOps)
	}

	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatal(applyErr)
	}

	newTarget, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(newTargetPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(newTarget), "# G-0001 — Renamed Target") {
		t.Errorf("H1 not synced to new title:\n%s", newTarget)
	}

	linkingBody, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(linkingPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(linkingBody), "("+newTargetPath+")") {
		t.Errorf("link to retitled gap not rewritten to %s:\n%s", newTargetPath, linkingBody)
	}
}

// TestRetitle_CompositeAC_RewritesNoLinkDestinations pins the
// composite-AC half of M-0247/AC-2: retitling an M-NNN/AC-N moves no
// file, so it must rewrite no other entity's link destinations — the
// Plan carries exactly the one existing write to the parent
// milestone's own AC heading.
func TestRetitle_CompositeAC_RewritesNoLinkDestinations(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache layer", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor))

	milestone := r.tree().ByID("M-0001")
	if milestone == nil {
		t.Fatal("M-0001 missing")
	}
	milestonePath := milestone.Path

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Linking gap", testActor, verb.AddOptions{
		BodyOverride: []byte("## What's missing\n\nSee [the milestone](" + milestonePath + ") for context.\n\n## Why it matters\n\nFixture.\n"),
	}))

	res, err := verb.Retitle(r.ctx, r.tree(), "M-0001/AC-1", "Renamed criterion", testActor, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan")
	}
	if len(res.Plan.Ops) != 1 {
		t.Fatalf("Ops = %+v, want exactly one write (the parent milestone's own AC heading) — a composite-AC retitle moves no file and rewrites no other entity's links", res.Plan.Ops)
	}
	if res.Plan.Ops[0].Type != verb.OpWrite || res.Plan.Ops[0].Path != milestonePath {
		t.Fatalf("Ops[0] = %+v, want a single OpWrite to %s", res.Plan.Ops[0], milestonePath)
	}
}

// TestRetitle_DirRename_ComposesH1AndLinkRewriteInOneWrite exercises
// the directory-shaped retitle case (epic/contract): the epic's own
// body links to a milestone nested inside it, and the epic's own file
// needs BOTH its H1 synced to the new title AND its own link to the
// co-moved milestone rewritten — in a single write to its new path,
// not two competing ones.
func TestRetitle_DirRename_ComposesH1AndLinkRewriteInOneWrite(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{
		BodyOverride: []byte("# E-0001 — Platform\n\n## Goal\n\nFixture.\n\n## Scope\n\n## Out of scope\n"),
	}))
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

	res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", "Renamed Platform", testActor, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Plan == nil {
		t.Fatal("expected plan")
	}

	var writeOps []verb.FileOp
	for _, op := range res.Plan.Ops {
		if op.Type == verb.OpWrite {
			writeOps = append(writeOps, op)
		}
	}
	if len(writeOps) != 1 {
		t.Fatalf("writeOps = %+v, want exactly ONE write to the epic's own new path — H1 sync and the link rewrite must compose into a single write, not two competing ones", writeOps)
	}

	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatal(applyErr)
	}

	wantEpicPath := filepath.Join(r.root, "work", "epics", "E-0001-renamed-platform", "epic.md")
	newEpic, err := os.ReadFile(wantEpicPath)
	if err != nil {
		t.Fatalf("retitled epic missing: %v", err)
	}
	got := string(newEpic)
	if !strings.Contains(got, "# E-0001 — Renamed Platform") {
		t.Errorf("H1 not synced to new title:\n%s", got)
	}
	wantLink := "work/epics/E-0001-renamed-platform/M-0001-cache-layer.md"
	if !strings.Contains(got, "("+wantLink+")") {
		t.Errorf("epic's own link to its co-moved milestone not recomputed:\n%s", got)
	}
}
