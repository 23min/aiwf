package check

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// Predicate-level + integration tests for M-0130/AC-4:
// manualEditFindings fires on observations whose (Prior, Next) is
// FSM-legal AND NOT a sovereign-act-shape AND whose commit lacks an
// aiwf-verb trailer AND whose commit is not a merge AND whose entity
// is not in the audit-only ack set (a separate later commit carrying
// aiwf-audit-only + aiwf-entity for this entity), per D-0008.
//
// AC-4 is the catch-all of D-0008's disjoint partition: it owns the
// FSM-legal non-sovereign space that AC-2 and AC-3 don't claim.
//
// Severity is WARNING, aligned with the parallel
// provenance-untrailered-entity-commit rule — the audit-only backfill
// is the intended cure, and warning-level avoids blocking pushes for
// state that is correct on disk pending acknowledgment.

// TestManualEditFindings_FiresOnLegalNonSovereignWithoutVerb is the
// load-bearing positive case: a milestone draft → in_progress
// transition (FSM-legal, not sovereign-act-shape) without an aiwf-verb
// trailer fires the finding. Mirrors the kernel-bypass shape — a hand-
// edit to the entity's frontmatter committed via plain `git commit`.
func TestManualEditFindings_FiresOnLegalNonSovereignWithoutVerb(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "M-0001",
			EntityKind: entity.KindMilestone,
			Commit:     "abc1234567890def",
			Parent:     "0000000000000000",
			Path:       "work/epics/E-0001-x/M-0001-x.md",
			Prior:      entity.StatusDraft,
			Next:       entity.StatusInProgress, // FSM-legal, not sovereign
			Trailers:   nil,                     // no aiwf-verb — manual edit
		},
	}
	got := manualEditFindings(obs, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Code != "fsm-history-consistent" {
		t.Errorf("code = %q, want fsm-history-consistent", got[0].Code)
	}
	if got[0].Subcode != "manual-edit" {
		t.Errorf("subcode = %q, want manual-edit", got[0].Subcode)
	}
	if got[0].Severity != SeverityWarning {
		t.Errorf("severity = %q, want warning", got[0].Severity)
	}
	if got[0].EntityID != "M-0001" {
		t.Errorf("entity = %q, want M-0001", got[0].EntityID)
	}
	if !strings.Contains(got[0].Message, "draft → in_progress") {
		t.Errorf("message should name (prior → next); got %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "milestone") {
		t.Errorf("message should name the kind; got %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "aiwf-verb") {
		t.Errorf("message should mention the missing aiwf-verb trailer; got %q", got[0].Message)
	}
}

// TestManualEditFindings_VerbTrailerExempts — the canonical exemption:
// any aiwf-verb trailer (key-present, value irrelevant) signals the
// commit went through the kernel and exempts the finding.
func TestManualEditFindings_VerbTrailerExempts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		verbs []string
	}{
		{"promote", []string{"promote"}},
		{"cancel", []string{"cancel"}},
		{"any string value", []string{"some-verb-name"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			obs := []statusChange{
				{
					EntityID:   "M-0001",
					EntityKind: entity.KindMilestone,
					Commit:     "abc1234567890def",
					Path:       "work/epics/E-0001-x/M-0001-x.md",
					Prior:      entity.StatusDraft,
					Next:       entity.StatusInProgress,
					Trailers:   map[string]string{gitops.TrailerVerb: c.verbs[0]},
				},
			}
			got := manualEditFindings(obs, nil)
			if len(got) != 0 {
				t.Errorf("expected 0 findings (aiwf-verb exempts), got %+v", got)
			}
		})
	}
}

