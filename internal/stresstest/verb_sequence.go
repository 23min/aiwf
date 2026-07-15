package stresstest

import (
	"fmt"
	"math/rand/v2"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
)

// verb_sequence.go — M-0241/AC-1: VerbSequenceScenario drives a
// random walk of `aiwf promote` calls, via the real compiled
// binary, against one real entity of every kind in one disposable
// git repo. It extends internal/entity/transition_property_test.go's
// pattern (which only exercises entity.ValidateTransition in-process)
// out to the real binary: after every step it confirms the FSM's
// legality verdict is honored and that `aiwf check` never regresses,
// per M-0241/AC-1.

// verbSequenceExpectedWarnings is the baseline of finding codes any
// check in this scenario's disposable repo is expected to carry,
// independent of anything the scenario itself probes:
//
//   - provenance-untrailered-scope-undefined: the repo never
//     configures an upstream remote, so the provenance audit range
//     is permanently undefined.
//   - epic-active-no-drafted-milestones: the scenario's one epic
//     reaches "active" (and stays there) while never carrying a
//     draft-status milestone — the milestone entity this scenario
//     also creates gets walked away from "draft" independently, and
//     no second milestone is ever added to replace it.
//   - terminal-entity-not-archived / archive-sweep-pending: the
//     random walk drives entities to terminal statuses (done,
//     rejected, wontfix, ...) routinely, and this scenario never
//     runs `aiwf archive` — both are advisory-only sweep reminders,
//     not evidence of anything this scenario probes.
//   - promote-on-wrong-branch (G-0270): this scenario's whole point
//     is exercising FSM transitions generically across every entity
//     kind in ONE disposable repo — it never cuts an epic or
//     milestone ritual branch (that's ADR-0010's branch-choreography
//     concern, a different scenario's domain). So every epic that
//     reaches "active" and every milestone that reaches
//     "in_progress" here lands on the single working branch, not the
//     ritual branch AC-8 expects — an accepted, structural side
//     effect of this walker's design, not evidence of a real
//     misplaced activation.
//
// Any OTHER finding — any error-severity finding, or a warning with
// a code not in this set — is a real violation this scenario reports.
var verbSequenceExpectedWarnings = map[string]bool{
	check.CodeProvenanceUntrailedScopeUndefined: true,
	check.CodeEpicActiveNoDraftedMilestones:     true,
	check.CodeTerminalEntityNotArchived:         true,
	check.CodeArchiveSweepPending:               true,
	check.CodePromoteOnWrongBranch.ID:           true,
}

// VerbSequenceScenario implements Scenario. steps is the number of
// walk operations run per entity kind (M-0250/AC-2: promote is one of
// several selectable operations now, not the only one).
type VerbSequenceScenario struct {
	aiwfBin        string
	rng            *rand.Rand
	steps          int
	violations     []Violation
	renameCounter  int
	retitleCounter int
	archiveCounter int
	moveCounter    int
}

// NewVerbSequenceScenario builds a scenario that walks `steps`
// promote attempts per kind, seeded for reproducibility.
func NewVerbSequenceScenario(aiwfBin string, seed int64, steps int) *VerbSequenceScenario {
	return &VerbSequenceScenario{
		aiwfBin: aiwfBin,
		rng:     rand.New(rand.NewPCG(uint64(seed), uint64(seed))), //nolint:gosec // seeded PCG for reproducible replay, not a security context
		steps:   steps,
	}
}

// Setup git-inits dir and sets a deterministic commit identity.
func (s *VerbSequenceScenario) Setup(dir string) error {
	return gitInitAndConfig(dir)
}

