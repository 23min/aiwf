package check

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// hasFindingCode reports whether the slice contains a finding with the
// given code. Used by the negative-case tests below.
func hasFindingCode(fs []Finding, code string) bool {
	for i := range fs {
		if fs[i].Code == code {
			return true
		}
	}
	return false
}

// TestEpicActiveNoDraftedMilestones_FiresOnActiveEpicWithNoDrafts pins
// M-0094/AC-1: an active epic with zero drafted milestones surfaces a
// `epic-active-no-drafted-milestones` warning naming the epic. The test
// drives through check.Run (not the helper directly) so the seam
// between the rule and the dispatcher is covered per CLAUDE.md
// *Test the seam, not just the layer*.
func TestEpicActiveNoDraftedMilestones_FiresOnActiveEpicWithNoDrafts(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: entity.StatusActive},
	)
	got := Run(tr, nil)

	var found *Finding
	for i := range got {
		if got[i].Code == "epic-active-no-drafted-milestones" {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected finding code epic-active-no-drafted-milestones, got codes %v", codes(got))
	}
	if found.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want warning", found.Severity)
	}
	if found.EntityID != "E-0001" {
		t.Errorf("EntityID = %q, want E-0001", found.EntityID)
	}
}

// TestEpicActiveNoDraftedMilestones_SilentWhenDraftPresent pins
// M-0094/AC-2: when the active epic has at least one milestone at
// status `draft`, the rule does not fire. Mixed sibling statuses
// (one in_progress alongside one draft) still satisfy the rule —
// the draft alone is enough.
func TestEpicActiveNoDraftedMilestones_SilentWhenDraftPresent(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: entity.StatusActive},
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "In flight", Status: entity.StatusInProgress, Parent: "E-0001"},
		&entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Title: "Queued", Status: entity.StatusDraft, Parent: "E-0001"},
	)
	got := Run(tr, nil)
	if hasFindingCode(got, "epic-active-no-drafted-milestones") {
		t.Errorf("rule fired despite a drafted milestone being present; codes: %v", codes(got))
	}
}

// TestEpicActiveNoDraftedMilestones_SilentForNonActiveEpic pins
// M-0094/AC-3: the rule's scope is exactly `active` epics. For any
// other epic status (proposed, done, cancelled), the rule does not
// fire even with zero drafted milestones — those statuses are not
// what the preflight signal is about.
func TestEpicActiveNoDraftedMilestones_SilentForNonActiveEpic(t *testing.T) {
	cases := []struct {
		name   string
		status string
	}{
		{"proposed", entity.StatusProposed},
		{"done", entity.StatusDone},
		{"cancelled", entity.StatusCancelled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := makeTree(
				&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: tc.status},
			)
			got := Run(tr, nil)
			if hasFindingCode(got, "epic-active-no-drafted-milestones") {
				t.Errorf("rule fired on status %q; codes: %v", tc.status, codes(got))
			}
		})
	}
}

// TestEpicActiveNoDraftedMilestones_IgnoresMilestonesUnderOtherEpics
// pins the parent-matching branch of the inner loop: a draft milestone
// with a *different* parent must not count toward the active epic's
// "has draft" check. Without this guard the rule would consider every
// draft milestone in the tree as satisfying every active epic.
func TestEpicActiveNoDraftedMilestones_IgnoresMilestonesUnderOtherEpics(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Target", Status: entity.StatusActive},
		// Draft milestone parented elsewhere — should not satisfy E-0001's preflight.
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Other epic's draft", Status: entity.StatusDraft, Parent: "E-0002"},
	)
	got := Run(tr, nil)
	if !hasFindingCode(got, "epic-active-no-drafted-milestones") {
		t.Errorf("rule should fire on E-0001 despite a draft milestone under a different parent; codes: %v", codes(got))
	}
}

// TestEpicActiveNoDraftedMilestones_HintReferencesStartEpicPreflight
// pins M-0094/AC-4: the hint surface mentions G-0063 (gap framing) and
// the start-epic preflight role, so a reader who lands on the finding
// via `aiwf check` can navigate to the framing without re-deriving it.
// Substring assertion is appropriate here per CLAUDE.md *Substring
// assertions are not structural assertions* — the hint is a single
// short string, not a structured document where placement matters.
func TestEpicActiveNoDraftedMilestones_HintReferencesStartEpicPreflight(t *testing.T) {
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Test", Status: entity.StatusActive},
	)
	got := Run(tr, nil)

	var found *Finding
	for i := range got {
		if got[i].Code == "epic-active-no-drafted-milestones" {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected finding code epic-active-no-drafted-milestones, got codes %v", codes(got))
	}
	if found.Hint == "" {
		t.Fatal("Hint is empty; expected non-empty hint text")
	}
	if !contains(found.Hint, "G-0063") {
		t.Errorf("Hint %q should reference G-0063", found.Hint)
	}
	if !contains(found.Hint, "start-epic") {
		t.Errorf("Hint %q should reference the start-epic preflight role", found.Hint)
	}
}
