package stresstest

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// concurrent_milestone_race.go — M-0258/AC-1 & AC-2: ConcurrentMilestoneRaceScenario
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
// AC-1 commits to two mechanical invariants — every actor's subprocess
// actually ran and returned a parseable envelope (surfaced as a Run
// error, not a Violation), and the resulting repo stays check-clean
// beyond a curated baseline. AC-2 adds the oracle proper:
// classifyMilestoneRaceOutcomes judges s.outcomes (already captured by
// AC-1) against two independent signals — the outcome-shape/refusal-
// reason each actor reported, and the real commit order
// readRaceCommitOrder reads back via git trailers — so a legitimate
// race (exactly one promote/cancel actor lands per mutually-exclusive
// transition, everyone else cleanly refused) is never flagged, while a
// guard that silently didn't hold (the G-0335 shape) is.

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
// classifyMilestoneRaceOutcomes (AC-2) classifies as a legitimate race
// outcome or a guard violation.
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

// raceCommit is one commit's aiwf-verb/aiwf-entity trailer pair, in
// oldest-first commit order — readRaceCommitOrder's own reduction of
// dir's git history, which classifyMilestoneRaceOutcomes' commit-order
// causality check (AC-2) uses to tell a legitimate race (a winning
// cancel commit landing strictly after the AC's own open->met commit)
// from the G-0335 shape (the open-AC guard silently not holding, so a
// cancel lands at or before it).
type raceCommit struct {
	verb   string
	entity string
}

// readRaceCommitOrder returns every commit in dir's history, oldest
// first, reduced to its aiwf-verb/aiwf-entity trailer pair. Lists the
// commit SHAs via `git log --reverse --format=%H`, then reuses this
// same package's commitTrailerValue (force_override_durability.go) —
// its own `git log -1 <ref> --format=%(trailers:key=...,valueonly,
// unfold=true)` idiom — once per SHA per trailer key, rather than
// combining both keys into one --format string: git's %(trailers:...)
// placeholder always appends its own trailing newline to a matched
// value, so two of them concatenated in one format string do not land
// on the same line — confirmed empirically against a real repo.
func readRaceCommitOrder(dir string) ([]raceCommit, error) {
	cmd := exec.Command("git", "log", "--reverse", "--format=%H")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing commit SHAs in %s: %w", dir, err)
	}
	shas := parseRaceCommitSHAs(out)
	commits := make([]raceCommit, len(shas))
	for i, sha := range shas {
		verbVal, verbErr := commitTrailerValue(dir, sha, gitops.TrailerVerb)
		if verbErr != nil { //coverage:ignore defensive: reading a trailer off a SHA readRaceCommitOrder itself just listed from the same repo has no realistic failure mode; commitTrailerValue's own error branch is unit-tested directly (TestCommitTrailerValue_ErrorsOnUnreadableRef)
			return nil, fmt.Errorf("reading %s trailer on %s: %w", gitops.TrailerVerb, sha, verbErr)
		}
		entityVal, entityErr := commitTrailerValue(dir, sha, gitops.TrailerEntity)
		if entityErr != nil { //coverage:ignore defensive: see the aiwf-verb read above
			return nil, fmt.Errorf("reading %s trailer on %s: %w", gitops.TrailerEntity, sha, entityErr)
		}
		commits[i] = raceCommit{verb: verbVal, entity: entityVal}
	}
	return commits, nil
}

// parseRaceCommitSHAs parses `git log --reverse --format=%H`'s raw
// output into an ordered list of commit SHAs, oldest first. Split out
// of readRaceCommitOrder so the parsing itself is directly
// unit-testable against fabricated bytes, without a real git
// subprocess.
func parseRaceCommitSHAs(out []byte) []string {
	var shas []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		shas = append(shas, line)
	}
	return shas
}