// Run creates one entity of every kind, plus one extra scratch epic
// dedicated to being the milestone's alternate move target
// (M-0250/AC-2), and walks each through s.steps random operations.
// The epic and milestone are special-cased: the milestone is created
// immediately after the epic — while the epic is still freshly
// "proposed", guaranteed non-terminal — and only then are both
// walked, rather than "walk the epic to completion, then create the
// milestone" the way entity.AllKinds() order would otherwise suggest.
// That ordering used to let the epic's own random walk (stepPromote
// draws its target uniformly from the kind's full closed status set,
// terminal statuses included) drive it to done/cancelled before the
// milestone ever existed, starving `move`'s walk coverage — see
// G-0401. The remaining four kinds (ADR, gap, decision, contract)
// have no creation-time dependency on another kind's walked state, so
// they keep the original create-then-walk-in-place shape, in
// entity.AllKinds() order.
func (s *VerbSequenceScenario) Run(dir string) error {
	// The scratch epic is created first and never walked itself
	// (no promote/rename/retitle/archive ever runs against it), so it
	// stays non-terminal for the whole scenario — always a valid,
	// live `move` target regardless of what the walked epic's own
	// walk later does to its own status.
	altEpicEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "epic",
		"--title", "walker move-target scratch epic",
		"--body", "dedicated aiwf move target for the verb-sequence stress scenario; never walked itself")
	if err != nil { //coverage:ignore defensive: covered by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing at the source (runAiwfJSON's own launch-failure branch), not by re-triggering it at every call site
		return fmt.Errorf("seeding the move-target scratch epic: %w", err)
	}
	if altEpicEnv.Status != "ok" {
		return fmt.Errorf("seeding the move-target scratch epic: aiwf did not report ok (status=%s, error=%+v)", altEpicEnv.Status, altEpicEnv.Error)
	}
	altEpicID := altEpicEnv.Metadata.EntityID

	epicID, epicStatus, err := s.createWalkerEntity(dir, entity.KindEpic, "")
	if err != nil {
		return err
	}

	// Created immediately after the epic and before the epic ever
	// takes a single walk step, so its --epic parent is guaranteed to
	// still be "proposed" (an entity is always born non-terminal) —
	// this add can no longer trip the epic-terminal-non-terminal-
	// children refusal G-0398 describes, because the epic it names has
	// had no opportunity yet to reach a terminal or archived status.
	milestoneID, milestoneStatus, err := s.createWalkerEntity(dir, entity.KindMilestone, epicID)
	if err != nil { //coverage:ignore defensive: createWalkerEntity's own error-forward is already covered at the epic call site above (TestVerbSequenceScenario_RealBinary_RunSurfacesAnAllKindsLoopCreationRefusal), not by re-triggering it at every call site
		return err
	}

	if err := s.walk(dir, entity.KindEpic, epicID, epicStatus, nil); err != nil { //coverage:ignore defensive: walk's own internal error branches are the same launch-failure class, pinned at their source
		return err
	}

	// move is selectable only for the milestone kind — it's the only
	// kind verb.Move accepts, and only the milestone entity has a
	// live, guaranteed-non-terminal second epic (altEpicID) to
	// alternate with (see this method's doc comment).
	mv := &moveState{current: epicID, other: altEpicID}
	if err := s.walk(dir, entity.KindMilestone, milestoneID, milestoneStatus, mv); err != nil { //coverage:ignore defensive: walk's own internal error branches are the same launch-failure class, pinned at their source
		return err
	}

	for _, kind := range entity.AllKinds() {
		if kind == entity.KindEpic || kind == entity.KindMilestone {
			continue
		}
		id, status, err := s.createWalkerEntity(dir, kind, "")
		if err != nil { //coverage:ignore defensive: createWalkerEntity's own error-forward is already covered at the epic call site above (TestVerbSequenceScenario_RealBinary_RunSurfacesAnAllKindsLoopCreationRefusal), not by re-triggering it at every call site
			return err
		}
		if err := s.walk(dir, kind, id, status, nil); err != nil { //coverage:ignore defensive: walk's own internal error branches are the same launch-failure class, pinned at their source
			return err
		}
	}
	return nil
}

// createWalkerEntity runs `aiwf add <kind>` (passing epicID as the
// milestone's --epic parent when kind is KindMilestone) followed by
// `aiwf show` to read the freshly-created entity's starting status.
// Shared by Run for every kind — the epic/milestone special-casing
// above and the generic per-kind loop below both create their entity
// through this one path, so the add/show wiring can't drift between
// the two call sites.
func (s *VerbSequenceScenario) createWalkerEntity(dir string, kind entity.Kind, epicID string) (id, status string, err error) {
	args := []string{
		"add", string(kind),
		"--title", fmt.Sprintf("stress %s", kind),
		"--body", "generated by the verb-sequence stress scenario",
	}
	if kind == entity.KindMilestone {
		args = append(args, "--epic", epicID, "--tdd", "none")
	}

	addEnv, err := runAiwfJSON(s.aiwfBin, dir, args...)
	if err != nil { //coverage:ignore defensive: covered by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing at the source (runAiwfJSON's own launch-failure branch), not by re-triggering it at every call site
		return "", "", fmt.Errorf("creating a %s entity: %w", kind, err)
	}
	if addEnv.Status != "ok" {
		return "", "", fmt.Errorf("creating a %s entity: aiwf did not report ok (status=%s, error=%+v, findings=%+v)",
			kind, addEnv.Status, addEnv.Error, addEnv.Findings)
	}
	id = addEnv.Metadata.EntityID

	showEnv, err := runAiwfJSON(s.aiwfBin, dir, "show", id)
	if err != nil { //coverage:ignore defensive: same launch-failure class as the `add` call above; `show` on a just-created valid id has no realistic failure mode of its own
		return "", "", fmt.Errorf("reading initial status of %s: %w", id, err)
	}
	status = showEnv.Result.Status
	if status == "" { //coverage:ignore defensive: `show`'s JSON contract always populates result.status for a valid entity id
		return "", "", fmt.Errorf("could not determine initial status of %s", id)
	}
	return id, status, nil
}

