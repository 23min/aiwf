package integration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/list"
	"github.com/23min/aiwf/internal/tree"
)

// setupPriorityTree writes a planning tree into a fresh tempdir and loads
// it (no git — tree.Load reads files directly, keeping the priority-filter
// unit tests fast). Shape:
//
//	G-0001 gap, priority urgent
//	G-0002 gap, priority high
//	G-0003 gap, no priority (untagged)
//	D-0001 decision, priority urgent
//	D-0002 decision, no priority (untagged)
//	ADR-0001 (proposed — never carries priority)
//	E-0001 epic, active (never carries priority)
//	M-0001 milestone under E-0001 (never carries priority)
//
// Shared by the list, status, and show priority-filter unit tests
// (G-0078, E-0066, M-0263).
func setupPriorityTree(t *testing.T) (string, *tree.Tree) {
	t.Helper()
	root := t.TempDir()
	w := func(rel, content string) { mustWriteFile(t, filepath.Join(root, rel), content) }
	w("work/gaps/G-0001-urgent-leak.md", "---\nid: G-0001\ntitle: Urgent gap\nstatus: open\npriority: urgent\n---\n")
	w("work/gaps/G-0002-high-leak.md", "---\nid: G-0002\ntitle: High gap\nstatus: open\npriority: high\n---\n")
	w("work/gaps/G-0003-untagged.md", "---\nid: G-0003\ntitle: Untagged gap\nstatus: open\n---\n")
	w("work/decisions/D-0001-urgent-choice.md", "---\nid: D-0001\ntitle: Urgent decision\nstatus: proposed\npriority: urgent\n---\n")
	w("work/decisions/D-0002-untagged-choice.md", "---\nid: D-0002\ntitle: Untagged decision\nstatus: proposed\n---\n")
	w("docs/adr/ADR-0001-shape.md", "---\nid: ADR-0001\ntitle: An ADR\nstatus: proposed\n---\n")
	w("work/epics/E-0001-main/epic.md", "---\nid: E-0001\ntitle: Main epic\nstatus: active\n---\n")
	w("work/epics/E-0001-main/M-0001-cache.md", "---\nid: M-0001\ntitle: Main milestone\nstatus: in_progress\nparent: E-0001\n---\n")

	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	return root, tr
}

// listIDsByPriority returns the set of ids `aiwf list --priority level`
// yields, with archived=true so terminality never interferes — the test
// isolates the priority axis.
func listIDsByPriority(tr *tree.Tree, level string) map[string]bool {
	got := map[string]bool{}
	rows := list.BuildListRows(context.Background(), tr, "", "", "", "", level, true)
	for i := range rows {
		got[rows[i].ID] = true
	}
	return got
}

// TestBuildListRows_PriorityFilter pins M-0263/AC-1: `aiwf list --priority`
// returns exactly the gaps and decisions whose own priority field matches.
// An empty level applies no filter.
func TestBuildListRows_PriorityFilter(t *testing.T) {
	t.Parallel()
	_, tr := setupPriorityTree(t)

	urgent := listIDsByPriority(tr, "urgent")
	assertExactIDSet(t, "urgent", urgent, []string{"G-0001", "D-0001"})

	high := listIDsByPriority(tr, "high")
	assertExactIDSet(t, "high", high, []string{"G-0002"})

	all := listIDsByPriority(tr, "")
	for _, id := range []string{"G-0003", "D-0002", "E-0001", "M-0001", "ADR-0001"} {
		if !all[id] {
			t.Errorf("empty --priority should not filter; %s missing from %v", id, all)
		}
	}
}

// TestBuildListRows_PriorityExcludesNonCarrying pins the negative space:
// entities with no priority set (untagged gap/decision) and entities whose
// kind never carries a priority (epic, milestone, ADR) never match a
// specific `--priority` level.
func TestBuildListRows_PriorityExcludesNonCarrying(t *testing.T) {
	t.Parallel()
	_, tr := setupPriorityTree(t)

	nonCarrying := []string{"G-0003", "D-0002", "E-0001", "M-0001", "ADR-0001"}
	for _, level := range []string{"urgent", "high", "medium", "low"} {
		got := listIDsByPriority(tr, level)
		for _, id := range nonCarrying {
			if got[id] {
				t.Errorf("--priority %s should exclude %s (no matching priority); got %v", level, id, got)
			}
		}
	}
}

// TestBuildListRows_PriorityFieldPopulated pins M-0263/AC-3's list-surface
// half: the row's own Priority field carries the entity's value (or is
// empty for a non-carrying / untagged entity), independent of whether
// --priority is used to filter.
func TestBuildListRows_PriorityFieldPopulated(t *testing.T) {
	t.Parallel()
	_, tr := setupPriorityTree(t)
	rows := list.BuildListRows(context.Background(), tr, "", "", "", "", "", true)
	byID := map[string]list.ListSummary{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	if got := byID["G-0001"].Priority; got != "urgent" {
		t.Errorf("G-0001 row Priority = %q, want urgent", got)
	}
	if got := byID["E-0001"].Priority; got != "" {
		t.Errorf("E-0001 row Priority = %q, want empty (epic never carries priority)", got)
	}
	if got := byID["G-0003"].Priority; got != "" {
		t.Errorf("G-0003 row Priority = %q, want empty (untagged)", got)
	}
}

// TestRunList_PriorityViaDispatcher pins the list dispatcher seam for
// AC-1: the --priority flag flows through cli.Execute and filters, and
// the JSON row carries the priority field (AC-3).
func TestRunList_PriorityViaDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Urgent gap", "--priority", "urgent", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Unprioritized gap", "--actor", "human/test", "--root", root)

	t.Run("priority filter flows through the dispatcher", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"list", "--priority", "urgent", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d, want ExitOK", rc)
		}
		var env struct {
			Result []list.ListSummary `json:"result"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, stdout)
		}
		if len(env.Result) != 1 || env.Result[0].Priority != "urgent" {
			t.Errorf("result = %+v, want exactly one row with priority=urgent", env.Result)
		}
	})

	t.Run("out-of-range value is a usage error, not an empty result", func(t *testing.T) {
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"list", "--priority", "critical", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitUsage {
			t.Errorf("rc=%d, want ExitUsage for an out-of-range --priority value", rc)
		}
		for _, want := range []string{"critical", "urgent", "high", "medium", "low"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("stderr %q missing %q", stderr, want)
			}
		}
	})
}
