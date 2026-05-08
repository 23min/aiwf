package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TestRun_List_CoreFlagsEndToEnd is M-072 AC-1 + AC-9: the verb-level
// integration test that drives `run([]string{"list", ...})` and asserts
// the rendered output for the V1 core flag set: --kind, --status,
// --parent, --format=text|json, --pretty. The helper-only path is not
// sufficient (CLAUDE.md "test the seam" rule); this test fires the
// dispatcher so a future implementation that wires the flags wrongly
// fails here, not just at the helper layer.
//
// Pre-implementation this test fails with exitUsage because Cobra
// reports `aiwf list` as an unknown verb. The red phase landed here is
// what the green phase has to clear.
func TestRun_List_CoreFlagsEndToEnd(t *testing.T) {
	root := setupCLITestRepo(t)

	// Fixture: two epics, two milestones — one per epic — exercising
	// the kind, status, and parent dimensions of the V1 flag set.
	//
	// E-01 active, E-02 proposed; M-001 (parent E-01, tdd none),
	// M-002 (parent E-02, tdd advisory).
	if rc := run([]string{"add", "epic", "--title", "Active epic", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic E-01: %d", rc)
	}
	if rc := run([]string{"add", "epic", "--title", "Planned epic", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic E-02: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-01", "active"}); rc != exitOK {
		t.Fatalf("promote E-01 active: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-01", "--title", "M one", "--tdd", "none", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone M-001: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-02", "--title", "M two", "--tdd", "advisory", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone M-002: %d", rc)
	}

	t.Run("no-args prints per-kind counts", func(t *testing.T) {
		var rc int
		out := captureStdout(t, func() {
			rc = run([]string{"list", "--root", root})
		})
		if rc != exitOK {
			t.Fatalf("rc = %d, want exitOK", rc)
		}
		s := string(out)
		// Structural per-kind assertions: each fixture-created kind shows
		// its exact count alongside the kind name. A bare `Contains(s,
		// "2")` was too loose — the digit floats freely in path strings,
		// json offsets, etc., and `99 milestones · 12 epics` would have
		// passed even though the values are wrong.
		for _, want := range []string{"2 epics", "2 milestones"} {
			if !strings.Contains(s, want) {
				t.Errorf("no-args output missing %q:\n%s", want, s)
			}
		}
		// Non-fixture kinds are zero — pin them so a regression that
		// counts terminal-status entities into the no-args summary
		// surfaces here.
		for _, want := range []string{"0 ADRs", "0 gaps", "0 decisions", "0 contracts"} {
			if !strings.Contains(s, want) {
				t.Errorf("no-args output missing %q:\n%s", want, s)
			}
		}
	})

	t.Run("--kind milestone lists only milestones", func(t *testing.T) {
		var rc int
		out := captureStdout(t, func() {
			rc = run([]string{"list", "--kind", "milestone", "--root", root})
		})
		if rc != exitOK {
			t.Fatalf("rc = %d, want exitOK", rc)
		}
		s := string(out)
		if !strings.Contains(s, "M-001") || !strings.Contains(s, "M-002") {
			t.Errorf("--kind milestone missing M-001 or M-002:\n%s", s)
		}
		// Epic titles must not leak — they would only appear if epic
		// rows were emitted. Plain `E-01` substring isn't a valid
		// negative because milestone rows carry their parent in the
		// parent column.
		if strings.Contains(s, "Active epic") || strings.Contains(s, "Planned epic") {
			t.Errorf("--kind milestone leaked epic rows:\n%s", s)
		}
	})

	t.Run("--status active scopes by status", func(t *testing.T) {
		var rc int
		out := captureStdout(t, func() {
			rc = run([]string{"list", "--kind", "epic", "--status", "active", "--root", root})
		})
		if rc != exitOK {
			t.Fatalf("rc = %d, want exitOK", rc)
		}
		s := string(out)
		if !strings.Contains(s, "E-01") {
			t.Errorf("--status active missing E-01 (the only active epic):\n%s", s)
		}
		if strings.Contains(s, "E-02") {
			t.Errorf("--status active leaked the proposed epic E-02:\n%s", s)
		}
	})

	t.Run("--parent scopes to children of an epic", func(t *testing.T) {
		var rc int
		out := captureStdout(t, func() {
			rc = run([]string{"list", "--kind", "milestone", "--parent", "E-01", "--root", root})
		})
		if rc != exitOK {
			t.Fatalf("rc = %d, want exitOK", rc)
		}
		s := string(out)
		if !strings.Contains(s, "M-001") {
			t.Errorf("--parent E-01 missing M-001:\n%s", s)
		}
		if strings.Contains(s, "M-002") {
			t.Errorf("--parent E-01 leaked M-002 (whose parent is E-02):\n%s", s)
		}
	})

	t.Run("--format=json --pretty parses as a JSON envelope", func(t *testing.T) {
		var rc int
		out := captureStdout(t, func() {
			rc = run([]string{"list", "--kind", "milestone", "--format=json", "--pretty", "--root", root})
		})
		if rc != exitOK {
			t.Fatalf("rc = %d, want exitOK", rc)
		}
		var envelope struct {
			Tool   string `json:"tool"`
			Status string `json:"status"`
			Result []struct {
				ID     string `json:"id"`
				Kind   string `json:"kind"`
				Status string `json:"status"`
				Title  string `json:"title"`
				Parent string `json:"parent"`
				Path   string `json:"path"`
			} `json:"result"`
		}
		if err := json.Unmarshal(out, &envelope); err != nil {
			t.Fatalf("json unmarshal: %v\nraw output:\n%s", err, out)
		}
		if envelope.Tool != "aiwf" {
			t.Errorf("envelope.tool = %q, want %q", envelope.Tool, "aiwf")
		}
		if len(envelope.Result) != 2 {
			t.Fatalf("envelope.result length = %d, want 2 (M-001 and M-002):\n%s", len(envelope.Result), out)
		}
		ids := []string{envelope.Result[0].ID, envelope.Result[1].ID}
		if ids[0] != "M-001" || ids[1] != "M-002" {
			t.Errorf("envelope.result ids (id-ascending) = %v, want [M-001 M-002]", ids)
		}
		if envelope.Result[0].Parent != "E-01" || envelope.Result[1].Parent != "E-02" {
			t.Errorf("envelope.result parents = [%q %q], want [E-01 E-02]",
				envelope.Result[0].Parent, envelope.Result[1].Parent)
		}
		// --pretty asks for indented JSON; sanity-check that the
		// rendered output is multi-line.
		if !strings.Contains(string(out), "\n  ") {
			t.Errorf("--pretty did not produce indented output:\n%s", out)
		}
	})
}

// TestRun_List_BadFormat covers the format-validation usage-error path
// (`--format=xml` and other unsupported values). Mirrors
// TestRunStatus_BadFormat for the same closed-set discipline.
func TestRun_List_BadFormat(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"list", "--root", root, "--format=xml"}); rc != exitUsage {
		t.Errorf("rc = %d, want exitUsage (%d)", rc, exitUsage)
	}
}

