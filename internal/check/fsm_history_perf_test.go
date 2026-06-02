package check

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/23min/aiwf/internal/entity"
)

// TestFSMHistoryConsistent_PerfBudget pins a wall-time regression
// budget for M-0137/AC-7. The retrofit collapses M-0130's per-entity
// `git log --follow` + per-(commit, parent) `git show` fan-out into
// one whole-repo `git log --all` (gitops.BulkRevwalk) + one long-
// lived `git cat-file --batch` pump (gitops.BlobReader). For an
// N-entity tree with K status-change commits per entity, M-0130's
// subprocess count was O(N·K) (~3000 on the kernel tree); M-0137's
// is O(2 + acks) — constant in the consumer-tree size for the hot
// path.
//
// The budget: 50 entities × 4 status-change commits each = ~200
// status-change observations. Post-retrofit completes in well under
// 1 second on a real devcontainer. We set 10 seconds as the
// regression budget — generous over the post-retrofit baseline, but
// tight enough that re-introducing per-entity exec.Command at the
// kernel-tree scale (~331 entities ⇒ ~1700 commits at this fixture's
// density) would push the fixture's runtime past 10s and fire the
// assertion.
//
// Flake risk: wall-time tests are clock-dependent. The 10s budget is
// deliberately 10× generous over the measured post-retrofit runtime
// to absorb CI runner variance. If this assertion fires without an
// obvious code change, suspect runner load before suspecting a
// regression — but DO investigate, because the silent-swallow this
// milestone closed was first detected as exactly this kind of
// "tests passed but performance degraded" signal.
//
// The aiwf-tests trailer on this AC's promote commit records the
// post-retrofit runtime (chosen at AC-7 wrap time) so future runs
// can compare against a concrete number, not just the budget cap.
func TestFSMHistoryConsistent_PerfBudget(t *testing.T) {
	t.Parallel()
	const (
		entityCount = 50
		budget      = 10 * time.Second
	)
	r := newRepoFixture(t)

	// Build a fixture of `entityCount` epics, each cycled through
	// 4 status-change commits (proposed → active → done, with a
	// retitle commit between to add a per-entity touch volume).
	// The transitions deliberately include proposed → active (a
	// sovereign-act-shape change) so the predicates fire on every
	// entity, exercising the full observation→finding path.
	for i := 0; i < entityCount; i++ {
		id := fmt.Sprintf("E-%04d", i+1)
		r.commitEntity(id, entity.KindEpic, entity.StatusProposed, "add "+id)
		r.commitEntity(id, entity.KindEpic, entity.StatusActive, "promote "+id)
		r.commitEntity(id, entity.KindEpic, entity.StatusDone, "complete "+id)
		// Retitle commit (no status change) to add walker noise.
		r.commitEntityWithBody(id, entity.KindEpic, entity.StatusDone, "retitled body content\n", "retitle "+id)
	}
	tr := r.tree()

	start := time.Now()
	findings := FSMHistoryConsistent(context.Background(), r.root, tr, nil)
	elapsed := time.Since(start)

	if elapsed > budget {
		t.Errorf("FSMHistoryConsistent took %v on %d-entity fixture (4 commits/entity), want < %v — suspect per-entity subprocess fan-out regression",
			elapsed, entityCount, budget)
	}

	// Sanity check: the fixture is designed to produce findings (each
	// entity's proposed → active fires forced-untrailered: sovereign-
	// act-shape transition by a non-human actor without aiwf-force).
	// A retrofit that emits ZERO findings would be silently broken.
	if len(findings) == 0 {
		t.Errorf("expected at least one finding on the perf fixture (forced-untrailered per entity); got 0 — suspect a finding-emission regression")
	}

	t.Logf("FSMHistoryConsistent on %d-entity fixture: %v elapsed, %d findings (budget: %v)",
		entityCount, elapsed, len(findings), budget)
}