// TestManualEditFindings_AckedEntityExempted — per D-0008, an entity
// in the audit-only ack set is exempt from the finding. The ack set
// is built by walkAuditOnlyAckedEntities from a SEPARATE later commit
// carrying aiwf-audit-only + aiwf-entity; the predicate consults the
// set via the second arg.
//
// At the predicate level we don't care HOW the ack got into the set —
// only that the entity is in it. The walker's correctness (composite
// rollup, canonicalization) is tested end-to-end below in
// TestFSMHistoryConsistent_ManualEditClearedByLaterAuditOnlyCommit.
//
// Important: aiwf-audit-only on the same commit as the flip does NOT
// exempt — only a later separate ack commit does. This is the
// retrospective-acknowledgment semantics the verb layer produces (the
// audit-only verbs emit empty commits; they never co-locate with a
// status flip).
func TestManualEditFindings_AckedEntityExempted(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "M-0001",
			EntityKind: entity.KindMilestone,
			Commit:     "abc1234567890def",
			Path:       "work/epics/E-0001-x/M-0001-x.md",
			Prior:      entity.StatusDraft,
			Next:       entity.StatusInProgress,
			Trailers:   nil, // no aiwf-verb on the flip itself
		},
	}
	acked := map[string]bool{"M-0001": true}
	got := manualEditFindings(obs, acked)
	if len(got) != 0 {
		t.Errorf("expected 0 findings (entity acked via audit-only ack set), got %+v", got)
	}
}

// TestManualEditFindings_AckedDifferentEntity_DoesNotExempt — the
// ack set is per-entity; an ack for M-0001 does not exempt M-0002.
func TestManualEditFindings_AckedDifferentEntity_DoesNotExempt(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "M-0002",
			EntityKind: entity.KindMilestone,
			Commit:     "abc1234567890def",
			Path:       "work/epics/E-0001-x/M-0002-x.md",
			Prior:      entity.StatusDraft,
			Next:       entity.StatusInProgress,
			Trailers:   nil,
		},
	}
	acked := map[string]bool{"M-0001": true} // wrong entity
	got := manualEditFindings(obs, acked)
	if len(got) != 1 {
		t.Errorf("expected 1 finding (ack for different entity does not transfer), got %d: %+v", len(got), got)
	}
}

// TestManualEditFindings_NoFireOnIllegalTransition — AC-4 doesn't fire
// on illegal transitions (AC-2 owns those). Disjointness invariant
// pinned at the predicate boundary.
func TestManualEditFindings_NoFireOnIllegalTransition(t *testing.T) {
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
	got := manualEditFindings(obs, nil)
	if len(got) != 0 {
		t.Errorf("expected 0 findings (illegal transition belongs to AC-2); got %+v", got)
	}
}

// TestManualEditFindings_NoFireOnSovereignActShape — AC-4 doesn't fire
// on sovereign-act-shape transitions (AC-3 owns those). Disjointness
// invariant pinned at the predicate boundary.
func TestManualEditFindings_NoFireOnSovereignActShape(t *testing.T) {
	t.Parallel()
	shapes := entity.SovereignActShapes()
	if len(shapes) == 0 {
		t.Skip("no sovereign-act-shapes registered; nothing to assert")
	}
	for _, s := range shapes {
		t.Run(string(s.Kind)+":"+s.From+"->"+s.To, func(t *testing.T) {
			t.Parallel()
			obs := []statusChange{
				{
					EntityID:   "X-0001",
					EntityKind: s.Kind,
					Commit:     "abc1234567890def",
					Path:       "x.md",
					Prior:      s.From,
					Next:       s.To,
					Trailers:   nil, // no aiwf-verb either — AC-3 territory regardless
				},
			}
			got := manualEditFindings(obs, nil)
			if len(got) != 0 {
				t.Errorf("sovereign-act-shape %s %s->%s should be owned by AC-3, not AC-4; got %+v",
					s.Kind, s.From, s.To, got)
			}
		})
	}
}

// TestManualEditFindings_MergeSkippedPerD0010 — per D-0010, AC-4 skips
// merge-commit observations (uniform across all three subcodes).
func TestManualEditFindings_MergeSkippedPerD0010(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:      "M-0001",
			EntityKind:    entity.KindMilestone,
			Commit:        "merge12345abc",
			Path:          "work/epics/E-0001-x/M-0001-x.md",
			Prior:         entity.StatusDraft,
			Next:          entity.StatusInProgress,
			Trailers:      nil,
			IsMergeCommit: true,
		},
	}
	got := manualEditFindings(obs, nil)
	if len(got) != 0 {
		t.Errorf("expected 0 findings on merge observation (D-0010); got %+v", got)
	}
}

