package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestBuildStatus_FiltersToInFlight verifies the in-flight rules:
// only `active` epics in InFlightEpics, only `proposed` epics in
// PlannedEpics, `done`/`cancelled` epics excluded entirely; only
// `proposed` ADRs/decisions in OpenDecisions; only `open` gaps in
// OpenGaps.
func TestBuildStatus_FiltersToInFlight(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Active", Status: "active"},
			{Kind: entity.KindEpic, ID: "E-0002", Title: "Done", Status: "done"},
			{Kind: entity.KindEpic, ID: "E-0003", Title: "Proposed", Status: "proposed"},

			{Kind: entity.KindMilestone, ID: "M-0001", Title: "First", Status: "done", Parent: "E-0001"},
			{Kind: entity.KindMilestone, ID: "M-0002", Title: "Second", Status: "in_progress", Parent: "E-0001"},
			{Kind: entity.KindMilestone, ID: "M-0003", Title: "Third", Status: "draft", Parent: "E-0001"},
			{Kind: entity.KindMilestone, ID: "M-0004", Title: "Planned", Status: "draft", Parent: "E-0003"},
			{Kind: entity.KindMilestone, ID: "M-0099", Title: "Stale", Status: "done", Parent: "E-0002"},

			{Kind: entity.KindADR, ID: "ADR-0001", Title: "Proposed ADR", Status: "proposed"},
			{Kind: entity.KindADR, ID: "ADR-0002", Title: "Accepted", Status: "accepted"},

			{Kind: entity.KindDecision, ID: "D-0001", Title: "Open D", Status: "proposed"},
			{Kind: entity.KindDecision, ID: "D-0002", Title: "Done D", Status: "accepted"},

			{Kind: entity.KindGap, ID: "G-0001", Title: "Open gap", Status: "open", DiscoveredIn: "M-0001"},
			{Kind: entity.KindGap, ID: "G-0002", Title: "Addressed gap", Status: "addressed"},
		},
	}
	r := buildStatus(tr, nil)

	if len(r.InFlightEpics) != 1 || r.InFlightEpics[0].ID != "E-0001" {
		t.Errorf("InFlightEpics: only E-01 expected, got %+v", r.InFlightEpics)
	}
	gotMS := r.InFlightEpics[0].Milestones
	if len(gotMS) != 3 {
		t.Errorf("expected 3 milestones under E-01 (all of them), got %d", len(gotMS))
	}

	if len(r.PlannedEpics) != 1 || r.PlannedEpics[0].ID != "E-0003" {
		t.Errorf("PlannedEpics: only E-03 expected, got %+v", r.PlannedEpics)
	}
	if len(r.PlannedEpics[0].Milestones) != 1 || r.PlannedEpics[0].Milestones[0].ID != "M-0004" {
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

	if len(r.OpenGaps) != 1 || r.OpenGaps[0].ID != "G-0001" {
		t.Errorf("OpenGaps: only G-001 expected, got %+v", r.OpenGaps)
	}

	if r.Health.Entities != len(tr.Entities) {
		t.Errorf("Health.Entities = %d, want %d", r.Health.Entities, len(tr.Entities))
	}
}

