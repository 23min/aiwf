package policies

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0124_PositiveDriver_LegalCells exercises every Legal cell in
// spec.Rules() through the real `aiwf` binary against a per-cell
// fixture. The driver derives the verb invocation (positional args,
// flags) from the cell's (Kind, FromState, Verb, Preconditions) plus
// the FSM's allowed targets, executes via subprocess, and asserts
// (a) exit 0, (b) the entity reached the expected post-state, and
// (c) HEAD carries the expected `aiwf-verb` and `aiwf-entity`
// trailers.
//
// Per-cell target derivation:
//
//   - `self.target-state == X` precondition pins target = X.
//   - Verb == "cancel" → target = entity.CancelTarget(kind, from).
//   - Promote on AC.open with `self.evidence non-empty` → target = met.
//   - Promote on Gap.open with `self.addressed_by non-empty` → target =
//     addressed (verb takes --by to populate the field atomically with
//     the transition).
//   - Promote on TDD-phase cells → next phase via tddPhaseTransitions.
//   - Otherwise: every reachable state in entity.AllowedTransitions
//     (multi-target cells like (epic, active, promote) expand into one
//     subtest per target).
//
// Fixture setup is in-process via cellcoverage; the cell-under-test
// runs via testutil.RunBin (the integration seam matters there).
func TestM0124_PositiveDriver_LegalCells(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	cases := enumerateLegalCases(t)
	if len(cases) == 0 {
		t.Fatal("no Legal cells enumerated from spec.Rules(); expected ~30")
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runPositiveCell(t, tc)
		})
	}
}

// positiveCase pins one driver row: one (Kind, FromState, Verb,
// target) tuple plus the precondition list that fixture setup must
// satisfy.
type positiveCase struct {
	name   string
	rule   spec.Rule
	target string
}

func enumerateLegalCases(t *testing.T) []positiveCase {
	t.Helper()
	var out []positiveCase
	rules := spec.Rules()
	for i := range rules {
		rule := rules[i]
		if rule.Outcome != spec.OutcomeLegal {
			continue
		}
		targets := deriveLegalTargets(t, rule)
		if len(targets) == 0 {
			t.Fatalf("no targets derived for Legal cell %+v", rule)
		}
		for _, tgt := range targets {
			name := caseName(rule, tgt)
			out = append(out, positiveCase{name: name, rule: rule, target: tgt})
		}
	}
	return out
}

func caseName(rule spec.Rule, target string) string {
	from := rule.FromState
	if from == "" {
		from = "empty"
	}
	name := fmt.Sprintf("%s-%s-%s-to-%s", rule.Kind, from, rule.Verb, target)
	// Append a precondition signature when distinct Legal cells share
	// the same (Kind, FromState, Verb, target) quadruple — happens for
	// AC.met where the cell splits on parent.tdd (G-0152). Without
	// this, the two split cells collide on t.Run subtest names.
	if sig := preconditionSignature(rule); sig != "" {
		name = name + "-" + sig
	}
	// Sanitize for t.Run name: no slashes (composite ids aren't in the
	// keys here, but defensive).
	name = strings.ReplaceAll(name, "/", "-")
	return name
}

// preconditionSignature builds a short identifier from preconditions
// that aren't already captured by the (Kind, FromState, Verb, target)
// quadruple — i.e., anything beyond `self.target-state`, `self.evidence`,
// `self.addressed_by`, and `self.superseded_by` (the verb-arg-shaped
// preconditions whose values either set the target or supply a flag).
// Returns "" when no disambiguating preconditions exist.
func preconditionSignature(rule spec.Rule) string {
	var parts []string
	for _, p := range rule.Preconditions {
		switch p.Subject {
		case "self.target-state", "self.evidence", "self.addressed_by", "self.superseded_by":
			continue
		}
		parts = append(parts, shortAtom(p))
	}
	return strings.Join(parts, "-")
}

func shortAtom(p spec.Predicate) string {
	subj := strings.ReplaceAll(p.Subject, "self.", "")
	subj = strings.ReplaceAll(subj, "parent.", "p")
	subj = strings.ReplaceAll(subj, "_", "")
	subj = strings.ReplaceAll(subj, "-", "")
	subj = strings.ReplaceAll(subj, ".", "")
	op := p.Op
	switch op {
	case "==":
		op = "eq"
	case "!=":
		op = "ne"
	case "∈":
		op = "in"
	case "∉":
		op = "notin"
	case "non-empty":
		op = "nonempty"
	}
	if p.Value == "" {
		return subj + op
	}
	val := strings.ReplaceAll(p.Value, "_", "")
	val = strings.ReplaceAll(val, "-", "")
	return subj + op + val
}

