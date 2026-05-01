package main

import (
	"os"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// TestBuildStatus_FiltersToInFlight verifies the in-flight rules:
// only `active` epics in InFlightEpics, only `proposed` epics in
// PlannedEpics, `done`/`cancelled` epics excluded entirely; only
// `proposed` ADRs/decisions in OpenDecisions; only `open` gaps in
// OpenGaps.
func TestBuildStatus_FiltersToInFlight(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-01", Title: "Active", Status: "active"},
			{Kind: entity.KindEpic, ID: "E-02", Title: "Done", Status: "done"},
			{Kind: entity.KindEpic, ID: "E-03", Title: "Proposed", Status: "proposed"},

			{Kind: entity.KindMilestone, ID: "M-001", Title: "First", Status: "done", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-002", Title: "Second", Status: "in_progress", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-003", Title: "Third", Status: "draft", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-004", Title: "Planned", Status: "draft", Parent: "E-03"},
			{Kind: entity.KindMilestone, ID: "M-099", Title: "Stale", Status: "done", Parent: "E-02"},

			{Kind: entity.KindADR, ID: "ADR-0001", Title: "Proposed ADR", Status: "proposed"},
			{Kind: entity.KindADR, ID: "ADR-0002", Title: "Accepted", Status: "accepted"},

			{Kind: entity.KindDecision, ID: "D-001", Title: "Open D", Status: "proposed"},
			{Kind: entity.KindDecision, ID: "D-002", Title: "Done D", Status: "accepted"},

			{Kind: entity.KindGap, ID: "G-001", Title: "Open gap", Status: "open", DiscoveredIn: "M-001"},
			{Kind: entity.KindGap, ID: "G-002", Title: "Addressed gap", Status: "addressed"},
		},
	}
	r := buildStatus(tr, nil)

	if len(r.InFlightEpics) != 1 || r.InFlightEpics[0].ID != "E-01" {
		t.Errorf("InFlightEpics: only E-01 expected, got %+v", r.InFlightEpics)
	}
	gotMS := r.InFlightEpics[0].Milestones
	if len(gotMS) != 3 {
		t.Errorf("expected 3 milestones under E-01 (all of them), got %d", len(gotMS))
	}

	if len(r.PlannedEpics) != 1 || r.PlannedEpics[0].ID != "E-03" {
		t.Errorf("PlannedEpics: only E-03 expected, got %+v", r.PlannedEpics)
	}
	if len(r.PlannedEpics[0].Milestones) != 1 || r.PlannedEpics[0].Milestones[0].ID != "M-004" {
		t.Errorf("PlannedEpics[E-03] milestones: expected [M-004], got %+v", r.PlannedEpics[0].Milestones)
	}

	if len(r.OpenDecisions) != 2 {
		t.Errorf("OpenDecisions: expected 2 (1 ADR + 1 D-NNN), got %d", len(r.OpenDecisions))
	}
	for _, d := range r.OpenDecisions {
		if d.Status != "proposed" {
			t.Errorf("non-proposed decision leaked: %+v", d)
		}
	}

	if len(r.OpenGaps) != 1 || r.OpenGaps[0].ID != "G-001" {
		t.Errorf("OpenGaps: only G-001 expected, got %+v", r.OpenGaps)
	}

	if r.Health.Entities != len(tr.Entities) {
		t.Errorf("Health.Entities = %d, want %d", r.Health.Entities, len(tr.Entities))
	}
}