// TestBuildStatus_MilestoneOrder verifies milestones under an in-flight
// epic come back in id order.
func TestBuildStatus_MilestoneOrder(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Active", Status: "active"},
			// Out-of-tree-walk-order on purpose; buildStatus must sort.
			{Kind: entity.KindMilestone, ID: "M-0003", Title: "Third", Status: "draft", Parent: "E-0001"},
			{Kind: entity.KindMilestone, ID: "M-0001", Title: "First", Status: "done", Parent: "E-0001"},
			{Kind: entity.KindMilestone, ID: "M-0002", Title: "Second", Status: "in_progress", Parent: "E-0001"},
		},
	}
	r := buildStatus(tr, nil)

	got := []string{}
	for _, m := range r.InFlightEpics[0].Milestones {
		got = append(got, m.ID)
	}
	want := []string{"M-0001", "M-0002", "M-0003"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("milestone order [%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestBuildStatus_EmptyTree: no entities — every section is empty,
// health has zero counts. No panic.
func TestBuildStatus_EmptyTree(t *testing.T) {
	t.Parallel()
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
			ID: "E-0001", Title: "Active", Status: "active",
			Milestones: []statusMilestone{
				{ID: "M-0001", Title: "First", Status: "done"},
				{ID: "M-0002", Title: "Second", Status: "in_progress"},
				{ID: "M-0003", Title: "Third", Status: "draft"},
			},
		}},
		Health: statusHealthCounts{Entities: 4},
	}
	out := captureStdout(t, func() {
		if err := renderStatusText(os.Stdout, &r, 0, false); err != nil {
			t.Fatalf("renderStatusText: %v", err)
		}
	})
	got := string(out)

	for _, want := range []string{
		"aiwf status — 2026-04-28",
		"E-0001 — Active",
		" ✓ M-0001",
		" → M-0002",
		"M-0003 — Third    [draft]",
		"Roadmap",
		"(nothing planned)",
		"Warnings",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

// TestStatusCmd_NoArchivedFlag — M-0087/AC-3: `aiwf status` exposes
// no `--archived` flag. ADR-0004 §"Display surfaces" pins this: "The
// narrative view is forward-looking; archive inspection lives in
// `aiwf list --archived`."
//
// The completion-drift test (cmd/aiwf/completion_drift_test.go)
// already enumerates every flag of every verb at CI time — that's the
// chokepoint. This AC-level test is the direct mechanical assertion
// on `newStatusCmd()`'s flag set, scoped to the one flag name the
// AC forbids, so a future change that adds `--archived` to status
// fails this test specifically and not just the drift summary.
func TestStatusCmd_NoArchivedFlag(t *testing.T) {
	t.Parallel()
	cmd := newStatusCmd()
	if cmd.Flags().Lookup("archived") != nil {
		t.Errorf("status has --archived flag; ADR-0004 §\"Display surfaces\" forbids it on the status verb (archive inspection lives in `aiwf list --archived`)")
	}
}

// TestBuildStatus_SweepPendingPopulatedWhenTerminalsLiveInActive
// — M-0087/AC-1: when the loaded tree carries terminal-status
// entities still in active dirs, buildStatus populates a dedicated
// SweepPending field on the report. The field carries the count and
// the operator-facing message naming the remediation verb.
//
// Decoupled from r.Warnings on purpose: per ADR-0004 §"Display
// surfaces", the sweep-pending one-liner lives in the tree-health
// section of `aiwf status`, not in the general warnings list. The
// per-file `terminal-entity-not-archived` warnings can still appear
// alongside, but the aggregate count is reported separately so the
// reader sees "Sweep pending: N" inline with health rather than
// mixed in with body-empty / resolver-missing warnings.
func TestBuildStatus_SweepPendingPopulatedWhenTerminalsLiveInActive(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			// Two terminal-status gaps, both in the active dir.
			// `addressed` is terminal-for-kind gap (entity.IsTerminal).
			{
				Kind: entity.KindGap, ID: "G-0001",
				Title: "Closed-but-active", Status: "addressed",
				Path:        "work/gaps/G-0001-closed-but-active.md",
				AddressedBy: []string{"M-0001"},
			},
			{
				Kind: entity.KindGap, ID: "G-0002",
				Title: "Also-closed-but-active", Status: "addressed",
				Path:        "work/gaps/G-0002-also-closed-but-active.md",
				AddressedBy: []string{"M-0001"},
			},
		},
	}
	r := buildStatus(tr, nil)

	if r.SweepPending == nil {
		t.Fatalf("SweepPending = nil, want non-nil (2 terminal entities in active dirs)")
	}
	if r.SweepPending.Count != 2 {
		t.Errorf("SweepPending.Count = %d, want 2", r.SweepPending.Count)
	}
	if r.SweepPending.Message == "" {
		t.Errorf("SweepPending.Message is empty; want operator-facing one-liner")
	}
	if !strings.Contains(r.SweepPending.Message, "aiwf archive") {
		t.Errorf("SweepPending.Message missing the remediation verb name: %q", r.SweepPending.Message)
	}
}

