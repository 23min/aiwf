package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/status"
)

// groupedStatusReport builds a StatusReport with two declared areas plus
// untagged epics, AreaMembers set so the renderers group.
func groupedStatusReport() status.StatusReport {
	return status.StatusReport{
		Date: "2026-06-23",
		InFlightEpics: []status.StatusEpic{
			{ID: "E-0001", Title: "Platform epic", Status: "active", Area: "platform"},
			{ID: "E-0002", Title: "Billing epic", Status: "active", Area: "billing"},
			{ID: "E-0003", Title: "Untagged epic", Status: "active", Area: ""},
		},
		PlannedEpics: []status.StatusEpic{
			{ID: "E-0004", Title: "Planned platform", Status: "proposed", Area: "platform"},
		},
		AreaMembers: []string{"platform", "billing"},
		AreaDefault: "Uncategorized",
	}
}

// indexBefore asserts a appears before b in s (structural ordering, not a
// bare substring presence check).
func indexBefore(t *testing.T, s, a, b string) {
	t.Helper()
	ia, ib := strings.Index(s, a), strings.Index(s, b)
	if ia < 0 {
		t.Errorf("missing %q in output:\n%s", a, s)
		return
	}
	if ib < 0 {
		t.Errorf("missing %q in output:\n%s", b, s)
		return
	}
	if ia >= ib {
		t.Errorf("expected %q before %q (got %d >= %d):\n%s", a, b, ia, ib, s)
	}
}

// TestRenderStatusText_GroupsByArea pins M-0175/AC-2 (text): when an areas
// block is declared, the In-flight and Roadmap epic sections are grouped
// into per-area subsections in members order, with each epic under its
// area heading.
func TestRenderStatusText_GroupsByArea(t *testing.T) {
	t.Parallel()
	r := groupedStatusReport()
	var buf bytes.Buffer
	if err := status.RenderStatusText(&buf, &r, 0, false); err != nil {
		t.Fatalf("RenderStatusText: %v", err)
	}
	out := buf.String()

	// In flight: platform heading → E-0001 → billing heading → E-0002 →
	// Uncategorized complement → E-0003 (untagged).
	indexBefore(t, out, "platform", "E-0001")
	indexBefore(t, out, "E-0001", "billing")
	indexBefore(t, out, "billing", "E-0002")
	indexBefore(t, out, "E-0002", "Uncategorized")
	indexBefore(t, out, "Uncategorized", "E-0003")
}

// TestRenderStatusText_FlatWithoutAreas pins M-0175/AC-6 (status text):
// with no areas block (AreaMembers empty), output is the flat,
// pre-grouping rendering — no area headings, no default-complement label.
func TestRenderStatusText_FlatWithoutAreas(t *testing.T) {
	t.Parallel()
	r := groupedStatusReport()
	r.AreaMembers = nil // no areas block
	r.AreaDefault = ""
	var buf bytes.Buffer
	if err := status.RenderStatusText(&buf, &r, 0, false); err != nil {
		t.Fatalf("RenderStatusText: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "Uncategorized") {
		t.Errorf("flat output must not carry the default-complement label:\n%s", out)
	}
	for _, id := range []string{"E-0001", "E-0002", "E-0003", "E-0004"} {
		if !strings.Contains(out, id) {
			t.Errorf("flat output missing epic %s:\n%s", id, out)
		}
	}
}

// TestRenderStatusMarkdown_GroupsByArea pins M-0175/AC-2 (markdown): the
// In-flight / Roadmap epic sections gain per-area subheadings when areas
// are declared.
func TestRenderStatusMarkdown_GroupsByArea(t *testing.T) {
	t.Parallel()
	r := groupedStatusReport()
	var buf bytes.Buffer
	if err := status.RenderStatusMarkdown(&buf, &r); err != nil {
		t.Fatalf("RenderStatusMarkdown: %v", err)
	}
	out := buf.String()
	indexBefore(t, out, "platform", "E-0001")
	indexBefore(t, out, "E-0001", "billing")
	indexBefore(t, out, "Uncategorized", "E-0003")
}

// TestRenderStatus_EmptyComplementAlwaysShown pins M-0175/AC-5 in the
// status renderer: when every in-flight epic is tagged, the complement
// section is still rendered (empty), and a declared area with no epics is
// suppressed.
func TestRenderStatus_EmptyComplementAlwaysShown(t *testing.T) {
	t.Parallel()
	r := status.StatusReport{
		Date: "2026-06-23",
		InFlightEpics: []status.StatusEpic{
			{ID: "E-0001", Title: "Platform epic", Status: "active", Area: "platform"},
		},
		AreaMembers: []string{"platform", "billing"}, // billing has no epics
		AreaDefault: "Uncategorized",
	}
	var buf bytes.Buffer
	if err := status.RenderStatusText(&buf, &r, 0, false); err != nil {
		t.Fatalf("RenderStatusText: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Uncategorized") {
		t.Errorf("complement must always render even when empty:\n%s", out)
	}
	// billing (declared, zero epics) is suppressed — its area heading must
	// not appear. Anchor on the "▸ " heading marker so the check is
	// heading-specific (not a bare substring that an epic title could trip).
	if strings.Contains(out, "▸ billing") {
		t.Errorf("empty declared area 'billing' must be suppressed:\n%s", out)
	}
}

// TestRunRoadmap_GroupsByAreaViaDispatcher pins M-0175/AC-3 through the
// dispatcher: `aiwf render roadmap` on a repo with an areas block emits
// per-area sections (the RunRoadmap grouped arm), and renders flat when
// no areas block exists (AC-6).
func TestRunRoadmap_GroupsByAreaViaDispatcher(t *testing.T) {
	t.Run("grouped when areas declared", func(t *testing.T) {
		root := setupAreaRepo(t)
		mustRun(t, "add", "epic", "--title", "Platform work", "--area", "platform", "--actor", "human/test", "--root", root)
		mustRun(t, "add", "epic", "--title", "Untagged work", "--actor", "human/test", "--root", root)
		captured := testutil.CaptureStdout(t, func() {
			mustRun(t, "render", "roadmap", "--root", root)
		})
		out := string(captured)
		if !strings.Contains(out, "## platform") {
			t.Errorf("roadmap should carry a platform area section:\n%s", out)
		}
		if !strings.Contains(out, "### E-0001") {
			t.Errorf("grouped roadmap epic should be demoted to h3:\n%s", out)
		}
		if !strings.Contains(out, "## Uncategorized") {
			t.Errorf("roadmap should carry the untagged complement section:\n%s", out)
		}
	})

	t.Run("flat when no areas block", func(t *testing.T) {
		root := setupCLITestRepo(t)
		mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
		mustRun(t, "add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root)
		captured := testutil.CaptureStdout(t, func() {
			mustRun(t, "render", "roadmap", "--root", root)
		})
		out := string(captured)
		if strings.Contains(out, "## Uncategorized") {
			t.Errorf("no areas block -> roadmap must not carry area sections:\n%s", out)
		}
		if !strings.Contains(out, "## E-0001 — Foundations") {
			t.Errorf("flat roadmap should render the epic at h2:\n%s", out)
		}
	})
}
