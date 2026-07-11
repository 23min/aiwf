package stresstest

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"
)

// concurrent_move.go — M-0250/AC-4: ConcurrentMoveScenario launches n
// real `aiwf move` subprocesses concurrently against one working
// copy, each relocating a distinct milestone from a shared source
// epic to a shared target epic. move is the one verb in
// verb_sequence.go's sequential-walk extension (M-0250/AC-2) whose
// cross-entity fan-out — source epic, target epic, the moved entity
// itself — makes a true concurrent race worth checking on top of
// that sequential coverage. Mirrors
// ConcurrentIDAllocationScenario's fan-out mechanism (M-0241/AC-2):
// goroutines racing real OS process scheduling, no artificial
// synchronization delay.

// ConcurrentMoveScenario implements Scenario.
type ConcurrentMoveScenario struct {
	aiwfBin      string
	n            int
	sourceEpic   string
	targetEpic   string
	milestoneIDs []string
	violations   []Violation
}

// NewConcurrentMoveScenario builds a scenario that races n concurrent
// `aiwf move` subprocesses, each relocating one of n milestones from
// a shared source epic to a shared target epic. seed matches
// RunRepeated's newScenario(seed int64) Scenario signature but is
// otherwise unused — this scenario's race jitter comes from real OS
// goroutine/process scheduling, not seeded pseudo-randomness (same
// rationale as ConcurrentIDAllocationScenario's own seed parameter).
func NewConcurrentMoveScenario(aiwfBin string, n int, _ int64) *ConcurrentMoveScenario {
	return &ConcurrentMoveScenario{aiwfBin: aiwfBin, n: n}
}

// Setup git-inits dir, creates the source and target epics, and n
// milestones under the source epic — one per concurrent actor Run
// will later launch.
func (s *ConcurrentMoveScenario) Setup(dir string) error {
	if err := gitInitAndConfig(dir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}

	sourceEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "epic", "--title", "move-race source", "--body", "source epic for the concurrent-move stress scenario")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("seeding the source epic: %w", err)
	}
	if sourceEnv.Status != "ok" { //coverage:ignore defensive: seeding the very first entity in a fresh disposable repo has no realistic refusal mode; the generic add-refusal mechanism is already exercised by verb_sequence_test.go's own creation-refusal tests
		return fmt.Errorf("seeding the source epic: aiwf did not report ok (status=%s, error=%+v)", sourceEnv.Status, sourceEnv.Error)
	}
	s.sourceEpic = sourceEnv.Metadata.EntityID

	targetEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "epic", "--title", "move-race target", "--body", "target epic for the concurrent-move stress scenario")
	if err != nil { //coverage:ignore defensive: see the source epic add above
		return fmt.Errorf("seeding the target epic: %w", err)
	}
	if targetEnv.Status != "ok" { //coverage:ignore defensive: see the source epic add above
		return fmt.Errorf("seeding the target epic: aiwf did not report ok (status=%s, error=%+v)", targetEnv.Status, targetEnv.Error)
	}
	s.targetEpic = targetEnv.Metadata.EntityID

	s.milestoneIDs = make([]string, s.n)
	for i := 0; i < s.n; i++ {
		msEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "milestone", "--epic", s.sourceEpic, "--tdd", "none",
			"--title", fmt.Sprintf("move-race milestone %d", i), "--body", "seeded for the concurrent-move stress scenario")
		if err != nil { //coverage:ignore defensive: see the source epic add above
			return fmt.Errorf("seeding milestone %d: %w", i, err)
		}
		if msEnv.Status != "ok" { //coverage:ignore defensive: seeding a milestone under a just-created, never-touched source epic has no realistic refusal mode
			return fmt.Errorf("seeding milestone %d: aiwf did not report ok (status=%s, error=%+v)", i, msEnv.Status, msEnv.Error)
		}
		s.milestoneIDs[i] = msEnv.Metadata.EntityID
	}
	return nil
}

// rawMoveActorResult is one actor's unparsed `aiwf move` subprocess
// result, before classification.
type rawMoveActorResult struct {
	execErr error
	out     []byte
}

// launchActor runs `aiwf move <milestoneID> --epic s.targetEpic`
// against dir. Factored out of Run's fan-out loop (rather than
// inlined in the goroutine literal) so the loop launching the n
// actors is a plain fan-out, not a retry — this is a single
// subprocess launch per actor, never retried on failure.
func (s *ConcurrentMoveScenario) launchActor(dir, milestoneID string) rawMoveActorResult {
	args := []string{"move", milestoneID, "--epic", s.targetEpic, "--format=json"}
	cmd := exec.Command(s.aiwfBin, args...) //nolint:gosec // s.aiwfBin is a path this package's own BuildBinary just produced, not attacker-controlled input
	cmd.Dir = dir
	out, err := cmd.Output()
	return rawMoveActorResult{execErr: err, out: out}
}

