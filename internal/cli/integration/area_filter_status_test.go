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

// epicIDSet collects the ids of a slice of StatusEpic.
func epicIDSet(es []status.StatusEpic) map[string]bool {
	got := map[string]bool{}
	for _, e := range es {
		got[e.ID] = true
	}
	return got
}

// decisionIDSet / gapIDSet collect ids of the open-decisions / open-gaps
// sections.
func decisionIDSet(ds []status.StatusEntity) map[string]bool {
	got := map[string]bool{}
	for _, d := range ds {
		got[d.ID] = true
	}
	return got
}

func gapIDSet(gs []status.StatusGap) map[string]bool {
	got := map[string]bool{}
	for _, g := range gs {
		got[g.ID] = true
	}
	return got
}

// TestFilterStatusByArea pins M-0174/AC-2: filtering the status report by
// area scopes the entity-derived sections (in-flight epics, planned
// epics, open decisions, open gaps) to that workstream — root kinds by
// their own field, epics carrying their derived milestones along. It also
// pins AC-6 on the status surface: untagged entities (E-0003, G-0002) are
// excluded from a named-area scope. Health stays global (AC-2: cross-
// cutting signals are not scoped).
func TestFilterStatusByArea(t *testing.T) {
	t.Parallel()
	_, tr := setupAreaTree(t)
	now := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)

	t.Run("platform scopes every entity section", func(t *testing.T) {
		t.Parallel()
		r := status.BuildStatus(tr, nil, now)
		entitiesBefore := r.Health.Entities
		status.FilterStatusByArea(tr, &r, "platform")

		if got := epicIDSet(r.InFlightEpics); !got["E-0001"] || got["E-0002"] || got["E-0003"] {
			t.Errorf("in-flight epics = %v, want only E-0001 (E-0002 billing, E-0003 untagged excluded)", got)
		}
		if got := epicIDSet(r.PlannedEpics); !got["E-0004"] || len(got) != 1 {
			t.Errorf("planned epics = %v, want only E-0004", got)
		}
		if got := decisionIDSet(r.OpenDecisions); !got["ADR-0001"] || got["D-0001"] {
			t.Errorf("open decisions = %v, want only ADR-0001 (D-0001 billing excluded)", got)
		}
		if got := gapIDSet(r.OpenGaps); !got["G-0001"] || got["G-0002"] {
			t.Errorf("open gaps = %v, want only G-0001 (G-0002 untagged excluded)", got)
		}
		if r.Health.Entities != entitiesBefore {
			t.Errorf("Health.Entities = %d, want %d unchanged (health is global, not scoped)", r.Health.Entities, entitiesBefore)
		}
	})

	t.Run("billing scopes to its own entities", func(t *testing.T) {
		t.Parallel()
		r := status.BuildStatus(tr, nil, now)
		status.FilterStatusByArea(tr, &r, "billing")
		if got := epicIDSet(r.InFlightEpics); !got["E-0002"] || len(got) != 1 {
			t.Errorf("in-flight epics = %v, want only E-0002", got)
		}
		if len(r.PlannedEpics) != 0 {
			t.Errorf("planned epics = %v, want none (E-0004 is platform)", epicIDSet(r.PlannedEpics))
		}
		if got := decisionIDSet(r.OpenDecisions); !got["D-0001"] || got["ADR-0001"] {
			t.Errorf("open decisions = %v, want only D-0001", got)
		}
		if len(r.OpenGaps) != 0 {
			t.Errorf("open gaps = %v, want none under billing", gapIDSet(r.OpenGaps))
		}
	})

	t.Run("empty area is a no-op", func(t *testing.T) {
		t.Parallel()
		r := status.BuildStatus(tr, nil, now)
		before := len(r.InFlightEpics)
		status.FilterStatusByArea(tr, &r, "")
		if len(r.InFlightEpics) != before || before != 3 {
			t.Errorf("empty area changed in-flight epics: got %d, want %d (all active epics)", len(r.InFlightEpics), before)
		}
	})
}

// TestRunStatus_AreaViaDispatcher pins the status dispatcher seam: AC-2
// (--area flows through cli.Execute and scopes the report) and AC-5 (an
// undeclared value notes to stderr and exits 0).
func TestRunStatus_AreaViaDispatcher(t *testing.T) {
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Platform work", "--area", "platform", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Billing work", "--area", "billing", "--actor", "human/test", "--root", root)
	// Activate both so they land in the in-flight section.
	mustRun(t, "promote", "E-0001", "active", "--actor", "human/test", "--root", root)
	mustRun(t, "promote", "E-0002", "active", "--actor", "human/test", "--root", root)

	t.Run("declared area scopes the in-flight section through the dispatcher", func(t *testing.T) {
		rc, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"status", "--area", "platform", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("rc=%d, want ExitOK", rc)
		}
		// Structural assertion on the scoped section (not a blanket
		// substring): the in-flight epics must be exactly the platform
		// epic. E-0002 legitimately still appears under the GLOBAL
		// warnings/recent-activity sections, which AC-2 deliberately
		// leaves un-scoped — so a flat substring check would be wrong.
		var env struct {
			Result struct {
				InFlightEpics []struct {
					ID string `json:"id"`
				} `json:"in_flight_epics"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("unmarshal status envelope: %v\n%s", err, stdout)
		}
		ids := map[string]bool{}
		for _, e := range env.Result.InFlightEpics {
			ids[e.ID] = true
		}
		if !ids["E-0001"] || ids["E-0002"] || len(ids) != 1 {
			t.Errorf("in_flight_epics = %v, want exactly {E-0001} (billing E-0002 scoped out)", ids)
		}
	})

	t.Run("undeclared value notes and exits 0", func(t *testing.T) {
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"status", "--area", "nonsense", "--format", "json", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Errorf("undeclared --area rc=%d, want ExitOK", rc)
		}
		if !strings.Contains(stderr, "nonsense") {
			t.Errorf("stderr should carry the undeclared-area note:\n%s", stderr)
		}
	})
}
