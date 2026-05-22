// Package cellcoverage provides per-cell test fixtures for the
// M-0124 + M-0125 positive/negative coverage milestones. A
// CellFixture is an isolated tmp git repo with `aiwf init` already
// run; its methods walk the kernel's verb surface in-process (no
// subprocess fork) to bring an entity to a (Kind, FromState) point
// or to mutate the fixture so it satisfies a spec.Predicate.
//
// Why in-process for fixture setup: M-0137 retrofit's insight is
// "don't fork when you don't have to." The cell-under-test still
// runs via subprocess (testutil.RunBin) — the integration seam
// matters there. Fixture preparation doesn't test flag parsing or
// exit codes; calling verb.Add / verb.Promote / verb.Apply directly
// produces the same on-disk + frontmatter state in ~10ms instead
// of ~80ms per fork.
package cellcoverage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/verb"
	"github.com/23min/aiwf/internal/workflows/spec"
)

const testActor = "human/test"

// CellFixture is a fresh isolated repo with aiwf init applied.
// Methods walk the kernel's verb surface in-process to drive the
// fixture to the required state. The cell-under-test in the
// per-cell driver runs against this same Root via subprocess.
//
// Carries the *testing.T it was constructed with so method calls
// can fail-fast via t.Fatalf without each call site marshaling the
// t parameter through. Pattern matches internal/verb/verb_test.go's
// runner.
type CellFixture struct {
	t    *testing.T
	Root string
	ctx  context.Context
}

// BringOpts controls knobs of BringEntityToState that some cells
// need (e.g., a non-default parent.tdd policy, an AC count to seed
// for the populated-fixture variant of Milestone.done).
//
// Zero-value fields mean "use the conventional default":
//   - ParentTDD: "required" for milestones (matches the parent.tdd
//     == "required" predicate used in AC sub-cells)
//   - ACs: 0 for milestone targets (the vacuous-satisfaction path
//     for Milestone.done; the spec's AntiRule says a milestone is
//     not required to have >=1 AC)
type BringOpts struct {
	ParentTDD string // override default "required" for milestone parents
	ACs       int    // populated-fixture knob: seed N met ACs
}

// NewCellFixture sets up an isolated repo:
//
//  1. t.TempDir for the root
//  2. git init + identity config
//  3. initrepo.Init to lay down aiwf.yaml + framework artifacts
//
// Returns a *CellFixture whose methods drive subsequent state.
// Bootstrap cost is dominated by initrepo.Init (~30-60ms; one-time
// per test). Compared to a subprocess `aiwf init`, the saving is
// the fork itself (~20-40ms).
func NewCellFixture(t *testing.T) *CellFixture {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()

	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	res, err := initrepo.Init(ctx, root, initrepo.Options{ActorOverride: testActor, SkipHook: true})
	if err != nil {
		t.Fatalf("initrepo.Init: %v", err)
	}
	_ = res

	return &CellFixture{t: t, Root: root, ctx: ctx}
}

// Tree re-loads the on-disk tree. Each call walks the filesystem;
// callers re-load after any state mutation so they see the post-
// mutation shape.
func (f *CellFixture) Tree() *tree.Tree {
	f.t.Helper()
	tr, loadErrs, err := tree.Load(f.ctx, f.Root)
	if err != nil {
		f.t.Fatalf("tree.Load: %v", err)
	}
	if len(loadErrs) != 0 {
		f.t.Fatalf("loadErrs: %+v", loadErrs)
	}
	return tr
}

// Must commits a verb.Result's plan. Mirrors the runner.must
// pattern from internal/verb/verb_test.go: asserts no Go error, no
// error-severity findings, plan present, then verb.Apply. Call
// sites pass the verb's (res, err) tuple directly:
//
//	f.Must(verb.Add(f.ctx, f.Tree(), ...))
func (f *CellFixture) Must(res *verb.Result, err error) *verb.Result {
	f.t.Helper()
	if err != nil {
		f.t.Fatalf("verb error: %v", err)
	}
	if check.HasErrors(res.Findings) {
		f.t.Fatalf("unexpected findings: %+v", res.Findings)
	}
	if res.Plan == nil {
		f.t.Fatal("no plan produced")
	}
	if applyErr := verb.Apply(f.ctx, f.Root, res.Plan); applyErr != nil {
		f.t.Fatalf("apply: %v", applyErr)
	}
	return res
}

