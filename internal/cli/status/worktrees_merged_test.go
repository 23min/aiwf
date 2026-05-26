package status

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// TestMergedStaleOverride covers the G-0172 decision: a fully-merged
// worktree branch (AheadOfTrunk == 0) whose driver is terminal on trunk
// should be re-classified stale with trunk's authoritative status. The
// override is gated so it never fires on a fresh fork (active driver) or
// when the branch still carries unmerged work.
func TestMergedStaleOverride(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		aheadOfTrunk int
		trunk        *entity.Entity
		wantStale    bool
		wantStatus   string
		wantTitle    string
	}{
		{
			name:         "branch ahead of trunk blocks override even when trunk terminal",
			aheadOfTrunk: 3,
			trunk:        &entity.Entity{Kind: entity.KindEpic, Status: entity.StatusDone, Title: "Done epic"},
			wantStale:    false,
		},
		{
			name:         "nil trunk entity (trunk unavailable or driver absent) skips override",
			aheadOfTrunk: 0,
			trunk:        nil,
			wantStale:    false,
		},
		{
			name:         "merged branch + active driver on trunk is a fresh fork, not a leftover",
			aheadOfTrunk: 0,
			trunk:        &entity.Entity{Kind: entity.KindEpic, Status: entity.StatusActive, Title: "Active epic"},
			wantStale:    false,
		},
		{
			name:         "merged branch + done driver on trunk is a retire-able leftover",
			aheadOfTrunk: 0,
			trunk:        &entity.Entity{Kind: entity.KindEpic, Status: entity.StatusDone, Title: "Done epic"},
			wantStale:    true,
			wantStatus:   entity.StatusDone,
			wantTitle:    "Done epic",
		},
		{
			name:         "merged branch + cancelled driver on trunk is also a leftover",
			aheadOfTrunk: 0,
			trunk:        &entity.Entity{Kind: entity.KindMilestone, Status: entity.StatusCancelled, Title: "Abandoned"},
			wantStale:    true,
			wantStatus:   entity.StatusCancelled,
			wantTitle:    "Abandoned",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, title, stale := mergedStaleOverride(tc.aheadOfTrunk, tc.trunk)
			if stale != tc.wantStale {
				t.Fatalf("stale = %v, want %v", stale, tc.wantStale)
			}
			if !stale {
				return
			}
			if status != tc.wantStatus {
				t.Errorf("status = %q, want %q", status, tc.wantStatus)
			}
			if title != tc.wantTitle {
				t.Errorf("title = %q, want %q", title, tc.wantTitle)
			}
		})
	}
}

// TestTrunkTreeOf_NoMainWorktree covers the no-trunk-reference branch:
// when no worktree is on `main`, trunkTreeOf returns nil and the caller
// falls back to branch-local terminality (G-0172 detection is skipped,
// no regression). nil tr is safe because the loop returns before any
// disk load.
func TestTrunkTreeOf_NoMainWorktree(t *testing.T) {
	t.Parallel()
	worktrees := []gitops.Worktree{
		{Path: "/repo/wt-a", Branch: "epic/E-0001-a"},
		{Path: "/repo/wt-b", Branch: "milestone/M-0002-b"},
	}
	if got := trunkTreeOf(context.Background(), worktrees, "/repo", nil); got != nil {
		t.Errorf("trunkTreeOf with no main worktree = %v, want nil", got)
	}
}

// gitDo runs a git command in dir, failing the test on error. Shared by
// the G-0172 seam tests that build real repos + worktrees.
func gitDo(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v (in %s): %v\n%s", args, dir, err, out)
	}
}

// writeEpic writes an epic entity file at work/epics/<slug>/epic.md
// under root with the given id/status. Synthetic, obviously-fictional
// content per the golden-fixture convention.
func writeEpic(t *testing.T, root, slug, id, status string) {
	t.Helper()
	dir := filepath.Join(root, "work", "epics", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: " + id + "\ntitle: " + id + " epic\nstatus: " + status + "\n---\n\n## Goal\nSynthetic fixture.\n"
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// viewForBranch finds the worktree view whose Branch matches, failing
// the test when absent.
func viewForBranch(t *testing.T, views []WorktreeView, branch string) *WorktreeView {
	t.Helper()
	for i := range views {
		if views[i].Branch == branch {
			return &views[i]
		}
	}
	t.Fatalf("no worktree view for branch %q", branch)
	return nil
}

// TestBuildWorktreeViews_MergedEpicTrunkTerminal is the core seam test
// for G-0172: a real repo where the driver epic is terminal on trunk but
// the linked worktree's branch tree still shows it active (the
// promote-done landed on main after the branch merged). BuildWorktreeViews
// must classify the worktree stale with trunk's status — not phantom
// in-flight work. Drives trunkTreeOf + the override wiring end-to-end
// (outer-if true, inner-if true → mutate).
func TestBuildWorktreeViews_MergedEpicTrunkTerminal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	// Base commit: epic active. Branch the worktree off this state.
	writeEpic(t, main, "E-9001-merged", "E-9001", "active")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9001 active")
	gitDo(t, main, "branch", "epic/E-9001-merged")

	// On main only: promote the epic to done. The branch is now an
	// ancestor of main (fully merged) and its tree lags trunk.
	writeEpic(t, main, "E-9001-merged", "E-9001", "done")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "promote E-9001 done")

	wtPath := filepath.Join(t.TempDir(), "wt-merged")
	gitDo(t, main, "worktree", "add", "-q", wtPath, "epic/E-9001-merged")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}

	// Precondition: the branch tree still shows the epic active — the
	// stale-branch-tree condition the fix has to see past.
	wt, _, err := tree.Load(ctx, wtPath)
	if err != nil {
		t.Fatalf("tree.Load(wt): %v", err)
	}
	if e := wt.ByID("E-9001"); e == nil || e.Status != entity.StatusActive {
		t.Fatalf("precondition: branch tree should show E-9001 active, got %#v", e)
	}

	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	got := viewForBranch(t, views, "epic/E-9001-merged")
	if got.AheadOfTrunk != 0 {
		t.Fatalf("AheadOfTrunk = %d, want 0 (branch fully merged)", got.AheadOfTrunk)
	}
	if !got.Stale {
		t.Errorf("merged worktree whose driver is terminal on trunk should be Stale; got Stale=false — the G-0172 phantom-in-flight bug")
	}
	if got.DriverStatus != entity.StatusDone {
		t.Errorf("DriverStatus = %q, want %q (trunk-authoritative, not the stale branch view)", got.DriverStatus, entity.StatusDone)
	}
}

