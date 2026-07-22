package check

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestDependsOnCancelled_FiresWhenNonTerminalDependsOnCancelled pins the
// primary case (G-0437): a non-terminal milestone lists a now-cancelled
// milestone in depends_on. The edge can never be satisfied, so this
// fires at error severity naming both milestones.
func TestDependsOnCancelled_FiresWhenNonTerminalDependsOnCancelled(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Cancelled", Status: entity.StatusCancelled},
		&entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Title: "Waiting", Status: entity.StatusInProgress, DependsOn: []string{"M-0001"}},
	)
	got := Run(tr, nil)

	var found *Finding
	for i := range got {
		if got[i].Code == CodeDependsOnCancelled {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected finding code depends-on-cancelled, got codes %v", codes(got))
	}
	if found.Severity != SeverityError {
		t.Errorf("Severity = %v, want error", found.Severity)
	}
	if found.EntityID != "M-0002" {
		t.Errorf("EntityID = %q, want M-0002", found.EntityID)
	}
	if !contains(found.Message, "M-0001") {
		t.Errorf("Message %q should name the cancelled referent M-0001", found.Message)
	}
	if found.Hint == "" {
		t.Errorf("expected a non-empty Hint")
	}
}

// TestDependsOnCancelled_SilentWhenReferentNotCancelled pins the
// negative case: an in-progress or done referent is a perfectly normal
// depends_on target and must not fire.
func TestDependsOnCancelled_SilentWhenReferentNotCancelled(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status string
	}{
		{"referent in_progress", entity.StatusInProgress},
		{"referent done", entity.StatusDone},
		{"referent draft", entity.StatusDraft},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := makeTree(
				&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Referent", Status: tc.status},
				&entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Title: "Waiting", Status: entity.StatusInProgress, DependsOn: []string{"M-0001"}},
			)
			got := Run(tr, nil)
			if hasFindingCode(got, CodeDependsOnCancelled) {
				t.Errorf("rule fired on referent status %q; codes: %v", tc.status, codes(got))
			}
		})
	}
}

// TestDependsOnCancelled_SilentWhenDependentTerminal pins the second
// negative case: once the dependent itself reaches a terminal status
// (done or cancelled), a cancelled referent is no longer this rule's
// concern — the dependent isn't waiting on anything anymore.
func TestDependsOnCancelled_SilentWhenDependentTerminal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status string
	}{
		{"dependent done", entity.StatusDone},
		{"dependent cancelled", entity.StatusCancelled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := makeTree(
				&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Cancelled", Status: entity.StatusCancelled},
				&entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Title: "Waiting", Status: tc.status, DependsOn: []string{"M-0001"}},
			)
			got := Run(tr, nil)
			if hasFindingCode(got, CodeDependsOnCancelled) {
				t.Errorf("rule fired despite terminal dependent status %q; codes: %v", tc.status, codes(got))
			}
		})
	}
}

// TestDependsOnCancelled_SkipsDependentWithEmptyOrUnknownStatus mirrors
// epicTerminalNonTerminalChildren's own guard test: a milestone with an
// empty or unrecognized status is frontmatterShape's/statusValid's
// concern, not this rule's — even when it has a depends_on pointing at
// a genuinely cancelled milestone, so the suppression is behavioral,
// not vacuous (a regressed guard would double-report alongside
// statusValid instead of staying silent here).
func TestDependsOnCancelled_SkipsDependentWithEmptyOrUnknownStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status string
	}{
		{"empty", ""},
		{"unknown", "bogus"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := makeTree(
				&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Title: "Cancelled", Status: entity.StatusCancelled},
				&entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Title: "Waiting", Status: tc.status, DependsOn: []string{"M-0001"}},
			)
			got := Run(tr, nil)
			if hasFindingCode(got, CodeDependsOnCancelled) {
				t.Errorf("rule should not fire on a dependent with status %q; codes: %v", tc.status, codes(got))
			}
		})
	}
}

// TestDependsOnCancelled_SilentOnUnresolvedReferent confirms the rule
// doesn't panic or misfire when depends_on names an id absent from the
// tree — refsResolve already reports that shape; this rule has nothing
// to add without a resolvable referent to inspect.
func TestDependsOnCancelled_SilentOnUnresolvedReferent(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Title: "Waiting", Status: entity.StatusInProgress, DependsOn: []string{"M-9999"}},
	)
	got := Run(tr, nil)
	if hasFindingCode(got, CodeDependsOnCancelled) {
		t.Errorf("rule fired on an unresolved referent; codes: %v", codes(got))
	}
}
