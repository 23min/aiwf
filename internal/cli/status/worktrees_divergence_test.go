package status

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestAnnotateWorktreeDivergence_FlagsDisagreement is the core G-0277
// unit test: a sibling epic-branch worktree reports a milestone status
// that disagrees with the current checkout's copy — the milestone must
// come back flagged, naming the sibling worktree's status and a label.
func TestAnnotateWorktreeDivergence_FlagsDisagreement(t *testing.T) {
	t.Parallel()
	report := &StatusReport{
		InFlightEpics: []StatusEpic{
			{
				ID: "E-0043",
				Milestones: []StatusMilestone{
					{ID: "M-0171", Status: entity.StatusDraft},
				},
			},
		},
	}
	views := []WorktreeView{
		{
			Path:           "/repo/wt-epic",
			Branch:         "epic/E-0043-area-tag",
			DriverKind:     string(entity.KindEpic),
			DriverEntityID: "E-0043",
			EpicMilestones: []EpicChildRow{
				{ID: "M-0171", Status: entity.StatusDone},
			},
		},
	}

	AnnotateWorktreeDivergence(report, views, "/repo/main")

	got := report.InFlightEpics[0].Milestones[0].WorktreeDivergence
	if got == nil {
		t.Fatalf("WorktreeDivergence = nil, want a flagged divergence")
	}
	if got.Status != entity.StatusDone {
		t.Errorf("Status = %q, want %q", got.Status, entity.StatusDone)
	}
	if !strings.Contains(got.Label, "E-0043") {
		t.Errorf("Label = %q, want it to name E-0043", got.Label)
	}
}

// TestAnnotateWorktreeDivergence_FlagsPlannedEpic covers the
// PlannedEpics branch: a proposed epic that already has a worktree
// (e.g. work started, then reverted to proposed) must be reconciled
// the same way an in-flight one is — the annotate helper applies
// uniformly to both report sections.
func TestAnnotateWorktreeDivergence_FlagsPlannedEpic(t *testing.T) {
	t.Parallel()
	report := &StatusReport{
		PlannedEpics: []StatusEpic{
			{
				ID: "E-0043",
				Milestones: []StatusMilestone{
					{ID: "M-0171", Status: entity.StatusDraft},
				},
			},
		},
	}
	views := []WorktreeView{
		{
			Path:           "/repo/wt-epic",
			Branch:         "epic/E-0043-area-tag",
			DriverKind:     string(entity.KindEpic),
			DriverEntityID: "E-0043",
			EpicMilestones: []EpicChildRow{
				{ID: "M-0171", Status: entity.StatusDone},
			},
		},
	}

	AnnotateWorktreeDivergence(report, views, "/repo/main")

	got := report.PlannedEpics[0].Milestones[0].WorktreeDivergence
	if got == nil {
		t.Fatalf("WorktreeDivergence = nil, want a flagged divergence on the PlannedEpics row")
	}
	if got.Status != entity.StatusDone {
		t.Errorf("Status = %q, want %q", got.Status, entity.StatusDone)
	}
}

// TestAnnotateWorktreeDivergence_MatchesNarrowEpicID covers the
// canonicalization on the epic-id map key: a worktree's driver id may
// be recorded at a narrower legacy width (`E-43`) than the report's
// canonical epic id (`E-0043`) — the lookup must still match.
func TestAnnotateWorktreeDivergence_MatchesNarrowEpicID(t *testing.T) {
	t.Parallel()
	report := &StatusReport{
		InFlightEpics: []StatusEpic{
			{
				ID: "E-0043",
				Milestones: []StatusMilestone{
					{ID: "M-0171", Status: entity.StatusDraft},
				},
			},
		},
	}
	views := []WorktreeView{
		{
			Path:           "/repo/wt-epic",
			Branch:         "epic/E-0043-area-tag",
			DriverKind:     string(entity.KindEpic),
			DriverEntityID: "E-43",
			EpicMilestones: []EpicChildRow{
				{ID: "M-0171", Status: entity.StatusDone},
			},
		},
	}

	AnnotateWorktreeDivergence(report, views, "/repo/main")

	if got := report.InFlightEpics[0].Milestones[0].WorktreeDivergence; got == nil {
		t.Fatalf("WorktreeDivergence = nil, want a narrow-width driver id (E-43) to still match canonical E-0043")
	}
}