// openRel opens a file relative to the fixture root.
func (f *CellFixture) openRel(rel string) (*os.File, error) {
	return os.Open(filepath.Join(f.Root, rel))
}

// BringEntityToState walks the verb sequence required to produce an
// entity of kind k at fromState, returning its id. The dispatch
// table is the per-kind FSM walk; the chosen path is the shortest
// legal sequence from a fresh repo.
//
// For sub-kinds (spec.KindAC, spec.KindTDDPhase) the helper first
// constructs the parent milestone (in_progress) then adds the AC
// and walks its sub-FSM.
//
// The Milestone.done path takes the zero-AC vacuous-satisfaction
// route by default (spec AntiRule: a milestone is not required to
// have >= 1 AC). BringOpts.ACs > 0 selects the populated path.
func (f *CellFixture) BringEntityToState(t *testing.T, k entity.Kind, fromState string, opts BringOpts) string {
	t.Helper()
	switch k {
	case entity.KindEpic:
		return f.epicAt(t, fromState)
	case entity.KindMilestone:
		return f.milestoneAt(t, fromState, opts)
	case entity.KindADR:
		return f.adrAt(t, fromState)
	case entity.KindGap:
		return f.gapAt(t, fromState)
	case entity.KindDecision:
		return f.decisionAt(t, fromState)
	case entity.KindContract:
		return f.contractAt(t, fromState)
	case spec.KindAC:
		return f.acAt(t, fromState, opts)
	}
	t.Fatalf("BringEntityToState: unsupported kind %q", k)
	return ""
}

func (f *CellFixture) epicAt(t *testing.T, fromState string) string {
	t.Helper()
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindEpic, "Cell-coverage Epic", testActor, verb.AddOptions{}))
	switch fromState {
	case entity.StatusProposed:
		return "E-0001"
	case entity.StatusActive:
		f.Must(verb.Promote(f.ctx, f.Tree(), "E-0001", entity.StatusActive, testActor, "", false, verb.PromoteOptions{}))
		return "E-0001"
	case entity.StatusDone:
		f.Must(verb.Promote(f.ctx, f.Tree(), "E-0001", entity.StatusActive, testActor, "", false, verb.PromoteOptions{}))
		f.Must(verb.Promote(f.ctx, f.Tree(), "E-0001", entity.StatusDone, testActor, "", false, verb.PromoteOptions{}))
		return "E-0001"
	case entity.StatusCancelled:
		f.Must(verb.Cancel(f.ctx, f.Tree(), "E-0001", testActor, "", false))
		return "E-0001"
	}
	t.Fatalf("epicAt: unsupported fromState %q", fromState)
	return ""
}

