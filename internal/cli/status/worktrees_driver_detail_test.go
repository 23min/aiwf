package status

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestBuildWorktreeViews_MilestoneACsDependsOnAndSurfacedGaps is the
// core seam test for the milestone-driver detail rows: a real repo
// where the driven milestone carries an AC, a depends_on reference to
// another milestone, and a gap discovered against it. BuildWorktreeViews
// must populate v.ACs, resolve the depends_on row's title/status from
// the loaded tree (not the "(unknown)"/"?" placeholder), and list the
// gap under v.SurfacedGaps.
func TestBuildWorktreeViews_MilestoneACsDependsOnAndSurfacedGaps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9060-detail", "E-9060", "active")
	// Dependency target: writeMilestone lays down the dir this driver
	// milestone also lives in.
	writeMilestone(t, main, "E-9060-detail", "M-9061", "done", "E-9060")

	// The driver milestone: writeMilestone doesn't support depends_on,
	// so its frontmatter is authored directly here, matching the same
	// synthetic-fixture convention.
	driverBody := "---\n" +
		"id: M-9060\n" +
		"title: M-9060 milestone\n" +
		"status: in_progress\n" +
		"parent: E-9060\n" +
		"depends_on:\n" +
		"  - M-9061\n" +
		"acs:\n" +
		"  - id: AC-1\n" +
		"    title: AC-1 criterion\n" +
		"    status: open\n" +
		"---\n"
	if err := os.WriteFile(filepath.Join(main, "work", "epics", "E-9060-detail", "M-9060.md"), []byte(driverBody), 0o644); err != nil {
		t.Fatal(err)
	}

	// A gap surfaced against the driver milestone.
	gapDir := filepath.Join(main, "work", "gaps")
	if err := os.MkdirAll(gapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gapBody := "---\nid: G-9060\ntitle: G-9060 surfaced gap\nstatus: open\ndiscovered_in: M-9060\n---\n"
	if err := os.WriteFile(filepath.Join(gapDir, "G-9060-surfaced.md"), []byte(gapBody), 0o644); err != nil {
		t.Fatal(err)
	}

	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9060 + M-9060/M-9061 + G-9060")
	gitDo(t, main, "branch", "milestone/M-9060-detail")

	wtPath := filepath.Join(t.TempDir(), "wt-detail")
	gitDo(t, main, "worktree", "add", "-q", wtPath, "milestone/M-9060-detail")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	got := viewForBranch(t, views, "milestone/M-9060-detail")

	if got.DriverKind != "milestone" || got.DriverEntityID != "M-9060" {
		t.Fatalf("driver = %s/%s, want milestone/M-9060", got.DriverKind, got.DriverEntityID)
	}

	// ACs: the e.ACs loop (worktrees.go BuildWorktreeViews) must
	// populate one ACRow per AC.
	if len(got.ACs) != 1 {
		t.Fatalf("ACs = %+v, want exactly one row", got.ACs)
	}
	if got.ACs[0].ID != "AC-1" || got.ACs[0].Title != "AC-1 criterion" || got.ACs[0].Status != "open" {
		t.Errorf("ACs[0] = %+v, want {ID: AC-1, Title: AC-1 criterion, Status: open}", got.ACs[0])
	}

	// DependsOn: the dep-resolution loop must find M-9061 in the
	// worktree's own tree and fill in its real title/status rather
	// than falling back to the "(unknown)"/"?" placeholder.
	if len(got.DependsOn) != 1 {
		t.Fatalf("DependsOn = %+v, want exactly one row", got.DependsOn)
	}
	dep := got.DependsOn[0]
	if dep.ID != "M-9061" || dep.Title != "M-9061 milestone" || dep.Status != "done" {
		t.Errorf("DependsOn[0] = %+v, want {ID: M-9061, Title: M-9061 milestone, Status: done} (dep resolved, not the unknown placeholder)", dep)
	}

	// SurfacedGaps: the discovered_in canonical-match loop must list
	// G-9060.
	if len(got.SurfacedGaps) != 1 {
		t.Fatalf("SurfacedGaps = %+v, want exactly one row", got.SurfacedGaps)
	}
	gap := got.SurfacedGaps[0]
	if gap.ID != "G-9060" || gap.Title != "G-9060 surfaced gap" || gap.Status != "open" {
		t.Errorf("SurfacedGaps[0] = %+v, want {ID: G-9060, Title: G-9060 surfaced gap, Status: open}", gap)
	}
}

// TestEpicExpansion_ClosesAndSurfacedGaps unit-tests epicExpansion
// directly (no worktree/BuildWorktreeViews plumbing needed): an epic
// with one gap it closes (addressed_by references the epic) and one
// gap surfaced against it (discovered_in references the epic, no
// addressed_by) must land in closesGaps and surfacedGaps respectively.
func TestEpicExpansion_ClosesAndSurfacedGaps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9070-gaps", "E-9070", "active")

	gapDir := filepath.Join(main, "work", "gaps")
	if err := os.MkdirAll(gapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	closedGapBody := "---\nid: G-9070\ntitle: G-9070 closed gap\nstatus: addressed\naddressed_by:\n  - E-9070\n---\n"
	if err := os.WriteFile(filepath.Join(gapDir, "G-9070-closed.md"), []byte(closedGapBody), 0o644); err != nil {
		t.Fatal(err)
	}
	surfacedGapBody := "---\nid: G-9071\ntitle: G-9071 surfaced gap\nstatus: open\ndiscovered_in: E-9070\n---\n"
	if err := os.WriteFile(filepath.Join(gapDir, "G-9071-surfaced.md"), []byte(surfacedGapBody), 0o644); err != nil {
		t.Fatal(err)
	}

	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9070 + closed/surfaced gaps")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}

	milestones, closesGaps, surfacedGaps := epicExpansion(tr, "E-9070", nil)

	if len(milestones) != 0 {
		t.Errorf("milestones = %+v, want none", milestones)
	}
	if len(closesGaps) != 1 || closesGaps[0].ID != "G-9070" || closesGaps[0].Status != "addressed" {
		t.Errorf("closesGaps = %+v, want exactly [{ID: G-9070, Status: addressed}]", closesGaps)
	}
	if len(surfacedGaps) != 1 || surfacedGaps[0].ID != "G-9071" || surfacedGaps[0].Status != "open" {
		t.Errorf("surfacedGaps = %+v, want exactly [{ID: G-9071, Status: open}]", surfacedGaps)
	}
}
