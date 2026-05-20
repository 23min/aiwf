package check

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// Predicate-level tests for M-0130/AC-2: illegalTransitionFindings
// fires on observations whose (Prior, Next) is outside the FSM AND
// whose commit lacks the aiwf-force trailer. Per D-0009 the predicate
// ignores IsMergeCommit (the AC-2 policy is "fire on merges too").

// TestIllegalTransitionFindings_FiresOnIllegalNoForce — the load-
// bearing positive case: illegal transition, no force trailer, fires.
func TestIllegalTransitionFindings_FiresOnIllegalNoForce(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "abc1234567890def",
			Parent:     "0000000000000000",
			Path:       "work/epics/E-0001-x/epic.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusDone, // epic FSM: proposed -> active|cancelled; proposed -> done is illegal
			Trailers:   map[string]string{"aiwf-verb": "promote"},
		},
	}
	got := illegalTransitionFindings(obs)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Subcode != "illegal-transition" {
		t.Errorf("subcode = %q, want illegal-transition", got[0].Subcode)
	}
	if got[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", got[0].Severity)
	}
}

// TestIllegalTransitionFindings_NoFireOnLegalTransition — legal FSM
// transition produces no finding regardless of trailers.
func TestIllegalTransitionFindings_NoFireOnLegalTransition(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "abc1234567890def",
			Path:       "work/epics/E-0001-x/epic.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusActive, // legal in epic FSM
			Trailers:   nil,
		},
	}
	got := illegalTransitionFindings(obs)
	if len(got) != 0 {
		t.Errorf("expected 0 findings for legal transition, got %+v", got)
	}
}

// TestIllegalTransitionFindings_ForceTrailerExempts — aiwf-force
// trailer presence (key-present, any value) exempts the transition
// from the finding. This is the kernel's sovereign override path.
func TestIllegalTransitionFindings_ForceTrailerExempts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		trailers map[string]string
	}{
		{"force with reason", map[string]string{gitops.TrailerForce: "operator override for migration"}},
		{"force with empty value", map[string]string{gitops.TrailerForce: ""}},
		{"force alongside other trailers", map[string]string{
			gitops.TrailerForce: "x",
			"aiwf-verb":         "promote",
			"aiwf-actor":        "human/peter",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			obs := []statusChange{
				{
					EntityID:   "E-0001",
					EntityKind: entity.KindEpic,
					Commit:     "abc1234567890def",
					Path:       "work/epics/E-0001-x/epic.md",
					Prior:      entity.StatusProposed,
					Next:       entity.StatusDone,
					Trailers:   c.trailers,
				},
			}
			got := illegalTransitionFindings(obs)
			if len(got) != 0 {
				t.Errorf("expected 0 findings (force-trailer exempts), got %+v", got)
			}
		})
	}
}

// TestIllegalTransitionFindings_VerbTrailerDoesNotExempt — an illegal
// transition routed through a verb (aiwf-verb trailer present, no
// aiwf-force) still fires. The kernel's "verb's FSM check drifted from
// entity.AllowedTransitions" case — the rule is the tree-level
// chokepoint that catches it.
func TestIllegalTransitionFindings_VerbTrailerDoesNotExempt(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "abc1234567890def",
			Path:       "work/epics/E-0001-x/epic.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusDone,
			Trailers: map[string]string{
				"aiwf-verb":   "promote",
				"aiwf-actor":  "human/peter",
				"aiwf-entity": "E-0001",
				// notably: no aiwf-force
			},
		},
	}
	got := illegalTransitionFindings(obs)
	if len(got) != 1 {
		t.Errorf("expected 1 finding (aiwf-verb does not exempt; only aiwf-force does); got %d: %+v", len(got), got)
	}
}

// TestIllegalTransitionFindings_MergeSkippedPerD0010 — per D-0010
// (supersedes D-0009), AC-2 skips merge-commit observations. The
// trade-off is documented in D-0010: an FSM-illegal merge
// resolution to a third state is silently accepted (the operator's
// accountability is the force trailer or aiwf-verb routing on any
// non-merge commit that would have produced the same edit).
func TestIllegalTransitionFindings_MergeSkippedPerD0010(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:      "E-0001",
			EntityKind:    entity.KindEpic,
			Commit:        "merge12345abc",
			Path:          "work/epics/E-0001-x/epic.md",
			Prior:         entity.StatusActive,
			Next:          entity.StatusProposed, // FSM: epic active -> proposed is illegal
			Trailers:      nil,
			IsMergeCommit: true,
		},
	}
	got := illegalTransitionFindings(obs)
	if len(got) != 0 {
		t.Errorf("expected 0 findings on merge observation (D-0010: AC-2 skips merges); got %+v", got)
	}
}

