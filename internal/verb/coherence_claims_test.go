package verb

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// ruleSpecClaimViolations checks every claim a declaration makes against
// the verdicts its rules actually produce across the domain that
// declaration generates. Returns one message per contradicted claim,
// sorted so the report does not depend on map iteration order.
//
// A rule that fires nowhere in the domain is skipped rather than
// reported: no claim about it can be checked against behavior it never
// exhibits, and reporting it here would duplicate — and bury — the
// reachability failure that is M-0294/AC-3's subject.
func ruleSpecClaimViolations(specs []coherenceRuleSpec) []string {
	flags := flagsFor(specs)

	// Keyed by actor role then trailer mask, so a case can be compared
	// with the one differing from it in exactly one trailer.
	verdicts := map[string]map[int]string{}
	for _, c := range coherenceDomainFrom(flags) {
		if verdicts[c.actorName] == nil {
			verdicts[c.actorName] = map[int]string{}
		}
		verdicts[c.actorName][c.mask] = verdict(c.trailers)
	}

	// Every trailer a rule declares is in the axis by construction — the
	// axis is the union of what the rules declare — so a Reads entry
	// always resolves to a bit here.
	bitOf := map[string]int{}
	for i, f := range flags {
		bitOf[f.key] = i
	}
	forceBit, forceOnAxis := bitOf[gitops.TrailerForce]

	var out []string
	for _, s := range specs {
		var fires, firesWithoutForce bool
		for _, byMask := range verdicts {
			for mask, v := range byMask {
				if v != s.Rule {
					continue
				}
				fires = true
				if !forceOnAxis || mask&(1<<forceBit) == 0 {
					firesWithoutForce = true
				}
			}
		}
		if !fires {
			continue
		}

		switch {
		case s.FiresOnlyWithForce && firesWithoutForce:
			out = append(out, fmt.Sprintf(
				"%s: declares FiresOnlyWithForce, but fires at a point carrying no %s trailer",
				s.Rule, gitops.TrailerForce))
		case !s.FiresOnlyWithForce && !firesWithoutForce:
			out = append(out, fmt.Sprintf(
				"%s: fires only where %s is present but does not declare FiresOnlyWithForce, so the seam would not enforce it",
				s.Rule, gitops.TrailerForce))
		}

		for _, key := range s.Reads {
			if !togglingChangesFiring(verdicts, s.Rule, bitOf[key]) {
				out = append(out, fmt.Sprintf(
					"%s: declares it reads %s, but toggling %s changes whether it fires at no point in the domain",
					s.Rule, key, key))
			}
		}
	}
	sort.Strings(out)
	return out
}

// togglingChangesFiring reports whether the domain holds any pair of
// points differing only in the trailer at bit where one has rule as its
// verdict and the other does not. That is what makes a declared input
// demonstrably an input rather than an assertion nobody checked.
func togglingChangesFiring(verdicts map[string]map[int]string, rule string, bit int) bool {
	for _, byMask := range verdicts {
		for mask, v := range byMask {
			// Flipping one bit of a mask lands on another mask the
			// domain enumerates, so this lookup always resolves.
			other := byMask[mask^(1<<bit)]
			if (v == rule) != (other == rule) {
				return true
			}
		}
	}
	return false
}

// TestCoherenceRuleSpecs_ClaimsHoldAgainstBehavior is M-0294/AC-2.
//
// The declaration is data a human writes beside each rule, so a test
// that reads its claims and believes them proves nothing about the
// rules. Every claim is checked against the verdicts the rules actually
// produce across the generated domain.
func TestCoherenceRuleSpecs_ClaimsHoldAgainstBehavior(t *testing.T) {
	t.Parallel()
	if got := ruleSpecClaimViolations(coherenceRuleSpecs); len(got) != 0 {
		t.Errorf("the declaration makes %d claim(s) the rules' behavior contradicts:\n  %s",
			len(got), strings.Join(got, "\n  "))
	}
}