// Run launches s.n `aiwf move` subprocesses concurrently — one per
// seeded milestone, all targeting s.targetEpic — waits for all of
// them, then classifies the outcomes.
func (s *ConcurrentMoveScenario) Run(dir string) error {
	before, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: git rev-list on a repo this scenario itself just created and is still driving has no realistic failure mode
		return fmt.Errorf("counting commits before the concurrent move: %w", err)
	}

	raw := make([]rawMoveActorResult, s.n)
	var wg sync.WaitGroup
	for i := 0; i < s.n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			raw[i] = s.launchActor(dir, s.milestoneIDs[i])
		}(i)
	}
	wg.Wait()

	after, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: see the "before" call above
		return fmt.Errorf("counting commits after the concurrent move: %w", err)
	}

	outcomes := make([]moveActorOutcome, s.n)
	for i, ro := range raw {
		var exitErr *exec.ExitError
		if ro.execErr != nil && !errors.As(ro.execErr, &exitErr) {
			return fmt.Errorf("actor %d: running aiwf move: %w", i, ro.execErr)
		}
		env, err := parseVerbEnvelope([]string{"move", s.milestoneIDs[i]}, ro.out)
		if err != nil { //coverage:ignore defensive: parseVerbEnvelope's own malformed-input branch is unit-tested directly against fabricated bytes; a real `move` invocation's stdout is never malformed
			return fmt.Errorf("actor %d: %w", i, err)
		}
		parent := ""
		if env.Status == "ok" {
			showEnv, showErr := runAiwfJSON(s.aiwfBin, dir, "show", s.milestoneIDs[i])
			if showErr != nil { //coverage:ignore defensive: `show` on a milestone whose own move attempt just reported ok has no realistic failure mode of its own; the binary-missing launch-failure class is already pinned at the earlier launchActor check above, which returns before any actor reaches this line in that test
				return fmt.Errorf("actor %d: reading post-move parent of %s: %w", i, s.milestoneIDs[i], showErr)
			}
			parent = showEnv.Result.Parent
		}
		outcomes[i] = moveActorOutcome{milestoneID: s.milestoneIDs[i], status: env.Status, parent: parent}
	}

	s.violations = classifyConcurrentMove(outcomes, s.n, s.targetEpic, before, after)
	return nil
}

// Verify returns every violation Run collected.
func (s *ConcurrentMoveScenario) Verify(_ string) []Violation {
	return s.violations
}

// moveActorOutcome is one concurrent `aiwf move` actor's result,
// reduced to the fields classifyConcurrentMove needs.
type moveActorOutcome struct {
	milestoneID string
	status      string
	parent      string // the milestone's parent after Run's move attempt; "" when the attempt didn't succeed
}

// classifyConcurrentMove judges n concurrent `aiwf move` attempts,
// each targeting a distinct milestone under the same source epic and
// moving to the same target epic: every non-"ok" status is its own
// violation (repolock should serialize every attempt to success
// within its timeout, since none of these n attempts logically
// conflicts with another — each targets its own milestone), any
// successfully-moved milestone that didn't actually end up parented
// under targetEpic is a violation (move's file-rename + frontmatter-
// write landed inconsistently under contention), and the total commit
// count must land exactly successCount more than before — any other
// delta means a commit was lost or duplicated.
func classifyConcurrentMove(outcomes []moveActorOutcome, n int, targetEpic string, before, after int) []Violation {
	var violations []Violation
	successCount := 0
	for _, oc := range outcomes {
		if oc.status != "ok" {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"%s: aiwf move did not report ok under concurrent contention (status=%s)", oc.milestoneID, oc.status,
			)})
			continue
		}
		successCount++
		if oc.parent != targetEpic {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"%s: move reported ok but final parent is %q, want %q", oc.milestoneID, oc.parent, targetEpic,
			)})
		}
	}
	if successCount != n {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"only %d/%d concurrent move actors succeeded — expected all to serialize successfully within repolock's timeout", successCount, n,
		)})
	}
	if after != before+successCount {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"commit count %d -> %d after %d successful moves, want exactly +%d", before, after, successCount, successCount,
		)})
	}
	return violations
}
