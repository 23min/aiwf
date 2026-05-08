package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/render"
)

// updateGolden lets a developer refresh the golden files via
// `go test -run TestRenderStatus_Goldens ./cmd/aiwf/ -update`. Without
// the flag, the test compares bytes and fails on drift — which is
// exactly what M-072 AC-7 ("status text and JSON goldens unchanged
// after refactor") demands.
var updateGolden = flag.Bool("update", false, "regenerate the status golden files")

// canonicalStatusReport returns a deterministic statusReport intended
// for golden-file rendering: hand-constructed entities covering every
// section (in-flight, planned, decisions, gaps, warnings, recent
// activity), fixed Date so the header doesn't drift with wall clock,
// no calls into the live tree.
//
// Synthetic content per CLAUDE.md test conventions — entity ids and
// titles read as obviously fictional, not anonymized copies.
func canonicalStatusReport() statusReport {
	return statusReport{
		Date: "2026-05-09",
		InFlightEpics: []statusEpic{{
			ID:     "E-99",
			Title:  "Goldenfix epic for refactor parity",
			Status: "active",
			Milestones: []statusMilestone{
				{ID: "M-901", Title: "Lay foundations", Status: "done"},
				{
					ID:     "M-902",
					Title:  "Wire the seam",
					Status: "in_progress",
					TDD:    "required",
					ACs: &statusACProgress{
						Total: 3, InScope: 3, Open: 1, Met: 2,
					},
				},
				{ID: "M-903", Title: "Polish edges", Status: "draft"},
			},
		}},
		PlannedEpics: []statusEpic{{
			ID:     "E-100",
			Title:  "Future work that hasn't started",
			Status: "proposed",
			Milestones: []statusMilestone{
				{ID: "M-910", Title: "First plan step", Status: "draft"},
			},
		}},
		OpenDecisions: []statusEntity{
			{ID: "ADR-0901", Title: "Adopt fictional convention", Status: "proposed", Kind: "adr"},
			{ID: "D-901", Title: "Pick approach A or B", Status: "proposed", Kind: "decision"},
		},
		OpenGaps: []statusGap{
			{ID: "G-901", Title: "Refactor leaves a seam", DiscoveredIn: "M-902"},
		},
		Warnings: []statusFinding{
			{
				Code:     "gap-resolved-has-resolver",
				EntityID: "G-902",
				Path:     "work/gaps/G-902-fictional-fix.md",
				Message:  "gap is marked addressed but addressed_by is empty",
			},
		},
		RecentActivity: []HistoryEvent{
			{
				Date:   "2026-05-09",
				Verb:   "promote",
				Detail: "aiwf promote M-902 in_progress",
				Actor:  "human/golden",
				Commit: "feedf00d",
				To:     "in_progress",
			},
			{
				Date:   "2026-05-08",
				Verb:   "add",
				Detail: "aiwf add milestone M-903",
				Actor:  "human/golden",
				Commit: "deadbeef",
			},
		},
		Health: statusHealthCounts{Entities: 11, Errors: 0, Warnings: 1},
	}
}

// TestRenderStatus_Goldens is M-072 AC-7's chokepoint: byte-equal
// goldens for status text and JSON output against a canonical
// in-memory report. The earlier TestBuildStatus_* and TestRenderStatus*
// suite asserts on substrings (and survived the AC-6 refactor); this
// test locks down the *exact* rendered bytes so a future change to
// any padding / wording / field order surfaces here too.
//
// CLAUDE.md §"Substring assertions are not structural assertions" —
// the substring tests prove "the literal exists somewhere"; the
// goldens prove "the literal exists in the right place, in the right
// order, with the right whitespace."
func TestRenderStatus_Goldens(t *testing.T) {
	r := canonicalStatusReport()

	t.Run("text", func(t *testing.T) {
		var buf bytes.Buffer
		if err := renderStatusText(&buf, &r); err != nil {
			t.Fatalf("renderStatusText: %v", err)
		}
		assertGolden(t, "status_text.golden", buf.Bytes())
	})

	t.Run("json", func(t *testing.T) {
		env := render.Envelope{
			Tool:    "aiwf",
			Version: "test-golden", // fixed so version drift doesn't churn the golden
			Status:  "ok",
			Result:  &r,
			Metadata: map[string]any{
				"root":     "/golden/fixture",
				"entities": r.Health.Entities,
			},
		}
		var buf bytes.Buffer
		if err := render.JSON(&buf, env, true); err != nil {
			t.Fatalf("render.JSON: %v", err)
		}
		assertGolden(t, "status_json.golden", buf.Bytes())
	})
}

// assertGolden compares got against the file at testdata/<name>. With
// the -update flag, writes got to the file (use to refresh after an
// intentional change). Without it, fails the test on any byte
// difference — that's the AC-7 contract.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\n(run with -update to create)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch for %s\n--- want %d bytes ---\n%s\n--- got %d bytes ---\n%s\n\n(run with -update to refresh after an intentional change)",
			path, len(want), want, len(got), got)
	}
}
