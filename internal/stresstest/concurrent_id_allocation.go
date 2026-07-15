package stresstest

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
)

// concurrent_id_allocation.go — M-0241/AC-2: ConcurrentIDAllocationScenario
// launches n real `aiwf add <kind>` subprocesses against ONE working
// copy, started close together via goroutine + OS process scheduling
// (no artificial synchronization delay — the race window is real, not
// simulated), and confirms repolock's mutual exclusion holds: every
// attempt serializes to a distinct id within the lock's 2-second
// timeout, and no two attempts ever allocate the same one.

// concurrentIDAllocationExpectedWarnings is the baseline of finding
// codes this scenario's post-run check is expected to carry
// (M-0257/AC-1), beyond the per-actor outcome/duplicate-id assertion
// classifyConcurrentIDAllocation already pins directly:
//
//   - provenance-untrailered-scope-undefined: this scenario's
//     disposable repo never configures an upstream remote.
//
// Any OTHER finding — any error-severity finding, or a warning with a
// code not in this set — is a real violation.
var concurrentIDAllocationExpectedWarnings = map[string]bool{
	check.CodeProvenanceUntrailedScopeUndefined: true,
}

// ConcurrentIDAllocationScenario implements Scenario.
type ConcurrentIDAllocationScenario struct {
	aiwfBin    string
	kind       entity.Kind
	n          int
	violations []Violation
}

// NewConcurrentIDAllocationScenario builds a scenario that races n
// concurrent `aiwf add <kind>` subprocesses against one disposable
// repo. seed matches RunRepeated's newScenario(seed int64) Scenario
// signature (M-0240/AC-5) but is otherwise unused — this scenario's
// race jitter comes from real OS goroutine/process scheduling, not
// seeded pseudo-randomness.
func NewConcurrentIDAllocationScenario(aiwfBin string, kind entity.Kind, n int, _ int64) *ConcurrentIDAllocationScenario {
	return &ConcurrentIDAllocationScenario{aiwfBin: aiwfBin, kind: kind, n: n}
}

// Setup git-inits dir and sets a deterministic commit identity.
func (s *ConcurrentIDAllocationScenario) Setup(dir string) error {
	return gitInitAndConfig(dir)
}

// rawActorResult is one actor's unparsed `aiwf add` subprocess
// result, before classification.
type rawActorResult struct {
	execErr error
	out     []byte
}

// launchActor runs one `aiwf add <kind>` invocation for actor i
// against dir. Factored out of Run's fan-out loop (rather than
// inlined in the goroutine literal) so the loop launching the n
// actors is a plain fan-out, not a retry — this is a single
// subprocess launch per actor, never retried on failure.
func (s *ConcurrentIDAllocationScenario) launchActor(dir string, i int) rawActorResult {
	args := []string{
		"add", string(s.kind),
		"--title", fmt.Sprintf("concurrent actor %d", i),
		"--body", "concurrent id-allocation stress actor",
		"--format=json",
	}
	cmd := exec.Command(s.aiwfBin, args...) //nolint:gosec // s.aiwfBin is a path this package's own BuildBinary just produced, not attacker-controlled input
	cmd.Dir = dir
	out, err := cmd.Output()
	return rawActorResult{execErr: err, out: out}
}

// Run launches s.n `aiwf add` subprocesses concurrently, waits for
// all of them, then classifies the outcomes.
func (s *ConcurrentIDAllocationScenario) Run(dir string) error {
	raw := make([]rawActorResult, s.n)
	var wg sync.WaitGroup
	for i := 0; i < s.n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			raw[i] = s.launchActor(dir, i)
		}(i)
	}
	wg.Wait()

	outcomes := make([]actorOutcome, s.n)
	for i, ro := range raw {
		var exitErr *exec.ExitError
		if ro.execErr != nil && !errors.As(ro.execErr, &exitErr) { //coverage:ignore defensive: same launch-failure class pinned at its source by TestConcurrentIDAllocationScenario_RealBinary_ErrorsWhenBinaryMissing
			return fmt.Errorf("actor %d: running aiwf add: %w", i, ro.execErr)
		}
		env, err := parseVerbEnvelope([]string{"add", string(s.kind)}, ro.out)
		if err != nil { //coverage:ignore defensive: parseVerbEnvelope's own malformed-input branch is unit-tested directly in verb_sequence_classify_test.go; a real `add` invocation's stdout is never malformed
			return fmt.Errorf("actor %d: %w", i, err)
		}
		outcomes[i] = actorOutcome{status: env.Status, entityID: env.Metadata.EntityID}
	}

	s.violations = append(s.violations, classifyConcurrentIDAllocation(outcomes, s.n)...)

	// M-0257/AC-1: alongside the per-actor outcome assertion above,
	// confirm the resulting tree stays check-clean beyond baseline
	// noise — this scenario never ran `aiwf check` at all before.
	checkEnv, err := runAiwfJSON(s.aiwfBin, dir, "check")
	if err != nil { //coverage:ignore defensive: same launch-failure class other scenarios pin at runAiwfJSON's own source; the actor loop above already exercised this binary successfully by the time this call runs
		return fmt.Errorf("running aiwf check after the concurrent add: %w", err)
	}
	s.violations = append(s.violations, classifyAgainstBaseline(checkEnv.Findings, concurrentIDAllocationExpectedWarnings)...)
	return nil
}

// Verify returns every violation Run collected.
func (s *ConcurrentIDAllocationScenario) Verify(_ string) []Violation {
	return s.violations
}

// actorOutcome is one concurrent actor's `aiwf add` result, reduced
// to the two fields classifyConcurrentIDAllocation needs.
type actorOutcome struct {
	status   string
	entityID string
}

// classifyConcurrentIDAllocation judges n concurrent `aiwf add`
// attempts: every non-"ok" status is its own violation (repolock
// should serialize every attempt to success within its timeout), any
// entity id allocated by more than one successful attempt is a
// violation (repolock's core mutual-exclusion promise broken), and an
// overall success-count shortfall is reported once more in aggregate
// so a partial-failure run can't slip through as "no duplicates
// found" when few or no ids were even allocated.
func classifyConcurrentIDAllocation(outcomes []actorOutcome, n int) []Violation {
	var violations []Violation
	seen := map[string]int{}
	successCount := 0
	for i, oc := range outcomes {
		if oc.status != "ok" {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"actor %d: aiwf add did not report ok under concurrent contention (status=%s)", i, oc.status)})
			continue
		}
		successCount++
		seen[oc.entityID]++
	}
	for id, count := range seen {
		if count > 1 {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"id %s was allocated by %d concurrent actors — repolock failed to serialize id allocation", id, count)})
		}
	}
	if successCount != n {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"only %d/%d concurrent actors succeeded — expected all to serialize successfully within repolock's timeout", successCount, n)})
	}
	return violations
}
