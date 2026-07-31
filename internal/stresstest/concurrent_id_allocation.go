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
// simulated), and confirms repolock's mutual exclusion holds: no two
// attempts ever allocate the same id, and any attempt that does not
// succeed was refused because another actor held the lock.

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
	before, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: counting commits in a repo this scenario's own Setup just created has no realistic failure mode
		return fmt.Errorf("counting commits before the concurrent add: %w", err)
	}

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

	after, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: see the "before" call above
		return fmt.Errorf("counting commits after the concurrent add: %w", err)
	}

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
		outcomes[i] = newActorOutcome(env)
	}

	s.violations = append(s.violations, classifyConcurrentIDAllocation(outcomes, s.n, before, after)...)

	// M-0257/AC-1: alongside the per-actor outcome assertion above,
	// confirm the resulting tree stays check-clean beyond baseline
	// noise — this scenario never ran `aiwf check` at all before.
	// checkErr, not err: this repo's govet config runs with
	// enable-all: true, and reusing err here trips its shadow check
	// against the per-actor loop's own inner `err` declaration above.
	checkEnv, checkErr := runAiwfJSON(s.aiwfBin, dir, "check")
	if checkErr != nil { //coverage:ignore defensive: same launch-failure class other scenarios pin at runAiwfJSON's own source; the actor loop above already exercised this binary successfully by the time this call runs
		return fmt.Errorf("running aiwf check after the concurrent add: %w", checkErr)
	}
	s.violations = append(s.violations, classifyAgainstBaseline(checkEnv.Findings, concurrentIDAllocationExpectedWarnings)...)
	return nil
}

// Verify returns every violation Run collected.
func (s *ConcurrentIDAllocationScenario) Verify(_ string) []Violation {
	return s.violations
}

// actorOutcome is one concurrent actor's `aiwf add` result, reduced
// to the fields classifyConcurrentIDAllocation needs. errorCode
// carries the envelope's error code, which is what separates a
// refusal the scenario expects from one it does not.
type actorOutcome struct {
	status    string
	entityID  string
	errorCode string
}

// newActorOutcome reduces one actor's envelope to the fields the
// classifier judges. Separate from Run's loop so that the reduction —
// in particular that the error code is carried across at all, without
// which every refusal looks alike to the classifier — is pinned
// against a fabricated envelope rather than only through a real
// subprocess race.
func newActorOutcome(env verbEnvelope) actorOutcome {
	return actorOutcome{
		status:    env.Status,
		entityID:  env.Metadata.EntityID,
		errorCode: envelopeErrorCode(env),
	}
}

// classifyConcurrentIDAllocation judges n concurrent `aiwf add`
// attempts against the properties that hold regardless of how loaded
// the machine is: every id allocated by more than one successful
// attempt breaks repolock's core mutual-exclusion promise; every
// failure that is not repolock's documented busy refusal is
// unexplained by contention; the commit count must land exactly
// successCount above before, since a refused actor takes the lock
// before it writes anything and so commits nothing (ADR-0036); and a
// run in which no actor at all succeeded is a deadlock rather than
// congestion, since the lock is released by whichever actor holds it.
//
// The commit-count arm is what keeps refusals honest. Once some
// actors are expected to fail, "it reported busy" and "it changed
// nothing" are separate claims, and only the second is checked here
// against git rather than against the actor's own account of itself.
//
// How many actors get through is deliberately not asserted. That
// count measures the machine's throughput against a two-second lock
// timeout, so on a contended runner the tail actors receive the busy
// refusal the verb promises and a count-based oracle reports the
// specification being honored as a defect.
func classifyConcurrentIDAllocation(outcomes []actorOutcome, n, before, after int) []Violation {
	var violations []Violation
	seen := map[string]int{}
	successCount := 0
	for i, oc := range outcomes {
		if oc.status != "ok" {
			if !isBusyRefusal(oc.errorCode) {
				violations = append(violations, Violation{Message: fmt.Sprintf(
					"actor %d: aiwf add failed for a reason other than repolock contention (status=%s, code=%q)", i, oc.status, oc.errorCode)})
			}
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
	if n > 0 && successCount == 0 {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"none of %d concurrent actors succeeded — contention explains some actors being refused, never all of them", n)})
	}
	if after != before+successCount {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"commit count %d -> %d after %d successful adds, want exactly +%d", before, after, successCount, successCount)})
	}
	return violations
}
