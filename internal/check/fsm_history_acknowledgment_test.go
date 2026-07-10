package check

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// M-0136/AC-2 + AC-3: the fsm-history-consistent predicate extends
// to walk HEAD's reachable history for `aiwf-force-for` trailers and
// exempt illegal-transition findings whose offending commit SHA
// appears as a target. These tests pin the predicate-level behavior
// directly via gitops.CommitAllowEmpty (no import of internal/verb,
// which would create a cycle); verb.AcknowledgeIllegal's own tests
// live under internal/verb/ alongside the verb.

// TestFSMHistoryConsistent_AC2_AcknowledgmentExemptsIllegalTransition
// pins AC-2: when a current-day commit carries
// `aiwf-force-for: <historical-sha>` and that historical SHA is the
// commit that flagged an illegal-transition, the predicate exempts
// the finding.
//
// RED today: the predicate has no exemption logic — it always emits
// for the illegal observation. GREEN once `walkAcknowledgedSHAs` +
// the predicate's per-Commit-SHA skip land.
func TestFSMHistoryConsistent_AC2_AcknowledgmentExemptsIllegalTransition(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	illegalSHA := r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone,
		"skip-ahead proposed->done (FSM-illegal)")

	writeAcknowledgmentCommit(t, r.root, illegalSHA,
		"pre-rule era; intermediate transition lost to a squash")

	// M-0159/AC-3: the rule no longer recomputes ackedSHAs
	// internally; the caller (CLI gather layer in production)
	// computes it once and passes it in. Test mirrors that shape
	// by calling WalkAcknowledgedSHAs and passing the result.
	ackedSHAs := WalkAcknowledgedSHAs(context.Background(), r.root, mustHead(t, r.root))
	got := FSMHistoryConsistent(context.Background(), r.root, r.tree(), ackedSHAs, mustHead(t, r.root))
	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent && f.Subcode == "illegal-transition" && f.EntityID == "E-0001" {
			t.Errorf("expected no illegal-transition for E-0001 (acknowledged via aiwf-force-for: %s); got finding %+v",
				illegalSHA[:8], f)
		}
	}
}

// TestFSMHistoryConsistent_AC3_NoAcknowledgmentStillFires pins AC-3:
// the exemption is targeted — the predicate still fires on illegal-
// transition commits that have NO matching `aiwf-force-for` trailer
// in HEAD's history. No false negatives from the AC-2 extension.
//
// Today this is PASSING (the predicate fires on everything illegal)
// because the AC-2 extension doesn't exist yet. After GREEN, the
// extension exempts only acknowledged SHAs and this test verifies
// un-acknowledged ones still emerge.
func TestFSMHistoryConsistent_AC3_NoAcknowledgmentStillFires(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone,
		"skip-ahead proposed->done (FSM-illegal, NOT acknowledged)")

	// M-0159/AC-3: the rule no longer recomputes ackedSHAs
	// internally; the caller (CLI gather layer in production)
	// computes it once and passes it in. Test mirrors that shape
	// by calling WalkAcknowledgedSHAs and passing the result.
	ackedSHAs := WalkAcknowledgedSHAs(context.Background(), r.root, mustHead(t, r.root))
	got := FSMHistoryConsistent(context.Background(), r.root, r.tree(), ackedSHAs, mustHead(t, r.root))
	var hasFinding bool
	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent && f.Subcode == "illegal-transition" && f.EntityID == "E-0001" {
			hasFinding = true
		}
	}
	if !hasFinding {
		t.Errorf("expected illegal-transition finding for E-0001 (not acknowledged); got 0 such findings: %+v", got)
	}
}

// TestFSMHistoryConsistent_AC2_AcknowledgmentScopedToTarget pins the
// per-SHA scoping of the exemption: an acknowledgment for one SHA
// does NOT exempt a different illegal commit's finding. Catches a
// "blanket exempt everything" regression.
//
// RED today (no acknowledgment logic exists yet) — the test fails
// at the "exemptSHA still fires" assertion because the predicate
// fires on EVERY illegal-transition regardless of acknowledgment.
// Post-GREEN, the exemption is keyed on the specific commit SHA so
// only the exempt SHA's finding goes away.
func TestFSMHistoryConsistent_AC2_AcknowledgmentScopedToTarget(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	exemptSHA := r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone,
		"skip-ahead 1 (will be acknowledged)")
	// A second illegal transition on a DIFFERENT entity — not covered
	// by exemptSHA's acknowledgment.
	r.commitEntity("E-0002", entity.KindEpic, entity.StatusProposed, "add E-0002")
	r.commitEntity("E-0002", entity.KindEpic, entity.StatusDone,
		"skip-ahead 2 (NOT acknowledged)")

	writeAcknowledgmentCommit(t, r.root, exemptSHA, "acknowledged first skip only")

	// M-0159/AC-3: the rule no longer recomputes ackedSHAs
	// internally; the caller (CLI gather layer in production)
	// computes it once and passes it in. Test mirrors that shape
	// by calling WalkAcknowledgedSHAs and passing the result.
	ackedSHAs := WalkAcknowledgedSHAs(context.Background(), r.root, mustHead(t, r.root))
	got := FSMHistoryConsistent(context.Background(), r.root, r.tree(), ackedSHAs, mustHead(t, r.root))
	var exemptStillFires, otherFires bool
	for _, f := range got {
		if f.Code != CodeFSMHistoryConsistent || f.Subcode != "illegal-transition" {
			continue
		}
		if strings.Contains(f.Message, exemptSHA[:8]) {
			exemptStillFires = true
		}
		if f.EntityID == "E-0002" {
			otherFires = true
		}
	}
	if exemptStillFires {
		t.Errorf("expected no illegal-transition finding mentioning acknowledged SHA %s; got %+v",
			exemptSHA[:8], got)
	}
	if !otherFires {
		t.Errorf("expected illegal-transition finding for E-0002 (un-acknowledged); got 0 such findings: %+v", got)
	}
}