// TestManualEditFindings_MultipleObservations — the predicate processes
// a slice; multiple offenders produce multiple findings; exempted
// observations don't accumulate.
func TestManualEditFindings_MultipleObservations(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "M-0001",
			EntityKind: entity.KindMilestone,
			Commit:     "aaa1111122223333",
			Path:       "m1.md",
			Prior:      entity.StatusDraft,
			Next:       entity.StatusInProgress, // legal, non-sovereign, no verb → fires
		},
		{
			EntityID:   "M-0002",
			EntityKind: entity.KindMilestone,
			Commit:     "bbb1111122223333",
			Path:       "m2.md",
			Prior:      entity.StatusInProgress,
			Next:       entity.StatusDone, // legal, non-sovereign, no verb → fires
		},
		{
			EntityID:   "M-0003",
			EntityKind: entity.KindMilestone,
			Commit:     "ccc1111122223333",
			Path:       "m3.md",
			Prior:      entity.StatusDraft,
			Next:       entity.StatusInProgress,
			Trailers:   map[string]string{gitops.TrailerVerb: "promote"}, // verb-exempt
		},
		{
			EntityID:   "M-0004",
			EntityKind: entity.KindMilestone,
			Commit:     "ddd1111122223333",
			Path:       "m4.md",
			Prior:      entity.StatusDraft,
			Next:       entity.StatusInProgress, // ack-exempt via acked map below
		},
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "eee1111122223333",
			Path:       "e1.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusDone, // illegal → AC-2's territory, not AC-4
		},
	}
	acked := map[string]bool{"M-0004": true}
	got := manualEditFindings(obs, acked)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings (M-0001 and M-0002; M-0003 verb-exempt, M-0004 ack-exempt, E-0001 illegal); got %d: %+v",
			len(got), got)
	}
	ids := map[string]bool{}
	for _, f := range got {
		ids[f.EntityID] = true
	}
	if !ids["M-0001"] || !ids["M-0002"] {
		t.Errorf("findings should name M-0001 and M-0002; got %+v", ids)
	}
}

// TestManualEditFindings_EmptyInput — nil and empty-slice inputs
// produce nil findings (matches AC-2/AC-3 predicate behavior).
func TestManualEditFindings_EmptyInput(t *testing.T) {
	t.Parallel()
	got := manualEditFindings(nil, nil)
	if got != nil {
		t.Errorf("expected nil findings on nil input, got %+v", got)
	}
	got = manualEditFindings([]statusChange{}, nil)
	if got != nil {
		t.Errorf("expected nil findings on empty slice, got %+v", got)
	}
}

