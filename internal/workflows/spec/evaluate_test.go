package spec

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestEvaluatePredicate covers the closed (Subject, Op, Value)
// vocabulary that actually appears in Rules(). The 10 unique atoms
// harvested from rules.go at M-0124 design time:
//
//	self.target-state == <state>
//	self.evidence non-empty
//	self.evidence == ""
//	self.addressed_by non-empty
//	self.addressed_by == ""
//	self.tdd_phase != <phase>
//	parent.tdd == <policy>
//	any-child.status ∉ <named-set>
//	any-child-ac.status == <state>
//	all-children-acs.status != <state>
//
// Each atom gets a positive and a negative case. Unknown Subject /
// Op / named-set Value return an error so the evaluator's surface
// matches the spec's closed vocabulary explicitly — adding a new
// atom in rules.go without wiring it here fails the matching atom
// test with a clear "unknown" error.
func TestEvaluatePredicate(t *testing.T) {
	t.Parallel()

	// Reusable entities. Each test references the ones it needs by
	// pointer; the tree holds the union.
	epicProposed := &entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Status: "proposed"}
	milestoneDraft := &entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Status: "draft", Parent: "E-0001", TDD: "required"}
	milestoneAdvisory := &entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Status: "draft", Parent: "E-0001", TDD: "advisory"}
	milestoneDone := &entity.Entity{ID: "M-0003", Kind: entity.KindMilestone, Status: "done", Parent: "E-0001"}
	milestoneCancelled := &entity.Entity{ID: "M-0004", Kind: entity.KindMilestone, Status: "cancelled", Parent: "E-0001"}
	gapWithResolver := &entity.Entity{ID: "G-0001", Kind: entity.KindGap, Status: "open", AddressedBy: []string{"M-0001"}}
	gapNoResolver := &entity.Entity{ID: "G-0002", Kind: entity.KindGap, Status: "open"}

	milestoneWithOpenAC := &entity.Entity{
		ID: "M-0010", Kind: entity.KindMilestone, Status: "in_progress", Parent: "E-0001", TDD: "required",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Status: "open", TDDPhase: "red"},
			{ID: "AC-2", Status: "met", TDDPhase: "done"},
		},
	}
	milestoneAllACsClosed := &entity.Entity{
		ID: "M-0011", Kind: entity.KindMilestone, Status: "in_progress", Parent: "E-0001", TDD: "required",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Status: "met", TDDPhase: "done"},
			{ID: "AC-2", Status: "deferred", TDDPhase: "done"},
		},
	}

	tr := &tree.Tree{
		Root: "/test",
		Entities: []*entity.Entity{
			epicProposed,
			milestoneDraft, milestoneAdvisory, milestoneDone, milestoneCancelled,
			gapWithResolver, gapNoResolver,
			milestoneWithOpenAC, milestoneAllACsClosed,
		},
	}

	cases := []struct {
		name    string
		pred    Predicate
		entity  *entity.Entity
		ctx     EvalContext
		want    bool
		wantErr string // non-empty means expect error containing this substring
	}{
		// self.target-state == <state>
		{"target-state-eq-positive", Predicate{Subject: "self.target-state", Op: "==", Value: "deferred"}, milestoneDraft, EvalContext{TargetState: "deferred"}, true, ""},
		{"target-state-eq-negative", Predicate{Subject: "self.target-state", Op: "==", Value: "deferred"}, milestoneDraft, EvalContext{TargetState: "met"}, false, ""},

		// self.evidence non-empty / == ""
		{"evidence-non-empty-positive", Predicate{Subject: "self.evidence", Op: "non-empty"}, milestoneDraft, EvalContext{Evidence: "covered by TestFoo"}, true, ""},
		{"evidence-non-empty-negative", Predicate{Subject: "self.evidence", Op: "non-empty"}, milestoneDraft, EvalContext{Evidence: ""}, false, ""},
		{"evidence-empty-positive", Predicate{Subject: "self.evidence", Op: "==", Value: ""}, milestoneDraft, EvalContext{Evidence: ""}, true, ""},
		{"evidence-empty-negative", Predicate{Subject: "self.evidence", Op: "==", Value: ""}, milestoneDraft, EvalContext{Evidence: "x"}, false, ""},

		// self.addressed_by non-empty / == ""
		{"addressed_by-non-empty-positive", Predicate{Subject: "self.addressed_by", Op: "non-empty"}, gapWithResolver, EvalContext{}, true, ""},
		{"addressed_by-non-empty-negative", Predicate{Subject: "self.addressed_by", Op: "non-empty"}, gapNoResolver, EvalContext{}, false, ""},
		{"addressed_by-empty-positive", Predicate{Subject: "self.addressed_by", Op: "==", Value: ""}, gapNoResolver, EvalContext{}, true, ""},
		{"addressed_by-empty-negative", Predicate{Subject: "self.addressed_by", Op: "==", Value: ""}, gapWithResolver, EvalContext{}, false, ""},

		// self.tdd_phase != <phase> — fires on an AC, which is a
		// sub-element of a milestone's ACs slice, not a tree entity.
		// EvalContext.AC carries the specific slot the caller is
		// reasoning about; without it the evaluator returns an
		// error rather than silently treating the predicate as
		// vacuously true.
		{"tdd_phase-neq-positive", Predicate{Subject: "self.tdd_phase", Op: "!=", Value: "done"}, milestoneWithOpenAC, EvalContext{AC: &milestoneWithOpenAC.ACs[0]}, true, ""},
		{"tdd_phase-neq-negative", Predicate{Subject: "self.tdd_phase", Op: "!=", Value: "done"}, milestoneAllACsClosed, EvalContext{AC: &milestoneAllACsClosed.ACs[0]}, false, ""},
		{"tdd_phase-missing-ac-ctx", Predicate{Subject: "self.tdd_phase", Op: "!=", Value: "done"}, milestoneWithOpenAC, EvalContext{}, false, "requires ctx.AC"},

		// parent.tdd == <policy>
		{"parent-tdd-required-positive", Predicate{Subject: "parent.tdd", Op: "==", Value: "required"}, &entity.Entity{ID: "AC-x", Parent: "M-0001"}, EvalContext{}, true, ""},
		{"parent-tdd-required-negative", Predicate{Subject: "parent.tdd", Op: "==", Value: "required"}, &entity.Entity{ID: "AC-x", Parent: "M-0002"}, EvalContext{}, false, ""},

		// any-child.status ∉ <named-set> (milestone-terminal-set = {done, cancelled})
		{"any-child-status-not-in-set-positive", Predicate{Subject: "any-child.status", Op: "∉", Value: "milestone-terminal-set"}, epicProposed, EvalContext{}, true, ""},
		{"any-child-status-not-in-set-negative", Predicate{Subject: "any-child.status", Op: "∉", Value: "milestone-terminal-set"}, &entity.Entity{ID: "E-0099", Kind: entity.KindEpic, Status: "active"}, EvalContext{}, false, ""},

		// any-child-ac.status == <state>
		{"any-child-ac-eq-positive", Predicate{Subject: "any-child-ac.status", Op: "==", Value: "open"}, milestoneWithOpenAC, EvalContext{}, true, ""},
		{"any-child-ac-eq-negative", Predicate{Subject: "any-child-ac.status", Op: "==", Value: "open"}, milestoneAllACsClosed, EvalContext{}, false, ""},

		// all-children-acs.status != <state>
		{"all-children-acs-neq-positive", Predicate{Subject: "all-children-acs.status", Op: "!=", Value: "open"}, milestoneAllACsClosed, EvalContext{}, true, ""},
		{"all-children-acs-neq-negative", Predicate{Subject: "all-children-acs.status", Op: "!=", Value: "open"}, milestoneWithOpenAC, EvalContext{}, false, ""},

		// Unknown Subject / Op / named-set Value
		{"unknown-subject", Predicate{Subject: "self.nonsense", Op: "==", Value: "x"}, milestoneDraft, EvalContext{}, false, "unknown subject"},
		{"unknown-op-string", Predicate{Subject: "self.target-state", Op: "≡", Value: "x"}, milestoneDraft, EvalContext{TargetState: "x"}, false, "unknown op"},
		{"unknown-op-slice", Predicate{Subject: "self.addressed_by", Op: "≡", Value: ""}, gapNoResolver, EvalContext{}, false, "unknown op"},
		{"unknown-op-named-set", Predicate{Subject: "any-child.status", Op: "≡", Value: "milestone-terminal-set"}, epicProposed, EvalContext{}, false, "unknown op"},
		{"unknown-named-set", Predicate{Subject: "any-child.status", Op: "∉", Value: "no-such-set"}, epicProposed, EvalContext{}, false, "unknown named set"},
		// Slice == comparison to non-empty value is explicitly
		// rejected (only "==" to "" is the supported empty-check).
		{"slice-eq-non-empty-value", Predicate{Subject: "self.addressed_by", Op: "==", Value: "M-0001"}, gapWithResolver, EvalContext{}, false, "requires empty Value"},
		// ∈ over a named set (Rules() uses ∉ today but ∈ is part of
		// the closed Op vocabulary; the evaluator supports both for
		// future widening).
		// epicProposed (E-0001) has milestoneDone among its children,
		// so any-child.status ∈ {done, cancelled} is true.
		{"in-named-set-positive", Predicate{Subject: "any-child.status", Op: "∈", Value: "milestone-terminal-set"}, epicProposed, EvalContext{}, true, ""},
		// An epic with no children: any-child is the empty existential
		// (false). Used both for the ∈ negative case here and as
		// the contrast for the ∉ test above.
		{"in-named-set-negative", Predicate{Subject: "any-child.status", Op: "∈", Value: "milestone-terminal-set"}, &entity.Entity{ID: "E-0099", Kind: entity.KindEpic, Status: "active"}, EvalContext{}, false, ""},
		// parent.<field> with a parent that doesn't exist returns
		// false (the predicate doesn't hold rather than erroring —
		// "no parent" is a runtime fact, not a vocabulary error).
		{"parent-tdd-missing-parent", Predicate{Subject: "parent.tdd", Op: "==", Value: "required"}, &entity.Entity{ID: "X", Parent: "M-9999"}, EvalContext{}, false, ""},
		// Error propagation through anyChildAC + allChildrenACs +
		// anyChild: an unknown op inside the per-child predicate
		// surfaces as an error rather than silently being treated
		// as "no match." Without these the propagation arms in the
		// children-iterator helpers are unexercised.
		{"any-child-ac-unknown-op", Predicate{Subject: "any-child-ac.status", Op: "≡", Value: "open"}, milestoneWithOpenAC, EvalContext{}, false, "unknown op"},
		{"all-children-acs-unknown-op", Predicate{Subject: "all-children-acs.status", Op: "≡", Value: "open"}, milestoneWithOpenAC, EvalContext{}, false, "unknown op"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := EvaluatePredicate(tc.pred, tc.entity, tr, tc.ctx)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got nil (result %v)", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("EvaluatePredicate(%+v, %s, ctx=%+v) = %v, want %v", tc.pred, tc.entity.ID, tc.ctx, got, tc.want)
			}
		})
	}
}