// TestAnnotateWorktreeDivergence_NoOpWhenStatusesAgree guards against
// over-firing: when the sibling worktree's copy agrees with the
// current checkout, no annotation is added.
func TestAnnotateWorktreeDivergence_NoOpWhenStatusesAgree(t *testing.T) {
	t.Parallel()
	report := &StatusReport{
		InFlightEpics: []StatusEpic{
			{
				ID: "E-0043",
				Milestones: []StatusMilestone{
					{ID: "M-0171", Status: entity.StatusDone},
				},
			},
		},
	}
	views := []WorktreeView{
		{
			Path:           "/repo/wt-epic",
			Branch:         "epic/E-0043-area-tag",
			DriverKind:     string(entity.KindEpic),
			DriverEntityID: "E-0043",
			EpicMilestones: []EpicChildRow{
				{ID: "M-0171", Status: entity.StatusDone},
			},
		},
	}

	AnnotateWorktreeDivergence(report, views, "/repo/main")

	if got := report.InFlightEpics[0].Milestones[0].WorktreeDivergence; got != nil {
		t.Errorf("WorktreeDivergence = %+v, want nil (statuses agree)", got)
	}
}

// TestAnnotateWorktreeDivergence_IgnoresCurrentCheckout guards against
// self-comparison: the current checkout's own worktree entry (Path ==
// rootDir) never drives a divergence, even when it happens to carry an
// epic driver, because it reflects the same tree the report was built
// from and can never disagree with itself.
func TestAnnotateWorktreeDivergence_IgnoresCurrentCheckout(t *testing.T) {
	t.Parallel()
	report := &StatusReport{
		InFlightEpics: []StatusEpic{
			{
				ID: "E-0043",
				Milestones: []StatusMilestone{
					{ID: "M-0171", Status: entity.StatusDraft},
				},
			},
		},
	}
	views := []WorktreeView{
		{
			Path:           "/repo/main",
			Branch:         "epic/E-0043-area-tag",
			DriverKind:     string(entity.KindEpic),
			DriverEntityID: "E-0043",
			EpicMilestones: []EpicChildRow{
				{ID: "M-0171", Status: entity.StatusDone},
			},
		},
	}

	AnnotateWorktreeDivergence(report, views, "/repo/main")

	if got := report.InFlightEpics[0].Milestones[0].WorktreeDivergence; got != nil {
		t.Errorf("WorktreeDivergence = %+v, want nil (self-comparison excluded)", got)
	}
}

// TestAnnotateWorktreeDivergence_IgnoresNonEpicDriver guards against a
// milestone-driver worktree being mistaken for a source of epic-level
// milestone status: only an epic-driver worktree carries EpicMilestones
// rows to reconcile against.
func TestAnnotateWorktreeDivergence_IgnoresNonEpicDriver(t *testing.T) {
	t.Parallel()
	report := &StatusReport{
		InFlightEpics: []StatusEpic{
			{
				ID: "E-0043",
				Milestones: []StatusMilestone{
					{ID: "M-0171", Status: entity.StatusDraft},
				},
			},
		},
	}
	views := []WorktreeView{
		{
			Path:           "/repo/wt-milestone",
			Branch:         "milestone/M-0171-area-tag",
			DriverKind:     string(entity.KindMilestone),
			DriverEntityID: "M-0171",
		},
	}

	AnnotateWorktreeDivergence(report, views, "/repo/main")

	if got := report.InFlightEpics[0].Milestones[0].WorktreeDivergence; got != nil {
		t.Errorf("WorktreeDivergence = %+v, want nil (non-epic driver)", got)
	}
}