// TestFSMHistoryConsistent_G0395_RevivedFindingCarriesDanglingAckHint
// pins the production wiring end-to-end (not illegalTransitionFindings
// and findDanglingAckHint in isolation, which their own test files
// already cover): FSMHistoryConsistent, called through its real public
// entry point against a repo where a rebase has dropped just the
// acknowledgment commit, must return the revived illegal-transition
// finding with its Hint field naming the dangling acknowledgment.
func TestFSMHistoryConsistent_G0395_RevivedFindingCarriesDanglingAckHint(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	illegalSHA := r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone,
		"skip-ahead proposed->done (FSM-illegal)")
	writeAcknowledgmentCommit(t, r.root, illegalSHA, "test ack")
	ackSHA := strings.TrimSpace(r.run("git", "rev-parse", "HEAD"))
	r.gitCommit("unrelated follow-up")
	// 2-arg form stays on and moves the current branch — see
	// acks_test.go's identical rationale for why the 3-arg "HEAD" form
	// would defeat this reproduction.
	r.run("git", "rebase", "-q", "--onto", illegalSHA, ackSHA)

	ctx := context.Background()
	ackedSHAs := WalkAcknowledgedSHAs(ctx, r.root, mustHead(t, r.root))
	got := FSMHistoryConsistent(ctx, r.root, r.tree(), ackedSHAs, mustHead(t, r.root))

	var found *Finding
	for i := range got {
		if got[i].Code == CodeFSMHistoryConsistent && got[i].Subcode == "illegal-transition" && got[i].EntityID == "E-0001" {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected a revived illegal-transition finding for E-0001; got %+v", got)
	}
	if found.Hint == "" {
		t.Fatal("expected the revived finding's Hint to name the dangling acknowledgment; got empty string")
	}
	if !strings.Contains(found.Hint, ackSHA[:8]) {
		t.Errorf("Hint %q does not name the dangling ack commit %s", found.Hint, ackSHA[:8])
	}
}

// TestFSMHistoryConsistent_G0395_ForcedUntrailered_RevivedFindingCarriesDanglingAckHint
// mirrors the illegal-transition case above for the forced-untrailered
// subcode: fsmHistoryConsistentWithDeps wires the same danglingHint
// closure to both predicates (see the production call site), so a
// revived forced-untrailered finding — a sovereign-act-shape
// transition by a non-human actor whose acknowledgment commit has
// gone dangling the same way — must also carry the hint. Without this
// test, dropping the forced-untrailered wiring alone (leaving
// illegal-transition's intact) would pass every other test in this
// package.
func TestFSMHistoryConsistent_G0395_ForcedUntrailered_RevivedFindingCarriesDanglingAckHint(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
	forcedSHA := r.commitEntityWithTrailers("E-0001", entity.KindEpic, entity.StatusActive,
		"activate E-0001 (sovereign-act-shape, non-human, no force)",
		map[string]string{gitops.TrailerActor: "ai/claude"})
	writeAcknowledgmentCommit(t, r.root, forcedSHA, "test ack")
	ackSHA := strings.TrimSpace(r.run("git", "rev-parse", "HEAD"))
	r.gitCommit("unrelated follow-up")
	// 2-arg form stays on and moves the current branch — see
	// acks_test.go's identical rationale for why the 3-arg "HEAD" form
	// would defeat this reproduction.
	r.run("git", "rebase", "-q", "--onto", forcedSHA, ackSHA)

	ctx := context.Background()
	ackedSHAs := WalkAcknowledgedSHAs(ctx, r.root, mustHead(t, r.root))
	got := FSMHistoryConsistent(ctx, r.root, r.tree(), ackedSHAs, mustHead(t, r.root))

	var found *Finding
	for i := range got {
		if got[i].Code == CodeFSMHistoryConsistent && got[i].Subcode == "forced-untrailered" && got[i].EntityID == "E-0001" {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected a revived forced-untrailered finding for E-0001; got %+v", got)
	}
	if found.Hint == "" {
		t.Fatal("expected the revived finding's Hint to name the dangling acknowledgment; got empty string")
	}
	if !strings.Contains(found.Hint, ackSHA[:8]) {
		t.Errorf("Hint %q does not name the dangling ack commit %s", found.Hint, ackSHA[:8])
	}
}

// writeAcknowledgmentCommit synthesizes an acknowledge-illegal commit
// directly via gitops.CommitAllowEmpty, sidestepping the verb package
// to avoid the check ↔ verb import cycle. The commit shape mirrors
// what verb.AcknowledgeIllegal produces — same four trailers, same
// AllowEmpty semantics.
func writeAcknowledgmentCommit(t *testing.T, root, targetSHA, reason string) {
	t.Helper()
	subject := "aiwf acknowledge illegal " + targetSHA[:min(8, len(targetSHA))]
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "acknowledge-illegal"},
		{Key: gitops.TrailerForceFor, Value: targetSHA},
		{Key: gitops.TrailerActor, Value: "human/test"},
		{Key: gitops.TrailerReason, Value: reason},
	}
	if err := gitops.CommitAllowEmpty(context.Background(), root, subject, reason, trailers); err != nil {
		t.Fatalf("CommitAllowEmpty (acknowledgment): %v", err)
	}
}