// TestBuildStatus_MilestoneOrder verifies milestones under an in-flight
// epic come back in id order.
func TestBuildStatus_MilestoneOrder(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-01", Title: "Active", Status: "active"},
			// Out-of-tree-walk-order on purpose; buildStatus must sort.
			{Kind: entity.KindMilestone, ID: "M-003", Title: "Third", Status: "draft", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-001", Title: "First", Status: "done", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-002", Title: "Second", Status: "in_progress", Parent: "E-01"},
		},
	}
	r := buildStatus(tr, nil)

	got := []string{}
	for _, m := range r.InFlightEpics[0].Milestones {
		got = append(got, m.ID)
	}
	want := []string{"M-001", "M-002", "M-003"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("milestone order [%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestBuildStatus_EmptyTree: no entities — every section is empty,
// health has zero counts. No panic.
func TestBuildStatus_EmptyTree(t *testing.T) {
	r := buildStatus(&tree.Tree{}, nil)
	if len(r.InFlightEpics) != 0 || len(r.PlannedEpics) != 0 ||
		len(r.OpenDecisions) != 0 || len(r.OpenGaps) != 0 ||
		len(r.Warnings) != 0 {
		t.Errorf("expected all empty: %+v", r)
	}
	if r.Health.Entities != 0 || r.Health.Errors != 0 || r.Health.Warnings != 0 {
		t.Errorf("expected zero health: %+v", r.Health)
	}
}

// TestRenderStatusText_MarksInProgress verifies the → marker on the
// in-progress milestone and the ✓ on the done one.
func TestRenderStatusText_MarksInProgress(t *testing.T) {
	r := statusReport{
		Date: "2026-04-28",
		InFlightEpics: []statusEpic{{
			ID: "E-01", Title: "Active", Status: "active",
			Milestones: []statusMilestone{
				{ID: "M-001", Title: "First", Status: "done"},
				{ID: "M-002", Title: "Second", Status: "in_progress"},
				{ID: "M-003", Title: "Third", Status: "draft"},
			},
		}},
		Health: statusHealthCounts{Entities: 4},
	}
	out := captureStdout(t, func() {
		if err := renderStatusText(os.Stdout, &r); err != nil {
			t.Fatalf("renderStatusText: %v", err)
		}
	})
	got := string(out)

	for _, want := range []string{
		"aiwf status — 2026-04-28",
		"E-01 — Active",
		" ✓ M-001",
		" → M-002",
		"M-003 — Third    [draft]",
		"Roadmap",
		"(nothing planned)",
		"Warnings",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

// TestBuildStatus_Warnings: a tree that trips a known warning rule
// (gap-resolved-has-resolver) populates the Warnings slice with the
// relevant fields, and Health.Warnings is incremented in lockstep.
func TestBuildStatus_Warnings(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{
				Kind: entity.KindGap, ID: "G-001",
				Title: "Half-resolved", Status: "addressed",
				Path: "work/gaps/G-001-half-resolved.md",
			},
		},
	}
	r := buildStatus(tr, nil)

	if r.Health.Warnings != 1 {
		t.Fatalf("Health.Warnings = %d, want 1", r.Health.Warnings)
	}
	if len(r.Warnings) != 1 {
		t.Fatalf("Warnings = %d, want 1", len(r.Warnings))
	}
	w := r.Warnings[0]
	if w.Code != "gap-resolved-has-resolver" {
		t.Errorf("Warnings[0].Code = %q, want %q", w.Code, "gap-resolved-has-resolver")
	}
	if w.EntityID != "G-001" {
		t.Errorf("Warnings[0].EntityID = %q, want G-001", w.EntityID)
	}
	if w.Message == "" {
		t.Error("Warnings[0].Message is empty")
	}
}

// TestRenderStatusText_Warnings: the text renderer surfaces each
// warning row with its code, [entity-id], and message.
func TestRenderStatusText_Warnings(t *testing.T) {
	r := statusReport{
		Date: "2026-05-01",
		Warnings: []statusFinding{
			{Code: "gap-resolved-has-resolver", EntityID: "G-001", Message: "gap is marked addressed but addressed_by is empty"},
		},
		Health: statusHealthCounts{Entities: 1, Warnings: 1},
	}
	out := captureStdout(t, func() {
		if err := renderStatusText(os.Stdout, &r); err != nil {
			t.Fatalf("renderStatusText: %v", err)
		}
	})
	got := string(out)

	for _, want := range []string{
		"Warnings",
		"gap-resolved-has-resolver",
		"[G-001]",
		"gap is marked addressed",
		"run `aiwf check` for details",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

// TestRenderStatusMarkdown_FullReport: a representative report renders
// every section with mermaid blocks for both in-flight and roadmap
// epics. Asserts on stable substrings rather than a byte-for-byte
// golden so the date line and small markdown formatting tweaks don't
// require golden refresh.
func TestRenderStatusMarkdown_FullReport(t *testing.T) {
	r := statusReport{
		Date: "2026-05-01",
		InFlightEpics: []statusEpic{{
			ID: "E-01", Title: "Active epic", Status: "active",
			Milestones: []statusMilestone{
				{ID: "M-001", Title: "First", Status: "done"},
				{ID: "M-002", Title: "Second", Status: "in_progress"},
			},
		}},
		PlannedEpics: []statusEpic{{
			ID: "E-02", Title: "Planned epic", Status: "proposed",
			Milestones: []statusMilestone{
				{ID: "M-010", Title: "Setup", Status: "draft"},
			},
		}},
		OpenDecisions: []statusEntity{
			{ID: "ADR-0001", Title: "Adopt X", Status: "proposed", Kind: "adr"},
		},
		OpenGaps: []statusGap{
			{ID: "G-001", Title: "Edge case", DiscoveredIn: "M-002"},
		},
		Warnings: []statusFinding{
			{Code: "gap-resolved-has-resolver", EntityID: "G-002", Message: "gap is marked addressed but addressed_by is empty"},
		},
		Health: statusHealthCounts{Entities: 7, Errors: 0, Warnings: 1},
	}
	out := captureStdout(t, func() {
		if err := renderStatusMarkdown(os.Stdout, &r); err != nil {
			t.Fatalf("renderStatusMarkdown: %v", err)
		}
	})
	got := string(out)

	for _, want := range []string{
		"# aiwf status — 2026-05-01",
		"_7 entities · 0 errors · 1 warnings · run `aiwf check` for details_",
		"## In flight",
		"### E-01 — Active epic _(active)_",
		"```mermaid\nflowchart LR",
		"E_01[\"E-01<br/>Active epic\"]:::epic_active",
		"M_002[\"M-002<br/>Second\"]:::ms_in_progress",
		"E_01 --> M_001",
		"## Roadmap",
		"### E-02 — Planned epic _(proposed)_",
		"E_02[\"E-02<br/>Planned epic\"]:::epic_proposed",
		"## Open decisions",
		"| ADR-0001 | adr | Adopt X | proposed |",
		"## Open gaps",
		"| G-001 | Edge case | M-002 |",
		"## Warnings",
		"| gap-resolved-has-resolver | G-002 |  | gap is marked addressed but addressed_by is empty |",
		"## Recent activity",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("markdown output missing %q:\nFULL OUTPUT:\n%s", want, got)
		}
	}
}

// TestRenderStatusMarkdown_EmptyReport: every section emits its
// italic empty-state line and no orphan mermaid block leaks.
func TestRenderStatusMarkdown_EmptyReport(t *testing.T) {
	r := statusReport{Date: "2026-05-01"}
	out := captureStdout(t, func() {
		if err := renderStatusMarkdown(os.Stdout, &r); err != nil {
			t.Fatalf("renderStatusMarkdown: %v", err)
		}
	})
	got := string(out)

	for _, want := range []string{
		"_(no active epics)_",
		"_(nothing planned)_",
		"## Open decisions\n\n_(none)_",
		"## Open gaps\n\n_(none)_",
		"## Warnings\n\n_(none)_",
		"## Recent activity\n\n_(none)_",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("empty-report output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "```mermaid") {
		t.Error("empty report should not contain a mermaid block")
	}
}

// TestMdEscape: pipes/backticks are escaped, double-quotes downgraded
// to single, newlines flattened to a space. Pipes break table rows;
// double-quotes break mermaid labels.
func TestMdEscape(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"a | b", "a \\| b"},
		{"a `code` b", "a \\`code\\` b"},
		{"a \"q\" b", "a 'q' b"},
		{"a\nb", "a b"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := mdEscape(tc.in); got != tc.want {
				t.Errorf("mdEscape(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRunStatus_BadFormat: --format with an unsupported value returns
// the usage exit code.
func TestRunStatus_BadFormat(t *testing.T) {
	rc := runStatus([]string{"--format=xml"})
	if rc != exitUsage {
		t.Errorf("rc = %d, want exitUsage (%d)", rc, exitUsage)
	}
}