func (f *CellFixture) milestoneAt(t *testing.T, fromState string, opts BringOpts) string {
	t.Helper()
	parentTDD := opts.ParentTDD
	if parentTDD == "" {
		parentTDD = "required"
	}
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindEpic, "Cell-coverage Epic", testActor, verb.AddOptions{}))
	f.Must(verb.Promote(f.ctx, f.Tree(), "E-0001", entity.StatusActive, testActor, "", false, verb.PromoteOptions{}))
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindMilestone, "Cell-coverage Milestone", testActor, verb.AddOptions{EpicID: "E-0001", TDD: parentTDD}))
	switch fromState {
	case entity.StatusDraft:
		return "M-0001"
	case entity.StatusInProgress:
		f.Must(verb.Promote(f.ctx, f.Tree(), "M-0001", entity.StatusInProgress, testActor, "", false, verb.PromoteOptions{}))
		return "M-0001"
	case entity.StatusDone:
		f.Must(verb.Promote(f.ctx, f.Tree(), "M-0001", entity.StatusInProgress, testActor, "", false, verb.PromoteOptions{}))
		// Optionally seed populated ACs (BringOpts.ACs > 0). Each AC
		// goes through open -> met -> phase done to match the
		// "all-children-acs.status != open" predicate without
		// tripping acs-tdd-audit under tdd: required.
		for i := 0; i < opts.ACs; i++ {
			f.Must(verb.AddAC(f.ctx, f.Tree(), "M-0001", fmt.Sprintf("AC %d", i+1), testActor, nil))
			acID := fmt.Sprintf("M-0001/AC-%d", i+1)
			// Walk the TDD phase first (red -> green -> done) so
			// the subsequent status open -> met clears
			// acs-tdd-audit. Then advance status; --evidence
			// satisfies D-0005's required-evidence precondition.
			f.Must(verb.PromoteACPhase(f.ctx, f.Tree(), acID, entity.TDDPhaseGreen, testActor, "", false, nil))
			f.Must(verb.PromoteACPhase(f.ctx, f.Tree(), acID, entity.TDDPhaseDone, testActor, "", false, nil))
			f.Must(verb.Promote(f.ctx, f.Tree(), acID, entity.StatusMet, testActor, "evidence: covered by Test"+fmt.Sprint(i+1), false, verb.PromoteOptions{}))
		}
		f.Must(verb.Promote(f.ctx, f.Tree(), "M-0001", entity.StatusDone, testActor, "", false, verb.PromoteOptions{}))
		return "M-0001"
	case entity.StatusCancelled:
		f.Must(verb.Cancel(f.ctx, f.Tree(), "M-0001", testActor, "", false))
		return "M-0001"
	}
	t.Fatalf("milestoneAt: unsupported fromState %q", fromState)
	return ""
}

func (f *CellFixture) adrAt(t *testing.T, fromState string) string {
	t.Helper()
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindADR, "Cell-coverage ADR", testActor, verb.AddOptions{}))
	switch fromState {
	case entity.StatusProposed:
		return "ADR-0001"
	case entity.StatusAccepted:
		f.Must(verb.Promote(f.ctx, f.Tree(), "ADR-0001", entity.StatusAccepted, testActor, "", false, verb.PromoteOptions{}))
		return "ADR-0001"
	case entity.StatusRejected:
		f.Must(verb.Cancel(f.ctx, f.Tree(), "ADR-0001", testActor, "", false))
		return "ADR-0001"
	}
	t.Fatalf("adrAt: unsupported fromState %q", fromState)
	return ""
}

func (f *CellFixture) gapAt(t *testing.T, fromState string) string {
	t.Helper()
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindGap, "Cell-coverage Gap", testActor, verb.AddOptions{}))
	switch fromState {
	case entity.StatusOpen:
		return "G-0001"
	case entity.StatusWontfix:
		f.Must(verb.Cancel(f.ctx, f.Tree(), "G-0001", testActor, "", false))
		return "G-0001"
	case entity.StatusAddressed:
		// addressed requires a resolver (--by); construct a milestone
		// to serve as the addressed-by target.
		f.Must(verb.Add(f.ctx, f.Tree(), entity.KindEpic, "Resolver Epic", testActor, verb.AddOptions{}))
		f.Must(verb.Promote(f.ctx, f.Tree(), "E-0001", entity.StatusActive, testActor, "", false, verb.PromoteOptions{}))
		f.Must(verb.Add(f.ctx, f.Tree(), entity.KindMilestone, "Resolver Milestone", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
		f.Must(verb.Promote(f.ctx, f.Tree(), "G-0001", entity.StatusAddressed, testActor, "", false, verb.PromoteOptions{AddressedBy: []string{"M-0001"}}))
		return "G-0001"
	}
	t.Fatalf("gapAt: unsupported fromState %q", fromState)
	return ""
}

func (f *CellFixture) decisionAt(t *testing.T, fromState string) string {
	t.Helper()
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindDecision, "Cell-coverage Decision", testActor, verb.AddOptions{}))
	switch fromState {
	case entity.StatusProposed:
		return "D-0001"
	case entity.StatusAccepted:
		f.Must(verb.Promote(f.ctx, f.Tree(), "D-0001", entity.StatusAccepted, testActor, "", false, verb.PromoteOptions{}))
		return "D-0001"
	case entity.StatusRejected:
		f.Must(verb.Cancel(f.ctx, f.Tree(), "D-0001", testActor, "", false))
		return "D-0001"
	}
	t.Fatalf("decisionAt: unsupported fromState %q", fromState)
	return ""
}