func deriveLegalTargets(t *testing.T, rule spec.Rule) []string {
	t.Helper()
	for _, p := range rule.Preconditions {
		if p.Subject == "self.target-state" && p.Op == "==" {
			return []string{p.Value}
		}
	}
	switch rule.Verb {
	case "cancel":
		// AC cancel is sub-kind; CancelTarget covers top-level kinds
		// only. The AC FSM lands cancel at "cancelled" regardless of
		// from-state.
		if rule.Kind == spec.KindAC {
			return []string{entity.StatusCancelled}
		}
		tgt := entity.CancelTarget(rule.Kind, rule.FromState)
		if tgt == "" {
			t.Fatalf("CancelTarget returned empty for (%s, %s)", rule.Kind, rule.FromState)
		}
		return []string{tgt}
	case "promote":
		return derivePromoteTargets(t, rule)
	}
	t.Fatalf("deriveLegalTargets: unsupported verb %q for cell %+v", rule.Verb, rule)
	return nil
}

func derivePromoteTargets(t *testing.T, rule spec.Rule) []string {
	t.Helper()
	switch rule.Kind {
	case entity.KindGap:
		for _, p := range rule.Preconditions {
			if p.Subject == "self.addressed_by" && p.Op == "non-empty" {
				return []string{entity.StatusAddressed}
			}
		}
	case spec.KindAC:
		for _, p := range rule.Preconditions {
			if p.Subject == "self.evidence" && p.Op == "non-empty" {
				return []string{entity.StatusMet}
			}
		}
	case spec.KindTDDPhase:
		return tddPhaseTargetsFromState(t, rule.FromState)
	}
	tgts := entity.AllowedTransitions(rule.Kind, rule.FromState)
	if len(tgts) == 0 {
		t.Fatalf("AllowedTransitions(%s, %s) returned empty for Legal cell %+v", rule.Kind, rule.FromState, rule)
	}
	return tgts
}

func tddPhaseTargetsFromState(t *testing.T, fromPhase string) []string {
	t.Helper()
	switch fromPhase {
	case "":
		return []string{entity.TDDPhaseRed}
	case entity.TDDPhaseRed:
		return []string{entity.TDDPhaseGreen}
	case entity.TDDPhaseGreen:
		return []string{entity.TDDPhaseRefactor, entity.TDDPhaseDone}
	case entity.TDDPhaseRefactor:
		return []string{entity.TDDPhaseDone}
	}
	t.Fatalf("tddPhaseTargetsFromState: unsupported phase %q", fromPhase)
	return nil
}

func runPositiveCell(t *testing.T, tc positiveCase) {
	t.Helper()
	f := cellcoverage.NewCellFixture(t)
	// Derive BringOpts from preconditions BEFORE fixture build —
	// parent.tdd is set at milestone-add time and the fixture's
	// default ("required") may not match the cell's precondition.
	opts := deriveBringOpts(tc.rule)
	id := bringEntityForCell(t, f, tc.rule, opts)

	// Materialize each precondition. Three shapes:
	//
	//  - Verb-arg-only (target-state, evidence): the cell-under-test
	//    supplies the value via positional arg / flag at run-time; no
	//    fixture mutation needed. Recorded in evalCtx for the post-
	//    state assertion.
	//  - Verb-arg-shaped field (addressed_by, superseded_by): the
	//    field is populated atomically with the transition via a
	//    flag (--by, --superseded-by). The fixture only needs to
	//    build the support entity the flag references; the driver
	//    appends the flag to verb args.
	//  - Fixture-state field (parent.tdd, any-child.status,
	//    any-child-ac.status, all-children-acs.status,
	//    self.tdd_phase): SatisfyPredicate mutates the fixture so
	//    the predicate holds before the cell-under-test runs.
	//    parent.tdd was already materialized at fixture-build time
	//    above; SatisfyPredicate's silent-drift guard confirms.
	evalCtx := spec.EvalContext{}
	extras := extraArgs{}
	for _, p := range tc.rule.Preconditions {
		switch p.Subject {
		case "self.target-state", "self.evidence":
			// Verb-arg only; no fixture mutation required.
		case "self.addressed_by":
			if p.Op == "non-empty" {
				extras.resolverID = ensureGapResolver(t, f)
			}
		case "self.superseded_by":
			if p.Op == "non-empty" {
				extras.supersedingID = ensureSupersedingADR(t, f)
			}
		default:
			f.SatisfyPredicate(t, p, id, &evalCtx)
		}
	}

	args := buildVerbArgs(t, tc, id, extras)
	out, err := testutil.RunBin(t, f.Root, "", nil, args...)
	if err != nil {
		t.Fatalf("aiwf %v failed:\nerr: %v\nout:\n%s", args, err, out)
	}

	assertPostState(t, f, tc, id)
	assertHeadTrailers(t, f.Root, tc.rule.Verb, id)
}