// Verify returns every violation walk collected across every kind.
func (s *VerbSequenceScenario) Verify(_ string) []Violation {
	return s.violations
}

// walk runs s.steps operations against id, selecting each step's
// operation via a weighted random draw over walkOperationsFor(mv !=
// nil) (M-0250/AC-2). mv is nil for every kind but milestone — move
// is only ever in the drawable set when mv is non-nil. After every
// step it re-runs `aiwf check` and classifies its findings, then
// cross-checks `aiwf list --archived` against ground truth via
// checkListInvariant (M-0250/AC-3) — a whole-tree check, not scoped
// to id, since a corruption `list` misses could affect any entity
// this or an earlier kind's walk created.
func (s *VerbSequenceScenario) walk(dir string, kind entity.Kind, id, current string, mv *moveState) error {
	ops := walkOperationsFor(mv != nil)
	for i := 0; i < s.steps; i++ {
		opName := pickWalkOperation(s.rng, ops)

		var stepViolations []Violation
		var err error
		switch opName {
		case "promote":
			current, stepViolations, err = s.stepPromote(dir, kind, id, current)
		case "rename":
			stepViolations, err = s.stepRename(dir, id)
		case "retitle":
			stepViolations, err = s.stepRetitle(dir, id)
		case "archive":
			stepViolations, err = s.stepArchive(dir)
		case moveOperationName:
			stepViolations, err = s.stepMove(dir, id, mv)
		}
		if err != nil { //coverage:ignore defensive: forwards whichever stepX method's own launch-failure error fired — each is already pinned at its own source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing's launch-failure class
			return err
		}
		s.violations = append(s.violations, stepViolations...)

		checkEnv, err := runAiwfJSON(s.aiwfBin, dir, "check")
		if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
			return fmt.Errorf("running aiwf check after %s step %q: %w", id, opName, err)
		}
		s.violations = append(s.violations, classifyCheckFindings(checkEnv.Findings)...)

		label := fmt.Sprintf("%s step %d (%s)", id, i+1, opName)
		listViolations, err := checkListInvariant(s.aiwfBin, dir, label)
		if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
			return fmt.Errorf("running the list-vs-ground-truth invariant after %s: %w", label, err)
		}
		s.violations = append(s.violations, listViolations...)
	}
	return nil
}

// stepPromote runs one promote attempt against id, picking a target
// status uniformly at random from kind's full closed status set (so
// both FSM-legal and FSM-illegal targets are exercised), and
// classifies the outcome via classifyVerbSequenceStep.
func (s *VerbSequenceScenario) stepPromote(dir string, kind entity.Kind, id, current string) (next string, violations []Violation, err error) {
	targets := entity.AllowedStatuses(kind)
	target := targets[s.rng.IntN(len(targets))]

	before, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: git rev-list on a repo this scenario itself just created and is still driving has no realistic failure mode
		return current, nil, fmt.Errorf("counting commits before %s %s -> %s: %w", id, current, target, err)
	}
	env, err := runAiwfJSON(s.aiwfBin, dir, "promote", id, target)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
		return current, nil, fmt.Errorf("running promote %s %s: %w", id, target, err)
	}
	after, err := gitHeadCommitCount(dir)
	if err != nil { //coverage:ignore defensive: see the "before" call above
		return current, nil, fmt.Errorf("counting commits after %s %s -> %s: %w", id, current, target, err)
	}

	next, violations = classifyVerbSequenceStep(kind, current, target, before, after, env)
	return next, violations, nil
}

