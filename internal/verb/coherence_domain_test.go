package verb

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

// coherenceFlags is the trailer-presence axis: the five trailers whose
// presence or absence any coherence rule reads. Values are irrelevant to
// every rule — each reads presence only — so one representative value
// per trailer covers the axis.
var coherenceFlags = []struct{ name, key, value string }{
	{"principal", gitops.TrailerPrincipal, "human/peter"},
	{"onbehalfof", gitops.TrailerOnBehalfOf, "human/peter"},
	{"authorizedby", gitops.TrailerAuthorizedBy, "abc1234"},
	{"force", gitops.TrailerForce, "override"},
	{"auditonly", gitops.TrailerAuditOnly, "true"},
}

// coherenceCase is one point in the domain.
type coherenceCase struct {
	name     string
	trailers []gitops.Trailer
	// present reports whether a given flag name is set in this case.
	present map[string]bool
	actor   string
}

// coherenceDomain enumerates the complete input domain: every actor role
// crossed with every subset of the presence-bearing trailers. Generated
// rather than enumerated, so coverage is a property of this function and
// a rule added against a new trailer is a one-line change here rather
// than a fresh set of hand-written cases someone must remember to add.
func coherenceDomain() []coherenceCase {
	var out []coherenceCase
	for _, actor := range coherenceActors {
		for mask := 0; mask < 1<<len(coherenceFlags); mask++ {
			trailers := []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
			}
			if actor.value != "" {
				trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerActor, Value: actor.value})
			}
			present := make(map[string]bool, len(coherenceFlags))
			parts := []string{"actor=" + actor.name}
			for i, f := range coherenceFlags {
				on := mask&(1<<i) != 0
				present[f.name] = on
				if on {
					trailers = append(trailers, gitops.Trailer{Key: f.key, Value: f.value})
					parts = append(parts, f.name)
				}
			}
			out = append(out, coherenceCase{
				name:     strings.Join(parts, "+"),
				trailers: trailers,
				present:  present,
				actor:    actor.value,
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

	var b strings.Builder
	for _, tc := range coherenceDomain() {
		fmt.Fprintf(&b, "%s => %s\n", tc.name, verdict(tc.trailers))
	}
	got := b.String()

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

// coherenceRules is every rule the package declares.
//
// Hand-maintained, and deliberately so for now: the rule space has no
// registry to enumerate, which is the gap M-0294 exists to close. Until
// it does, the golden is the backstop — a rule added or removed changes
// the golden's diff whether or not someone updates this list.
var coherenceRules = []string{
	CoherenceRuleOnBehalfOfMissingAuthorizedBy,
	CoherenceRuleAuthorizedByMissingOnBehalfOf,
	CoherenceRulePrincipalMissingForNonHumanActor,
	CoherenceRulePrincipalRequiresNonHumanActor,
	CoherenceRuleOnBehalfOfForbiddenForHumanActor,
	CoherenceRuleForceWithOnBehalfOf,
	CoherenceRuleForceNonHuman,
	CoherenceRuleAuditOnlyWithForce,
	CoherenceRuleAuditOnlyNonHuman,
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

	fired := make(map[string]bool, len(coherenceRules))
	for _, c := range coherenceDomain() {
		fired[verdict(c.trailers)] = true
	}
	for _, rule := range coherenceRules {
		if !fired[rule] {
			t.Errorf("rule %q fires at no point in the domain; it is shadowed by an earlier rule or unreachable", rule)
		}
	}
}

// forcePredicatedRules are the rules whose condition requires an
// aiwf-force trailer. Written out rather than derived from the
// function under test, so a rule silently added to or dropped from
// the seam's subset shows up as a failure here.
//
// Hand-maintained, like coherenceRules above, and retired by the same
// thing: M-0294's rule registry, which makes both derivable. Until
// then a fourth force-predicated rule must be added here by hand or
// this test stops covering it.
var forcePredicatedRules = map[string]bool{
	CoherenceRuleForceNonHuman:       true,
	CoherenceRuleForceWithOnBehalfOf: true,
	CoherenceRuleAuditOnlyWithForce:  true,
}

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