// classifyMilestoneRaceOutcomes judges one concurrent-milestone-race
// run's outcomes against AC-2's legitimate-race-vs-violation oracle.
// milestoneID is the shared milestone's id (order's entity fields are
// matched against "<milestoneID>/AC-1" and milestoneID itself).
//
// Two independent signals feed the judgment. The outcome-shape/
// refusal-reason signal: the promote group must land exactly one "ok"
// (the AC's open->met transition can only land once), with every other
// promote actor refused as CodeFSMTransitionIllegal; the cancel group
// must land zero or one "ok" (the milestone's draft->cancelled
// transition can only land once), with every other cancel actor
// refused as either CodeMilestoneCancelNonTerminalACs (raced while the
// AC was still open) or CodeFSMTransitionIllegal (raced after another
// actor already cancelled the milestone). The commit-order causality
// signal: when a cancel actor did land, that commit's real position in
// dir's git history must come strictly after the promote commit's —
// otherwise the open-AC guard did not actually hold at the moment that
// cancel committed, the G-0335 regression shape, indistinguishable
// from a legitimate race by final state alone.
func classifyMilestoneRaceOutcomes(outcomes []raceActorOutcome, order []raceCommit, milestoneID string) []Violation {
	var violations []Violation
	acEntity := milestoneID + "/AC-1"

	promoteOKCount := 0
	for _, oc := range outcomes {
		if oc.operation != raceOpPromote {
			continue
		}
		if oc.status == "ok" {
			promoteOKCount++
			continue
		}
		if oc.errorCode != entity.CodeFSMTransitionIllegal.ID {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"a promote actor was refused with error code %q, want %q — the refusal reason contradicts the FSM's own verdict",
				oc.errorCode, entity.CodeFSMTransitionIllegal.ID,
			)})
		}
	}
	if promoteOKCount != 1 {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"%d promote actors reported ok for %s (open -> met), want exactly 1",
			promoteOKCount, acEntity,
		)})
	}

	cancelOKCount := 0
	for _, oc := range outcomes {
		if oc.operation != raceOpCancel {
			continue
		}
		if oc.status == "ok" {
			cancelOKCount++
			continue
		}
		if oc.errorCode != verb.CodeMilestoneCancelNonTerminalACs.ID && oc.errorCode != entity.CodeFSMTransitionIllegal.ID {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"a cancel actor was refused with error code %q, want %q or %q — the refusal reason contradicts the open-AC guard or the FSM's own verdict",
				oc.errorCode, verb.CodeMilestoneCancelNonTerminalACs.ID, entity.CodeFSMTransitionIllegal.ID,
			)})
		}
	}
	if cancelOKCount > 1 {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"%d cancel actors reported ok for %s, want at most 1",
			cancelOKCount, milestoneID,
		)})
	}

	if cancelOKCount >= 1 {
		promoteIdx, cancelIdx := -1, -1
		for i, c := range order {
			if promoteIdx == -1 && c.verb == raceOpPromote && c.entity == acEntity {
				promoteIdx = i
			}
			if cancelIdx == -1 && c.verb == raceOpCancel && c.entity == milestoneID {
				cancelIdx = i
			}
		}
		if promoteIdx == -1 {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"a cancel actor reported ok but no %s commit for %s was found in the commit order — cannot verify commit-order causality",
				raceOpPromote, acEntity,
			)})
		} else if cancelIdx <= promoteIdx {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"a cancel actor's commit landed at or before the AC's own open -> met commit (cancel index %d, promote index %d) — the open-AC guard did not hold under concurrent contention",
				cancelIdx, promoteIdx,
			)})
		}
	}

	return violations
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

	// M-0258/AC-2: judge the race's own outcomes against the
	// legitimate-race-vs-violation oracle, reading back the real
	// commit order (once, on the final dir state — not re-run per
	// actor) so the commit-order causality signal can tell a
	// legitimate race from a guard that silently didn't hold.
	// s.violations accumulates both signals, mirroring every other
	// scenario's own append(s.violations, ...) idiom in this package.
	order, orderErr := readRaceCommitOrder(dir)
	if orderErr != nil { //coverage:ignore defensive: reading commit trailer order off a repo this scenario itself just produced commits in has no realistic failure mode; readRaceCommitOrder's own error branch is unit-tested directly against a non-git directory
		return fmt.Errorf("reading commit trailer order after the concurrent race: %w", orderErr)
	}
	s.violations = classifyMilestoneRaceOutcomes(s.outcomes, order, s.milestoneID)
	s.violations = append(s.violations, classifyAgainstBaseline(checkEnv.Findings, concurrentMilestoneRaceExpectedWarnings)...)
	return nil
}

// Verify returns every violation Run collected: AC-2's own
// legitimate-race-vs-violation oracle judging s.outcomes (classified
// in Run via classifyMilestoneRaceOutcomes), alongside AC-1's
// check-clean baseline — both signals stay live side by side.
func (s *ConcurrentMilestoneRaceScenario) Verify(_ string) []Violation {
	return s.violations
}