// TestThreeSubcodes_Disjoint_PerD0008 pins D-0008's closed-set
// disjointness across all three predicates at the check-package
// boundary: for any single observation, at most one of AC-2, AC-3,
// AC-4 can fire by construction.
//
// The test covers the partition by enumeration:
//
//  1. Illegal transition: only AC-2 fires.
//  2. Sovereign-act-shape without force, non-human actor: only AC-3 fires.
//  3. Legal non-sovereign without verb: only AC-4 fires.
//  4. Exempted (e.g., legal non-sovereign with aiwf-verb): no predicate fires.
//  5. Force trailer present (AC-2/AC-3 exempted), with aiwf-verb (AC-4 exempted): no predicate fires.
//
// Each case is a single observation; the test asserts the count
// across all three predicates' findings sums to ≤1.
func TestThreeSubcodes_Disjoint_PerD0008(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		obs       statusChange
		wantFires int
		wantOwner string // empty when wantFires == 0
	}{
		{
			name: "illegal-transition owns it",
			obs: statusChange{
				EntityKind: entity.KindEpic,
				Prior:      entity.StatusProposed,
				Next:       entity.StatusDone,
			},
			wantFires: 1,
			wantOwner: "illegal-transition",
		},
		{
			name: "forced-untrailered owns it",
			obs: statusChange{
				EntityKind: entity.KindEpic,
				Prior:      entity.StatusProposed,
				Next:       entity.StatusActive,
				Trailers:   map[string]string{gitops.TrailerActor: "ai/claude"},
			},
			wantFires: 1,
			wantOwner: "forced-untrailered",
		},
		{
			name: "manual-edit owns it",
			obs: statusChange{
				EntityKind: entity.KindMilestone,
				Prior:      entity.StatusDraft,
				Next:       entity.StatusInProgress,
			},
			wantFires: 1,
			wantOwner: "manual-edit",
		},
		{
			name: "legal non-sovereign with verb exempts all",
			obs: statusChange{
				EntityKind: entity.KindMilestone,
				Prior:      entity.StatusDraft,
				Next:       entity.StatusInProgress,
				Trailers:   map[string]string{gitops.TrailerVerb: "promote"},
			},
			wantFires: 0,
		},
		{
			name: "force trailer on illegal exempts AC-2; absent sovereign-shape exempts AC-3; verb exempts AC-4",
			obs: statusChange{
				EntityKind: entity.KindEpic,
				Prior:      entity.StatusProposed,
				Next:       entity.StatusDone,
				Trailers:   map[string]string{gitops.TrailerForce: "x", gitops.TrailerVerb: "promote"},
			},
			wantFires: 0,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			obs := []statusChange{c.obs}
			illegal := illegalTransitionFindings(obs)
			forced := forcedUntraileredFindings(obs)
			manual := manualEditFindings(obs, nil)
			total := len(illegal) + len(forced) + len(manual)
			if total != c.wantFires {
				t.Errorf("want %d total findings, got %d (illegal=%d forced=%d manual=%d): obs=%+v",
					c.wantFires, total, len(illegal), len(forced), len(manual), c.obs)
			}
			if c.wantFires == 1 {
				if c.wantOwner == "illegal-transition" && len(illegal) != 1 {
					t.Errorf("want illegal-transition owner; got 0 illegal, %d forced, %d manual", len(forced), len(manual))
				}
				if c.wantOwner == "forced-untrailered" && len(forced) != 1 {
					t.Errorf("want forced-untrailered owner; got %d illegal, 0 forced, %d manual", len(illegal), len(manual))
				}
				if c.wantOwner == "manual-edit" && len(manual) != 1 {
					t.Errorf("want manual-edit owner; got %d illegal, %d forced, 0 manual", len(illegal), len(forced))
				}
			}
		})
	}
}

// TestFSMHistoryConsistent_FiresManualEdit_OnLegalUntraileredCommit
// is the end-to-end test for AC-4: a real git fixture whose commit
// changed a milestone's status without an aiwf-verb trailer (plain
// `git commit`) produces exactly one manual-edit finding.
func TestFSMHistoryConsistent_FiresManualEdit_OnLegalUntraileredCommit(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
	r.commitEntity("M-0001", entity.KindMilestone, entity.StatusDraft, "add milestone")
	r.commitEntity("M-0001", entity.KindMilestone, entity.StatusInProgress, "hand-edit status; no verb trailer")

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree())
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	f := got[0]
	if f.Code != "fsm-history-consistent" {
		t.Errorf("code = %q, want fsm-history-consistent", f.Code)
	}
	if f.Subcode != "manual-edit" {
		t.Errorf("subcode = %q, want manual-edit", f.Subcode)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("severity = %q, want warning", f.Severity)
	}
	if f.EntityID != "M-0001" {
		t.Errorf("entity = %q, want M-0001", f.EntityID)
	}
	if !strings.Contains(f.Message, "draft → in_progress") {
		t.Errorf("message should name the transition; got %q", f.Message)
	}
	if !strings.Contains(f.Message, "aiwf-verb") {
		t.Errorf("message should mention the missing aiwf-verb trailer; got %q", f.Message)
	}
}

// TestFSMHistoryConsistent_NoManualEdit_WhenVerbTrailerPresent — same
// fixture as above but the status-change commit carries aiwf-verb;
// AC-4 stays silent (and AC-2/AC-3 also silent because the transition
// is FSM-legal and non-sovereign). Total findings: 0.
func TestFSMHistoryConsistent_NoManualEdit_WhenVerbTrailerPresent(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
	r.commitEntity("M-0001", entity.KindMilestone, entity.StatusDraft, "add milestone")
	r.commitEntityWithTrailers("M-0001", entity.KindMilestone, entity.StatusInProgress,
		"aiwf promote M-0001 draft -> in_progress",
		map[string]string{
			gitops.TrailerVerb:   "promote",
			gitops.TrailerEntity: "M-0001",
			gitops.TrailerActor:  "human/peter",
		})

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree())
	if len(got) != 0 {
		t.Errorf("expected 0 findings (verb-trailer exempts AC-4); got %d: %+v", len(got), got)
	}
}