// TestIllegalTransitionFindings_MultipleObservations — the predicate
// processes a slice; multiple offenders produce multiple findings.
func TestIllegalTransitionFindings_MultipleObservations(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "aaa1111122223333",
			Path:       "epic1.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusDone,
		},
		{
			EntityID:   "E-0002",
			EntityKind: entity.KindEpic,
			Commit:     "bbb1111122223333",
			Path:       "epic2.md",
			Prior:      entity.StatusDone, // terminal
			Next:       entity.StatusActive,
		},
		{
			EntityID:   "E-0003",
			EntityKind: entity.KindEpic,
			Commit:     "ccc1111122223333",
			Path:       "epic3.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusActive, // legal — no finding
		},
	}
	got := illegalTransitionFindings(obs)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings (E-0001 and E-0002; E-0003 is legal), got %d: %+v", len(got), got)
	}
	ids := map[string]bool{}
	for _, f := range got {
		ids[f.EntityID] = true
	}
	if !ids["E-0001"] || !ids["E-0002"] {
		t.Errorf("findings should name E-0001 and E-0002; got %+v", ids)
	}
}

// TestIllegalTransitionFindings_EmptyInput — empty observation slice
// produces no findings.
func TestIllegalTransitionFindings_EmptyInput(t *testing.T) {
	t.Parallel()
	got := illegalTransitionFindings(nil)
	if got != nil {
		t.Errorf("expected nil findings on nil input, got %+v", got)
	}
	got = illegalTransitionFindings([]statusChange{})
	if got != nil {
		t.Errorf("expected nil findings on empty slice, got %+v", got)
	}
}

// TestIsLegalTransition_Predicate exhaustively covers the helper
// across kinds, legal edges, illegal edges, terminal-state outbound,
// unknown kinds, and unknown prior states.
func TestIsLegalTransition_Predicate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		kind        entity.Kind
		prior, next string
		want        bool
	}{
		// Epic legal edges.
		{"epic proposed->active", entity.KindEpic, entity.StatusProposed, entity.StatusActive, true},
		{"epic active->done", entity.KindEpic, entity.StatusActive, entity.StatusDone, true},
		{"epic proposed->cancelled", entity.KindEpic, entity.StatusProposed, entity.StatusCancelled, true},
		// Epic illegal edges.
		{"epic proposed->done (skip-ahead)", entity.KindEpic, entity.StatusProposed, entity.StatusDone, false},
		{"epic done->active (terminal outbound)", entity.KindEpic, entity.StatusDone, entity.StatusActive, false},
		{"epic active->proposed (backwards)", entity.KindEpic, entity.StatusActive, entity.StatusProposed, false},
		// Milestone legal / illegal.
		{"milestone draft->in_progress", entity.KindMilestone, entity.StatusDraft, entity.StatusInProgress, true},
		{"milestone in_progress->draft (backwards)", entity.KindMilestone, entity.StatusInProgress, entity.StatusDraft, false},
		// Gap legal / illegal.
		{"gap open->addressed", entity.KindGap, entity.StatusOpen, entity.StatusAddressed, true},
		{"gap addressed->open (terminal outbound)", entity.KindGap, entity.StatusAddressed, entity.StatusOpen, false},
		// Unknown inputs.
		{"unknown kind", entity.Kind("widget"), entity.StatusProposed, entity.StatusActive, false},
		{"unknown prior", entity.KindEpic, "weird", entity.StatusActive, false},
		{"empty prior", entity.KindEpic, "", entity.StatusActive, false},
		{"empty next", entity.KindEpic, entity.StatusProposed, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := isLegalTransition(c.kind, c.prior, c.next); got != c.want {
				t.Errorf("isLegalTransition(%s, %q, %q) = %v, want %v", c.kind, c.prior, c.next, got, c.want)
			}
		})
	}
}

// TestShortHash pins the abbreviation helper.
func TestShortHash(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"abcdef0123456789", "abcdef01"},
		{"abc", "abc"},
		{"", ""},
		{"abcdef01", "abcdef01"}, // exact-8 stays exact
		{"abcdef012", "abcdef01"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			if got := shortHash(c.in); got != c.want {
				t.Errorf("shortHash(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestFSMHistoryConsistent_IllegalOnBranchPlusMergeFiresOnlyOnOriginal_PerD0010
// is the load-bearing test for D-0010's "skip merges" rule. An
// illegal transition on a feature branch, integrated via a merge
// into main, produces exactly ONE finding — at the original
// feature-branch commit. The merge integration is silent (per
// D-0010, AC-2 skips merge observations to avoid the routine
// feature-branch-integration noise that D-0009 produced).
func TestFSMHistoryConsistent_IllegalOnBranchPlusMergeFiresOnlyOnOriginal_PerD0010(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	// main: add at proposed.
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	// branch-bad: skip-ahead illegal proposed -> done
	r.gitCheckoutBranch("branch-bad")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "skip-ahead illegal on branch")
	// merge branch-bad into main; resolves to done (the illegal state).
	r.gitCheckout("main")
	r.gitMerge("branch-bad", "merge branch-bad into main")

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree())
	// D-0010: expect 1 finding — only the original branch-bad commit.
	// The merge integration is silent.
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (original commit only; merge skipped per D-0010), got %d: %+v", len(got), got)
	}
	f := got[0]
	if f.Subcode != "illegal-transition" {
		t.Errorf("subcode = %q, want illegal-transition", f.Subcode)
	}
	if !strings.Contains(f.Message, "proposed → done") {
		t.Errorf("message should name the (prior → next) transition; got %q", f.Message)
	}
}

