package stresstest

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"

	"github.com/23min/aiwf/internal/entity"
)

// reachability_isolation.go — M-0241/AC-5: ReachabilityIsolationScenario
// confirms the documented architectural fact behind
// internal/cli/integration/branch_scenarios_ac2_test.go's comment: aiwf
// check's provenance walk is `git log HEAD`, not `--all`, so a commit
// made in a sibling worktree is invisible to a check run in another
// worktree until the branches are actually merged. Beyond confirming
// the fact itself, this scenario checks whether that invisibility has
// any OTHER consequence: `aiwf check`'s full outcome (findings and
// entity count) must be byte-identical before and after the sibling's
// invisible commit — not just silent on the one rule
// (isolation-escape) that already documents this — and `aiwf
// show`/`aiwf history` must degrade in their own already-understood
// ways (show: not found; history: empty, never leaked) rather than
// erroring unexpectedly or leaking data across the branch boundary.

// ReachabilityIsolationScenario implements Scenario.
type ReachabilityIsolationScenario struct {
	aiwfBin    string
	kind       entity.Kind
	violations []Violation
}

// NewReachabilityIsolationScenario builds a scenario confirming
// cross-worktree reachability isolation for one entity kind. seed
// matches RunRepeated's newScenario(seed int64) Scenario signature
// (M-0240/AC-5) but is unused — this scenario is deterministic, not
// a race (contrast AC-2/AC-3): a sequential commit in one worktree,
// observed from another, always reproduces the same way.
func NewReachabilityIsolationScenario(aiwfBin string, kind entity.Kind, _ int64) *ReachabilityIsolationScenario {
	return &ReachabilityIsolationScenario{aiwfBin: aiwfBin, kind: kind}
}

// Setup creates a main repo with a seed commit, then adds two
// sibling worktrees (actor-a, actor-b) off it — dir/main, dir/wt-a,
// dir/wt-b. Only actor-b ever commits in this scenario; actor-a is
// the observing side.
func (s *ReachabilityIsolationScenario) Setup(dir string) error {
	return newSiblingWorktreesFixture(dir)
}

// Run captures a baseline `aiwf check` in worktree A, commits a new
// entity in worktree B, re-runs check in worktree A (still unmerged)
// and confirms it is unchanged, probes `show`/`history` for the
// sibling's entity from worktree A, then merges and confirms the
// same probes flip to "found" — closing the loop on "unreachable
// today, reachable once merged," never "unreachable forever."
func (s *ReachabilityIsolationScenario) Run(dir string) error {
	wtA := filepath.Join(dir, "wt-a")
	wtB := filepath.Join(dir, "wt-b")

	baselineEnv, err := runAiwfJSON(s.aiwfBin, wtA, "check")
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestReachabilityIsolationScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("baseline check in worktree A: %w", err)
	}

	addEnv, err := runAiwfJSON(s.aiwfBin, wtB, "add", string(s.kind), "--title", "sibling entity", "--body", "reachability isolation stress actor")
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestReachabilityIsolationScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("adding the sibling entity in worktree B: %w", err)
	}
	if addEnv.Status != "ok" {
		return fmt.Errorf("adding the sibling entity in worktree B: aiwf did not report ok (status=%s, error=%+v)", addEnv.Status, addEnv.Error)
	}
	bID := addEnv.Metadata.EntityID

	afterEnv, err := runAiwfJSON(s.aiwfBin, wtA, "check")
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestReachabilityIsolationScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("check in worktree A after the sibling's invisible commit: %w", err)
	}
	// probeShowFound, not runAiwfJSON: `aiwf show <missing-id>
	// --format=json` doesn't honor --format=json on its not-found
	// path (empty stdout, plain-text stderr instead of a JSON error
	// envelope — G-0389), so this classifies by exit status alone.
	showFoundBeforeMerge, err := probeShowFound(s.aiwfBin, wtA, bID)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestReachabilityIsolationScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("show in worktree A for the sibling's entity: %w", err)
	}
	historyEnv, err := runAiwfJSON(s.aiwfBin, wtA, "history", bID)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestReachabilityIsolationScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("history in worktree A for the sibling's entity: %w", err)
	}

	if mergeErr := runGit(wtA, "merge", "-q", "--no-edit", "actor-b"); mergeErr != nil { //coverage:ignore defensive: the two worktrees never touch overlapping paths, so this merge is always a clean fast path with no realistic conflict
		return fmt.Errorf("merging actor-b into actor-a's worktree: %w", mergeErr)
	}
	postMergeEnv, err := runAiwfJSON(s.aiwfBin, wtA, "check")
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestReachabilityIsolationScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("check in worktree A after merging: %w", err)
	}
	postMergeHistoryEnv, err := runAiwfJSON(s.aiwfBin, wtA, "history", bID)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestReachabilityIsolationScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("history in worktree A after merging: %w", err)
	}

	s.violations = append(s.violations, classifyReachabilityIsolation(baselineEnv, afterEnv, showFoundBeforeMerge, historyEnv, postMergeEnv, postMergeHistoryEnv)...)
	return nil
}