// extraArgs collects per-cell verb-arg flags the driver builds from
// the fixture. resolverID populates `--by` for gap.addressed;
// supersedingID populates `--superseded-by` for adr.superseded.
type extraArgs struct {
	resolverID    string
	supersedingID string
}

// deriveBringOpts inspects the rule's preconditions for any that
// must hold at fixture-creation time (today: parent.tdd, which sets
// the milestone's TDD field at `aiwf add milestone` and isn't
// changed afterward). Other preconditions are materialized after
// fixture build via SatisfyPredicate / verb-arg flags.
func deriveBringOpts(rule spec.Rule) cellcoverage.BringOpts {
	opts := cellcoverage.BringOpts{}
	for _, p := range rule.Preconditions {
		if p.Subject != "parent.tdd" {
			continue
		}
		switch p.Op {
		case "==":
			opts.ParentTDD = p.Value
		case "!=":
			// Pick any value other than the rejected one. The TDD
			// policy domain is {required, advisory, none}; rejecting
			// "required" leaves either of the others. "advisory" is
			// the closest operator-equivalent (still advisory-tracking,
			// just not blocking on the audit).
			if p.Value == "required" {
				opts.ParentTDD = "advisory"
			} else {
				opts.ParentTDD = "required"
			}
		}
	}
	return opts
}

// ensureSupersedingADR allocates a sibling ADR to serve as the
// --superseded-by target. Returns the new ADR's id by reading the
// aiwf-entity trailer the Add verb stamps on its plan.
func ensureSupersedingADR(t *testing.T, f *cellcoverage.CellFixture) string {
	t.Helper()
	ctx := context.Background()
	res, err := verb.Add(ctx, f.Tree(), entity.KindADR, "Superseding ADR", "human/test", verb.AddOptions{})
	f.Must(res, err)
	for _, tr := range res.Plan.Trailers {
		if tr.Key == gitops.TrailerEntity {
			return tr.Value
		}
	}
	t.Fatal("ensureSupersedingADR: no aiwf-entity trailer on Add plan")
	return ""
}

// bringEntityForCell brings the fixture to the (Kind, FromState) the
// rule reasons about. For top-level kinds + AC this is a direct
// BringEntityToState call honoring the caller-supplied opts (today:
// parent.tdd derived from preconditions). For TDD-phase cells the
// entity is the AC slot; setup builds an AC under a tdd:required
// milestone and advances its phase to FromState (or leaves it absent
// for the "" arm).
func bringEntityForCell(t *testing.T, f *cellcoverage.CellFixture, rule spec.Rule, opts cellcoverage.BringOpts) string {
	t.Helper()
	if rule.Kind == spec.KindTDDPhase {
		return bringTDDPhaseAC(t, f, rule.FromState)
	}
	return f.BringEntityToState(t, rule.Kind, rule.FromState, opts)
}

