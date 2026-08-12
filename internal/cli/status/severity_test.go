package status

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/23min/aiwf/internal/tree"
)

// severityFixture plants an aiwf.yaml plus one entity file in a fresh
// root, then loads the tree. relPath and content are caller-supplied so
// each test picks the entity whose finding its own knob escalates.
func severityFixture(t *testing.T, config, relPath, content string) (*tree.Tree, []tree.LoadError) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	full := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(relPath), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
	tr, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	return tr, loadErrs
}

// draftEpicWithEmptyGoal is the shape `tdd.strict` exists to escalate:
// an epic — a kind with a draft phase, so entity-body-empty fires at
// warning by default — carrying one empty load-bearing section. A
// born-complete kind would be an error without the knob and prove
// nothing about it.
const draftEpicWithEmptyGoal = "---\nid: E-0001\ntitle: Probe epic\nstatus: proposed\n---\n\n" +
	"## Goal\n\n## Scope\n\nReal prose.\n\n## Out of scope\n\nReal prose.\n"

// terminalGap is a swept-pending entity: terminal status, still in the
// active tree, so CountPendingSweep counts it.
const terminalGap = "---\nid: G-0001\ntitle: Probe gap\nstatus: wontfix\n---\n\n" +
	"## What's missing\n\nReal prose.\n\n## Why it matters\n\nReal prose.\n"

// TestBuildStatus_CountsTheSeveritiesAiwfCheckWouldReport pins the
// read-surface half of G-0567's divergence: before the shared seam,
// `aiwf status` counted as warnings, on a tree with `tdd.strict: true`,
// the findings `aiwf check` reported as errors and exited 1 for.
//
// The Health line's whole job is to tell a reader whether `aiwf check`
// will pass, and it read the findings at their config-agnostic default
// severities — so it reported a clean tree the pre-push hook blocks.
func TestBuildStatus_CountsTheSeveritiesAiwfCheckWouldReport(t *testing.T) {
	t.Parallel()
	tr, loadErrs := severityFixture(t, "tdd:\n  strict: true\n", "work/epics/E-0001-probe-epic/epic.md", draftEpicWithEmptyGoal)
	report := BuildStatus(tr, loadErrs, time.Now())
	if report.Health.Errors == 0 {
		t.Errorf("health = %+v; want a non-zero error count — `tdd.strict` escalates the entity-body-empty finding this fixture carries, and `aiwf check` reports it at error severity",
			report.Health)
	}

	// Negative control. Without it a BuildStatus that escalated
	// unconditionally — ignoring the config entirely — would satisfy the
	// assertion above, and the report would be wrong for every consumer
	// that never set the knob.
	lax, laxErrs := severityFixture(t, "tdd:\n  strict: false\n", "work/epics/E-0001-probe-epic/epic.md", draftEpicWithEmptyGoal)
	laxReport := BuildStatus(lax, laxErrs, time.Now())
	if laxReport.Health.Errors != 0 {
		t.Errorf("health = %+v with the knob off; want zero errors — the escalation is the operator's declared policy, not the report's own opinion",
			laxReport.Health)
	}
	if laxReport.Health.Warnings == 0 {
		t.Errorf("health = %+v with the knob off; want the finding still counted as a warning", laxReport.Health)
	}
}

// TestBuildStatus_SweepLineSurvivesItsOwnEscalation pins the display
// rule the escalation must not cost. ADR-0004 §"Display surfaces" puts
// the sweep-pending one-liner in the tree-health section; the sweep
// aggregate is exactly the finding `archive.sweep_threshold` raises to
// error, so lifting it only out of the warning stream would delete the
// line at the moment it matters most. The knob decides whether the
// aggregate blocks the push, not whether the reader is told about it.
func TestBuildStatus_SweepLineSurvivesItsOwnEscalation(t *testing.T) {
	t.Parallel()
	tr, loadErrs := severityFixture(t, "archive:\n  sweep_threshold: 0\n", "work/gaps/G-0001-probe-gap.md", terminalGap)
	report := BuildStatus(tr, loadErrs, time.Now())
	if report.SweepPending == nil {
		t.Fatal("SweepPending is nil; the tree-health sweep line disappeared once `archive.sweep_threshold` escalated the aggregate past warning severity")
	}
	if report.SweepPending.Count != 1 {
		t.Errorf("SweepPending.Count = %d, want 1", report.SweepPending.Count)
	}
	if report.Health.Errors == 0 {
		t.Errorf("health = %+v; want the escalated aggregate counted as an error", report.Health)
	}
}
