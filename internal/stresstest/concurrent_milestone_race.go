package stresstest

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"

	"github.com/23min/aiwf/internal/check"
)

// concurrent_milestone_race.go — M-0258/AC-1: ConcurrentMilestoneRaceScenario
// launches n real `aiwf` subprocess actors concurrently against ONE
// shared disposable repo, every actor targeting the SAME milestone's
// promote/cancel operations — the exact shape G-0335 exercised (the
// open-AC guard on milestone cancel, internal/verb/cancel_guards.go /
// internal/verb/promote.go's Cancel). Unlike ConcurrentMoveScenario
// (each actor its own milestone) or ConcurrentWriterAtScaleScenario
// (each actor its own gap), every actor here contests the same
// milestone+AC pair, reusing ConcurrentWriterAtScaleScenario's
// goroutine + sync.WaitGroup subprocess fan-out pattern: real OS
// process scheduling drives the race, no artificial delay.
//
// This scenario itself commits to two mechanical invariants only —
// every actor's subprocess actually ran and returned a parseable
// envelope (surfaced as a Run error, not a Violation), and the
// resulting repo stays check-clean beyond a curated baseline. Judging
// each actor's outcome as a legitimate race versus an actual guard
// violation is AC-2's own oracle, built on top of the raceActorOutcome
// shape Run already captures here.

// raceOpPromote and raceOpCancel name the two operations this
// scenario's actors race, doubling as the `aiwf` subcommand each
// launches.
const (
	raceOpPromote = "promote"
	raceOpCancel  = "cancel"
)

// concurrentMilestoneRaceExpectedWarnings is the baseline of finding
// codes this scenario's post-run check is expected to carry (M-0257/
// AC-1's baseline-map convention), derived empirically by running the
// scenario repeatedly:
//
//   - provenance-untrailered-scope-undefined: this scenario's
//     disposable repo never configures an upstream remote.
//   - archive-sweep-pending / terminal-entity-not-archived: a
//     legitimate race outcome can land the milestone at the terminal
//     `cancelled` status (when a cancel actor's attempt runs after the
//     AC has been promoted to `met`), and this scenario never sweeps
//     via `aiwf archive`. Absent when no cancel actor wins the race —
//     the milestone then stays non-terminal at `draft`.
//
// Any OTHER finding — any error-severity finding, or a warning with a
// code not in this set — is a real violation.
var concurrentMilestoneRaceExpectedWarnings = map[string]bool{
	check.CodeProvenanceUntrailedScopeUndefined: true,
	check.CodeArchiveSweepPending:               true,
	check.CodeTerminalEntityNotArchived:         true,
}

// ConcurrentMilestoneRaceScenario implements Scenario.
type ConcurrentMilestoneRaceScenario struct {
	aiwfBin     string
	n           int
	milestoneID string

	// before/after are the repo's HEAD commit count immediately
	// surrounding the fan-out — a harness sanity signal a test can
	// check directly (e.g. after == before + the number of "ok"
	// outcomes), independent of Verify's own check-clean assertion.
	before, after int

	// outcomes is every actor's reduced result, in launch order — the
	// shape AC-2's future oracle will classify.
	outcomes []raceActorOutcome

	// finalMilestoneStatus/finalACStatus are the milestone's and its
	// AC-1's status after the race settles, read back via `aiwf show`
	// — post-mutation state available to a future classifier (or a
	// test) beyond what any single actor's own envelope reported.
	finalMilestoneStatus string
	finalACStatus        string

	violations []Violation
}

// NewConcurrentMilestoneRaceScenario builds a scenario that races n
// concurrent actors — split roughly in half between `aiwf promote
// <milestone>/AC-1 met` and `aiwf cancel <milestone>` — against one
// shared, pre-seeded milestone carrying exactly one open AC. seed
// matches RunRepeated's newScenario(seed int64) Scenario signature but
// is otherwise unused — this scenario's race jitter comes from real OS
// goroutine/process scheduling, not seeded pseudo-randomness (same
// rationale as ConcurrentMoveScenario's own seed parameter).
func NewConcurrentMilestoneRaceScenario(aiwfBin string, n int, _ int64) *ConcurrentMilestoneRaceScenario {
	return &ConcurrentMilestoneRaceScenario{aiwfBin: aiwfBin, n: n}
}