// TestBuildWorktreeViews_PreservesGenuineInFlight guards against the
// override over-firing. Two worktrees exercise the two
// override-declines arms through the seam:
//
//   - epic/E-9100: active driver with an unmerged commit (AheadOfTrunk
//     > 0) — mergedStaleOverride declines (inner-if false); genuine
//     in-flight work must stay non-stale.
//   - epic/E-9101: driver promoted to done *on the branch* (branch-local
//     terminal) — v.Stale is already true so the override is skipped
//     entirely (outer-if false via !v.Stale); the existing G-0153 path
//     owns it.
func TestBuildWorktreeViews_PreservesGenuineInFlight(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9100-active", "E-9100", "active")
	writeEpic(t, main, "E-9101-localdone", "E-9101", "active")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9100 + E-9101 active")
	gitDo(t, main, "branch", "epic/E-9100-active")
	gitDo(t, main, "branch", "epic/E-9101-localdone")

	// epic/E-9100: an unmerged commit on the branch (ahead of trunk).
	wtActive := filepath.Join(t.TempDir(), "wt-active")
	gitDo(t, main, "worktree", "add", "-q", wtActive, "epic/E-9100-active")
	if err := os.WriteFile(filepath.Join(wtActive, "work", "epics", "E-9100-active", "note.md"), []byte("wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitDo(t, wtActive, "add", "-A")
	gitDo(t, wtActive, "commit", "-q", "-m", "wip on E-9100")

	// epic/E-9101: driver done on the branch tree itself (branch-local
	// terminal), committed on the branch so it's ahead of trunk.
	wtLocalDone := filepath.Join(t.TempDir(), "wt-localdone")
	gitDo(t, main, "worktree", "add", "-q", wtLocalDone, "epic/E-9101-localdone")
	writeEpic(t, wtLocalDone, "E-9101-localdone", "E-9101", "done")
	gitDo(t, wtLocalDone, "add", "-A")
	gitDo(t, wtLocalDone, "commit", "-q", "-m", "promote E-9101 done (on branch)")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}

	active := viewForBranch(t, views, "epic/E-9100-active")
	if active.AheadOfTrunk == 0 {
		t.Fatalf("precondition: epic/E-9100-active should be ahead of trunk, got AheadOfTrunk=0")
	}
	if active.Stale {
		t.Errorf("active driver with unmerged commits must stay non-stale; override over-fired (Stale=true)")
	}
	if active.DriverStatus != entity.StatusActive {
		t.Errorf("DriverStatus = %q, want %q (unchanged in-flight)", active.DriverStatus, entity.StatusActive)
	}

	localDone := viewForBranch(t, views, "epic/E-9101-localdone")
	if !localDone.Stale {
		t.Errorf("branch-local-terminal driver should be Stale via the existing G-0153 path; got Stale=false")
	}
	if localDone.DriverStatus != entity.StatusDone {
		t.Errorf("DriverStatus = %q, want %q (branch-local terminal)", localDone.DriverStatus, entity.StatusDone)
	}
}

// TestBuildWorktreeViews_NoTrunkWorktree_NoOverride covers the
// load-bearing nil guard: when no worktree is on `main` (the main
// checkout is itself parked on a feature branch), trunkTree is nil and
// the override must skip without dereferencing it (a nil trunkTree.ByID
// would panic). The driver stays branch-local — no reclassification.
func TestBuildWorktreeViews_NoTrunkWorktree_NoOverride(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9002-feature", "E-9002", "active")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9002 active")
	// Park the only checkout on a ritual branch so no worktree is on main.
	gitDo(t, main, "checkout", "-q", "-b", "epic/E-9002-feature")

	tr, _, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	// Must not panic even though trunkTree resolves to nil.
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	got := viewForBranch(t, views, "epic/E-9002-feature")
	if got.Stale {
		t.Errorf("with no trunk worktree the override must skip; driver should stay branch-local non-stale, got Stale=true")
	}
	if got.DriverStatus != entity.StatusActive {
		t.Errorf("DriverStatus = %q, want %q (branch-local, override skipped)", got.DriverStatus, entity.StatusActive)
	}
}
