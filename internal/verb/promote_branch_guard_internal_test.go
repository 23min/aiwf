package verb

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// promote_branch_guard_internal_test.go pins expectedActivationBranch's
// three fail-shut branches (M-0252/AC-2, promote_branch_guard.go:83/
// 87/91) directly against hand-built fixtures. These are pure,
// in-memory decision-logic branches over the milestone leg's parent
// resolution — no I/O, no git plumbing — and each needs a milestone
// shape verb.Add's own validation would refuse to create (a milestone
// with no parent, a dangling parent reference, a non-epic parent), so
// they're driven directly rather than through the public verb.Add /
// verb.Promote surface promote_branch_guard_test.go (package
// verb_test) already covers for the reachable-through-the-API cases.

// TestExpectedActivationBranch_MilestoneNoParent covers line 83: a
// milestone with an empty Parent field resolves to an unresolvable
// expectation (fail-shut, not a violation).
func TestExpectedActivationBranch_MilestoneNoParent(t *testing.T) {
	t.Parallel()
	m := &entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Status: entity.StatusDraft, Parent: ""}
	tr := &tree.Tree{Entities: []*entity.Entity{m}}

	branch, ok := expectedActivationBranch(tr, m, entity.StatusInProgress)
	if ok {
		t.Errorf("expectedActivationBranch = (%q, true), want (_, false) for a parentless milestone", branch)
	}
	if branch != "" {
		t.Errorf("branch = %q, want empty", branch)
	}
}

// TestExpectedActivationBranch_MilestoneParentLookupFails covers line
// 87 across both of its disjuncts: a Parent id that doesn't resolve
// in the tree at all, and a Parent id that resolves to a non-epic
// entity. Both are fail-shut, not a violation.
func TestExpectedActivationBranch_MilestoneParentLookupFails(t *testing.T) {
	t.Parallel()

	t.Run("dangling parent reference", func(t *testing.T) {
		t.Parallel()
		m := &entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Status: entity.StatusDraft, Parent: "E-9999"}
		tr := &tree.Tree{Entities: []*entity.Entity{m}}

		branch, ok := expectedActivationBranch(tr, m, entity.StatusInProgress)
		if ok {
			t.Errorf("expectedActivationBranch = (%q, true), want (_, false) for a dangling parent reference", branch)
		}
		if branch != "" {
			t.Errorf("branch = %q, want empty", branch)
		}
	})

	t.Run("parent resolves to a non-epic entity", func(t *testing.T) {
		t.Parallel()
		gap := &entity.Entity{ID: "G-0001", Kind: entity.KindGap, Status: entity.StatusOpen, Path: "work/gaps/G-0001-foo.md"}
		m := &entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Status: entity.StatusDraft, Parent: "G-0001"}
		tr := &tree.Tree{Entities: []*entity.Entity{gap, m}}

		branch, ok := expectedActivationBranch(tr, m, entity.StatusInProgress)
		if ok {
			t.Errorf("expectedActivationBranch = (%q, true), want (_, false) when Parent resolves to a non-epic entity", branch)
		}
		if branch != "" {
			t.Errorf("branch = %q, want empty", branch)
		}
	})
}

// TestExpectedActivationBranch_ParentEpicPathHasNoParentDir covers
// line 91: the parent epic's own Path has no directory component
// (filepath.Dir yields "."), so the derived branch name would be
// malformed — fail-shut rather than emit "epic/.".
func TestExpectedActivationBranch_ParentEpicPathHasNoParentDir(t *testing.T) {
	t.Parallel()
	epic := &entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Status: entity.StatusActive, Path: "epic.md"}
	m := &entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Status: entity.StatusDraft, Parent: "E-0001"}
	tr := &tree.Tree{Entities: []*entity.Entity{epic, m}}

	branch, ok := expectedActivationBranch(tr, m, entity.StatusInProgress)
	if ok {
		t.Errorf("expectedActivationBranch = (%q, true), want (_, false) when the parent epic's Path has no directory component", branch)
	}
	if branch != "" {
		t.Errorf("branch = %q, want empty", branch)
	}
}