// TestFSMHistoryConsistent_ManualEditClearedByLaterAuditOnlyCommit is
// the load-bearing test for the audit-only cooperation pattern that
// the hint table promises: an operator hand-edits a status (no
// aiwf-verb trailer), then later runs `aiwf <verb> --audit-only
// --reason "..."` to backfill the audit trail. The audit-only commit
// is a SEPARATE empty commit carrying aiwf-audit-only + aiwf-entity.
// On the next `aiwf check` invocation, AC-4's manual-edit finding
// for that entity clears.
//
// Pins:
//   - Pre-ack: manual-edit fires.
//   - Post-ack: manual-edit clears.
//   - The audit-only commit is on a separate empty commit, not on the
//     status-change commit (which is how the audit-only verbs produce
//     them in production — `internal/verb/auditonly.go`'s plan emits
//     `AllowEmpty: true`).
func TestFSMHistoryConsistent_ManualEditClearedByLaterAuditOnlyCommit(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
	r.commitEntity("M-0001", entity.KindMilestone, entity.StatusDraft, "add milestone")
	// Manual flip without aiwf-verb trailer — fires AC-4 absent ack.
	r.commitEntity("M-0001", entity.KindMilestone, entity.StatusInProgress, "hand-edit; no aiwf-verb trailer")

	pre := FSMHistoryConsistent(context.Background(), r.root, r.tree())
	if len(pre) != 1 || pre[0].Subcode != "manual-edit" {
		t.Fatalf("pre-ack: expected 1 manual-edit finding; got %+v", pre)
	}

	// Audit-only ack: separate empty commit; mirrors what `aiwf promote
	// M-0001 in_progress --audit-only --reason "..."` produces.
	r.gitCommitWithTrailers("aiwf promote M-0001 in_progress [audit-only]",
		map[string]string{
			gitops.TrailerVerb:      "promote",
			gitops.TrailerEntity:    "M-0001",
			gitops.TrailerActor:     "human/peter",
			gitops.TrailerAuditOnly: "post-hoc acknowledgment of hand-edit",
			gitops.TrailerTo:        entity.StatusInProgress,
		})

	post := FSMHistoryConsistent(context.Background(), r.root, r.tree())
	if len(post) != 0 {
		t.Errorf("post-ack: expected 0 findings (manual-edit cleared by audit-only ack); got %d: %+v", len(post), post)
	}
}

// TestFSMHistoryConsistent_AuditOnlyDoesNotClearIllegalTransition pins
// D-0008's scope-of-suppression contract: audit-only acknowledgment
// suppresses only the manual-edit subcode. An illegal-transition
// finding for the same entity is NOT cleared by an audit-only ack —
// the operator's accountability for FSM violations is `--force
// --reason`, not `--audit-only`.
func TestFSMHistoryConsistent_AuditOnlyDoesNotClearIllegalTransition(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add")
	// FSM-illegal hand-edit: proposed → done, skipping active. No
	// trailers on the flip commit.
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "hand-edit illegal jump")
	// Audit-only ack for E-0001 — should NOT clear illegal-transition.
	r.gitCommitWithTrailers("aiwf cancel E-0001 [audit-only]",
		map[string]string{
			gitops.TrailerVerb:      "cancel",
			gitops.TrailerEntity:    "E-0001",
			gitops.TrailerActor:     "human/peter",
			gitops.TrailerAuditOnly: "post-hoc acknowledgment (test fixture; should not actually clear illegal-transition)",
		})

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree())
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (illegal-transition still fires; audit-only doesn't apply per D-0008); got %d: %+v", len(got), got)
	}
	if got[0].Subcode != "illegal-transition" {
		t.Errorf("subcode = %q, want illegal-transition (audit-only doesn't suppress illegal-transition per D-0008)", got[0].Subcode)
	}
}

