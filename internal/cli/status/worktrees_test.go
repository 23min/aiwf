package status

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestParseEntityFromBranch covers the conventional ritual-branch
// prefixes plus negative cases (non-matching shapes return empty).
//
// G-0122.
func TestParseEntityFromBranch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		branch string
		want   string
	}{
		{"epic branch canonical width", "epic/E-0033-pin-legal-kernel-verb-workflows", "E-0033"},
		{"epic branch narrow width", "epic/E-33-old-shape", "E-33"},
		{"milestone branch canonical", "milestone/M-0123-pass-c-reconcile", "M-0123"},
		{"milestone branch with deep slug", "milestone/M-0130-implement-fsm-history-consistent-check-rule-for-fsm-tree-invariant", "M-0130"},
		{"patch with gap id lowercase", "patch/g-0122-worktree-view", "G-0122"},
		{"patch with gap id uppercase", "patch/G-0122-worktree-view", "G-0122"},
		{"main is not a ritual branch", "main", ""},
		{"trunk fix branch no entity", "fix/some-bug", ""},
		{"chore branch no entity", "chore/dep-bump", ""},
		{"patch without entity id", "patch/refactor-stuff", ""},
		{"feature branch no entity", "feature/x", ""},
		{"epic prefix but no id", "epic/some-name", ""},
		{"empty string", "", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseEntityFromBranch(tc.branch)
			if got != tc.want {
				t.Errorf("parseEntityFromBranch(%q) = %q, want %q", tc.branch, got, tc.want)
			}
		})
	}
}

// TestParentEntity verifies AC composite ids strip to their parent
// milestone id; non-composite ids pass through unchanged.
func TestParentEntity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"M-0123/AC-1", "M-0123"},
		{"M-0007/AC-3", "M-0007"},
		{"M-0123", "M-0123"},
		{"E-0033", "E-0033"},
		{"G-0122", "G-0122"},
		{"", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got := parentEntity(tc.in)
			if got != tc.want {
				t.Errorf("parentEntity(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestScopeDefiningEntity covers the cascade's first step: walking
// branchAiwfEventRecord events for scope-defining trailer patterns and
// returning the driver entity with proper multi-entity disambiguation.
func TestScopeDefiningEntity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		events []branchAiwfEventRecord
		want   string
	}{
		{
			name:   "empty events returns empty",
			events: nil,
			want:   "",
		},
		{
			name: "single authorize on epic",
			events: []branchAiwfEventRecord{
				{Verb: "authorize", Entity: "E-0033"},
			},
			want: "E-0033",
		},
		{
			name: "single promote to in_progress on milestone",
			events: []branchAiwfEventRecord{
				{Verb: "promote", Entity: "M-0123", To: "in_progress"},
			},
			want: "M-0123",
		},
		{
			name: "phase promote on AC strips to parent milestone",
			events: []branchAiwfEventRecord{
				{Verb: "promote", Entity: "M-0123/AC-1", To: "green"},
			},
			want: "M-0123",
		},
		{
			name: "non-scope verb (add) skipped",
			events: []branchAiwfEventRecord{
				{Verb: "add", Entity: "M-0123", To: ""},
			},
			want: "",
		},
		{
			name: "edit-body skipped",
			events: []branchAiwfEventRecord{
				{Verb: "edit-body", Entity: "M-0123", To: ""},
			},
			want: "",
		},
		{
			name: "multi-entity with one active wins active",
			events: []branchAiwfEventRecord{
				// Newest first per git log default ordering.
				{Verb: "promote", Entity: "M-0124", To: "done"},
				{Verb: "promote", Entity: "M-0123", To: "in_progress"},
			},
			want: "M-0123",
		},
		{
			name: "multi-entity all-active prefers most-recent",
			events: []branchAiwfEventRecord{
				{Verb: "promote", Entity: "M-0123", To: "in_progress"}, // newest
				{Verb: "promote", Entity: "M-0124", To: "in_progress"}, // older
			},
			want: "M-0123",
		},
		{
			name: "only done events fall through to most-recent done",
			events: []branchAiwfEventRecord{
				{Verb: "promote", Entity: "M-0123", To: "done"}, // newest
				{Verb: "promote", Entity: "M-0124", To: "done"},
			},
			want: "M-0123",
		},
		{
			name: "authorize on epic + phase work on milestone — authorize wins (epic scope)",
			events: []branchAiwfEventRecord{
				{Verb: "promote", Entity: "M-0123/AC-1", To: "green"}, // newest
				{Verb: "authorize", Entity: "E-0033"},                 // older
			},
			// The phase event is also active; the cascade picks
			// whichever active event is newest. M-0123 is newer.
			want: "M-0123",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := scopeDefiningEntity(tc.events)
			if got != tc.want {
				t.Errorf("scopeDefiningEntity = %q, want %q\nevents: %+v", got, tc.want, tc.events)
			}
		})
	}
}

