package verb

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// updateCoherenceGolden regenerates the domain golden. Refresh with
// `go test ./internal/verb/ -run TestCheckTrailerCoherence_FullDomain -update`
// and read the diff: every changed line is a verdict that changed.
var updateCoherenceGolden = flag.Bool("update", false, "regenerate the coherence domain golden file")

// coherenceActors is the actor-role axis. The absent actor is a real
// point in the domain, not a degenerate one: several rules turn on
// "non-human", which the implementation defines as present-and-not-
// human, so an absent actor is neither human nor non-human.
var coherenceActors = []struct{ name, value string }{
	{"absent", ""},
	{"human", "human/peter"},
	{"nonhuman", "ai/claude"},
}

// coherenceFlag is one trailer on the presence axis.
type coherenceFlag struct{ name, key, value string }

// coherenceFlags is the trailer-presence axis, derived from the rule
// declaration rather than listed. A rule reading a trailer no other
// rule reads widens this axis by construction, so it is exercised
// somewhere in the domain instead of firing at no point in it.
//
// Values are irrelevant to every rule — each reads presence only — so
// one representative value covers the axis.
var coherenceFlags = flagsFor(coherenceRuleSpecs)

// flagsFor renders a declaration's trailer axis as the presence axis
// the domain varies. Parameterized so AC-2's fixtures can build a
// domain from a deliberately mis-declared set.
func flagsFor(specs []coherenceRuleSpec) []coherenceFlag {
	axis := declaredCoherenceTrailerAxis(specs)
	out := make([]coherenceFlag, 0, len(axis))
	for _, key := range axis {
		out = append(out, coherenceFlag{name: coherenceFlagName(key), key: key, value: "set"})
	}
	return out
}

// coherenceFlagName renders a trailer key as the compact label the
// domain's case names use: aiwf-on-behalf-of becomes onbehalfof.
func coherenceFlagName(trailerKey string) string {
	return strings.ReplaceAll(strings.TrimPrefix(trailerKey, "aiwf-"), "-", "")
}

// coherenceCase is one point in the domain.
type coherenceCase struct {
	name     string
	trailers []gitops.Trailer
	// present reports whether a given flag name is set in this case.
	present map[string]bool
	actor   string
	// actorName and mask locate this case on the domain's two axes, so
	// AC-2 can find the case differing from it in exactly one trailer.
	actorName string
	mask      int
}

// coherenceDomain enumerates the complete input domain: every actor role
// crossed with every subset of the presence-bearing trailers. Generated
// rather than enumerated, so coverage is a property of this function and
// a rule added against a new trailer is a one-line change here rather
// than a fresh set of hand-written cases someone must remember to add.
func coherenceDomain() []coherenceCase { return coherenceDomainFrom(coherenceFlags) }

// coherenceDomainFrom builds the domain over an explicit presence axis,
// so a fixture can generate the domain a mis-declared set would produce.
func coherenceDomainFrom(flags []coherenceFlag) []coherenceCase {
	var out []coherenceCase
	for _, actor := range coherenceActors {
		for mask := 0; mask < 1<<len(flags); mask++ {
			trailers := []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
			}
			if actor.value != "" {
				trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerActor, Value: actor.value})
			}
			present := make(map[string]bool, len(flags))
			var set []string
			for i, f := range flags {
				on := mask&(1<<i) != 0
				present[f.name] = on
				if on {
					trailers = append(trailers, gitops.Trailer{Key: f.key, Value: f.value})
					set = append(set, f.name)
				}
			}
			// Sorted, so a case name states which trailers are present
			// and not the order the axis happens to carry them in. That
			// decouples the golden from the axis: a rule widening the
			// axis adds lines rather than rewriting every one.
			sort.Strings(set)
			parts := append([]string{"actor=" + actor.name}, set...)
			out = append(out, coherenceCase{
				name:      strings.Join(parts, "+"),
				trailers:  trailers,
				present:   present,
				actor:     actor.value,
				actorName: actor.name,
				mask:      mask,
			})
		}
	}
	return out
}

// verdict names the rule CheckTrailerCoherence reported, or "coherent".
func verdict(trailers []gitops.Trailer) string {
	err := CheckTrailerCoherence(trailers)
	if err == nil {
		return "coherent"
	}
	if _, rule := AsCoherenceError(err); rule != "" {
		return rule
	}
	return "unknown:" + err.Error()
}

