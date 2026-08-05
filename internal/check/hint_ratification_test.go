package check

import (
	"strings"
	"testing"
)

// hint_ratification_test.go pins M-0292's discoverability claim: an
// operator who meets a ratifiable provenance finding is told about
// `aiwf acknowledge illegal` at the moment they meet it.
//
// The roster is derived from hintTable rather than written out here.
// Every rule RunProvenance runs is cleared by an acknowledgment, so a
// provenance code added later is ratifiable the day it lands; deriving
// the list means it either carries the remedy or fails this test,
// rather than depending on whoever adds it noticing.

// TestHintFor_EveryRatifiableProvenanceCodeAdvertisesTheRemedy walks
// every provenance code carrying a hint and asserts the ratification
// sentence is part of what the operator reads.
//
// Why it matters that this is not per-code opt-in: before M-0292 the
// hints for these codes named only a re-run of the verb, a correction
// to git config, or an amend. The first two cannot be performed on a
// commit that has already landed, and the third performs it by
// rewriting history — the outcome this milestone exists to make
// unnecessary. A hint that names only those leaves the operator stuck
// while the remedy sits one command away.
func TestHintFor_EveryRatifiableProvenanceCodeAdvertisesTheRemedy(t *testing.T) {
	t.Parallel()
	seen := 0
	for code := range hintTable {
		if strings.Contains(code, "/") || !strings.HasPrefix(code, "provenance-") {
			continue
		}
		if !ratifiableByAcknowledgment(code) {
			continue
		}
		seen++
		hint := HintFor(code, "")
		if !strings.Contains(hint, "acknowledge illegal") {
			t.Errorf("%s is cleared by an acknowledgment but its hint never says so; got: %s", code, hint)
		}
		if !strings.Contains(hint, "every provenance finding against that one commit") {
			t.Errorf("%s's hint names the command without its scope; an operator ratifying one "+
				"finding needs to know the acknowledgment retires the commit's others too; got: %s", code, hint)
		}
		// The two sentences must not run together. Table hints carry no
		// trailing punctuation, so a bare concatenation produces
		// "...of the email A commit already in history..." — readable
		// enough to survive review and wrong in every rendered finding.
		if idx := strings.Index(hint, ratificationSentence); idx > 0 {
			if before := hint[idx-2 : idx]; before != ". " {
				t.Errorf("%s's hint joins the ratification sentence as %q, want a sentence break; got: %s",
					code, before, hint)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no ratifiable provenance codes found in hintTable; the derivation is broken and " +
			"this test would pass no matter what the hints said")
	}
}

// TestHintFor_PerEntityRuleDoesNotAdvertiseTheBlanketRemedy is the
// negative half. provenance-untrailered-entity-commit is cleared only
// by the `--for-entity` shape, whose (SHA, entity) binding the verb
// verifies against the commit's own diff — a blanket acknowledgment
// leaves it firing. Advertising the blanket command on it would send an
// operator at an invocation that changes nothing they can see.
func TestHintFor_PerEntityRuleDoesNotAdvertiseTheBlanketRemedy(t *testing.T) {
	t.Parallel()
	for _, code := range []string{CodeProvenanceUntrailedEntityCommit, CodeProvenanceUntrailedScopeUndefined} {
		if got := HintFor(code, ""); strings.Contains(got, ratificationSentence) {
			t.Errorf("%s advertises the blanket ratification, which does not clear it; got: %s", code, got)
		}
	}
}

// TestHintFor_NonProvenanceCodesAreUntouched guards the seam the
// derivation rests on: the sentence is appended by prefix, so a
// mis-scoped predicate would quietly append it to unrelated hints.
func TestHintFor_NonProvenanceCodesAreUntouched(t *testing.T) {
	t.Parallel()
	for _, code := range []string{"ids-unique", "acs-tdd-audit", "trailer-verb-unknown", "id-rename-untrailered"} {
		got := HintFor(code, "")
		if got == "" {
			t.Errorf("%s has no hint; the fixture no longer exercises what it claims to", code)
			continue
		}
		if strings.Contains(got, ratificationSentence) {
			t.Errorf("%s is not a provenance rule but its hint carries the ratification sentence; got: %s", code, got)
		}
	}
}

// TestWithRatificationHint_JoinsOnOneSentenceBreak covers both arms of
// the separator choice. No hint in the table ends with punctuation
// today, so the already-punctuated arm is unreachable through HintFor
// and is exercised here directly — it is what keeps the join correct
// if a hint is ever written with a closing period.
func TestWithRatificationHint_JoinsOnOneSentenceBreak(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, hint, want string
	}{
		{"unpunctuated gains a period", "run `aiwf doctor`", "run `aiwf doctor`. " + ratificationSentence},
		{"period is not doubled", "run `aiwf doctor`.", "run `aiwf doctor`. " + ratificationSentence},
		{"question mark is kept", "did you mean `aiwf doctor`?", "did you mean `aiwf doctor`? " + ratificationSentence},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := withRatificationHint(CodeProvenanceForceNonHuman, tc.hint); got != tc.want {
				t.Errorf("withRatificationHint(%q)\n got: %q\nwant: %q", tc.hint, got, tc.want)
			}
		})
	}
}

// TestHintFor_EmptyHintStaysEmpty pins that a code with no registered
// hint emits none. Appending the remedy to an otherwise-empty hint
// would produce advice to ratify a finding the reader was never told
// anything else about.
func TestHintFor_EmptyHintStaysEmpty(t *testing.T) {
	t.Parallel()
	if got := HintFor("provenance-not-a-real-code", ""); got != "" {
		t.Errorf("HintFor on an unregistered provenance code = %q, want empty", got)
	}
}

// TestHintFor_SubcodeResolutionRespectsRatifiability pins the subcode
// branch, which resolves through its own map lookup and so has to apply
// the same predicate the bare-code branch does.
//
// The one registered provenance subcode belongs to the rule a blanket
// acknowledgment does not clear, so this asserts the subcode hint
// resolves and stays free of the sentence. Were a ratifiable subcode
// registered later, the bare-code sibling would already be carrying it
// and the mismatch would show here.
func TestHintFor_SubcodeResolutionRespectsRatifiability(t *testing.T) {
	t.Parallel()
	const code, subcode = CodeProvenanceUntrailedEntityCommit, "squash-merge"
	if _, ok := hintTable[code+"/"+subcode]; !ok {
		t.Fatalf("no hint registered for %s/%s; this test's subject no longer exists", code, subcode)
	}
	got := HintFor(code, subcode)
	if got == HintFor(code, "") {
		t.Errorf("subcode lookup returned the bare-code hint; the subcode branch is not resolving")
	}
	if strings.Contains(got, ratificationSentence) {
		t.Errorf("the %s/%s hint advertises a blanket ratification that does not clear it; got: %s", code, subcode, got)
	}
}