func (f *CellFixture) contractAt(t *testing.T, fromState string) string {
	t.Helper()
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindContract, "Cell-coverage Contract", testActor, verb.AddOptions{}))
	switch fromState {
	case entity.StatusProposed:
		return "C-0001"
	case entity.StatusAccepted:
		// Contract proposed -> accepted normally requires ContractBind.
		// For fixture purposes, --force the transition (the binding
		// state is orthogonal to the cell-under-test for non-Contract
		// rule cells; Contract cells that depend on bindings are out
		// of M-0124's scope).
		f.Must(verb.Promote(f.ctx, f.Tree(), "C-0001", entity.StatusAccepted, testActor, "fixture-setup", true, verb.PromoteOptions{}))
		return "C-0001"
	case entity.StatusDeprecated:
		f.Must(verb.Promote(f.ctx, f.Tree(), "C-0001", entity.StatusAccepted, testActor, "fixture-setup", true, verb.PromoteOptions{}))
		f.Must(verb.Promote(f.ctx, f.Tree(), "C-0001", entity.StatusDeprecated, testActor, "fixture-setup", true, verb.PromoteOptions{}))
		return "C-0001"
	case entity.StatusRetired:
		f.Must(verb.Promote(f.ctx, f.Tree(), "C-0001", entity.StatusAccepted, testActor, "fixture-setup", true, verb.PromoteOptions{}))
		f.Must(verb.Promote(f.ctx, f.Tree(), "C-0001", entity.StatusDeprecated, testActor, "fixture-setup", true, verb.PromoteOptions{}))
		// deprecated -> retired via cancel (post-M-0131: state-aware
		// CancelTarget routes Contract.deprecated to retired).
		f.Must(verb.Cancel(f.ctx, f.Tree(), "C-0001", testActor, "", false))
		return "C-0001"
	case entity.StatusRejected:
		f.Must(verb.Cancel(f.ctx, f.Tree(), "C-0001", testActor, "", false))
		return "C-0001"
	}
	t.Fatalf("contractAt: unsupported fromState %q", fromState)
	return ""
}

func (f *CellFixture) acAt(t *testing.T, fromState string, opts BringOpts) string {
	t.Helper()
	parentTDD := opts.ParentTDD
	if parentTDD == "" {
		parentTDD = "required"
	}
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindEpic, "Cell-coverage Epic", testActor, verb.AddOptions{}))
	f.Must(verb.Promote(f.ctx, f.Tree(), "E-0001", entity.StatusActive, testActor, "", false, verb.PromoteOptions{}))
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindMilestone, "Cell-coverage Milestone", testActor, verb.AddOptions{EpicID: "E-0001", TDD: parentTDD}))
	f.Must(verb.Promote(f.ctx, f.Tree(), "M-0001", entity.StatusInProgress, testActor, "", false, verb.PromoteOptions{}))
	f.Must(verb.AddAC(f.ctx, f.Tree(), "M-0001", "Cell-coverage AC", testActor, nil))
	acID := "M-0001/AC-1"
	switch fromState {
	case entity.StatusOpen:
		return acID
	case entity.StatusMet:
		// Under tdd: required, AC.open -> met is illegal while
		// tdd_phase != done (acs-tdd-audit fires). Phase changes
		// go through PromoteACPhase (NOT Promote, which is for
		// status). Advance red -> green -> done, then status
		// open -> met.
		f.Must(verb.PromoteACPhase(f.ctx, f.Tree(), acID, entity.TDDPhaseGreen, testActor, "", false, nil))
		f.Must(verb.PromoteACPhase(f.ctx, f.Tree(), acID, entity.TDDPhaseDone, testActor, "", false, nil))
		f.Must(verb.Promote(f.ctx, f.Tree(), acID, entity.StatusMet, testActor, "evidence: covered by TestCellCoverage", false, verb.PromoteOptions{}))
		return acID
	case entity.StatusDeferred:
		f.Must(verb.Promote(f.ctx, f.Tree(), acID, entity.StatusDeferred, testActor, "deferred for test setup", false, verb.PromoteOptions{}))
		return acID
	case entity.StatusCancelled:
		f.Must(verb.Cancel(f.ctx, f.Tree(), acID, testActor, "fixture setup", false))
		return acID
	}
	t.Fatalf("acAt: unsupported fromState %q", fromState)
	return ""
}

