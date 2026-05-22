package spec

import (
	"fmt"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// EvalContext carries the bits of state that don't live on the
// passed-in *entity.Entity:
//
//	TargetState — the second positional arg of `aiwf promote <id>
//	              <new-status>` (or AC's `--phase` value). Used by the
//	              "self.target-state" predicate.
//	Evidence    — the `--evidence` flag value on
//	              `aiwf promote M-NNN/AC-N met --evidence "..."`. Used
//	              by the "self.evidence" predicate.
//	AC          — populated when the rule's Kind is KindAC and the
//	              predicate references "self.tdd_phase" (which is a
//	              field on AcceptanceCriterion, not on Entity). ACs
//	              are sub-elements of their parent milestone, not
//	              standalone tree entities; the caller resolves the
//	              specific AC slot and hands it in.
//
// Widening EvalContext requires widening the Subject vocabulary,
// which is itself constrained per the M-0123 body's predicate-
// vocabulary commitment.
type EvalContext struct {
	TargetState string
	Evidence    string
	AC          *entity.AcceptanceCriterion
}

// EvaluatePredicate reports whether p holds against e in t, with
// verb-invocation context ctx. The closed (Subject, Op) vocabulary
// covered here is exactly what appears in Rules() at the M-0124 ship
// boundary:
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
// Unknown Subject, Op, or named-set Value return a typed error so a
// future rule that widens the vocabulary fails the matching atom's
// per-cell test with a clear "unknown subject/op/named set" message
// — discoverable, fixable.
//
// The named-set vocabulary is its own closed table; today it carries
// exactly one entry ("milestone-terminal-set" = {done, cancelled}).
// New named sets land here as the spec grows.
func EvaluatePredicate(p Predicate, e *entity.Entity, t *tree.Tree, ctx EvalContext) (bool, error) {
	switch p.Subject {
	case "self.target-state":
		return cmpString(p.Op, ctx.TargetState, p.Value)
	case "self.evidence":
		return cmpString(p.Op, ctx.Evidence, p.Value)
	case "self.addressed_by":
		return cmpStringSlice(p.Op, e.AddressedBy, p.Value)
	case "self.tdd_phase":
		if ctx.AC == nil {
			return false, fmt.Errorf("evaluate predicate: self.tdd_phase requires ctx.AC (AC predicate on a milestone slot)")
		}
		return cmpString(p.Op, ctx.AC.TDDPhase, p.Value)
	case "parent.tdd":
		parent := t.ByID(e.Parent)
		if parent == nil {
			return false, nil
		}
		return cmpString(p.Op, parent.TDD, p.Value)
	case "any-child.status":
		return anyChild(t, e, func(c *entity.Entity) (bool, error) {
			return cmpStatusNamedSet(p.Op, c.Status, p.Value)
		})
	case "any-child-ac.status":
		return anyChildAC(e, func(ac entity.AcceptanceCriterion) (bool, error) {
			return cmpString(p.Op, ac.Status, p.Value)
		})
	case "all-children-acs.status":
		return allChildrenACs(e, func(ac entity.AcceptanceCriterion) (bool, error) {
			return cmpString(p.Op, ac.Status, p.Value)
		})
	}
	return false, fmt.Errorf("evaluate predicate: unknown subject %q", p.Subject)
}

// cmpString applies an Op to two strings. The closed set of Ops at
// this surface is {==, !=, non-empty}. == "" is the empty-equality
// shape used to assert "field is empty"; non-empty is its symmetric.
func cmpString(op, got, want string) (bool, error) {
	switch op {
	case "==":
		return got == want, nil
	case "!=":
		return got != want, nil
	case "non-empty":
		return got != "", nil
	}
	return false, fmt.Errorf("evaluate predicate: unknown op %q for string subject", op)
}

// cmpStringSlice is the string-slice variant: "==" with want "" means
// the slice is empty; non-empty means len > 0. The Rules() surface
// uses these two forms only.
func cmpStringSlice(op string, got []string, want string) (bool, error) {
	switch op {
	case "==":
		if want == "" {
			return len(got) == 0, nil
		}
		// Slice equality to a non-empty string isn't a shape Rules()
		// uses today; reject explicitly to avoid silent semantics.
		return false, fmt.Errorf("evaluate predicate: == comparison on slice subject requires empty Value (got %q)", want)
	case "non-empty":
		return len(got) > 0, nil
	}
	return false, fmt.Errorf("evaluate predicate: unknown op %q for slice subject", op)
}

// cmpStatusNamedSet applies the ∈ / ∉ ops to a status value against
// a named set. The named-set table is closed; new entries land here
// as the spec grows.
func cmpStatusNamedSet(op, status, setName string) (bool, error) {
	set, ok := namedStatusSets[setName]
	if !ok {
		return false, fmt.Errorf("evaluate predicate: unknown named set %q", setName)
	}
	switch op {
	case "∈":
		return setContains(set, status), nil
	case "∉":
		return !setContains(set, status), nil
	}
	return false, fmt.Errorf("evaluate predicate: unknown op %q for named-set subject", op)
}

// namedStatusSets is the closed table of named sets referenced by
// Rules(). Each name maps to the set of statuses it covers. New
// rules that introduce a new named-set Value add their entry here.
//
// milestone-terminal-set — the set of Milestone statuses that are
// terminal (no outgoing FSM edges). Used by epic-cancel-non-terminal-
// children: cancelling an epic is illegal if any child milestone is
// outside this set.
var namedStatusSets = map[string][]string{
	"milestone-terminal-set": {"done", "cancelled"},
}

func setContains(set []string, want string) bool {
	for _, s := range set {
		if s == want {
			return true
		}
	}
	return false
}

// anyChild iterates the tree's entities whose Parent == e.ID and
// returns the OR of pred over them. The predicate receives the child
// entity; its return value short-circuits on the first true.
func anyChild(t *tree.Tree, e *entity.Entity, pred func(*entity.Entity) (bool, error)) (bool, error) {
	for _, c := range t.Entities {
		if c.Parent != e.ID {
			continue
		}
		ok, err := pred(c)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// anyChildAC iterates e's ACs slice (an AC is structurally a sub-
// element of its milestone, not a separate tree entity) and returns
// the OR of pred over them.
func anyChildAC(e *entity.Entity, pred func(entity.AcceptanceCriterion) (bool, error)) (bool, error) {
	for _, ac := range e.ACs {
		ok, err := pred(ac)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// allChildrenACs iterates e's ACs slice and returns the AND of pred.
// A milestone with zero ACs satisfies any all-ACs predicate
// vacuously (the empty universal); whether that matches the rule's
// intent is the rule-author's concern, not the evaluator's.
func allChildrenACs(e *entity.Entity, pred func(entity.AcceptanceCriterion) (bool, error)) (bool, error) {
	for _, ac := range e.ACs {
		ok, err := pred(ac)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}
