package check

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// Predicate-level + integration tests for M-0130/AC-3:
// forcedUntraileredFindings fires on observations whose (Prior, Next)
// is a sovereign-act-shape (per entity.IsSovereignActShape) AND whose
// commit lacks the aiwf-force trailer AND whose commit is not a merge
// (per D-0010).
//
// AC-3 is disjoint with AC-2 by construction (D-0008's closed-set
// invariant): sovereign-act-shapes are FSM-legal, illegal-transitions
// are not. A single observation can satisfy at most one of the two
// predicates' core gates.

// TestForcedUntraileredFindings_FiresOnSovereignActByNonHumanWithoutForce
// is the load-bearing positive case: epic proposed → active (the
// kernel's only sovereign-act-shape today) by a non-human actor without
// aiwf-force fires the finding. Mirrors M-0095's verb gate (refuses
// non-human actor without --force).
func TestForcedUntraileredFindings_FiresOnSovereignActByNonHumanWithoutForce(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "abc1234567890def",
			Parent:     "0000000000000000",
			Path:       "work/epics/E-0001-x/epic.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusActive, // sovereign-act-shape
			Trailers:   map[string]string{"aiwf-verb": "promote", gitops.TrailerActor: "ai/claude"},
		},
	}
	got := forcedUntraileredFindings(obs)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Code != CodeFSMHistoryConsistent {
		t.Errorf("code = %q, want fsm-history-consistent", got[0].Code)
	}
	if got[0].Subcode != "forced-untrailered" {
		t.Errorf("subcode = %q, want forced-untrailered", got[0].Subcode)
	}
	if got[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", got[0].Severity)
	}
	if got[0].EntityID != "E-0001" {
		t.Errorf("entity = %q, want E-0001", got[0].EntityID)
	}
	if !strings.Contains(got[0].Message, "proposed → active") {
		t.Errorf("message should name (prior → next); got %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "sovereign-act-shape") {
		t.Errorf("message should classify the change as sovereign-act-shape; got %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "non-human actor") {
		t.Errorf("message should classify the actor as non-human; got %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "aiwf-force") {
		t.Errorf("message should mention the missing aiwf-force trailer; got %q", got[0].Message)
	}
}

// TestForcedUntraileredFindings_HumanActorExempts pins the corrected
// predicate (mirrors M-0095's verb gate): an aiwf-actor: human/...
// trailer satisfies the sovereign-act discipline by default and exempts
// the finding even when aiwf-force is absent. The kernel's provenance
// doctrine treats the human-actor case as the default-authorized path;
// --force is the alternative for non-human actors. Either gesture is
// accepted.
func TestForcedUntraileredFindings_HumanActorExempts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		actor string
	}{
		{"human/peter", "human/peter"},
		{"human/anna", "human/anna"},
		{"human with role suffix", "human/peterbru@gmail.com"},
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
					Next:       entity.StatusActive,
					Trailers: map[string]string{
						"aiwf-verb":         "promote",
						gitops.TrailerActor: c.actor,
						// notably: no aiwf-force — the human-actor IS the discipline
					},
				},
			}
			got := forcedUntraileredFindings(obs)
			if len(got) != 0 {
				t.Errorf("expected 0 findings (human-actor exempts), got %+v", got)
			}
		})
	}
}

// TestForcedUntraileredFindings_NonHumanActors_AllFire pins that
// ai/, bot/, and missing actor each fire (the gate inverse of the
// human-actor exemption).
func TestForcedUntraileredFindings_NonHumanActors_AllFire(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		actor string
	}{
		{"ai actor", "ai/claude"},
		{"bot actor", "bot/ci"},
		{"missing actor (empty trailer)", ""},
		{"malformed actor (no slash)", "peter"},
		{"actor with human as substring not prefix", "ai/human-helper"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			trailers := map[string]string{"aiwf-verb": "promote"}
			if c.actor != "" {
				trailers[gitops.TrailerActor] = c.actor
			}
			obs := []statusChange{
				{
					EntityID:   "E-0001",
					EntityKind: entity.KindEpic,
					Commit:     "abc1234567890def",
					Path:       "work/epics/E-0001-x/epic.md",
					Prior:      entity.StatusProposed,
					Next:       entity.StatusActive,
					Trailers:   trailers,
				},
			}
			got := forcedUntraileredFindings(obs)
			if len(got) != 1 {
				t.Errorf("expected 1 finding (non-human actor without force), got %d: %+v", len(got), got)
			}
		})
	}
}