// Setup git-inits dir, creates one epic, one milestone under it
// (--tdd none, left at its default `draft` status), and exactly one AC
// on that milestone (left `open`) — the shared entity+AC pair every
// actor Run launches will later race promote/cancel against. Draft is
// a legal `aiwf cancel` source per the milestone FSM
// (internal/entity/transition.go), so the only thing standing between
// a cancel attempt and success is the open-AC guard this scenario
// exists to race.
func (s *ConcurrentMilestoneRaceScenario) Setup(dir string) error {
	if err := gitInitAndConfig(dir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}

	epicEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "epic", "--title", "milestone-race epic", "--body", "parent epic for the concurrent-milestone-race stress scenario")
	if err != nil {
		return fmt.Errorf("seeding the epic: %w", err)
	}
	if epicEnv.Status != "ok" { //coverage:ignore defensive: seeding the very first entity in a fresh disposable repo has no realistic refusal mode
		return fmt.Errorf("seeding the epic: aiwf did not report ok (status=%s, error=%+v)", epicEnv.Status, epicEnv.Error)
	}

	msEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "milestone", "--epic", epicEnv.Metadata.EntityID, "--tdd", "none",
		"--title", "milestone-race milestone", "--body", "the single shared milestone this scenario races promote/cancel actors against")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("seeding the milestone: %w", err)
	}
	if msEnv.Status != "ok" { //coverage:ignore defensive: seeding a milestone under a just-created, never-touched epic has no realistic refusal mode
		return fmt.Errorf("seeding the milestone: aiwf did not report ok (status=%s, error=%+v)", msEnv.Status, msEnv.Error)
	}
	s.milestoneID = msEnv.Metadata.EntityID

	acEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "ac", s.milestoneID, "--title", "concurrent-milestone-race probe AC")
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return fmt.Errorf("seeding the AC: %w", err)
	}
	if acEnv.Status != "ok" { //coverage:ignore defensive: seeding the first AC on a milestone this scenario itself just created has no realistic refusal mode
		return fmt.Errorf("seeding the AC: aiwf did not report ok (status=%s, error=%+v)", acEnv.Status, acEnv.Error)
	}
	return nil
}

// raceActorArgs builds the argv for one actor's operation ("promote"
// or "cancel") against milestoneID. Split out of launchActor so the
// argument-construction branch is directly unit-testable without a
// real subprocess.
func raceActorArgs(operation, milestoneID string) []string {
	if operation == raceOpPromote {
		return []string{"promote", milestoneID + "/AC-1", "met", "--format=json"}
	}
	return []string{"cancel", milestoneID, "--reason", "concurrent-milestone-race probe", "--format=json"}
}

// rawRaceActorResult is one actor's unparsed subprocess result, before
// classification, tagged with the operation it ran.
type rawRaceActorResult struct {
	operation string
	execErr   error
	out       []byte
}

// launchActor runs one promote-or-cancel attempt against s.milestoneID.
// Factored out of Run's fan-out loop (rather than inlined in the
// goroutine literal) so the loop launching the n actors is a plain
// fan-out, not a retry — one subprocess launch per actor, never
// retried on failure.
func (s *ConcurrentMilestoneRaceScenario) launchActor(dir, operation string) rawRaceActorResult {
	cmd := exec.Command(s.aiwfBin, raceActorArgs(operation, s.milestoneID)...) //nolint:gosec // s.aiwfBin is a path this package's own BuildBinary just produced, not attacker-controlled input
	cmd.Dir = dir
	out, err := cmd.Output()
	return rawRaceActorResult{operation: operation, execErr: err, out: out}
}

// raceActorOutcome is one concurrent actor's reduced result: which
// operation it ran, the verb's reported envelope status, and the
// typed error code it carried (empty when it succeeded) — the shape
// AC-2's future oracle will classify as a legitimate race outcome or a
// guard violation.
type raceActorOutcome struct {
	operation string
	status    string
	errorCode string
}