// TestFSMHistoryConsistent_LegalFeatureBranchMergeNoFire_PerD0010
// is the load-bearing positive test for D-0010: a normal feature-
// branch workflow (add at draft on main → branch off → progress
// through legal FSM steps on the branch → merge back to main)
// emits zero findings across AC-2/3/4, even though main's pre-
// merge view of the entity differs from the merge result. This
// pattern is the source of the 44 false positives D-0009 produced
// on aiwf's own repo.
//
// The branch's promote commits carry the aiwf-verb trailer so
// AC-4's manual-edit predicate doesn't fire (a verb-mediated
// commit by definition signals the kernel was not bypassed).
func TestFSMHistoryConsistent_LegalFeatureBranchMergeNoFire_PerD0010(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	// Build a milestone fixture (not epic) so we can exercise the
	// full draft → in_progress → done legal progression on a branch.
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
	r.commitEntity("M-0001", entity.KindMilestone, entity.StatusDraft, "add milestone at draft on main")

	// Branch off, progress legally on the branch. Verb trailers
	// pin the kernel-routed provenance so AC-4 (manual-edit) stays
	// silent.
	r.gitCheckoutBranch("feature-branch")
	r.commitEntityWithTrailers("M-0001", entity.KindMilestone, entity.StatusInProgress,
		"aiwf promote M-0001 draft -> in_progress on branch",
		map[string]string{gitops.TrailerVerb: "promote", gitops.TrailerActor: "human/peter"})
	r.commitEntityWithTrailers("M-0001", entity.KindMilestone, entity.StatusDone,
		"aiwf promote M-0001 in_progress -> done on branch",
		map[string]string{gitops.TrailerVerb: "promote", gitops.TrailerActor: "human/peter"})

	// Merge back to main.
	r.gitCheckout("main")
	r.gitMerge("feature-branch", "merge feature-branch into main")

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree())
	// D-0010: every per-commit observation on the branch is legal
	// (draft -> in_progress, in_progress -> done) with aiwf-verb
	// trailers; the merge integration is skipped. Total: 0 findings.
	if len(got) != 0 {
		t.Errorf("expected 0 findings on routine legal feature-branch merge; got %d: %+v", len(got), got)
	}
}

// TestFSMHistoryConsistent_MergeResolvingToLegalNoFire — a merge that
// integrates a feature branch's LEGAL promote produces NO finding from
// any AC-2..4 subcode. The original promote is legal in the FSM (so
// AC-2 doesn't fire), is NOT a sovereign-act-shape (so AC-3 doesn't
// fire — milestone draft → in_progress is FSM-legal but not
// sovereign), and the merge integration is skipped per D-0010 across
// all subcodes.
//
// Uses milestone draft → in_progress (rather than epic
// proposed → active) precisely to keep the test focused on merge-
// integration silence — epic proposed → active is a sovereign-act-
// shape and would entangle this test with AC-3's force-trailer
// exemption surface, which has its own dedicated tests in
// fsm_history_forced_untrailered_test.go.
func TestFSMHistoryConsistent_MergeResolvingToLegalNoFire(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
	r.commitEntity("M-0001", entity.KindMilestone, entity.StatusDraft, "add milestone")
	r.gitCheckoutBranch("branch-good")
	r.commitEntityWithTrailers("M-0001", entity.KindMilestone, entity.StatusInProgress,
		"aiwf promote M-0001 draft -> in_progress (legal, non-sovereign)",
		map[string]string{gitops.TrailerVerb: "promote", gitops.TrailerActor: "human/peter"})
	r.gitCheckout("main")
	r.gitMerge("branch-good", "merge branch-good into main")

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree())
	if len(got) != 0 {
		t.Errorf("expected 0 findings (legal non-sovereign transition with verb trailer, integrated via merge); got %d: %+v", len(got), got)
	}
}
