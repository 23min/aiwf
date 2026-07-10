package check

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestEpicTerminalNonTerminalChildren_FiresOnDoneEpicWithOpenMilestone
// pins the primary case (G-0393's standing backstop): an epic at
// terminal status `done` with a non-terminal child milestone surfaces
// an error-severity `epic-terminal-non-terminal-children` finding
// naming the epic. Driven through check.Run (not the helper directly)
// per CLAUDE.md *Test the seam, not just the layer*.
func TestEpicTerminalNonTerminalChildren_FiresOnDoneEpicWithOpenMilestone(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: entity.StatusDone},
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Still open", Status: entity.StatusInProgress, Parent: "E-0001"},
	)
	got := Run(tr, nil)

	var found *Finding
	for i := range got {
		if got[i].Code == CodeEpicTerminalNonTerminalChildren {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected finding code epic-terminal-non-terminal-children, got codes %v", codes(got))
	}
	if found.Severity != SeverityError {
		t.Errorf("Severity = %v, want error", found.Severity)
	}
	if found.EntityID != "E-0001" {
		t.Errorf("EntityID = %q, want E-0001", found.EntityID)
	}
	if !contains(found.Message, "M-0001") {
		t.Errorf("Message %q should name the offending milestone M-0001", found.Message)
	}
}

// TestEpicTerminalNonTerminalChildren_FiresOnCancelledEpicToo confirms
// the rule covers both of Promote's legal terminal targets for
// KindEpic (done and cancelled), not just done.
func TestEpicTerminalNonTerminalChildren_FiresOnCancelledEpicToo(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: entity.StatusCancelled},
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Still open", Status: entity.StatusDraft, Parent: "E-0001"},
	)
	got := Run(tr, nil)
	if !hasFindingCode(got, CodeEpicTerminalNonTerminalChildren) {
		t.Errorf("expected the rule to fire on a cancelled epic with an open child; codes: %v", codes(got))
	}
}

// TestEpicTerminalNonTerminalChildren_SilentWhenAllChildrenTerminal
// pins the negative case: every child milestone terminal (whatever
// their exact terminal statuses) keeps the rule silent.
func TestEpicTerminalNonTerminalChildren_SilentWhenAllChildrenTerminal(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: entity.StatusDone},
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Done", Status: entity.StatusDone, Parent: "E-0001"},
		&entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Title: "Cancelled", Status: entity.StatusCancelled, Parent: "E-0001"},
	)
	got := Run(tr, nil)
	if hasFindingCode(got, CodeEpicTerminalNonTerminalChildren) {
		t.Errorf("rule fired despite every child milestone being terminal; codes: %v", codes(got))
	}
}

// TestEpicTerminalNonTerminalChildren_SilentForNonTerminalEpic pins
// the rule's scope: a non-terminal epic (active, proposed) with open
// children is not this rule's concern — that's the ordinary, expected
// in-flight state.
func TestEpicTerminalNonTerminalChildren_SilentForNonTerminalEpic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status string
	}{
		{"proposed", entity.StatusProposed},
		{"active", entity.StatusActive},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := makeTree(
				&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: tc.status},
				&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Open", Status: entity.StatusInProgress, Parent: "E-0001"},
			)
			got := Run(tr, nil)
			if hasFindingCode(got, CodeEpicTerminalNonTerminalChildren) {
				t.Errorf("rule fired on epic status %q; codes: %v", tc.status, codes(got))
			}
		})
	}
}

// TestEpicTerminalNonTerminalChildren_IgnoresMilestonesUnderOtherEpics
// pins the parent-matching branch: a non-terminal milestone parented
// to a different epic must not count toward this epic's check.
func TestEpicTerminalNonTerminalChildren_IgnoresMilestonesUnderOtherEpics(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Target", Status: entity.StatusDone},
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Other epic's open work", Status: entity.StatusInProgress, Parent: "E-0002"},
	)
	got := Run(tr, nil)
	if hasFindingCode(got, CodeEpicTerminalNonTerminalChildren) {
		t.Errorf("rule fired on E-0001 despite the open milestone belonging to a different parent; codes: %v", codes(got))
	}
}

// TestEpicTerminalNonTerminalChildren_SkipsUnknownOrEmptyChildStatus
// pins the double-report guard: a child milestone with an empty or
// unrecognized status is already reported by frontmatterShape /
// statusValid, so this rule does not also treat it as "non-terminal."
func TestEpicTerminalNonTerminalChildren_SkipsUnknownOrEmptyChildStatus(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: entity.StatusDone},
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Malformed", Status: "", Parent: "E-0001"},
	)
	got := Run(tr, nil)
	if hasFindingCode(got, CodeEpicTerminalNonTerminalChildren) {
		t.Errorf("rule should not treat an empty child status as non-terminal (already reported elsewhere); codes: %v", codes(got))
	}
}

// TestEpicTerminalNonTerminalChildren_SkipsEpicWithEmptyOrUnknownStatus
// mirrors the child-side guard on the epic itself: an epic with an
// empty or unrecognized status is frontmatterShape's/statusValid's
// concern, not this rule's.
func TestEpicTerminalNonTerminalChildren_SkipsEpicWithEmptyOrUnknownStatus(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: ""},
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Open", Status: entity.StatusInProgress, Parent: "E-0001"},
	)
	got := Run(tr, nil)
	if hasFindingCode(got, CodeEpicTerminalNonTerminalChildren) {
		t.Errorf("rule should not fire on an epic with an empty status; codes: %v", codes(got))
	}
}