// TestWalkAuditOnlyAckedEntities_PicksUpAcks pins the walker's
// behavior end-to-end: a repo with one ack commit produces a non-empty
// set; a repo with none produces an empty set; composite ids roll up.
func TestWalkAuditOnlyAckedEntities_PicksUpAcks(t *testing.T) {
	t.Parallel()

	t.Run("no acks → empty set", func(t *testing.T) {
		t.Parallel()
		r := newRepoFixture(t)
		r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
		acked := walkAuditOnlyAckedEntities(context.Background(), r.root)
		if len(acked) != 0 {
			t.Errorf("expected empty ack set; got %+v", acked)
		}
	})

	t.Run("single ack picked up", func(t *testing.T) {
		t.Parallel()
		r := newRepoFixture(t)
		r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
		r.gitCommitWithTrailers("aiwf cancel G-0042 [audit-only]",
			map[string]string{
				gitops.TrailerVerb:      "cancel",
				gitops.TrailerEntity:    "G-0042",
				gitops.TrailerActor:     "human/peter",
				gitops.TrailerAuditOnly: "test ack",
			})
		acked := walkAuditOnlyAckedEntities(context.Background(), r.root)
		if !acked["G-0042"] {
			t.Errorf("expected G-0042 in ack set; got %+v", acked)
		}
	})

	t.Run("composite id rolls up to parent", func(t *testing.T) {
		t.Parallel()
		r := newRepoFixture(t)
		r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
		// Audit-only on M-0001/AC-1 should map to M-0001 in the ack set
		// (mirrors the existing RunUntrailedAudit's compositeRoot rollup).
		r.gitCommitWithTrailers("aiwf promote M-0001/AC-1 met [audit-only]",
			map[string]string{
				gitops.TrailerVerb:      "promote",
				gitops.TrailerEntity:    "M-0001/AC-1",
				gitops.TrailerActor:     "human/peter",
				gitops.TrailerAuditOnly: "test ack",
			})
		acked := walkAuditOnlyAckedEntities(context.Background(), r.root)
		if !acked["M-0001"] {
			t.Errorf("expected M-0001 in ack set (composite rollup); got %+v", acked)
		}
	})

	t.Run("audit-only without aiwf-entity not picked up", func(t *testing.T) {
		t.Parallel()
		r := newRepoFixture(t)
		r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
		// Malformed: audit-only present, aiwf-entity absent. Should be ignored.
		r.gitCommitWithTrailers("malformed audit-only commit",
			map[string]string{
				gitops.TrailerAuditOnly: "no entity trailer",
				gitops.TrailerActor:     "human/peter",
			})
		acked := walkAuditOnlyAckedEntities(context.Background(), r.root)
		if len(acked) != 0 {
			t.Errorf("expected empty ack set (audit-only without aiwf-entity ignored); got %+v", acked)
		}
	})
}

// TestFSMHistoryConsistent_ManualEdit_MergeIntegrationSilent pins
// D-0010 at the AC-4 surface: a manual-edit commit on a feature
// branch (no verb trailer) fires AC-4 at the ORIGINAL commit; the
// merge integration is silent.
func TestFSMHistoryConsistent_ManualEdit_MergeIntegrationSilent(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add epic")
	r.commitEntity("M-0001", entity.KindMilestone, entity.StatusDraft, "add milestone")
	r.gitCheckoutBranch("branch-handedit")
	r.commitEntity("M-0001", entity.KindMilestone, entity.StatusInProgress, "hand-edit on branch (no verb trailer)")
	r.gitCheckout("main")
	r.gitMerge("branch-handedit", "merge branch-handedit into main")

	got := FSMHistoryConsistent(context.Background(), r.root, r.tree())
	if len(got) != 1 {
		t.Fatalf("expected 1 finding (original commit only; merge skipped per D-0010), got %d: %+v", len(got), got)
	}
	if got[0].Subcode != "manual-edit" {
		t.Errorf("subcode = %q, want manual-edit", got[0].Subcode)
	}
}