// SatisfyPredicate mutates the fixture so that p holds against the
// entity identified by entityID. Each supported (Subject, Op) atom
// has a hand-rolled mutation; after applying, the helper re-loads
// the tree and calls spec.EvaluatePredicate to self-verify — the
// silent-drift guard. evalCtx is updated in place when the predicate
// is verb-arg-shaped (self.target-state, self.evidence).
func (f *CellFixture) SatisfyPredicate(t *testing.T, p spec.Predicate, entityID string, evalCtx *spec.EvalContext) {
	t.Helper()
	switch p.Subject {
	case "self.target-state":
		if p.Op == "==" {
			evalCtx.TargetState = p.Value
			return
		}
	case "self.evidence":
		switch p.Op {
		case "non-empty":
			evalCtx.Evidence = "fixture-provided evidence"
		case "==":
			evalCtx.Evidence = p.Value
		}
		return
	case "self.addressed_by":
		switch p.Op {
		case "non-empty":
			f.satisfyGapAddressed(t, entityID)
		case "==":
			// Default state after `aiwf add gap`: AddressedBy empty.
			// No mutation needed; the gap exists with the default.
		}
	case "self.superseded_by":
		switch p.Op {
		case "non-empty":
			f.satisfyADRSuperseded(t, entityID)
		case "==":
			// Default state after `aiwf add adr`: SupersededBy empty.
			// No mutation needed; the ADR exists with the default.
		}
	case "self.tdd_phase":
		// AC slot. Default phase after `aiwf add ac` is "red" under
		// tdd:required (matches != done by default) or "" under
		// tdd:none / advisory. For `== done`, walk the phase along
		// the TDD FSM via PromoteACPhase.
		f.populateEvalCtxAC(t, entityID, evalCtx)
		if p.Op == "==" && p.Value == entity.TDDPhaseDone {
			f.walkACToPhase(t, entityID, entity.TDDPhaseDone)
			f.populateEvalCtxAC(t, entityID, evalCtx)
		}
	case "parent.tdd":
		// Parent.tdd is fixed at BringEntityToState time. The driver
		// derives BringOpts.ParentTDD from this predicate before
		// fixture build, so by the time SatisfyPredicate runs the
		// milestone's TDD field already matches. The silent-drift
		// guard below confirms.
	case "any-child.status":
		// Epic with any non-terminal child. Default state after
		// epicAt + milestoneAt: epic has a draft milestone child
		// (non-terminal). Already satisfies any-child.status not in
		// terminal-set. Mutation: ensure a milestone child exists.
		f.ensureNonTerminalMilestoneChild(t, entityID)
	case "any-child-ac.status":
		// Milestone with at least one open AC. Mutation: add an AC.
		if p.Op == "==" && p.Value == entity.StatusOpen {
			f.ensureOpenACChild(t, entityID)
		}
	case "all-children-acs.status":
		// Milestone with zero ACs (vacuous) OR all ACs at met. The
		// zero-AC path is the default for Milestone.done per the
		// spec AntiRule — a fresh milestone with no ACs vacuously
		// satisfies "all children ACs have status != open." No
		// mutation needed.
	}
	// Self-verify: re-load tree, ensure the predicate now holds.
	tr := f.Tree()
	e, ctx, err := resolveForEval(tr, entityID, *evalCtx)
	if err != nil {
		t.Fatalf("SatisfyPredicate: resolveForEval %q: %v", entityID, err)
	}
	ok, evalErr := spec.EvaluatePredicate(p, e, tr, ctx)
	if evalErr != nil {
		t.Fatalf("SatisfyPredicate: post-mutation evaluate failed: %v", evalErr)
	}
	if !ok {
		t.Fatalf("SatisfyPredicate: fixture does not satisfy %+v after mutation (silent-drift guard fired)", p)
	}
	*evalCtx = ctx
}

