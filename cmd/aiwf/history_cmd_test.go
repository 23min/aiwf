package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestRun_HistoryShowsAddPromoteCancel exercises the full chain: init,
// add, promote, cancel, then history. The output should list three
// events for the entity, oldest-first.
func TestRun_HistoryShowsAddPromoteCancel(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "active"}); rc != exitOK {
		t.Fatalf("promote: %d", rc)
	}
	if rc := run([]string{"cancel", "--actor", "human/test", "--root", root, "E-0001"}); rc != exitOK {
		t.Fatalf("cancel: %d", rc)
	}

	events, err := readHistory(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatalf("readHistory: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3:\n%+v", len(events), events)
	}
	wantVerbs := []string{"add", "promote", "cancel"}
	for i, w := range wantVerbs {
		if events[i].Verb != w {
			t.Errorf("[%d] verb %q, want %q", i, events[i].Verb, w)
		}
		if events[i].Actor != "human/test" {
			t.Errorf("[%d] actor %q, want human/test", i, events[i].Actor)
		}
	}
}

// TestRun_HistoryJSON exercises the --format=json path. Capturing
// stdout requires redirecting os.Stdout for the duration of the call;
// we then parse the envelope and assert its shape.
func TestRun_HistoryJSON(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "active"}); rc != exitOK {
		t.Fatalf("promote: %d", rc)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"history", "--root", root, "--format=json", "E-0001"}); rc != exitOK {
			t.Fatalf("history: %d", rc)
		}
	})

	var env struct {
		Tool    string `json:"tool"`
		Status  string `json:"status"`
		Version string `json:"version"`
		Result  struct {
			ID     string         `json:"id"`
			Events []HistoryEvent `json:"events"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, captured)
	}
	if env.Tool != "aiwf" {
		t.Errorf("tool = %q", env.Tool)
	}
	if env.Status != "ok" {
		t.Errorf("status = %q", env.Status)
	}
	if env.Result.ID != "E-0001" {
		t.Errorf("result.id = %q", env.Result.ID)
	}
	if len(env.Result.Events) != 2 {
		t.Fatalf("events len = %d, want 2:\n%s", len(env.Result.Events), captured)
	}
	if env.Result.Events[0].Verb != "add" || env.Result.Events[1].Verb != "promote" {
		t.Errorf("verbs = [%q,%q], want [add,promote]",
			env.Result.Events[0].Verb, env.Result.Events[1].Verb)
	}
	for i, e := range env.Result.Events {
		if e.Date == "" || e.Actor == "" || e.Commit == "" {
			t.Errorf("event[%d] missing required field: %+v", i, e)
		}
	}
}

// TestRun_HistoryMilestonePrefixMatchesACs: querying the bare
// milestone id matches every commit whose aiwf-entity is the bare id
// OR M-NNN/AC-N (path-prefix anchored on `/`). The composite-id query
// matches only that AC.
func TestRun_HistoryMilestonePrefixMatchesACs(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "First", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	if rc := run([]string{"add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "AC one"}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "M-0001/AC-1", "met"}); rc != exitOK {
		t.Fatalf("promote AC: %d", rc)
	}

	// Bare milestone query matches both milestone and AC events.
	events, err := readHistory(context.Background(), root, "M-0001")
	if err != nil {
		t.Fatalf("readHistory M-001: %v", err)
	}
	// Expected events for the bare milestone query: 3 total — add of
	// M-001, add of M-001/AC-1, promote of M-001/AC-1.
	if len(events) != 3 {
		t.Errorf("M-001 history len = %d, want 3:\n%+v", len(events), events)
	}

	// Composite query matches only the AC events.
	events, err = readHistory(context.Background(), root, "M-0001/AC-1")
	if err != nil {
		t.Fatalf("readHistory M-001/AC-1: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("M-0001/AC-1 history len = %d, want 2 (add + promote):\n%+v", len(events), events)
	}
}

// TestRun_HistoryReadsAiwfToAndForce confirms readHistory pulls the
// I2 trailers (`aiwf-to:` and `aiwf-force:`) into HistoryEvent.To and
// .Force, and renders dashes / blanks for events that don't carry
// them. The mix of add (no aiwf-to), promote (with aiwf-to), and
// promote --force (with aiwf-to AND aiwf-force) covers the load-
// bearing field-projection paths.
func TestRun_HistoryReadsAiwfToAndForce(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "active"}); rc != exitOK {
		t.Fatalf("promote 1: %d", rc)
	}
	// Force-jump from active straight to cancelled — illegal for epics
	// (active→cancelled is legal so let's pick proposed→done... but
	// E-01 is now active). Force the FSM-illegal active→done jump
	// using a different epic to keep this test focused.
	if rc := run([]string{"add", "epic", "--title", "Bar", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add 2: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-0002", "done", "--force", "--reason", "sandbox emergency"}); rc != exitOK {
		t.Fatalf("forced promote: %d", rc)
	}

	// E-01: add (no to/force), promote → active (to=active, no force).
	events, err := readHistory(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatalf("readHistory E-01: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("E-01 got %d events, want 2:\n%+v", len(events), events)
	}
	if events[0].To != "" || events[0].Force != "" {
		t.Errorf("E-01 add event should have empty To/Force, got %+v", events[0])
	}
	if events[1].To != "active" {
		t.Errorf("E-01 promote To = %q, want active", events[1].To)
	}
	if events[1].Force != "" {
		t.Errorf("E-01 promote Force = %q, want empty", events[1].Force)
	}

	// E-02: add (no to/force), forced promote → done (to=done, force=reason).
	events, err = readHistory(context.Background(), root, "E-0002")
	if err != nil {
		t.Fatalf("readHistory E-02: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("E-02 got %d events, want 2:\n%+v", len(events), events)
	}
	if events[1].To != "done" {
		t.Errorf("E-02 forced promote To = %q, want done", events[1].To)
	}
	if events[1].Force != "sandbox emergency" {
		t.Errorf("E-02 forced promote Force = %q, want %q", events[1].Force, "sandbox emergency")
	}
}

// TestRun_HistoryRenderToDash confirms the text renderer produces a
// dash (`-`) for events without an `aiwf-to:` trailer, and `→ <to>`
// when one is present. This is the load-bearing backwards-compat
// rendering for pre-I2 commits.
func TestRun_HistoryRenderToDash(t *testing.T) {
	t.Parallel()
	tests := []struct {
		to   string
		want string
	}{
		{"", "-"},
		{"active", "→ active"},
		{"in_progress", "→ in_progress"},
		{"green", "→ green"},
	}
	for _, tt := range tests {
		t.Run(tt.to, func(t *testing.T) {
			if got := renderTo(tt.to); got != tt.want {
				t.Errorf("renderTo(%q) = %q, want %q", tt.to, got, tt.want)
			}
		})
	}
}

// TestRun_HistoryTextOutputIncludesForceLine: when an event has a
// force reason, the renderer emits a `[forced: <reason>]` line below
// the main row. Pinned via captured stdout.
func TestRun_HistoryTextOutputIncludesForceLine(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "done", "--force", "--reason", "policy override"}); rc != exitOK {
		t.Fatalf("forced promote: %d", rc)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"history", "--root", root, "E-0001"}); rc != exitOK {
			t.Fatalf("history: %d", rc)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "→ done") {
		t.Errorf("history text should contain `→ done` for forced promote; got:\n%s", out)
	}
	if !strings.Contains(out, "[forced: policy override]") {
		t.Errorf("history text should contain `[forced: policy override]`; got:\n%s", out)
	}
	// The add event has no aiwf-to — its column should render dash.
	if !strings.Contains(out, "-           ") && !strings.Contains(out, "-  ") {
		t.Errorf("history text should contain a dash for the add row's to-column; got:\n%s", out)
	}
}

// TestRun_HistoryUnknownIDIsEmpty: querying a never-allocated id
// returns no events and exits cleanly.
func TestRun_HistoryUnknownIDIsEmpty(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"history", "--root", root, "E-0099"}); rc != exitOK {
		t.Errorf("got %d, want %d", rc, exitOK)
	}
}

// TestRun_HistoryReallocateBridgesBothIDs verifies the
// `aiwf-prior-entity:` trailer is queryable: after reallocating, the
// old id still surfaces a final event.
func TestRun_HistoryReallocateBridgesBothIDs(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Bar", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add: %d", rc)
	}
	if rc := run([]string{"reallocate", "--actor", "human/test", "--root", root, "E-0001"}); rc != exitOK {
		t.Fatalf("reallocate: %d", rc)
	}

	// Old id sees the reallocate via aiwf-prior-entity.
	old, err := readHistory(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatal(err)
	}
	if len(old) < 2 {
		t.Fatalf("expected at least 2 events for E-01 (add + reallocate), got %d", len(old))
	}
	if old[len(old)-1].Verb != "reallocate" {
		t.Errorf("last event for E-01 verb = %q, want reallocate", old[len(old)-1].Verb)
	}

	// New id sees the reallocate via aiwf-entity.
	newH, err := readHistory(context.Background(), root, "E-0002")
	if err != nil {
		t.Fatal(err)
	}
	if len(newH) != 1 || newH[0].Verb != "reallocate" {
		t.Errorf("E-02 history = %+v, want one reallocate event", newH)
	}
}