// TestRuleSpecClaimViolations_CatchesMisdeclaredClaims is AC-2's
// falsifiability half. Each fixture mis-declares one entry in a way a
// reader could plausibly get wrong, and asserts the check reports it by
// rule name — a check that cannot fail would certify the declaration
// without reading it.
func TestRuleSpecClaimViolations_CatchesMisdeclaredClaims(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		// mutate returns a copy of the real declaration with one claim
		// altered.
		mutate   func([]coherenceRuleSpec) []coherenceRuleSpec
		wantRule string
	}{
		{
			// Claims a rule needs a force trailer when it fires without
			// one. Believing this would put the rule in the subset
			// verb.Apply enforces at the seam, refusing verbs that
			// cannot satisfy it.
			name: "force claimed for a rule that fires without force",
			mutate: func(in []coherenceRuleSpec) []coherenceRuleSpec {
				return withSpec(in, CoherenceRulePrincipalMissingForNonHumanActor, func(s coherenceRuleSpec) coherenceRuleSpec {
					s.FiresOnlyWithForce = true
					return s
				})
			},
			wantRule: CoherenceRulePrincipalMissingForNonHumanActor,
		},
		{
			// Drops the force claim from a rule that only ever fires
			// with one. Believing this would drop the rule out of the
			// seam's subset, letting a sovereign act commit.
			name: "force claim dropped from a force-only rule",
			mutate: func(in []coherenceRuleSpec) []coherenceRuleSpec {
				return withSpec(in, CoherenceRuleForceNonHuman, func(s coherenceRuleSpec) coherenceRuleSpec {
					s.FiresOnlyWithForce = false
					return s
				})
			},
			wantRule: CoherenceRuleForceNonHuman,
		},
		{
			// Declares an input the rule's condition never consults.
			// Harmless to the axis, but it is a false statement about
			// the rule, and the next reader takes it for true.
			name: "an input the rule does not consult",
			mutate: func(in []coherenceRuleSpec) []coherenceRuleSpec {
				return withSpec(in, CoherenceRulePrincipalRequiresNonHumanActor, func(s coherenceRuleSpec) coherenceRuleSpec {
					s.Reads = append(append([]string(nil), s.Reads...), gitops.TrailerAuditOnly)
					return s
				})
			},
			wantRule: CoherenceRulePrincipalRequiresNonHumanActor,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ruleSpecClaimViolations(tc.mutate(coherenceRuleSpecs))
			if len(got) == 0 {
				t.Fatalf("mis-declared entry accepted; the check certifies the declaration without reading it")
			}
			joined := strings.Join(got, "\n  ")
			if !strings.Contains(joined, tc.wantRule) {
				t.Errorf("violation does not name %q, so the reader must search for which entry is wrong:\n  %s",
					tc.wantRule, joined)
			}
		})
	}
}

// TestRuleSpecClaimViolations_ForceOffAxis covers a declaration in
// which no rule reads a force trailer, so the domain never varies one.
//
// The force claim is then unfalsifiable rather than false: with no
// force-bearing point, every rule that fires does so without force, and
// asking whether a rule fires "only with force" has no answer. The check
// must report nothing rather than accuse every rule of the same thing —
// and the force rules, which fire nowhere in such a domain, are AC-3's
// to report.
func TestRuleSpecClaimViolations_ForceOffAxis(t *testing.T) {
	t.Parallel()

	var specs []coherenceRuleSpec
	for _, s := range coherenceRuleSpecs {
		var reads []string
		for _, key := range s.Reads {
			if key != gitops.TrailerForce {
				reads = append(reads, key)
			}
		}
		if len(reads) == 0 {
			// A rule reading force alone has nothing left to declare.
			continue
		}
		s.Reads = reads
		s.FiresOnlyWithForce = false
		specs = append(specs, s)
	}

	for _, f := range flagsFor(specs) {
		if f.key == gitops.TrailerForce {
			t.Fatalf("fixture still varies %s; it cannot exercise the off-axis path", gitops.TrailerForce)
		}
	}
	if got := ruleSpecClaimViolations(specs); len(got) != 0 {
		t.Errorf("reported %d violation(s) against a declaration that reads no force trailer:\n  %s",
			len(got), strings.Join(got, "\n  "))
	}
}

// TestRuleSpecClaimViolations_ShadowingSatisfiesTheRelevanceCheck
// records the limit of the relevance check, so a reader meets it here
// rather than discovering it against a declaration they trusted.
//
// This pins current behavior; it does not endorse it. A check that
// established "the condition reads this trailer" would fail this case,
// and strengthening it — by parsing each rule's condition rather than
// watching the verdict — should break this test and be updated with it.
//
// audit-only-non-human's condition is `hasAuditOnly && actorIsNonHuman`
// and never consults a force trailer. Declaring that it reads one is
// nonetheless accepted, because force-non-human is checked first: at a
// non-human point carrying audit-only, adding force changes the
// reported verdict from audit-only-non-human to force-non-human. The
// rule's own firing is unchanged; only which rule reports it moves.
func TestRuleSpecClaimViolations_ShadowingSatisfiesTheRelevanceCheck(t *testing.T) {
	t.Parallel()

	specs := withSpec(coherenceRuleSpecs, CoherenceRuleAuditOnlyNonHuman, func(s coherenceRuleSpec) coherenceRuleSpec {
		s.Reads = append(append([]string(nil), s.Reads...), gitops.TrailerForce)
		return s
	})
	// Scoped to violations naming this rule. Asserting over the whole
	// declaration would report any unrelated mis-declaration under this
	// test's message, which names a cause it did not measure.
	for _, v := range ruleSpecClaimViolations(specs) {
		if strings.Contains(v, CoherenceRuleAuditOnlyNonHuman) {
			t.Errorf("the relevance check now rejects a shadow-satisfied input, which is stronger than"+
				" it was; update this characterization and the Reads doc on coherenceRuleSpec:\n  %s", v)
		}
	}
}

// withSpec returns a copy of in with the named rule's entry replaced by
// f applied to it. Copies rather than mutating, so a table subtest
// cannot disturb the package-level declaration other tests read.
func withSpec(in []coherenceRuleSpec, rule string, f func(coherenceRuleSpec) coherenceRuleSpec) []coherenceRuleSpec {
	out := make([]coherenceRuleSpec, 0, len(in))
	for _, s := range in {
		if s.Rule == rule {
			s = f(s)
		}
		out = append(out, s)
	}
	return out
}