// TestForcedUntraileredFindings_NoFireOnNonSovereignActShape — a legal
// FSM transition that is NOT a sovereign-act-shape (e.g., milestone
// draft → in_progress) produces no AC-3 finding regardless of trailers.
func TestForcedUntraileredFindings_NoFireOnNonSovereignActShape(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "M-0001",
			EntityKind: entity.KindMilestone,
			Commit:     "abc1234567890def",
			Path:       "work/epics/E-0001-x/M-0001-x.md",
			Prior:      entity.StatusDraft,
			Next:       entity.StatusInProgress, // legal, NOT sovereign
			Trailers:   nil,
		},
	}
	got := forcedUntraileredFindings(obs)
	if len(got) != 0 {
		t.Errorf("expected 0 findings (non-sovereign transition), got %+v", got)
	}
}

// TestForcedUntraileredFindings_NoFireOnIllegalTransition pins the
// disjointness invariant from D-0008: AC-3 does not fire on observations
// AC-2 owns. An illegal transition (e.g., epic proposed → done) is not
// in the sovereign-act-shape closed set, so AC-3 stays silent.
//
// The cooperation is by construction (the closed-set invariant guards
// it), but the test pins the behavior at the predicate boundary so a
// future hand-edit to sovereignActShapes that violates the invariant
// is caught here too.
func TestForcedUntraileredFindings_NoFireOnIllegalTransition(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "abc1234567890def",
			Path:       "work/epics/E-0001-x/epic.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusDone, // illegal in epic FSM
			Trailers:   nil,
		},
	}
	got := forcedUntraileredFindings(obs)
	if len(got) != 0 {
		t.Errorf("expected 0 findings (illegal transition belongs to AC-2); got %+v", got)
	}
}

// TestForcedUntraileredFindings_ForceTrailerExempts — aiwf-force
// trailer presence (key-present, any value) exempts the sovereign-act-
// shape transition. The trailer is the kernel's record of the human's
// accountability per the provenance model.
func TestForcedUntraileredFindings_ForceTrailerExempts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		trailers map[string]string
	}{
		{"force with reason", map[string]string{gitops.TrailerForce: "scope was wrong from the start"}},
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
					Next:       entity.StatusActive,
					Trailers:   c.trailers,
				},
			}
			got := forcedUntraileredFindings(obs)
			if len(got) != 0 {
				t.Errorf("expected 0 findings (force-trailer exempts), got %+v", got)
			}
		})
	}
}

// TestForcedUntraileredFindings_MergeSkippedPerD0010 — per D-0010, AC-3
// skips merge-commit observations (uniform across all three subcodes).
// A sovereign-act-shape that surfaces only on a merge edge is not
// caught; the operator's accountability path is the force trailer on
// the original non-merge commit that introduced the change.
func TestForcedUntraileredFindings_MergeSkippedPerD0010(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:      "E-0001",
			EntityKind:    entity.KindEpic,
			Commit:        "merge12345abc",
			Path:          "work/epics/E-0001-x/epic.md",
			Prior:         entity.StatusProposed,
			Next:          entity.StatusActive, // sovereign-act-shape
			Trailers:      nil,
			IsMergeCommit: true,
		},
	}
	got := forcedUntraileredFindings(obs)
	if len(got) != 0 {
		t.Errorf("expected 0 findings on merge observation (D-0010); got %+v", got)
	}
}

