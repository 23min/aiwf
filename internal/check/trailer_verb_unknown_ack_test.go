package check

import (
	"testing"

	"github.com/23min/aiwf/internal/scope"
)

// trailer_verb_unknown_ack_test.go — M-0159/AC-3 red phase: pin
// that RunTrailerVerbUnknown accepts an ackedSHAs map[string]bool
// 4th parameter and silences findings whose Commit appears in it.
//
// Compile-RED today: current RunTrailerVerbUnknown has 3 params
// (commits, registeredVerbs, ritualVerbs); these tests call it
// with 4. The red signal is `too many arguments in call to
// RunTrailerVerbUnknown`.
//
// Green phase: lift walkAcknowledgedSHAs to internal/check/acks.go,
// add the 4th param here, exempt commits whose SHA matches in the
// ackedSHAs map. The lift unblocks AC-5's real-git E2E (which
// converts the docstring promise at trailer_verb_unknown.go:25-29
// into mechanical truth end-to-end); this file pins only the
// rule's signature consumption side — that the rule HAS the
// param and uses it correctly per-SHA.
//
// Per-SHA scoping mirrors the M-0136/AC-2 illegal-transition ack
// shape: an ack for SHA-X does NOT exempt other unknown-verb
// commits. A nil/empty map means "no acknowledgments" so the rule
// polices as usual.

// TestRunTrailerVerbUnknown_AC3_AckedSHASilencesUnknownVerb pins
// the happy path: an ackedSHAs map containing the offending
// commit's SHA silences the trailer-verb-unknown warning. Mirrors
// the canonical fixture from TestRunTrailerVerbUnknown_FiresOnFabricatedVerb
// so the only behavioral delta is the ackedSHAs presence.
func TestRunTrailerVerbUnknown_AC3_AckedSHASilencesUnknownVerb(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{
		"add":     {},
		"promote": {},
	}
	commits := []scope.Commit{
		commitWithVerb("aaa1111", "implement"), // fabricated verb, would fire normally
	}
	ackedSHAs := map[string]bool{
		"aaa1111": true,
	}

	got := RunTrailerVerbUnknown(commits, registered, nil, ackedSHAs)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (ack silences the unknown-verb commit per AC-3 lift consumption); got %d: %+v", len(got), got)
	}
}

// TestRunTrailerVerbUnknown_AC3_AckedMapWithoutCommitSHA_StillFires
// is the positive control: an ackedSHAs map that does NOT contain
// the offending commit's SHA must NOT silence the finding. Pins
// per-SHA closed-set scoping — a green-phase regression that
// silenced on "ackedSHAs is non-empty" would pass the happy-path
// test above and silently over-exempt every unknown-verb commit.
func TestRunTrailerVerbUnknown_AC3_AckedMapWithoutCommitSHA_StillFires(t *testing.T) {
	t.Parallel()
	registered := map[string]struct{}{
		"add":     {},
		"promote": {},
	}
	commits := []scope.Commit{
		commitWithVerb("aaa1111", "implement"),
	}
	ackedSHAs := map[string]bool{
		"unrelated-sha-xyz": true,
	}

	got := RunTrailerVerbUnknown(commits, registered, nil, ackedSHAs)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding (per-SHA scoping: ack on unrelated SHA must not exempt aaa1111); got %d: %+v", len(got), got)
	}
	if got[0].Code != CodeTrailerVerbUnknown {
		t.Errorf("Code = %q; want %q", got[0].Code, CodeTrailerVerbUnknown)
	}
}