// TestCheckTrailerCoherence_FullDomain_MatchesGolden is M-0291/AC-2.
//
// Pins the verdict at every point of the domain. The golden records
// which rule fires, which the invariant test below deliberately does not
// assert: CheckTrailerCoherence reports its first violation only, so for
// a set violating several rules the choice is an ordering decision. The
// golden is where that ordering is visible and reviewable; a changed
// line in its diff is a changed verdict.
func TestCheckTrailerCoherence_FullDomain_MatchesGolden(t *testing.T) {
	t.Parallel()

	domain := coherenceDomain()
	lines := make([]string, 0, len(domain))
	for _, tc := range domain {
		lines = append(lines, fmt.Sprintf("%s => %s", tc.name, verdict(tc.trailers)))
	}
	// Sorted for the same reason the case names are: the golden records
	// a set of verdicts, so its diff shows a changed verdict rather than
	// a changed iteration order.
	sort.Strings(lines)
	got := strings.Join(lines, "\n") + "\n"

	path := filepath.Join("testdata", "coherence_domain.golden")
	if *updateCoherenceGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\n(run with -update to create)", path, err)
	}
	if string(want) != got {
		t.Errorf("coherence domain golden mismatch\n(run with -update to refresh after an intentional change)\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

// TestCheckTrailerCoherence_FullDomain_SatisfiesDesignDocRules is
// M-0291/AC-2's semantic half.
//
// The golden above pins what the implementation does; on its own a
// golden can be regenerated past a real regression without anyone
// reading the diff. These invariants come from provenance-model.md
// §"Required-together and mutually-exclusive rules" — the rules as the
// design states them, not as the code arranges them — so they fail if a
// refresh ever silently drops one.
//
// Each asserts only that *some* violation is reported, never which. The
// rules are checked in an order the design doc does not fix, so naming
// one would pin the implementation's arrangement rather than the
// documented rule.
func TestCheckTrailerCoherence_FullDomain_SatisfiesDesignDocRules(t *testing.T) {
	t.Parallel()

	invariants := []struct {
		name string
		// applies selects the domain points the rule governs.
		applies func(c coherenceCase) bool
		// wantCoherent is true when the selected points must report no
		// violation, false when every one must report some violation.
		wantCoherent bool
	}{
		{
			name: "on-behalf-of and authorized-by are required together",
			applies: func(c coherenceCase) bool {
				return c.present["onbehalfof"] != c.present["authorizedby"]
			},
		},
		{
			name: "a principal requires a non-human actor",
			applies: func(c coherenceCase) bool {
				return c.present["principal"] && !isNonHumanActor(c.actor)
			},
		},
		{
			name: "a non-human actor requires a principal",
			applies: func(c coherenceCase) bool {
				return isNonHumanActor(c.actor) && !c.present["principal"]
			},
		},
		{
			name: "force and on-behalf-of are mutually exclusive",
			applies: func(c coherenceCase) bool {
				return c.present["force"] && c.present["onbehalfof"]
			},
		},
		{
			name: "on-behalf-of is forbidden for a human actor",
			applies: func(c coherenceCase) bool {
				return c.present["onbehalfof"] && strings.HasPrefix(c.actor, "human/")
			},
		},
		{
			name: "force requires a human actor",
			applies: func(c coherenceCase) bool {
				return c.present["force"] && isNonHumanActor(c.actor)
			},
		},
		{
			name: "audit-only and force are mutually exclusive",
			applies: func(c coherenceCase) bool {
				return c.present["auditonly"] && c.present["force"]
			},
		},
		{
			name: "audit-only requires a human actor",
			applies: func(c coherenceCase) bool {
				return c.present["auditonly"] && isNonHumanActor(c.actor)
			},
		},
		{
			name: "a human actor with no provenance trailers is coherent",
			applies: func(c coherenceCase) bool {
				return strings.HasPrefix(c.actor, "human/") &&
					!c.present["principal"] && !c.present["onbehalfof"] &&
					!c.present["authorizedby"] && !c.present["force"] &&
					!c.present["auditonly"]
			},
			wantCoherent: true,
		},
	}

	domain := coherenceDomain()
	for _, inv := range invariants {
		t.Run(inv.name, func(t *testing.T) {
			t.Parallel()
			selected := 0
			for _, c := range domain {
				if !inv.applies(c) {
					continue
				}
				selected++
				got := verdict(c.trailers)
				switch {
				case inv.wantCoherent && got != "coherent":
					t.Errorf("%s: got %s, want coherent", c.name, got)
				case !inv.wantCoherent && got == "coherent":
					t.Errorf("%s: got coherent, want a violation", c.name)
				}
			}
			// A selector matching nothing reports success while proving
			// nothing — the same shape as a scan over an empty population.
			if selected == 0 {
				t.Errorf("invariant selected no domain points; it cannot fail and so pins nothing")
			}
		})
	}
}

// isNonHumanActor mirrors the domain's own notion of a non-human actor:
// present, and not a human/ role. An absent actor is neither.
func isNonHumanActor(actor string) bool {
	return actor != "" && !strings.HasPrefix(actor, "human/")
}

// TestCheckTrailerCoherence_EveryRuleIsReachable asserts each declared
// rule fires at some point in the domain.
//
// A rule can be made unreachable without being deleted: the checks run
// in order and return the first violation, so a broadened earlier
// condition can shadow a later rule entirely. That leaves a rule that
// reads as enforced, is covered by its own unit test through a direct
// call, and never fires for any real trailer set — the same shape of
// claim-without-enforcement this milestone exists to remove.
func TestCheckTrailerCoherence_EveryRuleIsReachable(t *testing.T) {
	t.Parallel()

	fired := make(map[string]bool, len(coherenceRuleSpecs))
	for _, c := range coherenceDomain() {
		fired[verdict(c.trailers)] = true
	}
	for _, rule := range declaredCoherenceRules(coherenceRuleSpecs) {
		if !fired[rule] {
			t.Errorf("rule %q fires at no point in the domain; it is shadowed by an earlier rule or unreachable", rule)
		}
	}
}

// forcePredicatedRules are the rules whose condition requires an
// aiwf-force trailer, derived from the rule declaration rather than
// from the function under test. Deriving it from
// CheckForceTrailerCoherence would make the subset assertion below
// compare that function to itself; the declaration is data stated
// beside each rule, and M-0294/AC-2 holds it to the rule's behavior.
var forcePredicatedRules = func() map[string]bool {
	out := map[string]bool{}
	for _, rule := range declaredForcePredicatedRules(coherenceRuleSpecs) {
		out[rule] = true
	}
	return out
}()

// TestCheckForceTrailerCoherence_IsTheForcePredicatedSubset pins what
// verb.Apply enforces, across the same generated domain the full rule
// set is pinned over.
//
// The seam's scope can be wrong in two directions and only one of them
// is visible from a refusal. Refusing too little would let a forced act
// by an agent commit — the defect this milestone exists to close.
// Refusing too much is the quieter failure: every rule enforced here
// beyond the force-predicated ones refuses verbs for reasons unrelated
// to sovereignty, and a verb that never passes through the
// provenance-decoration layer cannot satisfy those rules by any
// invocation. So both directions are asserted, and the expectation is
// derived from the rules' own statements rather than from the function.
func TestCheckForceTrailerCoherence_IsTheForcePredicatedSubset(t *testing.T) {
	t.Parallel()
	for _, tc := range coherenceDomain() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actorIsNonHuman := tc.actor != "" && !strings.HasPrefix(tc.actor, "human/")
			err := CheckForceTrailerCoherence(tc.trailers)

			// Derived from the three rule statements, not from the code:
			// force is sovereign, so it cannot be wielded by an agent, on
			// an agent's behalf, or alongside audit-only.
			wantViolation := tc.present["force"] &&
				(actorIsNonHuman || tc.present["onbehalfof"] || tc.present["auditonly"])

			ce, rule := AsCoherenceError(err)
			switch {
			case wantViolation && ce == nil:
				t.Fatalf("accepted a sovereign-force violation; the seam would let this act commit")
			case !wantViolation && ce != nil:
				t.Fatalf("refused %v, which violates no force-predicated rule; "+
					"a verb outside the provenance-decoration layer cannot satisfy this by any invocation", err)
			case ce == nil:
				return
			}

			if !forcePredicatedRules[rule] {
				t.Errorf("reported %q, which is not predicated on a force trailer; "+
					"the seam refuses only sovereign acts", rule)
			}
			// Subset-ness: the seam must never refuse a set the full rule
			// set considers coherent, or the two disagree about legality.
			if full := CheckTrailerCoherence(tc.trailers); full == nil {
				t.Errorf("refused with %q a set the full rule set accepts", rule)
			}
		})
	}
}