// TestForcedUntraileredFindings_VerbTrailerDoesNotExempt — an aiwf-verb
// trailer alone does not exempt the finding. M-0095's verb gate is
// the write-time chokepoint that refuses non-human-actor sovereign
// promotes; AC-3 is the tree-level chokepoint behind it, catching
// anything that slipped past (older binary, frontmatter hand-edit
// followed by a verb-trailered commit, etc.). Only aiwf-force exempts.
func TestForcedUntraileredFindings_VerbTrailerDoesNotExempt(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "abc1234567890def",
			Path:       "work/epics/E-0001-x/epic.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusActive,
			Trailers: map[string]string{
				"aiwf-verb":   "promote",
				"aiwf-actor":  "ai/claude",
				"aiwf-entity": "E-0001",
				// notably: no aiwf-force
			},
		},
	}
	got := forcedUntraileredFindings(obs)
	if len(got) != 1 {
		t.Errorf("expected 1 finding (aiwf-verb does not exempt; only aiwf-force does); got %d: %+v", len(got), got)
	}
}

// TestForcedUntraileredFindings_MultipleObservations — the predicate
// processes a slice; multiple offenders produce multiple findings.
func TestForcedUntraileredFindings_MultipleObservations(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "aaa1111122223333",
			Path:       "epic1.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusActive, // sovereign, untrailered
		},
		{
			EntityID:   "E-0002",
			EntityKind: entity.KindEpic,
			Commit:     "bbb1111122223333",
			Path:       "epic2.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusActive, // sovereign, untrailered
		},
		{
			EntityID:   "E-0003",
			EntityKind: entity.KindEpic,
			Commit:     "ccc1111122223333",
			Path:       "epic3.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusActive,
			Trailers:   map[string]string{gitops.TrailerForce: "yes"}, // exempted
		},
		{
			EntityID:   "M-0001",
			EntityKind: entity.KindMilestone,
			Commit:     "ddd1111122223333",
			Path:       "milestone.md",
			Prior:      entity.StatusDraft,
			Next:       entity.StatusInProgress, // not sovereign
		},
	}
	got := forcedUntraileredFindings(obs)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings (E-0001 and E-0002; E-0003 force-exempt, M-0001 non-sovereign), got %d: %+v", len(got), got)
	}
	ids := map[string]bool{}
	for _, f := range got {
		ids[f.EntityID] = true
	}
	if !ids["E-0001"] || !ids["E-0002"] {
		t.Errorf("findings should name E-0001 and E-0002; got %+v", ids)
	}
}

// TestForcedUntraileredFindings_EmptyInput — nil and empty-slice
// inputs produce nil findings (matches AC-2's predicate behavior).
func TestForcedUntraileredFindings_EmptyInput(t *testing.T) {
	t.Parallel()
	got := forcedUntraileredFindings(nil)
	if got != nil {
		t.Errorf("expected nil findings on nil input, got %+v", got)
	}
	got = forcedUntraileredFindings([]statusChange{})
	if got != nil {
		t.Errorf("expected nil findings on empty slice, got %+v", got)
	}
}

// TestForcedUntraileredAndIllegal_DisjointPerD0008 pins D-0008's
// closed-set invariant from the predicate side: for every entry in
// entity.SovereignActShapes(), the (from, to) pair is FSM-legal, so
// AC-2's illegal-transition predicate does NOT fire on a sovereign-
// act-shape observation. Conversely, AC-3 does NOT fire on illegal
// transitions (covered above in NoFireOnIllegalTransition).
//
// The deep invariant is also pinned by TestSovereignActShapes_AllFSMLegal
// in internal/entity/sovereign_test.go; this test pins the cooperation
// at the check-package boundary so a future predicate refactor that
// breaks disjointness is caught here.
func TestForcedUntraileredAndIllegal_DisjointPerD0008(t *testing.T) {
	t.Parallel()
	shapes := entity.SovereignActShapes()
	if len(shapes) == 0 {
		t.Skip("no sovereign-act-shapes registered; nothing to assert")
	}
	for _, s := range shapes {
		t.Run(string(s.Kind)+":"+s.From+"->"+s.To, func(t *testing.T) {
			t.Parallel()
			// Build an observation matching the sovereign-act-shape
			// with no trailers — the worst-case for both predicates.
			obs := []statusChange{
				{
					EntityID:   "X-0001",
					EntityKind: s.Kind,
					Commit:     "abc1234567890def",
					Path:       "x.md",
					Prior:      s.From,
					Next:       s.To,
					Trailers:   nil,
				},
			}
			illegal := illegalTransitionFindings(obs, nil)
			forced := forcedUntraileredFindings(obs)
			if len(illegal) != 0 {
				t.Errorf("sovereign-act-shape %s %s->%s should be FSM-legal (no illegal-transition finding); got %+v",
					s.Kind, s.From, s.To, illegal)
			}
			if len(forced) != 1 {
				t.Errorf("sovereign-act-shape %s %s->%s without force should produce 1 forced-untrailered finding; got %d",
					s.Kind, s.From, s.To, len(forced))
			}
		})
	}
}