func bringTDDPhaseAC(t *testing.T, f *cellcoverage.CellFixture, fromPhase string) string {
	t.Helper()
	ctx := context.Background()
	if fromPhase == "" {
		// AC under tdd:none milestone has no auto-seeded phase.
		// The cell-under-test verb is `promote --phase red`.
		f.Must(verb.Add(ctx, f.Tree(), entity.KindEpic, "TDD Epic", "human/test", verb.AddOptions{}))
		f.Must(verb.Promote(ctx, f.Tree(), "E-0001", entity.StatusActive, "human/test", "", false, verb.PromoteOptions{}))
		f.Must(verb.Add(ctx, f.Tree(), entity.KindMilestone, "TDD Milestone (none)", "human/test", verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
		f.Must(verb.Promote(ctx, f.Tree(), "M-0001", entity.StatusInProgress, "human/test", "", false, verb.PromoteOptions{}))
		f.Must(verb.AddAC(ctx, f.Tree(), "M-0001", "TDD-phase AC", "human/test", nil))
		return "M-0001/AC-1"
	}
	// red/green/refactor: AC under tdd:required milestone, advance phases.
	f.Must(verb.Add(ctx, f.Tree(), entity.KindEpic, "TDD Epic", "human/test", verb.AddOptions{}))
	f.Must(verb.Promote(ctx, f.Tree(), "E-0001", entity.StatusActive, "human/test", "", false, verb.PromoteOptions{}))
	f.Must(verb.Add(ctx, f.Tree(), entity.KindMilestone, "TDD Milestone", "human/test", verb.AddOptions{EpicID: "E-0001", TDD: "required"}))
	f.Must(verb.Promote(ctx, f.Tree(), "M-0001", entity.StatusInProgress, "human/test", "", false, verb.PromoteOptions{}))
	f.Must(verb.AddAC(ctx, f.Tree(), "M-0001", "TDD-phase AC", "human/test", nil))
	acID := "M-0001/AC-1"
	// AC starts at red under tdd:required. Walk to fromPhase via
	// PromoteACPhase.
	advanceTo := []string{}
	switch fromPhase {
	case entity.TDDPhaseRed:
		// already there
	case entity.TDDPhaseGreen:
		advanceTo = []string{entity.TDDPhaseGreen}
	case entity.TDDPhaseRefactor:
		advanceTo = []string{entity.TDDPhaseGreen, entity.TDDPhaseRefactor}
	default:
		t.Fatalf("bringTDDPhaseAC: unsupported fromPhase %q", fromPhase)
	}
	for _, p := range advanceTo {
		f.Must(verb.PromoteACPhase(ctx, f.Tree(), acID, p, "human/test", "", false, nil))
	}
	return acID
}

func ensureGapResolver(t *testing.T, f *cellcoverage.CellFixture) string {
	t.Helper()
	ctx := context.Background()
	tr := f.Tree()
	if tr.ByID("E-0001") == nil {
		f.Must(verb.Add(ctx, f.Tree(), entity.KindEpic, "Resolver Epic", "human/test", verb.AddOptions{}))
		f.Must(verb.Promote(ctx, f.Tree(), "E-0001", entity.StatusActive, "human/test", "", false, verb.PromoteOptions{}))
	}
	if tr.ByID("M-0001") == nil {
		f.Must(verb.Add(ctx, f.Tree(), entity.KindMilestone, "Resolver Milestone", "human/test", verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	}
	return "M-0001"
}

// buildVerbArgs constructs the CLI args for the cell-under-test.
//
//	promote <id> <target>                                  (default)
//	promote <id> <target> --by <resolverID>                (gap.addressed)
//	promote <id> <target> --superseded-by <id>             (adr.superseded)
//	promote <id> --phase <target>                          (TDD-phase)
//	cancel  <id>                                           (cancel)
func buildVerbArgs(t *testing.T, tc positiveCase, id string, extras extraArgs) []string {
	t.Helper()
	switch tc.rule.Verb {
	case "cancel":
		return []string{"cancel", id}
	case "promote":
		if tc.rule.Kind == spec.KindTDDPhase {
			return []string{"promote", id, "--phase", tc.target}
		}
		args := []string{"promote", id, tc.target}
		if extras.resolverID != "" {
			args = append(args, "--by", extras.resolverID)
		}
		if extras.supersedingID != "" {
			args = append(args, "--superseded-by", extras.supersedingID)
		}
		return args
	}
	t.Fatalf("buildVerbArgs: unsupported verb %q", tc.rule.Verb)
	return nil
}

func assertPostState(t *testing.T, f *cellcoverage.CellFixture, tc positiveCase, id string) {
	t.Helper()
	tr := f.Tree()
	if tc.rule.Kind == spec.KindTDDPhase {
		_, ac, err := lookupCompositeForDriver(tr, id)
		if err != nil {
			t.Fatalf("lookup composite %q after verb: %v", id, err)
		}
		if ac.TDDPhase != tc.target {
			t.Errorf("AC tdd_phase = %q, want %q", ac.TDDPhase, tc.target)
		}
		return
	}
	if entity.IsCompositeID(id) {
		_, ac, err := lookupCompositeForDriver(tr, id)
		if err != nil {
			t.Fatalf("lookup composite %q after verb: %v", id, err)
		}
		if ac.Status != tc.target {
			t.Errorf("AC status = %q, want %q", ac.Status, tc.target)
		}
		return
	}
	e := tr.ByID(id)
	if e == nil {
		t.Fatalf("entity %q not in tree after verb", id)
	}
	if e.Status != tc.target {
		t.Errorf("entity %q status = %q, want %q", id, e.Status, tc.target)
	}
}

func assertHeadTrailers(t *testing.T, root, wantVerb, wantEntity string) {
	t.Helper()
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var sawVerb, sawEntity bool
	for _, trailer := range tr {
		if trailer.Key == gitops.TrailerVerb && trailer.Value == wantVerb {
			sawVerb = true
		}
		if trailer.Key == gitops.TrailerEntity && trailer.Value == wantEntity {
			sawEntity = true
		}
	}
	if !sawVerb {
		t.Errorf("HEAD missing aiwf-verb: %q trailer; got %v", wantVerb, tr)
	}
	if !sawEntity {
		t.Errorf("HEAD missing aiwf-entity: %q trailer; got %v", wantEntity, tr)
	}
}

// lookupCompositeForDriver is a thin wrapper around tree lookup that
// resolves M-NNNN/AC-N → (parent, AC slot). Mirrors the cellcoverage
// helper but lives here so the driver doesn't depend on cellcoverage's
// internals.
func lookupCompositeForDriver(tr interface {
	ByID(string) *entity.Entity
}, compositeID string,
) (*entity.Entity, *entity.AcceptanceCriterion, error) {
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