func (f *CellFixture) satisfyGapAddressed(t *testing.T, gapID string) {
	t.Helper()
	// Build a resolver milestone (the gap's addressed_by target).
	tr := f.Tree()
	if tr.ByID("E-0001") == nil {
		f.Must(verb.Add(f.ctx, f.Tree(), entity.KindEpic, "Resolver Epic", testActor, verb.AddOptions{}))
		f.Must(verb.Promote(f.ctx, f.Tree(), "E-0001", entity.StatusActive, testActor, "", false, verb.PromoteOptions{}))
	}
	if tr.ByID("M-0001") == nil {
		f.Must(verb.Add(f.ctx, f.Tree(), entity.KindMilestone, "Resolver Milestone", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	}
	f.Must(verb.Promote(f.ctx, f.Tree(), gapID, entity.StatusAddressed, testActor, "", false, verb.PromoteOptions{AddressedBy: []string{"M-0001"}}))
}

// satisfyADRSuperseded supersedes the named ADR by a sibling ADR
// allocated for this purpose. Mirrors satisfyGapAddressed's shape:
// builds the support entity (a second ADR), then runs the verb that
// populates SupersededBy atomically with the transition.
//
// The sibling ADR must reach `accepted` first because the FSM only
// allows accepted → superseded — but the supersession target itself
// doesn't need to be accepted (the trailer just records the id). In
// practice though, the kernel doesn't gate target-state, so the
// sibling stays at proposed.
func (f *CellFixture) satisfyADRSuperseded(t *testing.T, adrID string) {
	t.Helper()
	// Allocate a sibling ADR to serve as the supersession target.
	// Determine its id by reading the aiwf-entity trailer the verb
	// stamps on its plan.
	res, err := verb.Add(f.ctx, f.Tree(), entity.KindADR, "Superseding ADR", testActor, verb.AddOptions{})
	f.Must(res, err)
	supersedingID := trailerValue(res.Plan, gitops.TrailerEntity)
	if supersedingID == "" {
		t.Fatalf("satisfyADRSuperseded: no %s trailer on Add plan", gitops.TrailerEntity)
	}
	f.Must(verb.Promote(f.ctx, f.Tree(), adrID, entity.StatusSuperseded, testActor, "", false, verb.PromoteOptions{SupersededBy: supersedingID}))
}

// trailerValue returns the value of the first trailer matching key in
// the plan, or "" if absent.
func trailerValue(plan *verb.Plan, key string) string {
	if plan == nil {
		return ""
	}
	for _, tr := range plan.Trailers {
		if tr.Key == key {
			return tr.Value
		}
	}
	return ""
}

// walkACToPhase advances the AC's tdd_phase to target via the TDD
// FSM (“ → red → green → {refactor, done}; refactor → done).
// Skips no-op transitions; tolerates the AC already being at target.
func (f *CellFixture) walkACToPhase(t *testing.T, compositeID, target string) {
	t.Helper()
	for {
		tr := f.Tree()
		_, ac, err := LookupComposite(tr, compositeID)
		if err != nil {
			t.Fatalf("walkACToPhase: %v", err)
		}
		if ac.TDDPhase == target {
			return
		}
		next := nextTDDPhaseTowards(ac.TDDPhase, target)
		if next == "" {
			t.Fatalf("walkACToPhase: no TDD path from %q to %q", ac.TDDPhase, target)
		}
		f.Must(verb.PromoteACPhase(f.ctx, f.Tree(), compositeID, next, testActor, "", false, nil))
	}
}

// nextTDDPhaseTowards returns the next phase to advance to given a
// current phase and a target. The TDD FSM is linear with a single
// branch at green ({refactor, done}); when target is done, the
// helper picks the green→done shortcut.
func nextTDDPhaseTowards(current, target string) string {
	switch current {
	case "":
		return entity.TDDPhaseRed
	case entity.TDDPhaseRed:
		return entity.TDDPhaseGreen
	case entity.TDDPhaseGreen:
		if target == entity.TDDPhaseRefactor {
			return entity.TDDPhaseRefactor
		}
		return entity.TDDPhaseDone
	case entity.TDDPhaseRefactor:
		return entity.TDDPhaseDone
	}
	return ""
}

func (f *CellFixture) ensureNonTerminalMilestoneChild(t *testing.T, epicID string) {
	t.Helper()
	tr := f.Tree()
	for _, e := range tr.Entities {
		if e.Parent == epicID && e.Kind == entity.KindMilestone && !entity.IsTerminal(entity.KindMilestone, e.Status) {
			return
		}
	}
	// No non-terminal milestone child found; add one.
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindMilestone, "Non-terminal child", testActor, verb.AddOptions{EpicID: epicID, TDD: "none"}))
}