// TestRun_List_BadKind covers the --kind validation usage-error path.
// A value outside entity.AllKinds() must not cause a tree walk; the
// verb returns exitUsage before loading anything.
func TestRun_List_BadKind(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"list", "--root", root, "--kind", "milestoneish"}); rc != exitUsage {
		t.Errorf("rc = %d, want exitUsage (%d)", rc, exitUsage)
	}
}

// TestUnionAllStatuses asserts the --status completion fallback
// returns the de-duplicated, sorted union of every kind's allowed
// statuses. Pure helper; unit-test only.
func TestUnionAllStatuses(t *testing.T) {
	got := unionAllStatuses()
	if len(got) == 0 {
		t.Fatalf("unionAllStatuses returned empty slice")
	}

	// De-dup invariant: every value appears at most once.
	seen := map[string]int{}
	for _, s := range got {
		seen[s]++
	}
	for s, n := range seen {
		if n > 1 {
			t.Errorf("status %q appears %d times; expected 1", s, n)
		}
	}

	// Sort invariant: result is sorted ascending.
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Errorf("not sorted at index %d: %q > %q", i, got[i-1], got[i])
		}
	}

	// Membership invariant: a representative per-kind status appears.
	// Picks one well-known status from each kind so a future kind that
	// drops one of these (or the helper that filters one out) surfaces
	// here.
	want := []string{
		"accepted",  // ADR
		"active",    // epic
		"addressed", // gap
		"draft",     // milestone
		"open",      // gap
		"proposed",  // multiple kinds
	}
	missing := []string{}
	for _, w := range want {
		found := false
		for _, s := range got {
			if s == w {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, w)
		}
	}
	if diff := cmp.Diff([]string(nil), missing, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("missing expected statuses (-want +got):\n%s\n\nfull union: %v", diff, got)
	}
}