// TestBuildStatus_SweepPendingNilWhenNoTerminalsInActive — M-0087/
// AC-2: when no entity is terminal-in-active, SweepPending is nil
// (not a zero-count struct), so the text renderer can skip the
// section entirely with a single nil-check. The branch is exercised
// here as the explicit zero-case partner to AC-1's non-zero case.
//
// Active-only entities + an already-archived terminal both stay
// silent: the archive-sweep-pending rule only counts terminals in
// the active dir.
func TestBuildStatus_SweepPendingNilWhenNoTerminalsInActive(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindGap, ID: "G-0001", Title: "Active", Status: "open", Path: "work/gaps/G-0001-active.md"},
			// Already-archived terminal: under archive/, so the
			// finding-rule's path filter skips it.
			{Kind: entity.KindGap, ID: "G-0002", Title: "Done", Status: "addressed", Path: "work/gaps/archive/G-0002-done.md", AddressedBy: []string{"M-0001"}},
		},
	}
	r := buildStatus(tr, nil)

	if r.SweepPending != nil {
		t.Fatalf("SweepPending = %+v, want nil (no terminals in active dirs)", r.SweepPending)
	}
}

// TestRenderStatusText_SweepPendingLineAppearsInHealthSection —
// M-0087/AC-1 render-side: when SweepPending is non-nil, the text
// renderer emits the one-liner inside the Health section,
// positioned so the reader sees it inline with the entity / errors
// / warnings count summary rather than buried in the Warnings list.
//
// Per CLAUDE.md "Substring assertions are not structural
// assertions": the assertion scopes the substring match to the
// Health section by splitting the output on the "Health\n" header
// and asserting the message appears after that split. A plain
// flat-substring match would not distinguish "line lands in Health"
// from "line lands in Warnings."
func TestRenderStatusText_SweepPendingLineAppearsInHealthSection(t *testing.T) {
	r := statusReport{
		Date: "2026-05-10",
		SweepPending: &statusSweepPending{
			Count:   3,
			Message: "Sweep pending: 3 terminal entities not yet archived (run `aiwf archive --dry-run` to preview)",
		},
		Health: statusHealthCounts{Entities: 5},
	}
	out := captureStdout(t, func() {
		if err := renderStatusText(os.Stdout, &r, 0, false); err != nil {
			t.Fatalf("renderStatusText: %v", err)
		}
	})
	got := string(out)

	const healthHeader = "Health\n"
	idx := strings.Index(got, healthHeader)
	if idx < 0 {
		t.Fatalf("output missing %q header:\n%s", healthHeader, got)
	}
	healthBody := got[idx+len(healthHeader):]
	if !strings.Contains(healthBody, "Sweep pending: 3") {
		t.Errorf("Health section missing sweep-pending one-liner:\n--- Health body ---\n%s\n--- full output ---\n%s", healthBody, got)
	}
	if !strings.Contains(healthBody, "aiwf archive --dry-run") {
		t.Errorf("Health section missing remediation verb name:\n%s", healthBody)
	}
}

// TestRenderStatusText_SweepPendingLineHiddenWhenNil —
// M-0087/AC-2 render-side: when SweepPending is nil, the renderer
// emits no "Sweep pending:" line anywhere in the output. Zero-count
// stays silent; the operator should not see "Sweep pending: 0".
func TestRenderStatusText_SweepPendingLineHiddenWhenNil(t *testing.T) {
	r := statusReport{
		Date:         "2026-05-10",
		SweepPending: nil,
		Health:       statusHealthCounts{Entities: 5},
	}
	out := captureStdout(t, func() {
		if err := renderStatusText(os.Stdout, &r, 0, false); err != nil {
			t.Fatalf("renderStatusText: %v", err)
		}
	})
	got := string(out)

	if strings.Contains(got, "Sweep pending") {
		t.Errorf("output contains \"Sweep pending\" when SweepPending is nil:\n%s", got)
	}
}