func (f *CellFixture) ensureOpenACChild(t *testing.T, milestoneID string) {
	t.Helper()
	tr := f.Tree()
	m := tr.ByID(milestoneID)
	if m == nil {
		t.Fatalf("ensureOpenACChild: %q not found", milestoneID)
	}
	for _, ac := range m.ACs {
		if ac.Status == entity.StatusOpen {
			return
		}
	}
	f.Must(verb.AddAC(f.ctx, f.Tree(), milestoneID, "Open AC for predicate satisfaction", testActor, nil))
}

func (f *CellFixture) populateEvalCtxAC(t *testing.T, compositeID string, evalCtx *spec.EvalContext) {
	t.Helper()
	tr := f.Tree()
	_, ac, err := LookupComposite(tr, compositeID)
	if err != nil {
		t.Fatalf("populateEvalCtxAC: %v", err)
	}
	evalCtx.AC = ac
}

// LookupComposite resolves a composite id (M-NNNN/AC-N) to its
// parent milestone + the specific AC slot. Returns an error when
// the id shape is malformed, the milestone is missing, or the AC
// slot doesn't exist. Exported so per-cell drivers (M-0124's
// positive driver, M-0125's negative driver) reuse the same logic
// rather than carrying their own copies — closes G-0159.
func LookupComposite(tr *tree.Tree, compositeID string) (*entity.Entity, *entity.AcceptanceCriterion, error) {
	if !entity.IsCompositeID(compositeID) {
		return nil, nil, fmt.Errorf("not a composite id: %q", compositeID)
	}
	parts := strings.SplitN(compositeID, "/", 2)
	parentID, slot := parts[0], parts[1]
	m := tr.ByID(parentID)
	if m == nil {
		return nil, nil, fmt.Errorf("milestone %q not found", parentID)
	}
	for i := range m.ACs {
		if m.ACs[i].ID == slot {
			return m, &m.ACs[i], nil
		}
	}
	return nil, nil, fmt.Errorf("AC slot %q not found on %q", slot, parentID)
}

// resolveForEval picks the *entity.Entity that EvaluatePredicate
// reasons against given an id. For top-level ids the entity is the
// tree lookup; for composite ids it's the parent milestone and the
// evalCtx is augmented with the AC slot.
func resolveForEval(tr *tree.Tree, id string, evalCtx spec.EvalContext) (*entity.Entity, spec.EvalContext, error) {
	if entity.IsCompositeID(id) {
		_, ac, err := LookupComposite(tr, id)
		if err != nil {
			return nil, evalCtx, err
		}
		evalCtx.AC = ac
		parts := strings.SplitN(id, "/", 2)
		parent := tr.ByID(parts[0])
		return parent, evalCtx, nil
	}
	e := tr.ByID(id)
	if e == nil {
		return nil, evalCtx, fmt.Errorf("entity %q not found", id)
	}
	return e, evalCtx, nil
}