// buildRaceOutcome reduces one actor's decoded envelope to a
// raceActorOutcome. Split out of Run so the error-code-extraction
// branch is directly unit-testable against a fabricated envelope,
// rather than depending on real race timing to exercise both the
// "carries an error" and "no error" cases.
func buildRaceOutcome(operation string, env verbEnvelope) raceActorOutcome {
	errorCode := ""
	if env.Error != nil {
		errorCode = env.Error.Code
	}
	return raceActorOutcome{operation: operation, status: env.Status, errorCode: errorCode}
}

// Run launches s.n actors concurrently against s.milestoneID — the
// first s.n/2 (by launch index) running `aiwf promote
// <milestoneID>/AC-1 met`, the rest running `aiwf cancel
// <milestoneID>` — waits for all of them, decodes every actor's
// envelope, reads back the milestone's and its AC's post-race status,
// then confirms the resulting tree stays check-clean beyond baseline
// noise.
func (s *ConcurrentMilestoneRaceScenario) Run(dir string) error {
	before, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: git rev-list on a repo this scenario itself just created and is still driving has no realistic failure mode
		return fmt.Errorf("counting commits before the concurrent race: %w", err)
	}

	promoteCount := s.n / 2
	raw := make([]rawRaceActorResult, s.n)
	var wg sync.WaitGroup
	for i := 0; i < s.n; i++ {
		operation := raceOpCancel
		if i < promoteCount {
			operation = raceOpPromote
		}
		wg.Add(1)
		go func(i int, operation string) {
			defer wg.Done()
			raw[i] = s.launchActor(dir, operation)
		}(i, operation)
	}
	wg.Wait()

	after, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: see the "before" call above
		return fmt.Errorf("counting commits after the concurrent race: %w", err)
	}
	s.before, s.after = before, after

	outcomes := make([]raceActorOutcome, s.n)
	for i, ro := range raw {
		var exitErr *exec.ExitError
		if ro.execErr != nil && !errors.As(ro.execErr, &exitErr) {
			return fmt.Errorf("actor %d (%s): running aiwf %s: %w", i, ro.operation, ro.operation, ro.execErr)
		}
		env, err := parseVerbEnvelope([]string{ro.operation, s.milestoneID}, ro.out)
		if err != nil { //coverage:ignore defensive: parseVerbEnvelope's own malformed-input branch is unit-tested directly against fabricated bytes; a real promote/cancel invocation's stdout is never malformed
			return fmt.Errorf("actor %d (%s): %w", i, ro.operation, err)
		}
		outcomes[i] = buildRaceOutcome(ro.operation, env)
	}
	s.outcomes = outcomes

	msEnv, msErr := runAiwfJSON(s.aiwfBin, dir, "show", s.milestoneID)
	if msErr != nil { //coverage:ignore defensive: `show` on the milestone this scenario itself just raced has no realistic failure mode of its own; the binary-missing launch-failure class is already pinned at the earlier per-actor loop, which returns before any actor reaches this line in that test
		return fmt.Errorf("reading post-race milestone status: %w", msErr)
	}
	s.finalMilestoneStatus = msEnv.Result.Status

	acEnv, acErr := runAiwfJSON(s.aiwfBin, dir, "show", s.milestoneID+"/AC-1")
	if acErr != nil { //coverage:ignore defensive: see the milestone show above
		return fmt.Errorf("reading post-race AC status: %w", acErr)
	}
	s.finalACStatus = acEnv.Result.Status

	// M-0257/AC-1: alongside the per-actor outcome capture above,
	// confirm the resulting tree stays check-clean beyond baseline
	// noise — this scenario never ran `aiwf check` at all before.
	checkEnv, checkErr := runAiwfJSON(s.aiwfBin, dir, "check")
	if checkErr != nil { //coverage:ignore defensive: same launch-failure class other scenarios pin at runAiwfJSON's own source; the actor loop above already exercised this binary successfully by the time this call runs
		return fmt.Errorf("running aiwf check after the concurrent race: %w", checkErr)
	}
	s.violations = classifyAgainstBaseline(checkEnv.Findings, concurrentMilestoneRaceExpectedWarnings)
	return nil
}

// Verify returns every violation Run collected. AC-1 commits to
// exactly one violation source — the check-clean baseline classified
// in Run — deliberately narrower than AC-2's future oracle, which will
// classify s.outcomes itself.
func (s *ConcurrentMilestoneRaceScenario) Verify(_ string) []Violation {
	return s.violations
}
