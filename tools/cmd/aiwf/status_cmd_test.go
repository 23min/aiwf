package main

import (
	"os"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// TestBuildStatus_FiltersToInFlight verifies the in-flight rules:
// only `active` epics, only `proposed` ADRs/decisions, only `open`
// gaps. Closed entities are excluded.
func TestBuildStatus_FiltersToInFlight(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-01", Title: "Active", Status: "active"},
			{Kind: entity.KindEpic, ID: "E-02", Title: "Done", Status: "done"},
			{Kind: entity.KindEpic, ID: "E-03", Title: "Proposed", Status: "proposed"},

			{Kind: entity.KindMilestone, ID: "M-001", Title: "First", Status: "done", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-002", Title: "Second", Status: "in_progress", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-003", Title: "Third", Status: "draft", Parent: "E-01"},
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
	if len(r.InFlightEpics) != 0 || len(r.OpenDecisions) != 0 || len(r.OpenGaps) != 0 {
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
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}