// TestFSMHistoryConsistent_FiresForcedUntrailered_OnEpicActivateWithoutForce
// is the end-to-end test for AC-3: a real git fixture with an epic
// proposed → active commit (no aiwf-force trailer) produces exactly
// one forced-untrailered finding.
func TestFSMHistoryConsistent_FiresForcedUntrailered_OnEpicActivateWithoutForce(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	r.commitEntityWithTrailers("E-0001", entity.KindEpic, entity.StatusActive,
		"promote epic E-0001 proposed -> active",
		map[string]string{
			"aiwf-verb":   "promote",
			"aiwf-entity": "E-0001",
			"aiwf-actor":  "ai/claude",
			// notably: no aiwf-force
		})

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree(), nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	f := got[0]
	if f.Code != CodeFSMHistoryConsistent {
		t.Errorf("code = %q, want fsm-history-consistent", f.Code)
	}
	if f.Subcode != "forced-untrailered" {
		t.Errorf("subcode = %q, want forced-untrailered", f.Subcode)
	}
	if f.Severity != SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
	if f.EntityID != "E-0001" {
		t.Errorf("entity = %q, want E-0001", f.EntityID)
	}
	if !strings.Contains(f.Message, "proposed → active") {
		t.Errorf("message should name the transition; got %q", f.Message)
	}
	if !strings.Contains(f.Message, "aiwf-force") {
		t.Errorf("message should mention the missing force trailer; got %q", f.Message)
	}
}

// TestFSMHistoryConsistent_NoForcedUntrailered_WhenForceTrailerPresent
// pins the override path end-to-end: same fixture as above but the
// promote commit carries `aiwf-force: <reason>`; AC-3 stays silent.
// AC-2 also stays silent (FSM-legal). Total findings: 0.
func TestFSMHistoryConsistent_NoForcedUntrailered_WhenForceTrailerPresent(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	r.commitEntityWithTrailers("E-0001", entity.KindEpic, entity.StatusActive,
		"promote epic E-0001 proposed -> active (sovereign override)",
		map[string]string{
			"aiwf-verb":   "promote",
			"aiwf-entity": "E-0001",
			"aiwf-actor":  "human/peter",
			"aiwf-force":  "ai/claude session — operator override per kernel doc",
		})

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree(), nil)
	if len(got) != 0 {
		t.Errorf("expected 0 findings (force-trailer exempts AC-3); got %d: %+v", len(got), got)
	}
}

// TestFSMHistoryConsistent_ForcedUntrailered_MergeIntegrationSilent
// pins D-0010 at the AC-3 surface: a sovereign-act-shape commit on a
// feature branch (without force) fires AC-3 at the ORIGINAL commit;
// the merge integration is silent. Total findings: 1.
func TestFSMHistoryConsistent_ForcedUntrailered_MergeIntegrationSilent(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	r.gitCheckoutBranch("branch-activate")
	r.commitEntityWithTrailers("E-0001", entity.KindEpic, entity.StatusActive,
		"activate on branch (untrailered sovereign)",
		map[string]string{
			"aiwf-verb":  "promote",
			"aiwf-actor": "ai/claude",
		})
	r.gitCheckout("main")
	r.gitMerge("branch-activate", "merge branch-activate into main")

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree(), nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (original commit only; merge skipped per D-0010), got %d: %+v", len(got), got)
	}
	if got[0].Subcode != "forced-untrailered" {
		t.Errorf("subcode = %q, want forced-untrailered", got[0].Subcode)
	}
}