// TestStatusRun_FlagsCrossWorktreeDivergence is the end-to-end G-0277
// seam test: a real repo where an epic's milestone is `draft` on the
// current checkout but `done` on a sibling epic-branch worktree. The
// default `aiwf status` text output must flag the divergence inline on
// the milestone's own row, not just the soft worktree-summary footer.
func TestStatusRun_FlagsCrossWorktreeDivergence(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	main := t.TempDir()
	gitDo(t, main, "init", "-q", "-b", "main")

	writeEpic(t, main, "E-9002-divergence", "E-9002", entity.StatusActive)
	writeMilestone(t, main, "E-9002-divergence", "M-9002", entity.StatusDraft, "E-9002")
	gitDo(t, main, "add", "-A")
	gitDo(t, main, "commit", "-q", "-m", "base: E-9002 active, M-9002 draft")
	gitDo(t, main, "branch", "epic/E-9002-divergence")

	wtPath := filepath.Join(t.TempDir(), "wt-epic")
	gitDo(t, main, "worktree", "add", "-q", wtPath, "epic/E-9002-divergence")

	// Advance the milestone to done only on the sibling worktree's
	// branch — the main checkout's own copy stays draft, reproducing
	// the exact G-0277 scenario (an unmerged epic branch with a more
	// current milestone status than trunk/main sees).
	writeMilestone(t, wtPath, "E-9002-divergence", "M-9002", entity.StatusDone, "E-9002")
	gitDo(t, wtPath, "add", "-A")
	gitDo(t, wtPath, "commit", "-q", "-m", "promote M-9002 done")

	tr, loadErrs, err := tree.Load(ctx, main)
	if err != nil {
		t.Fatalf("tree.Load(main): %v", err)
	}
	// Precondition: the main checkout's own tree still reads M-9002 as
	// draft — the stale copy the fix must flag rather than trust.
	if got := tr.ByID("M-9002"); got == nil || got.Status != entity.StatusDraft {
		t.Fatalf("precondition: main tree should show M-9002 draft, got %#v", got)
	}

	report := BuildStatus(tr, loadErrs, time.Now())
	views, err := BuildWorktreeViews(ctx, main, tr)
	if err != nil {
		t.Fatalf("BuildWorktreeViews: %v", err)
	}
	AnnotateWorktreeDivergence(&report, views, main)

	var b strings.Builder
	if err := RenderStatusText(&b, &report, 0, false); err != nil {
		t.Fatalf("RenderStatusText: %v", err)
	}
	out := b.String()

	// "M-9002 — " (id followed by the row's title separator) is unique
	// to the milestone's own status row — a bare "M-9002" also appears
	// in this fixture's tdd-undeclared warning line.
	line := findLine(t, out, "M-9002 — ")
	if !strings.Contains(line, entity.StatusDone) || !strings.Contains(line, "E-9002") {
		t.Errorf("milestone row = %q, want it to flag %q as available on the E-9002 worktree", line, entity.StatusDone)
	}
	// The row must still show the checkout's own (stale) status too —
	// this is an annotation, not a silent overwrite of the branch-local
	// read.
	if !strings.Contains(line, entity.StatusDraft) {
		t.Errorf("milestone row = %q, want it to still show the checkout-local %q status", line, entity.StatusDraft)
	}
}

// findLine returns the single line in out containing needle, failing
// the test when there isn't exactly one match.
func findLine(t *testing.T, out, needle string) string {
	t.Helper()
	var match string
	found := 0
	for line := range strings.SplitSeq(out, "\n") {
		if strings.Contains(line, needle) {
			match = line
			found++
		}
	}
	if found != 1 {
		t.Fatalf("found %d lines containing %q, want exactly 1; output:\n%s", found, needle, out)
	}
	return match
}
