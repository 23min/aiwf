package stresstest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// force_override_durability_test.go — real-subprocess coverage for
// ForceOverrideDurabilityScenario (M-0243/AC-4). The pure decision
// logic (classifyForceOverrideDurability) is pinned exhaustively in
// force_override_durability_classify_test.go against fabricated
// outcomes; this is the actual scenario, driving a real
// acknowledge-illegal/rebase sequence and a real force-promote/
// cherry-pick sequence.

// TestForceOverrideDurabilityScenario_RealBinary_ConfirmsAckRevocationByRebase
// is the scenario's real-binary happy path — and, per this AC's own
// framing, the expected pass state is that the confirmed defect (item
// 5's ack revocation) IS the one violation reported, with item 6's
// premise checks all clean.
func TestForceOverrideDurabilityScenario_RealBinary_ConfirmsAckRevocationByRebase(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewForceOverrideDurabilityScenario(bin)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if result.Passed {
		t.Fatal("expected the scenario to report the confirmed ack-revocation-by-rebase defect as a violation, not pass cleanly")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected exactly 1 violation (the confirmed ack revocation), got %d: %+v", len(result.Violations), result.Violations)
	}
	if !strings.Contains(result.Violations[0].Message, "not durable against history rewrites") {
		t.Fatalf("expected the violation to name the ack-durability defect, got: %+v", result.Violations[0])
	}
}

func TestForceOverrideDurabilityScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewForceOverrideDurabilityScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"))
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "seeding the ack-target epic") {
		t.Fatalf("expected the failure to name the seeding step, got: %v", err)
	}
}

// TestHeadSHA_ErrorsOnUnreadableDir pins headSHA's own error branch via
// a directory with no git repo at all.
func TestHeadSHA_ErrorsOnUnreadableDir(t *testing.T) {
	t.Parallel()
	if _, err := headSHA(t.TempDir()); err == nil {
		t.Fatal("expected headSHA to error against a non-git directory")
	}
}

// TestCommitTrailerValue_ErrorsOnUnreadableRef pins
// commitTrailerValue's own error branch via a nonexistent ref.
func TestCommitTrailerValue_ErrorsOnUnreadableRef(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := gitInitAndConfig(dir); err != nil {
		t.Fatalf("gitInitAndConfig: %v", err)
	}
	if _, err := commitTrailerValue(dir, "no-such-ref", "aiwf-force"); err == nil {
		t.Fatal("expected commitTrailerValue to error against a nonexistent ref")
	}
}

// TestHasFindingSubcodeForEntity_MatchesOnAllThreeDimensions pins
// hasFindingSubcodeForEntity's discriminating branches.
func TestHasFindingSubcodeForEntity_MatchesOnAllThreeDimensions(t *testing.T) {
	t.Parallel()
	findings := []verbEnvelopeFinding{
		{Code: check.CodeFSMHistoryConsistent, Subcode: "illegal-transition", EntityID: "E-0001"},
		{Code: check.CodeFSMHistoryConsistent, Subcode: "manual-edit", EntityID: "E-0001"},
		{Code: "some-other-code", Subcode: "illegal-transition", EntityID: "E-0001"},
	}
	if hasFindingSubcodeForEntity(findings, check.CodeFSMHistoryConsistent, "illegal-transition", "E-0002") {
		t.Fatal("expected no match: entity differs")
	}
	if hasFindingSubcodeForEntity(findings, check.CodeFSMHistoryConsistent, "forced-untrailered", "E-0001") {
		t.Fatal("expected no match: subcode differs")
	}
	if hasFindingSubcodeForEntity(findings, "different-code", "illegal-transition", "E-0001") {
		t.Fatal("expected no match: code differs")
	}
	if !hasFindingSubcodeForEntity(findings, check.CodeFSMHistoryConsistent, "illegal-transition", "E-0001") {
		t.Fatal("expected a match on the exact code+subcode+entity triple")
	}
}