// TestMostRecentEntity verifies the fallback step: when no scope-
// defining events fire, the most recent aiwf-entity trailer wins.
func TestMostRecentEntity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		events []branchAiwfEventRecord
		want   string
	}{
		{"empty events", nil, ""},
		{"single event", []branchAiwfEventRecord{{Verb: "add", Entity: "G-0122"}}, "G-0122"},
		{"newest-first picks first", []branchAiwfEventRecord{
			{Verb: "edit-body", Entity: "M-0123"}, // newest
			{Verb: "add", Entity: "G-0146"},
		}, "M-0123"},
		{"composite id strips to parent", []branchAiwfEventRecord{
			{Verb: "edit-body", Entity: "M-0123/AC-1"},
		}, "M-0123"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mostRecentEntity(tc.events)
			if got != tc.want {
				t.Errorf("mostRecentEntity = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRenderAge covers the relative-time formatting across the grain
// breakpoints (just-now, minutes, hours, days, months, years) plus
// the clock-skew (future-time) and zero-time edge cases.
//
// G-0122 age display.
func TestRenderAge(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name      string
		t         time.Time
		wantMatch string // substring expected in the rendered result
	}{
		{"zero time returns empty", time.Time{}, ""},
		{"just now (30s ago)", now.Add(-30 * time.Second), "just now"},
		{"15 minutes ago", now.Add(-15 * time.Minute), "15m ago"},
		{"2 hours ago", now.Add(-2 * time.Hour), "2h ago"},
		{"1 day ago (singular)", now.Add(-24 * time.Hour), "1 day ago"},
		{"5 days ago (plural)", now.Add(-5 * 24 * time.Hour), "5 days ago"},
		{"2 months ago", now.Add(-60 * 24 * time.Hour), "2 months ago"},
		{"3 years ago", now.Add(-3 * 365 * 24 * time.Hour), "3 years ago"},
		{"future time (clock skew) renders date without rel suffix", now.Add(time.Hour), "2026-05-20"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := renderAge(tc.t, now)
			if tc.wantMatch == "" {
				if got != "" {
					t.Errorf("renderAge(zero) = %q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, tc.wantMatch) {
				t.Errorf("renderAge = %q, want substring %q", got, tc.wantMatch)
			}
			// Every non-zero case includes the date prefix (YYYY-MM-DD).
			if !strings.Contains(got, "-") {
				t.Errorf("renderAge = %q, missing date component", got)
			}
		})
	}
}

// TestWorktreeMetadataLine covers the suppression logic: each
// metric (created, last entity touch) renders only when its relative-
// age label would differ from HeadTime's label — same-display metrics
// collapse to avoid useless repetition.
//
// G-0122 user-feedback extension.
func TestWorktreeMetadataLine(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		v    WorktreeView
		want string
	}{
		{
			name: "all zero (no metrics) returns empty",
			v:    WorktreeView{},
			want: "",
		},
		{
			name: "created == head (same rendered label) suppressed",
			v: WorktreeView{
				HeadTime:    now.Add(-2 * time.Hour),
				CreatedTime: now.Add(-2 * time.Hour),
			},
			want: "",
		},
		{
			name: "created differs from head shows",
			v: WorktreeView{
				HeadTime:    now.Add(-2 * time.Hour),
				CreatedTime: now.Add(-5 * 24 * time.Hour),
			},
			want: "created 5 days ago",
		},
		{
			name: "last entity differs from head shows",
			v: WorktreeView{
				HeadTime:       now.Add(-30 * time.Minute),
				LastEntityTime: now.Add(-3 * time.Hour),
			},
			want: "last entity touch 3h ago",
		},
		{
			name: "both differ: joined by bullet",
			v: WorktreeView{
				HeadTime:       now.Add(-2 * time.Hour),
				CreatedTime:    now.Add(-5 * 24 * time.Hour),
				LastEntityTime: now.Add(-24 * time.Hour),
			},
			want: "created 5 days ago  •  last entity touch 1 day ago",
		},
		{
			name: "head zero with metrics shows them",
			v: WorktreeView{
				CreatedTime:    now.Add(-2 * time.Hour),
				LastEntityTime: now.Add(-1 * time.Hour),
			},
			want: "created 2h ago  •  last entity touch 1h ago",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := worktreeMetadataLine(&tc.v, now)
			if got != tc.want {
				t.Errorf("worktreeMetadataLine = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRenderWorktreeViews covers the per-worktree section layout: one
// section per worktree with branch on its own line, driver row, and
// kind-specific expansion (epic gets Milestones+Gaps; milestone gets
// breadcrumb+ACs; gap and trunk get just the driver row; stale adds an
// inline marker).
func TestRenderWorktreeViews(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		views    []WorktreeView
		mustHave []string
		mustNot  []string
	}{
		{
			name:     "no worktrees",
			views:    nil,
			mustHave: []string{"No worktrees found."},
		},
		{
			name: "milestone-driver: epic header, → driven, depends_on + ACs + surfaced gaps",
			views: []WorktreeView{
				{
					Path: "/repo/wt-m123", Branch: "milestone/M-0123-pass-c",
					DriverEntityID: "M-0123", DriverKind: "milestone",
					DriverStatus: "in_progress", DriverTitle: "Pass C reconcile",
					ParentEpicID: "E-0033", ParentEpicTitle: "Pin legal workflows", ParentEpicStatus: "active",
					DependsOn: []EpicChildRow{
						{ID: "M-0001", Title: "Bootstrap", Status: "done"},
					},
					ACs: []ACRow{
						{ID: "AC-1", Title: "Spec types", Status: "open", TDDPhase: "red"},
						{ID: "AC-3", Title: "Antirules", Status: "met", TDDPhase: "done"},
					},
					SurfacedGaps: []EpicChildRow{
						{ID: "G-0129", Title: "Typed finding-code constants", Status: "open"},
					},
				},
			},
			mustHave: []string{
				"Worktree: /repo/wt-m123",
				"⎇ milestone/M-0123-pass-c",
				"E-0033 — Pin legal workflows [active]",
				"→ M-0123 — Pass C reconcile [in_progress]  (driven)",
				"depends on:",
				"M-0001 — Bootstrap [done]",
				"ACs:",
				"AC-1 — Spec types [open, red]",
				"AC-3 — Antirules [met, done]",
				"Surfaced gaps:",
				"G-0129 — Typed finding-code constants [open]",
			},
			mustNot: []string{"In-flight worktrees", "Trunk", "Driving", "Under E-0033"},
		},
		{
			name: "trunk worktree with no other in-flight gets the default marker",
			views: []WorktreeView{
				{Path: "/repo", Branch: "main"},
			},
			mustHave: []string{
				"Worktree: /repo",
				"⎇ main",
				"No in-flight scope (trunk)",
			},
			mustNot: []string{"Driving", "Trunk (no in-flight scope)", "Other in-flight"},
		},
		{
			name: "trunk worktree surfaces Other in-flight with branch + age and no-branch cases",
			views: []WorktreeView{
				{
					Path: "/repo", Branch: "main",
					OtherInFlight: []OtherInFlightRow{
						{
							ID: "E-0034", Title: "Retire docs/pocv3/", Status: "active",
							Branch: "epic/E-0034-retire-pocv3", BranchTime: time.Now().Add(-5 * 24 * time.Hour),
						},
						{
							ID: "M-0136", Title: "aiwf acknowledge-illegal", Status: "in_progress",
							// no Branch — work directly on trunk
						},
					},
				},
			},
			mustHave: []string{
				"Worktree: /repo",
				"⎇ main",
				"Other in-flight:",
				"→ E-0034 — Retire docs/pocv3/ [active]",
				"branch: epic/E-0034-retire-pocv3 (no worktree, ",
				"5 days ago)",
				"→ M-0136 — aiwf acknowledge-illegal [in_progress]",
				"(no branch, on trunk)",
			},
			mustNot: []string{"No in-flight scope (trunk)"},
		},
		{
			name: "stale worktree inline marker + cleanup hint",
			views: []WorktreeView{
				{
					Path: "/repo/wt-old", Branch: "milestone/M-0099-old",
					DriverEntityID: "M-0099", DriverKind: "milestone",
					DriverStatus: "done", DriverTitle: "Old work", Stale: true,
				},
			},
			mustHave: []string{
				"Worktree: /repo/wt-old",
				"⎇ milestone/M-0099-old",
				"M-0099 — Old work [done]",
				"STALE — driver is terminal",
				"git worktree remove /repo/wt-old",
			},
		},
		{
			name: "gap-driver worktree (wf-patch) shows just the gap",
			views: []WorktreeView{
				{
					Path: "/repo/wt-g0122", Branch: "patch/g-0122-worktree-view",
					DriverEntityID: "G-0122", DriverKind: "gap",
					DriverStatus: "open", DriverTitle: "Worktree-aware view",
				},
			},
			mustHave: []string{
				"Worktree: /repo/wt-g0122",
				"⎇ patch/g-0122-worktree-view",
				"G-0122 — Worktree-aware view [open]",
			},
			mustNot: []string{"Milestones:", "Gaps:", "ACs:", "Under"},
		},
		{
			name: "epic-driver expands milestones (ordered) + gaps",
			views: []WorktreeView{
				{
					Path: "/repo/wt-e0033", Branch: "epic/E-0033-pin",
					DriverEntityID: "E-0033", DriverKind: "epic",
					DriverStatus: "active", DriverTitle: "Pin legal workflows",
					EpicMilestones: []EpicChildRow{
						{ID: "M-0124", Title: "Positive coverage", Status: "draft"},
						{ID: "M-0123", Title: "Pass C", Status: "in_progress"},
						{ID: "M-0125", Title: "Negative coverage", Status: "done"},
					},
					EpicClosesGaps: []EpicChildRow{
						{ID: "G-0121", Title: "legal workflows", Status: "open"},
					},
				},
			},
			mustHave: []string{
				"E-0033 — Pin legal workflows [active]",
				"Milestones:",
				"M-0123 — Pass C [in_progress]",
				"M-0124 — Positive coverage [draft]",
				"M-0125 — Negative coverage [done]",
				"Closes gaps:",
				"G-0121 — legal workflows [open]",
			},
		},
		{
			name: "epic milestone driven-by wraps to next line, indented",
			views: []WorktreeView{
				{
					Path: "/repo/wt-e0033", Branch: "epic/E-0033-pin",
					DriverEntityID: "E-0033", DriverKind: "epic",
					DriverStatus: "active", DriverTitle: "Pin",
					EpicMilestones: []EpicChildRow{
						{ID: "M-0123", Title: "Pass C", Status: "in_progress", DrivenByPath: "/other/wt-m123"},
					},
				},
			},
			mustHave: []string{
				"M-0123 — Pass C [in_progress]",
				"        (driven by /other/wt-m123)",
			},
		},
		{
			name: "blank line separator between two worktree sections",
			views: []WorktreeView{
				{Path: "/repo/wt-a", Branch: "milestone/M-0001-a", DriverEntityID: "M-0001", DriverKind: "milestone", DriverStatus: "in_progress", DriverTitle: "A"},
				{Path: "/repo/wt-b", Branch: "milestone/M-0002-b", DriverEntityID: "M-0002", DriverKind: "milestone", DriverStatus: "in_progress", DriverTitle: "B"},
			},
			mustHave: []string{
				"Worktree: /repo/wt-a",
				"Worktree: /repo/wt-b",
				"→ M-0001 — A [in_progress]  (driven)",
				"\nWorktree: /repo/wt-b", // blank line precedes the second section
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := RenderWorktreeViews(&buf, tc.views, false); err != nil {
				t.Fatalf("RenderWorktreeViews: %v", err)
			}
			got := buf.String()
			for _, want := range tc.mustHave {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\n---output---\n%s", want, got)
				}
			}
			for _, forbidden := range tc.mustNot {
				if strings.Contains(got, forbidden) {
					t.Errorf("output should not contain %q\n---output---\n%s", forbidden, got)
				}
			}
		})
	}
}
