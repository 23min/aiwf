package integration

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/status"
)

// TestFilterStatusByPriority pins M-0263/AC-2: filtering the status report
// by priority scopes the priority-bearing sections (open decisions, open
// gaps) to the matching level. Epics and milestones never carry priority,
// so they are left untouched — unlike area, which scopes epics too.
func TestFilterStatusByPriority(t *testing.T) {
	t.Parallel()
	_, tr := setupPriorityTree(t)
	now := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)

	t.Run("urgent scopes gaps and decisions", func(t *testing.T) {
		t.Parallel()
		r := status.BuildStatus(tr, nil, now)
		epicsBefore := epicIDSet(r.InFlightEpics)
		status.FilterStatusByPriority(tr, &r, "urgent")

		if got := gapIDSet(r.OpenGaps); !got["G-0001"] || got["G-0003"] || len(got) != 1 {
			t.Errorf("open gaps = %v, want only G-0001", got)
		}
		if got := decisionIDSet(r.OpenDecisions); !got["D-0001"] || got["D-0002"] || got["ADR-0001"] || len(got) != 1 {
			t.Errorf("open decisions = %v, want only D-0001 (ADR-0001 never carries priority)", got)
		}
		if got := epicIDSet(r.InFlightEpics); len(got) != len(epicsBefore) || !got["E-0001"] {
			t.Errorf("in-flight epics = %v, want unchanged %v (epics never carry priority)", got, epicsBefore)
		}
	})

	t.Run("high scopes to its own gap only", func(t *testing.T) {
		t.Parallel()
		r := status.BuildStatus(tr, nil, now)
		status.FilterStatusByPriority(tr, &r, "high")
		if got := gapIDSet(r.OpenGaps); !got["G-0002"] || len(got) != 1 {
			t.Errorf("open gaps = %v, want only G-0002", got)
		}
		if len(r.OpenDecisions) != 0 {
			t.Errorf("open decisions = %v, want none under high", decisionIDSet(r.OpenDecisions))
		}
	})

	t.Run("empty priority is a no-op", func(t *testing.T) {
		t.Parallel()
		r := status.BuildStatus(tr, nil, now)
		gapsBefore := len(r.OpenGaps)
		decisionsBefore := len(r.OpenDecisions)
		status.FilterStatusByPriority(tr, &r, "")
		if len(r.OpenGaps) != gapsBefore || len(r.OpenDecisions) != decisionsBefore {
			t.Errorf("empty priority mutated report: gaps %d->%d, decisions %d->%d",
				gapsBefore, len(r.OpenGaps), decisionsBefore, len(r.OpenDecisions))
		}
	})
}

// TestFilterStatusByPriority_UnknownIDExcluded pins the defensive
// e != nil guard: a report entry whose id doesn't resolve against the
// tree passed to FilterStatusByPriority (a stale report / mismatched
// tree, not reachable through BuildStatus's own same-tree call
// pattern, but not compiler-proven unreachable either) is excluded
// rather than causing a nil-pointer dereference.
func TestFilterStatusByPriority_UnknownIDExcluded(t *testing.T) {
	t.Parallel()
	_, tr := setupPriorityTree(t)
	r := status.StatusReport{
		OpenGaps:      []status.StatusGap{{ID: "G-9999", Title: "Ghost gap"}},
		OpenDecisions: []status.StatusEntity{{ID: "D-9999", Title: "Ghost decision", Kind: "decision"}},
	}
	status.FilterStatusByPriority(tr, &r, "urgent")
	if len(r.OpenGaps) != 0 {
		t.Errorf("open gaps = %+v, want the unknown id excluded", r.OpenGaps)
	}
	if len(r.OpenDecisions) != 0 {
		t.Errorf("open decisions = %+v, want the unknown id excluded", r.OpenDecisions)
	}
}

// TestBuildStatus_PriorityFieldPopulated pins M-0263/AC-3's status-surface
// half: OpenGaps/OpenDecisions rows carry the entity's own priority (or
// empty for an untagged gap/decision or an ADR, which never carries one).
func TestBuildStatus_PriorityFieldPopulated(t *testing.T) {
	t.Parallel()
	_, tr := setupPriorityTree(t)
	now := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	r := status.BuildStatus(tr, nil, now)

	gapByID := map[string]status.StatusGap{}
	for _, g := range r.OpenGaps {
		gapByID[g.ID] = g
	}
	if got := gapByID["G-0001"].Priority; got != "urgent" {
		t.Errorf("G-0001 gap Priority = %q, want urgent", got)
	}
	if got := gapByID["G-0003"].Priority; got != "" {
		t.Errorf("G-0003 gap Priority = %q, want empty (untagged)", got)
	}

	decByID := map[string]status.StatusEntity{}
	for _, d := range r.OpenDecisions {
		decByID[d.ID] = d
	}
	if got := decByID["D-0001"].Priority; got != "urgent" {
		t.Errorf("D-0001 decision Priority = %q, want urgent", got)
	}
	if got := decByID["ADR-0001"].Priority; got != "" {
		t.Errorf("ADR-0001 Priority = %q, want empty (ADR never carries priority)", got)
	}
}

// TestRunStatus_PriorityViaDispatcher pins the status dispatcher seam:
// AC-2 (--priority flows through cli.Execute and scopes the report) and
// AC-3 (the JSON envelope's gap/decision rows carry their priority).
func TestRunStatus_PriorityViaDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Urgent gap", "--priority", "urgent", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Unprioritized gap", "--actor", "human/test", "--root", root)

	t.Run("priority scopes open gaps through the dispatcher", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"status", "--priority", "urgent", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d, want ExitOK", rc)
		}
		var env struct {
			Result struct {
				OpenGaps []status.StatusGap `json:"open_gaps"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("unmarshal status envelope: %v\n%s", err, stdout)
		}
		if len(env.Result.OpenGaps) != 1 || env.Result.OpenGaps[0].Priority != "urgent" {
			t.Errorf("open_gaps = %+v, want exactly one urgent gap", env.Result.OpenGaps)
		}
	})

	t.Run("out-of-range value is a usage error", func(t *testing.T) {
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"status", "--priority", "critical", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitUsage {
			t.Errorf("rc=%d, want ExitUsage for an out-of-range --priority value", rc)
		}
		if !strings.Contains(stderr, "critical") {
			t.Errorf("stderr should name the bad value:\n%s", stderr)
		}
	})
}
