package check

import (
	"sort"
	"strings"
	"testing"
)

// M-0293/AC-2. A hint that offers `--force` as the remedy states what
// the override relaxes and what it leaves standing.
//
// The caveat is appended in HintFor from one constant rather than
// written into each hint, for the reason M-0292 settled for the
// ratification sentence: four hints across three rule families offer
// the flag today, and a fifth author would otherwise have to remember.
//
// Deriving it costs the obvious assertion — "every offering hint has
// the caveat" cannot fail when the appender uses the same predicate the
// assertion does. So the assertions below are aimed at the three things
// that can: whether the predicate classifies the real table correctly,
// whether the join produces a readable sentence, and whether the wiring
// reaches HintFor's output at all.

// forceOfferingHintKeys are the hintTable keys whose remedy an operator
// can act on by typing `--force`. Hand-written on purpose: this is the
// expectation offersForceAsRemedy is measured against, so deriving it
// from that predicate would leave nothing to compare.
var forceOfferingHintKeys = []string{
	"fsm-history-consistent/forced-untrailered",
	"fsm-history-consistent/illegal-transition",
	"milestone-done-incomplete-acs",
	"provenance-force-non-human",
}

// TestOffersForceAsRemedy_ClassifiesTheWholeHintTable is the assertion
// over the table AC-2 asks for, rather than a spot-check of one hint.
//
// It fails in both directions: a hint that starts offering the flag
// without being listed here, and a listed hint whose wording drifted
// out of the predicate's reach. The second is the quiet one — the hint
// would simply stop carrying the caveat, with nothing else changing.
func TestOffersForceAsRemedy_ClassifiesTheWholeHintTable(t *testing.T) {
	t.Parallel()

	var got []string
	for key, hint := range hintTable {
		if offersForceAsRemedy(hint) {
			got = append(got, key)
		}
	}
	sort.Strings(got)

	want := append([]string(nil), forceOfferingHintKeys...)
	sort.Strings(want)

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("offersForceAsRemedy classified a different set than expected.\n got: %v\nwant: %v\n"+
			"A key in `got` but not `want` is a hint that began offering --force without the "+
			"caveat being considered; a key in `want` but not `got` is a hint whose wording "+
			"drifted out of the predicate, which silently drops its caveat", got, want)
	}
}

// TestOffersForceAsRemedy_MentioningForceIsNotOfferingIt pins the
// discriminator against the case that motivates it.
//
// milestone-cancelled-incomplete-acs names `--force` in order to say the
// override does not exist for that transition. Appending "here is what
// --force relaxes" to a hint whose point is that there is no --force
// would contradict it. The pairing with --reason is the kernel's own
// signal for the sovereign meaning, quoted from internal/policies/
// sovereign.go, and it is what separates the two.
func TestOffersForceAsRemedy_MentioningForceIsNotOfferingIt(t *testing.T) {
	t.Parallel()

	const mentions = "milestone-cancelled-incomplete-acs"
	hint, ok := hintTable[mentions]
	if !ok {
		t.Fatalf("%s is not in hintTable; the control case is stale and this test asserts nothing", mentions)
	}
	if !strings.Contains(hint, "--force") {
		t.Fatalf("%s no longer mentions --force at all, so it can no longer serve as the "+
			"mentions-but-does-not-offer control", mentions)
	}
	if offersForceAsRemedy(hint) {
		t.Errorf("%s was classified as offering --force. Its hint says the override does not "+
			"exist for this transition, so appending the caveat would contradict it", mentions)
	}
}

// TestHintFor_EveryForceOfferingHintStatesWhatForceLeavesStanding is the
// end-to-end half: the caveat has to survive HintFor's composition, not
// merely exist as a constant.
func TestHintFor_EveryForceOfferingHintStatesWhatForceLeavesStanding(t *testing.T) {
	t.Parallel()

	if len(forceOfferingHintKeys) == 0 {
		t.Fatal("no force-offering hints listed; this test would pass over an empty set")
	}

	for _, key := range forceOfferingHintKeys {
		code, subcode, _ := strings.Cut(key, "/")
		rendered := HintFor(code, subcode)
		if rendered == "" {
			t.Errorf("%s: HintFor returned an empty hint; the key does not resolve", key)
			continue
		}
		if !strings.Contains(rendered, forceCaveatSentence) {
			t.Errorf("%s: the rendered hint offers --force without saying what the override "+
				"leaves standing. An operator reads it as a general escape from the finding "+
				"it is attached to. Rendered: %q", key, rendered)
		}
	}
}

// TestHintFor_HintsThatDoNotOfferForceAreUntouched keeps the caveat off
// the hints it would only lengthen.
func TestHintFor_HintsThatDoNotOfferForceAreUntouched(t *testing.T) {
	t.Parallel()

	offering := make(map[string]bool, len(forceOfferingHintKeys))
	for _, k := range forceOfferingHintKeys {
		offering[k] = true
	}

	checked := 0
	for key := range hintTable {
		if offering[key] {
			continue
		}
		code, subcode, _ := strings.Cut(key, "/")
		if strings.Contains(HintFor(code, subcode), forceCaveatSentence) {
			t.Errorf("%s does not offer --force but its hint carries the caveat anyway", key)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("hintTable held no non-offering hints; the walk found nothing to check")
	}
}

// TestWithForceCaveat_JoinsOnOneSentenceBreak covers the join directly,
// with inputs written here rather than taken from the table. The table
// hints are overwhelmingly unpunctuated, so the punctuated arm would
// otherwise go untested until a hint that ends in a period started
// offering the flag — and the failure it produces is a run-on sentence
// in an operator's terminal, which nothing else would catch.
func TestWithForceCaveat_JoinsOnOneSentenceBreak(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		hint string
		want string
	}{
		{
			name: "unpunctuated hint gains a sentence break",
			hint: "re-run it as `aiwf <verb> <id> --force --reason \"...\"`",
			want: "re-run it as `aiwf <verb> <id> --force --reason \"...\"`. " + forceCaveatSentence,
		},
		{
			name: "punctuated hint is joined by a space alone",
			hint: "re-run it with `--force --reason \"...\"`.",
			want: "re-run it with `--force --reason \"...\"`. " + forceCaveatSentence,
		},
		{
			name: "hint that does not offer force is returned unchanged",
			hint: "promote each open AC first",
			want: "promote each open AC first",
		},
		{
			name: "empty hint stays empty",
			hint: "",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := withForceCaveat(tc.hint); got != tc.want {
				t.Errorf("withForceCaveat(%q)\n got: %q\nwant: %q", tc.hint, got, tc.want)
			}
		})
	}
}

// TestForceCaveatSentence_StatesBothHalves pins the content AC-2 names.
// A caveat that said only "force is human-only" would leave an operator
// believing the flag waives the rest of the finding, which is the
// reading the AC exists to close.
func TestForceCaveatSentence_StatesBothHalves(t *testing.T) {
	t.Parallel()

	// What it relaxes.
	if !strings.Contains(forceCaveatSentence, "transition rule") {
		t.Error("the caveat does not name what --force relaxes; without it the reader cannot " +
			"tell how far the override reaches")
	}
	// What it leaves standing: the actor constraint, and every other check.
	if !strings.Contains(forceCaveatSentence, "human/") {
		t.Error("the caveat does not name the actor constraint --force does not relax")
	}
	if !strings.Contains(forceCaveatSentence, "every other check still runs") {
		t.Error("the caveat does not say the remaining checks still run, so it reads as a " +
			"general escape from the finding it is attached to")
	}
}