// stepRename runs one `aiwf rename` against id with a fresh,
// monotonically-unique slug (s.renameCounter), so repeated rename
// steps against the same entity never collide with an earlier one's
// resulting slug.
func (s *VerbSequenceScenario) stepRename(dir, id string) ([]Violation, error) {
	s.renameCounter++
	slug := fmt.Sprintf("walk-rename-%d", s.renameCounter)
	env, err := runAiwfJSON(s.aiwfBin, dir, "rename", id, slug)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
		return nil, fmt.Errorf("running rename %s -> %q: %w", id, slug, err)
	}
	return classifySimpleStep(fmt.Sprintf("%s: rename to %q", id, slug), env), nil
}

// stepRetitle runs one `aiwf retitle` against id with a fresh,
// monotonically-unique title (s.retitleCounter).
func (s *VerbSequenceScenario) stepRetitle(dir, id string) ([]Violation, error) {
	s.retitleCounter++
	title := fmt.Sprintf("walker retitle %d", s.retitleCounter)
	env, err := runAiwfJSON(s.aiwfBin, dir, "retitle", id, title)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
		return nil, fmt.Errorf("running retitle %s -> %q: %w", id, title, err)
	}
	return classifySimpleStep(fmt.Sprintf("%s: retitle to %q", id, title), env), nil
}

// stepArchive runs `aiwf archive --apply`, a repo-wide sweep rather
// than an id-targeted operation — it may be a no-op (nothing
// currently terminal) depending on what earlier steps in this or
// other kinds' walks have done; a no-op sweep is a legitimate `ok`,
// not a violation. s.archiveCounter records every dispatch attempt
// (success or not), so a test can confirm walk's switch actually
// reached this case without depending on a sweep finding anything.
func (s *VerbSequenceScenario) stepArchive(dir string) ([]Violation, error) {
	s.archiveCounter++
	env, err := runAiwfJSON(s.aiwfBin, dir, "archive", "--apply")
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
		return nil, fmt.Errorf("running archive --apply: %w", err)
	}
	return classifySimpleStep("archive --apply", env), nil
}

// stepMove runs one `aiwf move` against id, relocating it to mv's
// current alternate target. mv.target() is always a live, non-
// terminal epic (see Run's doc comment), so an unexpected refusal is
// always a violation; on success mv.applyMoved() swaps current/other
// so the next move step alternates back. s.moveCounter mirrors
// s.archiveCounter's dispatch-attempt bookkeeping.
func (s *VerbSequenceScenario) stepMove(dir, id string, mv *moveState) ([]Violation, error) {
	s.moveCounter++
	target := mv.target()
	env, err := runAiwfJSON(s.aiwfBin, dir, "move", id, "--epic", target)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
		return nil, fmt.Errorf("running move %s -> %s: %w", id, target, err)
	}
	violations := classifySimpleStep(fmt.Sprintf("%s: move -> %s", id, target), env)
	if len(violations) == 0 {
		mv.applyMoved()
	}
	return violations, nil
}

// moveState tracks a milestone's current parent epic across a walk's
// move steps: current is its live parent, other is the alternate
// target the next move step selects. Both epic ids are guaranteed
// non-terminal for the milestone's entire walk — see Run's doc
// comment for why.
type moveState struct {
	current string
	other   string
}

// target returns the epic id a move step should relocate id to.
func (mv *moveState) target() string { return mv.other }

// applyMoved records a successful move: current and other swap, so
// the next move step's target alternates back to where id started.
func (mv *moveState) applyMoved() { mv.current, mv.other = mv.other, mv.current }

// walkOperation is one verb-shaped action VerbSequenceScenario.walk
// may select at a given step, alongside its selection weight.
type walkOperation struct {
	Name   string
	Weight int
}

// moveOperationName is move's own entry name in the operation table,
// named once so walk's switch and walkOperationsFor's conditional
// append can't drift apart on what selecting "move" means.
const moveOperationName = "move"

// baseWalkOperations are the operations every kind's walk can select
// regardless of kind. promote carries most of the weight (it is the
// walker's original, still-primary operation per M-0241/AC-1); the
// other three are weighted low enough to stay occasional perturbations
// rather than dominating the walk.
var baseWalkOperations = []walkOperation{
	{Name: "promote", Weight: 6},
	{Name: "rename", Weight: 1},
	{Name: "retitle", Weight: 1},
	{Name: "archive", Weight: 1},
}

// walkOperationsFor returns the operation set for one kind's walk:
// baseWalkOperations, plus move when moveEnabled (kind == milestone,
// verb.Move's only accepted kind, and a second epic exists to move
// between — see Run's doc comment). M-0250/AC-2's own acceptance
// criterion is pinned directly against this table's shape by
// TestWalkOperationsFor_NamesAllFourExtensionOpsWithNonzeroWeight.
func walkOperationsFor(moveEnabled bool) []walkOperation {
	ops := append([]walkOperation(nil), baseWalkOperations...)
	if moveEnabled {
		ops = append(ops, walkOperation{Name: moveOperationName, Weight: 1})
	}
	return ops
}