// TestRunStatusCmd_SweepPendingSeam — M-0087/AC-1 seam: drives the
// dispatcher end-to-end (resolveRoot → tree.Load → buildStatus →
// renderStatusText), so an integration regression in the wiring
// would fail this test. Fixture: a synthetic on-disk tree carrying
// one active gap and one terminal-in-active gap. The text output
// must show "Sweep pending: 1" inside the Health section.
//
// Per CLAUDE.md "Test the seam, not just the layer." Unit coverage
// of buildStatus + renderStatusText is necessary but not sufficient
// — this test pins that the verb dispatcher actually routes through
// the new field.
func TestRunStatusCmd_SweepPendingSeam(t *testing.T) {
	root := setupCLITestRepo(t)

	// One active gap to keep the tree non-empty.
	if err := os.MkdirAll(filepath.Join(root, "work", "gaps"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", "gaps", "G-0001-active.md"), []byte(`---
id: G-0001
title: Active gap
status: open
---
## What's missing

Active gap body.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	// One terminal gap still in active dir (pending sweep).
	if err := os.WriteFile(filepath.Join(root, "work", "gaps", "G-0002-closed.md"), []byte(`---
id: G-0002
title: Closed-but-active gap
status: addressed
addressed_by:
    - M-0001
---
## What's missing

Closed-but-active gap body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if rc := run([]string{"status", "--root", root}); rc != cliutil.ExitOK {
			t.Fatalf("status: rc = %d", rc)
		}
	})

	got := string(out)
	const healthHeader = "Health\n"
	idx := strings.Index(got, healthHeader)
	if idx < 0 {
		t.Fatalf("output missing %q header:\n%s", healthHeader, got)
	}
	healthBody := got[idx+len(healthHeader):]
	if !strings.Contains(healthBody, "Sweep pending: 1") {
		t.Errorf("Health section missing \"Sweep pending: 1\":\n--- Health body ---\n%s\n--- full output ---\n%s", healthBody, got)
	}
}

// TestBuildStatus_Warnings: a tree that trips a known warning rule
// (gap-resolved-has-resolver) populates the Warnings slice with the
// relevant fields, and Health.Warnings is incremented in lockstep.
//
// Path is under archive/ so the M-0086 terminal-entity-not-archived
// and archive-sweep-pending rules do not pile on — gap-resolved-
// has-resolver is not in AC-4's archive-skip list and still fires
// on archive entities.
func TestBuildStatus_Warnings(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{
				Kind: entity.KindGap, ID: "G-0001",
				Title: "Half-resolved", Status: "addressed",
				Path: "work/gaps/archive/G-0001-half-resolved.md",
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
	if w.EntityID != "G-0001" {
		t.Errorf("Warnings[0].EntityID = %q, want G-0001", w.EntityID)
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
			{Code: "gap-resolved-has-resolver", EntityID: "G-0001", Message: "gap is marked addressed but addressed_by is empty"},
		},
		Health: statusHealthCounts{Entities: 1, Warnings: 1},
	}
	out := captureStdout(t, func() {
		if err := renderStatusText(os.Stdout, &r, 0, false); err != nil {
			t.Fatalf("renderStatusText: %v", err)
		}
	})
	got := string(out)

	for _, want := range []string{
		"Warnings",
		"gap-resolved-has-resolver",
		"[G-0001]",
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
			ID: "E-0001", Title: "Active epic", Status: "active",
			Milestones: []statusMilestone{
				{ID: "M-0001", Title: "First", Status: "done"},
				{ID: "M-0002", Title: "Second", Status: "in_progress"},
			},
		}},
		PlannedEpics: []statusEpic{{
			ID: "E-0002", Title: "Planned epic", Status: "proposed",
			Milestones: []statusMilestone{
				{ID: "M-0010", Title: "Setup", Status: "draft"},
			},
		}},
		OpenDecisions: []statusEntity{
			{ID: "ADR-0001", Title: "Adopt X", Status: "proposed", Kind: "adr"},
		},
		OpenGaps: []statusGap{
			{ID: "G-0001", Title: "Edge case", DiscoveredIn: "M-0002"},
		},
		Warnings: []statusFinding{
			{Code: "gap-resolved-has-resolver", EntityID: "G-0002", Message: "gap is marked addressed but addressed_by is empty"},
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
		"### E-0001 — Active epic _(active)_",
		"```mermaid\nflowchart LR",
		"E_0001[\"E-0001<br/>Active epic\"]:::epic_active",
		"M_0002[\"M-0002<br/>Second\"]:::ms_in_progress",
		"E_0001 --> M_0001",
		"## Roadmap",
		"### E-0002 — Planned epic _(proposed)_",
		"E_0002[\"E-0002<br/>Planned epic\"]:::epic_proposed",
		"## Open decisions",
		"| ADR-0001 | adr | Adopt X | proposed |",
		"## Open gaps",
		"| G-0001 | Edge case | M-0002 |",
		"## Warnings",
		"| gap-resolved-has-resolver | G-0002 |  | gap is marked addressed but addressed_by is empty |",
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
// TestSummarizeACs covers the per-status counter and the InScope
// derivation (Total minus Cancelled).
func TestSummarizeACs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		acs  []entity.AcceptanceCriterion
		want *statusACProgress
	}{
		{
			name: "nil/empty returns nil",
			acs:  nil,
			want: nil,
		},
		{
			name: "single open",
			acs:  []entity.AcceptanceCriterion{{Status: "open"}},
			want: &statusACProgress{Total: 1, InScope: 1, Open: 1},
		},
		{
			name: "mixed with cancelled",
			acs: []entity.AcceptanceCriterion{
				{Status: "open"},
				{Status: "met"},
				{Status: "deferred"},
				{Status: "cancelled"},
			},
			want: &statusACProgress{Total: 4, InScope: 3, Open: 1, Met: 1, Deferred: 1, Cancelled: 1},
		},
		{
			name: "all cancelled",
			acs: []entity.AcceptanceCriterion{
				{Status: "cancelled"},
				{Status: "cancelled"},
			},
			want: &statusACProgress{Total: 2, InScope: 0, Cancelled: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeACs(tt.acs)
			switch {
			case got == nil && tt.want == nil:
				return
			case got == nil || tt.want == nil:
				t.Fatalf("got %+v, want %+v", got, tt.want)
			case *got != *tt.want:
				t.Errorf("got %+v, want %+v", *got, *tt.want)
			}
		})
	}
}

// TestRenderACProgress covers the badge text rendered next to each
// milestone in the status output.
func TestRenderACProgress(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		p    *statusACProgress
		want string
	}{
		{"nil", nil, ""},
		{"all met", &statusACProgress{Total: 2, InScope: 2, Met: 2}, "ACs 2/2 met"},
		{"in progress", &statusACProgress{Total: 3, InScope: 3, Open: 1, Met: 2}, "ACs 2/3 met (1 open)"},
		{"all cancelled", &statusACProgress{Total: 2, InScope: 0, Cancelled: 2}, "ACs all cancelled"},
		{"with deferred", &statusACProgress{Total: 3, InScope: 3, Met: 2, Deferred: 1}, "ACs 2/3 met"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderACProgress(tt.p); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderStatusText_ACProgressInline confirms the AC progress
// badge appears on the milestone row in the text renderer.
func TestRenderStatusText_ACProgressInline(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Foo", Status: "active"},
			{
				Kind: entity.KindMilestone, ID: "M-0001", Title: "First", Status: "in_progress", Parent: "E-0001",
				TDD: "required",
				ACs: []entity.AcceptanceCriterion{
					{ID: "AC-1", Title: "x", Status: "met", TDDPhase: "done"},
					{ID: "AC-2", Title: "y", Status: "open", TDDPhase: "red"},
				},
			},
		},
	}
	r := buildStatus(tr, nil)
	var b strings.Builder
	if err := renderStatusText(&b, &r, 0, false); err != nil {
		t.Fatalf("renderStatusText: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "ACs 1/2 met (1 open)") {
		t.Errorf("text output missing AC progress badge:\n%s", out)
	}
	if !strings.Contains(out, "tdd: required") {
		t.Errorf("text output missing tdd: badge:\n%s", out)
	}
}

// TestRenderStatusMarkdown_MermaidACBadge confirms the (M/T) badge
// appears in the mermaid milestone label, and the bullet row carries
// the inline progress text.
func TestRenderStatusMarkdown_MermaidACBadge(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Foo", Status: "active"},
			{
				Kind: entity.KindMilestone, ID: "M-0007", Title: "Engine", Status: "in_progress", Parent: "E-0001",
				ACs: []entity.AcceptanceCriterion{
					{ID: "AC-1", Status: "met"},
					{ID: "AC-2", Status: "met"},
					{ID: "AC-3", Status: "open"},
				},
			},
		},
	}
	r := buildStatus(tr, nil)
	var b strings.Builder
	if err := renderStatusMarkdown(&b, &r); err != nil {
		t.Fatalf("renderStatusMarkdown: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "ACs 2/3 met (1 open)") {
		t.Errorf("markdown bullet missing AC progress:\n%s", out)
	}
	if !strings.Contains(out, "M-0007 (2/3)") {
		t.Errorf("mermaid label missing (2/3) badge:\n%s", out)
	}
}

func TestMdEscape(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	rc := run([]string{"status", "--format=xml"})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want cliutil.ExitUsage (%d)", rc, cliutil.ExitUsage)
	}
}

// TestReadRecentActivity_SkipsProseMentions covers G30: a hand-
// authored commit whose body wraps such that a line starts with
// `aiwf-verb:` (a prose mention of the trailer name) must NOT
// appear in `aiwf status` Recent activity. Pre-fix the `--grep`
// matched the wrapped line and the row landed with empty Verb /
// Actor columns; the fix post-filters on the parsed-trailer column
// (Git's structured trailer parser correctly finds no trailer).
func TestReadRecentActivity_SkipsProseMentions(t *testing.T) {
	t.Parallel()
	root := setupGitRepoWithUpstream(t, "peter@example.com")
	// Real trailered commit — must show up.
	realMsg := "feat(aiwf): add a thing\n\n" +
		"aiwf-verb: add\n" +
		"aiwf-entity: G-001\n" +
		"aiwf-actor: human/peter\n"
	if out, err := runGit(root, "commit", "--allow-empty", "-m", realMsg); err != nil {
		t.Fatalf("git commit (real): %v\n%s", err, out)
	}
	// Prose mention — wrap puts `aiwf-verb:` at the start of a body
	// line, but it's mid-sentence prose. The naïve --grep matched;
	// Git's trailer parser correctly does not.
	proseMsg := "docs(aiwf): note about trailers\n\n" +
		"This commit folds the audit-trail manual-commit gap (no\n" +
		"aiwf-verb: trailers) into the followup discussion.\n"
	if out, err := runGit(root, "commit", "--allow-empty", "-m", proseMsg); err != nil {
		t.Fatalf("git commit (prose): %v\n%s", err, out)
	}

	events, err := readRecentActivity(context.Background(), root, 10)
	if err != nil {
		t.Fatalf("readRecentActivity: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (prose-mention should be filtered); got %+v", len(events), events)
	}
	if events[0].Verb != "add" || events[0].Actor != "human/peter" {
		t.Errorf("kept event: verb=%q actor=%q (want add/human/peter)", events[0].Verb, events[0].Actor)
	}
}

// TestReadHistory_SkipsProseMentions: same prose-mention class
// applied to `aiwf history <id>`. A wrapped body line starting
// with `aiwf-entity: G-001` must not render as an event for G-001.
func TestReadHistory_SkipsProseMentions(t *testing.T) {
	t.Parallel()
	root := setupGitRepoWithUpstream(t, "peter@example.com")
	realMsg := "feat(aiwf): real add\n\n" +
		"aiwf-verb: add\n" +
		"aiwf-entity: G-001\n" +
		"aiwf-actor: human/peter\n"
	if out, err := runGit(root, "commit", "--allow-empty", "-m", realMsg); err != nil {
		t.Fatalf("git commit (real): %v\n%s", err, out)
	}
	proseMsg := "docs: prose mention\n\n" +
		"refer to the previous note about\n" +
		"aiwf-entity: G-001 in the dispatcher.\n"
	if out, err := runGit(root, "commit", "--allow-empty", "-m", proseMsg); err != nil {
		t.Fatalf("git commit (prose): %v\n%s", err, out)
	}

	events, err := readHistory(context.Background(), root, "G-0001")
	if err != nil {
		t.Fatalf("readHistory: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (prose-mention should be filtered); got %+v", len(events), events)
	}
	if events[0].Verb != "add" || events[0].Actor != "human/peter" {
		t.Errorf("kept event: verb=%q actor=%q", events[0].Verb, events[0].Actor)
	}
}
