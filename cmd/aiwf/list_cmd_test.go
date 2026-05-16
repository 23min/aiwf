package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
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
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "active"}); rc != exitOK {
		t.Fatalf("promote E-01 active: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "M one", "--tdd", "none", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone M-001: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-0002", "--title", "M two", "--tdd", "advisory", "--actor", "human/test", "--root", root}); rc != exitOK {
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
		if !strings.Contains(s, "M-0001") || !strings.Contains(s, "M-0002") {
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
		if !strings.Contains(s, "E-0001") {
			t.Errorf("--status active missing E-01 (the only active epic):\n%s", s)
		}
		if strings.Contains(s, "E-0002") {
			t.Errorf("--status active leaked the proposed epic E-02:\n%s", s)
		}
	})

	t.Run("--parent scopes to children of an epic", func(t *testing.T) {
		var rc int
		out := captureStdout(t, func() {
			rc = run([]string{"list", "--kind", "milestone", "--parent", "E-0001", "--root", root})
		})
		if rc != exitOK {
			t.Fatalf("rc = %d, want exitOK", rc)
		}
		s := string(out)
		if !strings.Contains(s, "M-0001") {
			t.Errorf("--parent E-01 missing M-001:\n%s", s)
		}
		if strings.Contains(s, "M-0002") {
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
		if ids[0] != "M-0001" || ids[1] != "M-0002" {
			t.Errorf("envelope.result ids (id-ascending) = %v, want [M-001 M-002]", ids)
		}
		if envelope.Result[0].Parent != "E-0001" || envelope.Result[1].Parent != "E-0002" {
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

// TestRun_List_JSONResultIsArrayOfSummaryObjects is M-072 AC-2: the
// envelope's `result` is an array whose elements carry the documented
// six-field summary shape {id, kind, status, title, parent, path}.
// Stricter than the AC-1 JSON subtest, which only asserts id and
// parent — a regression that drops `kind`, `status`, `title`, or
// `path` from the Summary struct silently passes there but fails here.
func TestRun_List_JSONResultIsArrayOfSummaryObjects(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"add", "epic", "--title", "Active epic", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic E-01: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "active"}); rc != exitOK {
		t.Fatalf("promote E-01: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "M one", "--tdd", "none", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}

	out := captureStdout(t, func() {
		if rc := run([]string{"list", "--kind", "milestone", "--format=json", "--root", root}); rc != exitOK {
			t.Fatalf("list rc != exitOK")
		}
	})

	var envelope struct {
		Result []map[string]any `json:"result"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatalf("json unmarshal: %v\nraw:\n%s", err, out)
	}
	if len(envelope.Result) != 1 {
		t.Fatalf("expected 1 milestone row, got %d:\n%s", len(envelope.Result), out)
	}

	row := envelope.Result[0]

	// Every documented field is present and string-typed. Pin the
	// expected values for the load-bearing ones (id, kind, status,
	// parent) and assert non-emptiness for the descriptive ones (title,
	// path) so a future renumbering of the slug doesn't churn the test.
	wantStrings := map[string]string{
		"id":     "M-0001",
		"kind":   "milestone",
		"status": "draft",
		"parent": "E-0001",
	}
	for k, want := range wantStrings {
		got, ok := row[k].(string)
		if !ok {
			t.Errorf("field %q missing or not a string in row: %#v", k, row)
			continue
		}
		if got != want {
			t.Errorf("field %q = %q, want %q", k, got, want)
		}
	}
	for _, k := range []string{"title", "path"} {
		got, ok := row[k].(string)
		if !ok || got == "" {
			t.Errorf("field %q missing, empty, or not a string in row: %#v", k, row)
		}
	}

	// `path` must point at the milestone file under work/ — proof that
	// the loader-set Entity.Path made it into the Summary verbatim.
	if path, _ := row["path"].(string); !strings.HasSuffix(path, ".md") {
		t.Errorf("path %q does not end in .md", path)
	}
	if path, _ := row["path"].(string); !strings.Contains(path, "M-0001") {
		t.Errorf("path %q does not name M-001", path)
	}
}

// TestRun_List_ArchivedFlag is M-072 AC-3: the default filter excludes
// entities whose status is terminal under their kind's FSM; passing
// --archived widens to include them. Pre-ADR-0004 the behavior is
// driven by entity.IsTerminal; post-ADR-0004 the same flag walks
// archive/ subdirs without a list-side change.
//
// Fixture: one active epic with two milestones — one in_progress
// (non-terminal), one cancelled (terminal). The default invocation
// must surface only the in_progress one; --archived must surface both.
func TestRun_List_ArchivedFlag(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"add", "epic", "--title", "Active epic", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "active"}); rc != exitOK {
		t.Fatalf("promote epic active: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "Live", "--tdd", "none", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add M-001: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "Doomed", "--tdd", "none", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add M-002: %d", rc)
	}
	if rc := run([]string{"promote", "--actor", "human/test", "--root", root, "M-0001", "in_progress"}); rc != exitOK {
		t.Fatalf("promote M-001 in_progress: %d", rc)
	}
	if rc := run([]string{"cancel", "--actor", "human/test", "--root", root, "M-0002"}); rc != exitOK {
		t.Fatalf("cancel M-002: %d", rc)
	}

	t.Run("default excludes terminal-status entities", func(t *testing.T) {
		out := captureStdout(t, func() {
			if rc := run([]string{"list", "--kind", "milestone", "--root", root}); rc != exitOK {
				t.Fatalf("list rc != exitOK")
			}
		})
		s := string(out)
		if !strings.Contains(s, "M-0001") {
			t.Errorf("default list missing in_progress milestone M-001:\n%s", s)
		}
		if strings.Contains(s, "M-0002") {
			t.Errorf("default list leaked cancelled milestone M-002:\n%s", s)
		}
	})

	t.Run("--archived includes terminal-status entities", func(t *testing.T) {
		out := captureStdout(t, func() {
			if rc := run([]string{"list", "--kind", "milestone", "--archived", "--root", root}); rc != exitOK {
				t.Fatalf("list --archived rc != exitOK")
			}
		})
		s := string(out)
		if !strings.Contains(s, "M-0001") {
			t.Errorf("--archived list missing M-001:\n%s", s)
		}
		if !strings.Contains(s, "M-0002") {
			t.Errorf("--archived list missing the cancelled M-002 (the entire point of the flag):\n%s", s)
		}
	})

	t.Run("no-args counts exclude terminal entities", func(t *testing.T) {
		out := captureStdout(t, func() {
			if rc := run([]string{"list", "--root", root}); rc != exitOK {
				t.Fatalf("no-args rc != exitOK")
			}
		})
		s := string(out)
		// One non-terminal milestone (M-001); the cancelled M-002 is
		// excluded from the count.
		if !strings.Contains(s, "1 milestone") {
			t.Errorf("no-args output missing `1 milestone` (the cancelled one should not count):\n%s", s)
		}
	})
}

// TestRun_List_BadFormat covers the format-validation usage-error path
// (`--format=xml` and other unsupported values). Mirrors
// TestRunStatus_BadFormat for the same closed-set discipline.
func TestRun_List_BadFormat(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"list", "--root", root, "--format=xml"}); rc != exitUsage {
		t.Errorf("rc = %d, want exitUsage (%d)", rc, exitUsage)
	}
}

// TestRun_List_BadKind covers the --kind validation usage-error path.
// A value outside entity.AllKinds() must not cause a tree walk; the
// verb returns exitUsage before loading anything.
func TestRun_List_BadKind(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"list", "--root", root, "--kind", "milestoneish"}); rc != exitUsage {
		t.Errorf("rc = %d, want exitUsage (%d)", rc, exitUsage)
	}
}

// TestSeam_ListAndStatusAgreeOnOpenGaps is M-072 AC-6's chokepoint:
// `aiwf list --kind gap --status open` and the *Open gaps* slice
// produced by `buildStatus` must agree on the same fixture tree.
// Both routes through tree.FilterByKindStatuses; if a future change
// re-introduces parallel filter logic in either site, the agreement
// breaks here even when each verb's own tests still pass.
func TestSeam_ListAndStatusAgreeOnOpenGaps(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{Entities: []*entity.Entity{
		{Kind: entity.KindGap, ID: "G-0001", Status: "open", Title: "open one"},
		{Kind: entity.KindGap, ID: "G-0002", Status: "addressed", Title: "addressed (terminal)"},
		{Kind: entity.KindGap, ID: "G-0003", Status: "open", Title: "open two", DiscoveredIn: "M-0007"},
		{Kind: entity.KindGap, ID: "G-0004", Status: "wontfix", Title: "wontfix (terminal)"},
		// Non-gap noise that must not leak into either result.
		{Kind: entity.KindEpic, ID: "E-0001", Status: "active"},
		{Kind: entity.KindMilestone, ID: "M-0001", Status: "draft", Parent: "E-0001"},
	}}

	listIDs := make([]string, 0)
	for _, r := range buildListRows(tr, "gap", "open", "", false) {
		listIDs = append(listIDs, r.ID)
	}

	report := buildStatus(tr, nil)
	statusIDs := make([]string, 0, len(report.OpenGaps))
	for _, g := range report.OpenGaps {
		statusIDs = append(statusIDs, g.ID)
	}

	if diff := cmp.Diff(statusIDs, listIDs); diff != "" {
		t.Errorf("list and status disagree on open gaps (-status +list):\n%s\n\nlist=%v status=%v",
			diff, listIDs, statusIDs)
	}
	// And both must equal the documented expected set; otherwise both
	// could agree on the wrong answer.
	want := []string{"G-0001", "G-0003"}
	if diff := cmp.Diff(want, listIDs); diff != "" {
		t.Errorf("agreed result is wrong (-want +got):\n%s", diff)
	}
}

// TestNewListCmd_CompletionWiring is M-072 AC-5: --kind and --status
// have closed-set completion functions bound, returning the canonical
// kind list and (kind-aware) status list. The completion-drift policy
// (TestPolicy_FlagsHaveCompletion) catches "wired or opt-out", but a
// regression that wires a completion func returning the wrong values
// (e.g., the empty slice, or a different closed set) compiles, passes
// the drift test, and silently degrades the shell-completion UX. This
// test pins the closed-set semantics directly.
func TestNewListCmd_CompletionWiring(t *testing.T) {
	t.Parallel()
	cmd := newListCmd()

	t.Run("--kind returns entity.AllKinds", func(t *testing.T) {
		fn, ok := cmd.GetFlagCompletionFunc("kind")
		if !ok {
			t.Fatal("--kind has no completion function bound")
		}
		got, dir := fn(cmd, nil, "")
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", dir)
		}
		if diff := cmp.Diff(allKindNames(), got); diff != "" {
			t.Errorf("--kind completions mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("--status without --kind returns the de-duplicated union", func(t *testing.T) {
		fn, ok := cmd.GetFlagCompletionFunc("status")
		if !ok {
			t.Fatal("--status has no completion function bound")
		}
		got, dir := fn(cmd, nil, "")
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", dir)
		}
		if diff := cmp.Diff(unionAllStatuses(), got); diff != "" {
			t.Errorf("--status (no kind) completions mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("--status with --kind=milestone is kind-aware", func(t *testing.T) {
		// Same fresh command per subtest so flag state is isolated.
		c := newListCmd()
		if err := c.Flags().Set("kind", "milestone"); err != nil {
			t.Fatalf("set --kind: %v", err)
		}
		fn, ok := c.GetFlagCompletionFunc("status")
		if !ok {
			t.Fatal("--status has no completion function bound")
		}
		got, dir := fn(c, nil, "")
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", dir)
		}
		want := entity.AllowedStatuses(entity.KindMilestone)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("--status (kind=milestone) completions mismatch (-want +got):\n%s", diff)
		}
		// Cross-check: ADR statuses must NOT appear in the
		// kind=milestone completion set, proving the closure actually
		// branches on --kind rather than always returning the union.
		for _, s := range got {
			for _, adrOnly := range []string{"accepted", "superseded"} {
				if s == adrOnly {
					t.Errorf("ADR-only status %q leaked into kind=milestone completion: %v", adrOnly, got)
				}
			}
		}
	})
}

// TestUnionAllStatuses asserts the --status completion fallback
// returns the de-duplicated, sorted union of every kind's allowed
// statuses. Pure helper; unit-test only.
func TestUnionAllStatuses(t *testing.T) {
	t.Parallel()
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