// Verify returns every violation Run collected.
func (s *ReachabilityIsolationScenario) Verify(_ string) []Violation {
	return s.violations
}

// probeShowFound runs `aiwf show <id>` in dir and reports whether it
// found the entity, classified by exit status alone (see the
// not-found-path comment above Run's call site: G-0389).
func probeShowFound(aiwfBin, dir, id string) (bool, error) {
	cmd := exec.Command(aiwfBin, "show", id, "--format=json") //nolint:gosec // aiwfBin is a path this package's own BuildBinary just produced, not attacker-controlled input
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, fmt.Errorf("running aiwf show %s: %w", id, err) //coverage:ignore defensive: same launch-failure class pinned at its source by TestReachabilityIsolationScenario_RealBinary_ErrorsWhenBinaryMissing
}

// classifyReachabilityIsolation judges the six real `aiwf` calls
// Run makes:
//
//   - after must be identical to baseline — the sibling's invisible
//     commit must have ZERO observable effect on any check rule, not
//     just silence from isolation-escape specifically.
//   - showFoundBeforeMerge must be false — show must NOT find the
//     sibling's entity before the merge.
//   - history must return "ok" with zero events before the merge —
//     an empty (unreached) result, not an error and not a leaked
//     event from across the branch boundary.
//   - postMerge's entity count must exceed after's — the merge must
//     actually expose the entity, proving "unreachable today" is not
//     "unreachable forever."
//   - postMergeHistory must show at least one event — same closing-
//     the-loop confirmation for history specifically.
func classifyReachabilityIsolation(baseline, after verbEnvelope, showFoundBeforeMerge bool, history, postMerge, postMergeHistory verbEnvelope) []Violation {
	var violations []Violation

	if !reflect.DeepEqual(baseline.Findings, after.Findings) || baseline.Metadata.Entities != after.Metadata.Entities {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"aiwf check's outcome in worktree A changed after the sibling worktree's invisible commit: before=%+v after=%+v", baseline, after)})
	}

	if showFoundBeforeMerge {
		violations = append(violations, Violation{Message: "aiwf show found the sibling worktree's entity before any merge — reachability isolation broken"})
	}

	if history.Status != "ok" {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"aiwf history errored instead of returning an empty (unreached) history for the sibling worktree's entity: %+v", history)})
	} else if history.Metadata.Events != 0 {
		violations = append(violations, Violation{Message: "aiwf history found events for a commit unreachable from this worktree's HEAD — a broader-reachability leak"})
	}

	if postMerge.Metadata.Entities <= after.Metadata.Entities {
		violations = append(violations, Violation{Message: "aiwf check's entity count did not increase after merging the sibling branch — the merge did not actually expose the entity"})
	}

	if postMergeHistory.Status != "ok" || postMergeHistory.Metadata.Events == 0 {
		violations = append(violations, Violation{Message: "aiwf history still shows no events for the sibling's entity after merging — reachability never actually closed"})
	}

	return violations
}
