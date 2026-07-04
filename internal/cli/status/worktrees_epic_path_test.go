package status

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// writeMilestone writes a milestone entity file at
// work/epics/<epicSlug>/<id>.md under root, parented to epic id
// `parent`, optionally carrying ACs (each just an id — title/status are
// synthetic filler). Shares the golden-fixture convention writeEpic
// uses (worktrees_merged_test.go).
func writeMilestone(t *testing.T, root, epicSlug, id, status, parent string, acIDs ...string) {
	t.Helper()
	dir := filepath.Join(root, "work", "epics", epicSlug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: " + id + "\ntitle: " + id + " milestone\nstatus: " + status + "\nparent: " + parent + "\n"
	if len(acIDs) > 0 {
		body += "acs:\n"
		for _, ac := range acIDs {
			body += "  - id: " + ac + "\n    title: " + ac + " criterion\n    status: open\n"
		}
	}
	body += "---\n"
	if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// epicChildRow finds the row with the given id, failing the test when
// absent.
func epicChildRow(t *testing.T, rows []EpicChildRow, id string) EpicChildRow {
	t.Helper()
	for _, r := range rows {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no EpicChildRow for id %q in %+v", id, rows)
	return EpicChildRow{}
}

// TestBuildWorktreeViews_EpicPathOverridesMilestoneBranch is the core
// G-0332 seam test: a worktree placed at the in-repo ritual path
// `.../epic/E-NNNN-.../` with a *milestone* branch checked out inside
// it must still render at epic altitude, with the checked-out
// milestone's row flagged and its ACs nested underneath while its
// sibling milestone stays collapsed.
func TestBuildWorktreeViews_EpicPathOverridesMilestoneBranch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9040-altitude", "E-9040", "active")
	writeMilestone(t, main, "E-9040-altitude", "M-9040", "in_progress", "E-9040")
	writeMilestone(t, main, "E-9040-altitude", "M-9041", "in_progress", "E-9040", "AC-1")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9040 + two milestones")
	gitDo(t, main, "branch", "milestone/M-9041-altitude")

	// The worktree directory itself encodes the epic id; the branch
	// checked out inside it is the milestone's, not the epic's.
	wtPath := filepath.Join(t.TempDir(), "epic", "E-9040-altitude")
	gitDo(t, main, "worktree", "add", "-q", wtPath, "milestone/M-9041-altitude")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	got := viewForBranch(t, views, "milestone/M-9041-altitude")

	if got.DriverKind != "epic" || got.DriverEntityID != "E-9040" {
		t.Fatalf("driver = %s/%s, want epic/E-9040 (path signal should win over the milestone branch)", got.DriverKind, got.DriverEntityID)
	}
	if got.DriverStatus != "active" {
		t.Errorf("DriverStatus = %q, want %q", got.DriverStatus, "active")
	}

	active := epicChildRow(t, got.EpicMilestones, "M-9041")
	if !active.CheckedOut {
		t.Errorf("checked-out milestone M-9041 row should have CheckedOut=true")
	}
	if len(active.ACs) != 1 || active.ACs[0].ID != "AC-1" {
		t.Errorf("checked-out milestone M-9041 ACs = %+v, want one AC-1 row", active.ACs)
	}

	sibling := epicChildRow(t, got.EpicMilestones, "M-9040")
	if sibling.CheckedOut {
		t.Errorf("sibling milestone M-9040 should stay CheckedOut=false")
	}
	if len(sibling.ACs) != 0 {
		t.Errorf("sibling milestone M-9040 should carry no ACs, got %+v", sibling.ACs)
	}
}

// TestBuildWorktreeViews_EpicPathWithNonRitualBranch covers the
// driverID=="" arm of the override: a non-ritual branch (no
// aiwf-verb-trailered commits, doesn't parse to any entity) checked
// out inside an epic-path worktree still renders at epic altitude —
// the path signal doesn't depend on the branch resolving to anything.
func TestBuildWorktreeViews_EpicPathWithNonRitualBranch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9042-scratch", "E-9042", "active")
	writeMilestone(t, main, "E-9042-scratch", "M-9042", "in_progress", "E-9042")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9042 + one milestone")
	gitDo(t, main, "branch", "scratch")

	wtPath := filepath.Join(t.TempDir(), "epic", "E-9042-scratch")
	gitDo(t, main, "worktree", "add", "-q", wtPath, "scratch")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	got := viewForBranch(t, views, "scratch")

	if got.DriverKind != "epic" || got.DriverEntityID != "E-9042" {
		t.Fatalf("driver = %s/%s, want epic/E-9042 (path signal alone should drive it)", got.DriverKind, got.DriverEntityID)
	}
	if row := epicChildRow(t, got.EpicMilestones, "M-9042"); row.CheckedOut {
		t.Errorf("no milestone should be flagged CheckedOut when the checked-out branch resolves to no driver")
	}
}

// TestBuildWorktreeViews_EpicPathWithEpicBranchCheckedOut covers the
// case where the checked-out branch already IS the epic's own ritual
// branch: the override reassigns driverID to the same epic id, and no
// milestone is flagged CheckedOut (there's no milestone driver to
// overlay).
func TestBuildWorktreeViews_EpicPathWithEpicBranchCheckedOut(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9043-onbranch", "E-9043", "active")
	writeMilestone(t, main, "E-9043-onbranch", "M-9043", "in_progress", "E-9043")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9043 + one milestone")
	gitDo(t, main, "branch", "epic/E-9043-onbranch")

	wtPath := filepath.Join(t.TempDir(), "epic", "E-9043-onbranch")
	gitDo(t, main, "worktree", "add", "-q", wtPath, "epic/E-9043-onbranch")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	got := viewForBranch(t, views, "epic/E-9043-onbranch")

	if got.DriverKind != "epic" || got.DriverEntityID != "E-9043" {
		t.Fatalf("driver = %s/%s, want epic/E-9043", got.DriverKind, got.DriverEntityID)
	}
	if row := epicChildRow(t, got.EpicMilestones, "M-9043"); row.CheckedOut {
		t.Errorf("no milestone should be flagged CheckedOut when the epic's own branch is checked out")
	}
}

// TestBuildWorktreeViews_EpicPathUnresolvedIDFallsBack covers the
// epicEntity==nil decline arm: the worktree path parses to an epic id
// that doesn't exist in the tree (e.g. renamed/never-allocated). The
// override must decline and fall back to the ordinary branch-driven
// resolution — here, a real milestone under a different, real epic.
func TestBuildWorktreeViews_EpicPathUnresolvedIDFallsBack(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9044-real", "E-9044", "active")
	writeMilestone(t, main, "E-9044-real", "M-9044", "in_progress", "E-9044")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9044 + one milestone")
	gitDo(t, main, "branch", "milestone/M-9044-real")

	// Path claims E-9999 — an id that was never allocated.
	wtPath := filepath.Join(t.TempDir(), "epic", "E-9999-ghost")
	gitDo(t, main, "worktree", "add", "-q", wtPath, "milestone/M-9044-real")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	got := viewForBranch(t, views, "milestone/M-9044-real")

	if got.DriverKind != "milestone" || got.DriverEntityID != "M-9044" {
		t.Fatalf("driver = %s/%s, want milestone/M-9044 (unresolved path id must not override)", got.DriverKind, got.DriverEntityID)
	}
	if got.ParentEpicID != "E-9044" {
		t.Errorf("ParentEpicID = %q, want %q (ordinary milestone-driver resolution)", got.ParentEpicID, "E-9044")
	}
}

// TestBuildWorktreeViews_EpicPathWrongKindFallsBack covers the
// epicEntity.Kind!=epic decline arm: the worktree path parses to an id
// that exists in the tree but under a different kind (a malformed tree
// state — check would flag the mismatch, but the loader tolerates it
// per "errors are findings, not parse failures"). The override must
// decline rather than mislabel a gap as an epic.
func TestBuildWorktreeViews_EpicPathWrongKindFallsBack(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9045-real", "E-9045", "active")
	writeMilestone(t, main, "E-9045-real", "M-9045", "in_progress", "E-9045")
	// E-9050 is mislabeled: an epic-shaped id stored as a gap.
	if err := os.MkdirAll(filepath.Join(main, "work", "gaps"), 0o755); err != nil {
		t.Fatal(err)
	}
	gapBody := "---\nid: E-9050\ntitle: Mislabeled\nstatus: open\n---\n"
	if err := os.WriteFile(filepath.Join(main, "work", "gaps", "E-9050-mislabeled.md"), []byte(gapBody), 0o644); err != nil {
		t.Fatal(err)
	}
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9045 + one milestone + mislabeled E-9050")
	gitDo(t, main, "branch", "milestone/M-9045-real")

	wtPath := filepath.Join(t.TempDir(), "epic", "E-9050-mislabeled")
	gitDo(t, main, "worktree", "add", "-q", wtPath, "milestone/M-9045-real")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	got := viewForBranch(t, views, "milestone/M-9045-real")

	if got.DriverKind != "milestone" || got.DriverEntityID != "M-9045" {
		t.Fatalf("driver = %s/%s, want milestone/M-9045 (wrong-kind path id must not override)", got.DriverKind, got.DriverEntityID)
	}
}

// TestBuildWorktreeViews_EpicPathOverrideInteractsWithMergedStaleOverride
// covers the highest-risk interaction: once the path override
// reassigns driverID to the epic, the later G-0172 mergedStaleOverride
// check (trunkTree.ByID(driverID)) must evaluate the *epic's*
// trunk-terminality, not the milestone branch's — driverID has already
// been reassigned by the time that check runs. Fully-merged milestone
// branch (AheadOfTrunk==0) + epic promoted to done *on trunk only*
// (the milestone branch's own tree still shows it active, mirroring
// TestBuildWorktreeViews_MergedEpicTrunkTerminal's setup) must
// reclassify the worktree Stale with the epic's trunk-authoritative
// status — and the epic expansion (with the checked-out overlay) must
// still populate, since the switch-on-kind runs unconditionally after
// the stale determination.
func TestBuildWorktreeViews_EpicPathOverrideInteractsWithMergedStaleOverride(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9046-wrap", "E-9046", "active")
	writeMilestone(t, main, "E-9046-wrap", "M-9046", "in_progress", "E-9046")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9046 active + one milestone")
	gitDo(t, main, "branch", "milestone/M-9046-wrap")

	// On main only: promote the epic to done. The milestone branch never
	// receives this commit, so it stays a strict ancestor of main
	// (AheadOfTrunk==0) while its own tree still shows the epic active.
	writeEpic(t, main, "E-9046-wrap", "E-9046", "done")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "promote E-9046 done (trunk only)")

	wtPath := filepath.Join(t.TempDir(), "epic", "E-9046-wrap")
	gitDo(t, main, "worktree", "add", "-q", wtPath, "milestone/M-9046-wrap")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	got := viewForBranch(t, views, "milestone/M-9046-wrap")

	if got.DriverKind != "epic" || got.DriverEntityID != "E-9046" {
		t.Fatalf("driver = %s/%s, want epic/E-9046 (path signal should still win)", got.DriverKind, got.DriverEntityID)
	}
	if got.AheadOfTrunk != 0 {
		t.Fatalf("precondition: AheadOfTrunk = %d, want 0 (branch fully merged)", got.AheadOfTrunk)
	}
	if !got.Stale {
		t.Errorf("mergedStaleOverride should fire against the reassigned epic driver; got Stale=false")
	}
	if got.DriverStatus != "done" {
		t.Errorf("DriverStatus = %q, want %q (trunk-authoritative epic status, not the branch tree's stale active view)", got.DriverStatus, "done")
	}
	// The epic expansion (and its checked-out overlay) still populates
	// even though the worktree is Stale — the switch-on-kind runs
	// unconditionally after the stale determination.
	if row := epicChildRow(t, got.EpicMilestones, "M-9046"); !row.CheckedOut {
		t.Errorf("M-9046 should still be flagged CheckedOut even though the worktree is Stale")
	}
}
