package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// TestRun_ShowMilestoneAggregatesACsHistoryFindings exercises the
// full top-level path: a milestone with two ACs, a TDD phase walk,
// and a status promotion. The text output must contain the header,
// both AC rows, the recent-history block, and the no-findings line.
func TestRun_ShowMilestoneAggregatesACsHistoryFindings(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-01", "--title", "Engine warning", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-001", "--title", "AC one"}); rc != exitOK {
		t.Fatalf("add ac 1: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-001", "--title", "AC two"}); rc != exitOK {
		t.Fatalf("add ac 2: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "M-001/AC-1", "met"}); rc != exitOK {
		t.Fatalf("promote: %d", rc)
	}

	out := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "M-001"}); rc != exitOK {
			t.Fatalf("show: %d", rc)
		}
	})
	s := string(out)
	for _, want := range []string{
		"M-001 · Engine warning · status: draft",
		"parent: E-01",
		"ACs:",
		"AC-1 [met]",
		"AC-2 [open]",
		"Recent history",
		"Findings: (none)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("show output missing %q in:\n%s", want, s)
		}
	}
}

// TestRun_ShowCompositeIDRendersACSlice: querying a composite id
// renders just that AC plus its history, with the AC's parent
// milestone shown as "parent: M-NNN".
func TestRun_ShowCompositeIDRendersACSlice(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-01", "--title", "First", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-001", "--title", "Just one"}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "M-001/AC-1", "--phase", "red"}); rc != exitOK {
		t.Fatalf("promote phase: %d", rc)
	}

	out := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "M-001/AC-1"}); rc != exitOK {
			t.Fatalf("show: %d", rc)
		}
	})
	s := string(out)
	for _, want := range []string{
		"M-001/AC-1",
		`"Just one"`,
		"status: open",
		"phase: red",
		"parent: M-001",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("composite show output missing %q in:\n%s", want, s)
		}
	}
}

// TestRun_ShowJSONEnvelope confirms --format=json emits a structured
// envelope with the right shape.
func TestRun_ShowJSONEnvelope(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"show", "--root", root, "--format=json", "E-01"}); rc != exitOK {
			t.Fatalf("show: %d", rc)
		}
	})
	var env struct {
		Tool   string `json:"tool"`
		Status string `json:"status"`
		Result struct {
			ID     string `json:"id"`
			Kind   string `json:"kind"`
			Status string `json:"status"`
			Title  string `json:"title"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, captured)
	}
	if env.Tool != "aiwf" || env.Status != "ok" {
		t.Errorf("envelope tool/status = %q/%q", env.Tool, env.Status)
	}
	if env.Result.ID != "E-01" || env.Result.Kind != "epic" {
		t.Errorf("result.id/kind = %q/%q", env.Result.ID, env.Result.Kind)
	}
}

// TestRun_ShowUnknownIDIsUsageError surfaces a clean error and
// usage exit code.
func TestRun_ShowUnknownIDIsUsageError(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"show", "--root", root, "E-99"}); rc != exitUsage {
		t.Errorf("expected exitUsage, got %d", rc)
	}
}

// TestRun_ShowFindingsScopedToEntity: when the entity has a real
// finding, show surfaces it. The standing check
// `milestone-done-incomplete-acs` catches the inconsistent state on
// every check pass — even when the file landed via a hand-edit
// rather than the verb path. This is the load-bearing reason that
// finding runs at check time, not just at verb-projection time.
//
// We can't get a milestone into status: done with an open AC via the
// verb path (the projection check that becomes the standing finding
// also blocks the verb), so the test hand-edits the file on disk
// and commits — exactly the scenario the standing check exists to
// catch.
func TestRun_ShowFindingsScopedToEntity(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-01", "--title", "Done milestone", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-001", "--title", "Open AC"}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}

	// Hand-edit the milestone to status: done while AC-1 is still
	// open — the inconsistent state the standing check exists to
	// catch. The verb path would refuse this; that's the point.
	mPath := filepath.Join(root, "work", "epics", "E-01-foo", "M-001-done-milestone.md")
	raw, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	patched := strings.Replace(string(raw), "status: draft", "status: done", 1)
	if writeErr := os.WriteFile(mPath, []byte(patched), 0o644); writeErr != nil {
		t.Fatalf("write patched: %v", writeErr)
	}

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	view, ok := buildShowView(context.Background(), root, tr, nil, "M-001", 5)
	if !ok {
		t.Fatal("show view missing")
	}
	if len(view.Findings) == 0 {
		t.Fatal("expected milestone-done-incomplete-acs finding")
	}
	found := false
	for _, f := range view.Findings {
		if f.Code == "milestone-done-incomplete-acs" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected milestone-done-incomplete-acs in findings; got %+v", view.Findings)
	}
}