// totalWeight sums ops' weights.
func totalWeight(ops []walkOperation) int {
	total := 0
	for _, op := range ops {
		total += op.Weight
	}
	return total
}

// pickWalkOperation draws a uniformly random operation name from ops,
// weighted by each entry's Weight.
func pickWalkOperation(rng *rand.Rand, ops []walkOperation) string {
	return weightedPick(ops, rng.IntN(totalWeight(ops)))
}

// weightedPick returns the name of the operation whose cumulative
// weight range contains r (0 <= r < totalWeight(ops)) — the
// deterministic core of pickWalkOperation, factored out so the
// cumulative-boundary logic is testable without a random source.
func weightedPick(ops []walkOperation, r int) string {
	for _, op := range ops {
		if r < op.Weight {
			return op.Name
		}
		r -= op.Weight
	}
	return ops[len(ops)-1].Name //coverage:ignore defensive: unreachable when 0 <= r < totalWeight(ops), the only way pickWalkOperation constructs r
}

// classifySimpleStep judges one non-FSM walker step (rename, retitle,
// archive, move): none of these four carries a walker-chosen target
// that might deliberately be illegal (unlike promote's random target
// status), so an unexpected refusal is always a violation — there is
// no symmetrical "refused for a legitimate reason" branch the way
// classifyVerbSequenceStep has for promote.
func classifySimpleStep(label string, env verbEnvelope) []Violation {
	if env.Status == "ok" {
		return nil
	}
	return []Violation{{Message: fmt.Sprintf(
		"%s unexpectedly refused (status=%s, error=%+v)", label, env.Status, env.Error)}}
}

// classifyVerbSequenceStep judges one promote attempt's outcome
// against the FSM's own legality verdict (entity.ValidateTransition),
// returning the resulting current status and any violations found.
//
// An FSM-illegal target must always be refused specifically with
// CodeFSMTransitionIllegal, and must never land a commit — that
// direction is unconditional. An FSM-legal target may still be
// refused for an orthogonal business rule (e.g. a gap's
// addressed-status resolver requirement) that sits outside the FSM
// proper; that refusal is legitimate as long as it isn't also
// tagged fsm-transition-illegal and it lands no commit. Whenever
// the verb reports success, exactly one commit must land.
func classifyVerbSequenceStep(kind entity.Kind, current, target string, before, after int, env verbEnvelope) (next string, violations []Violation) {
	legal := entity.ValidateTransition(kind, current, target) == nil
	refusedAsIllegal := env.Status == "error" && env.Error != nil && env.Error.Code == entity.CodeFSMTransitionIllegal.ID

	if env.Status == "ok" {
		if !legal {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"%s: FSM-illegal %s -> %s was accepted (status ok) instead of refused", kind, current, target)})
		}
		if after != before+1 {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"%s: promote %s -> %s reported success but landed %d commits, want exactly 1", kind, current, target, after-before)})
		}
		if legal {
			return target, violations
		}
		return current, violations
	}

	// Refused. Legitimate for either reason (FSM-illegal, or an
	// orthogonal business rule) as long as the refusal reason
	// matches the FSM's own verdict and no commit landed.
	if !legal && !refusedAsIllegal {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"%s: FSM-illegal %s -> %s was not refused as %s (status=%s, error=%+v)",
			kind, current, target, entity.CodeFSMTransitionIllegal.ID, env.Status, env.Error)})
	}
	if legal && refusedAsIllegal {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"%s: FSM-legal %s -> %s was refused as %s", kind, current, target, entity.CodeFSMTransitionIllegal.ID)})
	}
	if after != before {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"%s: refused promote %s -> %s still landed a commit (%d -> %d)", kind, current, target, before, after)})
	}
	return current, violations
}

// classifyCheckFindings reports a violation for every finding that
// isn't part of verbSequenceExpectedWarnings — any error-severity
// finding always violates, regardless of code. Thin wrapper over
// classifyAgainstBaseline (M-0257/AC-2, checkclean.go), the
// generalized form of this same loop every other scenario's own
// check-clean baseline assertion shares.
func classifyCheckFindings(findings []verbEnvelopeFinding) []Violation {
	return classifyAgainstBaseline(findings, verbSequenceExpectedWarnings)
}
