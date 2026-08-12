package show_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/show"
	"github.com/23min/aiwf/internal/tree"
)

// TestBuildShowView_ReportsTheSeveritiesAiwfCheckWouldReport pins the
// agreement an entity page owes the gate. Measured before the shared
// severity seam, on a tree with `tdd.strict: true` and one epic whose
// `## Goal` is empty:
//
//	aiwf check          error   entity-body-empty
//	aiwf show E-0001    warning entity-body-empty
//
// Two surfaces describing the same finding on the same entity, one of
// them wrong about whether the push is blocked.
//
// The epic kind matters: it has a draft phase, so entity-body-empty
// fires at warning by default and only `tdd.strict` raises it. A
// born-complete kind is an error with or without the knob and would
// pass this test against a seam that was never wired.
func TestBuildShowView_ReportsTheSeveritiesAiwfCheckWouldReport(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "aiwf.yaml"), "tdd:\n  strict: true\n")
	writeFile(t, filepath.Join(root, "work", "epics", "E-0001-probe-epic", "epic.md"),
		"---\nid: E-0001\ntitle: Probe epic\nstatus: proposed\n---\n\n"+
			"## Goal\n\n## Scope\n\nReal prose.\n\n## Out of scope\n\nReal prose.\n")

	tr, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	view, found, err := show.BuildShowView(context.Background(), root, tr, loadErrs, "E-0001", 5)
	if err != nil {
		t.Fatalf("BuildShowView: %v", err)
	}
	if !found {
		t.Fatal("E-0001 not found")
	}

	var seen bool
	for _, f := range view.Findings {
		if f.Code != check.CodeEntityBodyEmpty {
			continue
		}
		seen = true
		if f.Severity != check.SeverityError {
			t.Errorf("entity-body-empty severity = %q, want %q — the entity page reports its own severity rather than the one `aiwf check` applies under `tdd.strict`",
				f.Severity, check.SeverityError)
		}
	}
	if !seen {
		t.Fatalf("no entity-body-empty finding on the view; got %+v", view.Findings)
	}
}

func writeFile(t *testing.T, full, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}
