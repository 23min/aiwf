// Package verb — I2.5 trailer-coherence rules.
package verb

import (
	"errors"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/gitops"
)

// CoherenceError is the typed error CheckTrailerCoherence returns when
// a trailer set violates one of the I2.5 required-together or mutually-
// exclusive rules. Rule names a single canonical violation per error
// so the caller (verb refusal path or aiwf check standing rule) can
// map it to a finding code without parsing prose.
//
// The Rule strings are the load-bearing identifiers: do not change one
// without updating the corresponding `aiwf check` standing-rule
// subcode in internal/check/provenance.go (added in step 7).
type CoherenceError struct {
	Rule    string
	Message string
}

func (e *CoherenceError) Error() string { return e.Message }

// Code maps the violated rule to the `aiwf check` finding code that
// reports the same violation class, so one identifier routes a refusal
// at the verb and a finding on a commit that already landed. Being
// Coded is also what lands the refusal at the legality exit rather
// than the internal-failure one — see the exit-code contract on
// cliutil.FinishVerb.
//
// Two rules have a finding code of their own check-side; the rest map
// to the one incoherent-trailer code, and the refusal follows that
// rather than inventing a second vocabulary.
//
// The mapping says what the audit calls a violation it reports, not
// that it reports every one. audit-only-with-force has no
// history-walking counterpart at all, so its code here names the class
// the audit would use rather than one it ever emits (G-0551).
func (e *CoherenceError) Code() string {
	switch e.Rule {
	case CoherenceRuleForceNonHuman:
		return check.CodeProvenanceForceNonHuman
	case CoherenceRuleAuditOnlyNonHuman:
		return check.CodeProvenanceAuditOnlyNonHuman
	default:
		return check.CodeProvenanceTrailerIncoherent
	}
}

// Coherence rule names. AsCoherenceError returns one to callers that
// switch on the rule rather than on message text, so they are part of
// this package's surface.
const (
	CoherenceRuleOnBehalfOfMissingAuthorizedBy    = "on-behalf-of-missing-authorized-by"
	CoherenceRuleAuthorizedByMissingOnBehalfOf    = "authorized-by-missing-on-behalf-of"
	CoherenceRulePrincipalMissingForNonHumanActor = "principal-missing-for-non-human-actor"
	CoherenceRulePrincipalRequiresNonHumanActor   = "principal-requires-non-human-actor"
	CoherenceRuleOnBehalfOfForbiddenForHumanActor = "on-behalf-of-forbidden-for-human-actor"
	CoherenceRuleForceWithOnBehalfOf              = "force-with-on-behalf-of"
	CoherenceRuleForceNonHuman                    = "force-non-human"
	CoherenceRuleAuditOnlyWithForce               = "audit-only-with-force"
	CoherenceRuleAuditOnlyNonHuman                = "audit-only-non-human"
)

// coherenceRuleSpec declares one coherence rule.
//
// This describes the rules; it does not drive them. CheckTrailerCoherence
// evaluates its conditions directly, so an entry here is a claim about a
// rule rather than the rule itself. Each claim is checked against the
// rules' behavior rather than trusted — but only as far as the verdict
// exposes it, which is less than the claims read as. Each field states
// what its check does and does not establish (D-0062).
type coherenceRuleSpec struct {
	// Rule is the name AsCoherenceError reports for this rule.
	Rule string

	// Reads are the presence-bearing trailer keys this rule turns on.
	// The generated domain varies exactly their union, so a rule
	// declaring a trailer no other rule reads widens the domain instead
	// of going unexercised in it. Widening the domain is what the field
	// is for.
	//
	// What the check establishes is narrower than "the condition reads
	// this trailer": it establishes that toggling the trailer changes
	// whether this rule is the *reported* verdict. Only the first
	// violation is reported, so an earlier rule shadowing this one
	// satisfies that without this rule's condition touching the trailer
	// at all. A declared input is load-bearing for the domain; it is not
	// proven to be read.
	//
	// The actor is not listed. It is the domain's other axis, and unlike
	// the trailer axis it is a closed three-valued classification no new
	// rule can widen — one splitting it finer would fire nowhere in the
	// domain and be named by the bijection check.
	Reads []string

	// FiresOnlyWithForce reports that the rule cannot fire without an
	// aiwf-force trailer. It is a statement about the rule.
	//
	// verb.Apply enforces exactly the rules carrying this flag today,
	// but the flag is not the membership criterion — satisfiability is
	// (D-0060, ADR-0040): a rule belongs at the seam only if every verb
	// reaching it has some invocation that satisfies it. Recording
	// membership here would key the seam on a property that merely
	// coincides with it, so a rule admitted on satisfiability grounds
	// without being force-predicated has no way to be recorded in this
	// declaration. That is deliberate: seam membership is a D-0060
	// decision, not a fact about the rule.
	FiresOnlyWithForce bool
}

// coherenceRuleSpecs declares every rule this package checks. It is the
// single source the rule roster, the seam's enforced subset, and the
// generated domain's trailer axis all derive from, so a rule added here
// enters all three at once rather than in three separate edits that can
// each be forgotten.
//
// Order is for reading and carries no meaning: every consumer treats
// these as a set.
var coherenceRuleSpecs = []coherenceRuleSpec{
	{Rule: CoherenceRuleOnBehalfOfMissingAuthorizedBy, Reads: []string{gitops.TrailerOnBehalfOf, gitops.TrailerAuthorizedBy}},
	{Rule: CoherenceRuleAuthorizedByMissingOnBehalfOf, Reads: []string{gitops.TrailerAuthorizedBy, gitops.TrailerOnBehalfOf}},
	{Rule: CoherenceRulePrincipalMissingForNonHumanActor, Reads: []string{gitops.TrailerPrincipal}},
	{Rule: CoherenceRulePrincipalRequiresNonHumanActor, Reads: []string{gitops.TrailerPrincipal}},
	{Rule: CoherenceRuleOnBehalfOfForbiddenForHumanActor, Reads: []string{gitops.TrailerOnBehalfOf}},
	{Rule: CoherenceRuleForceWithOnBehalfOf, Reads: []string{gitops.TrailerForce, gitops.TrailerOnBehalfOf}, FiresOnlyWithForce: true},
	{Rule: CoherenceRuleForceNonHuman, Reads: []string{gitops.TrailerForce}, FiresOnlyWithForce: true},
	{Rule: CoherenceRuleAuditOnlyWithForce, Reads: []string{gitops.TrailerAuditOnly, gitops.TrailerForce}, FiresOnlyWithForce: true},
	{Rule: CoherenceRuleAuditOnlyNonHuman, Reads: []string{gitops.TrailerAuditOnly}},
}

// declaredCoherenceRules returns every rule name the given declaration
// carries.
func declaredCoherenceRules(specs []coherenceRuleSpec) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Rule)
	}
	return out
}

// declaredForcePredicatedRules returns the rules that cannot fire
// without a force trailer. That is the set verb.Apply enforces today;
// see FiresOnlyWithForce for why it is not the membership criterion.
func declaredForcePredicatedRules(specs []coherenceRuleSpec) []string {
	var out []string
	for _, s := range specs {
		if s.FiresOnlyWithForce {
			out = append(out, s.Rule)
		}
	}
	return out
}

// declaredCoherenceTrailerAxis returns every presence-bearing trailer
// any rule reads, deduplicated in first-appearance order. This is the
// axis the generated domain varies.
func declaredCoherenceTrailerAxis(specs []coherenceRuleSpec) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range specs {
		for _, key := range s.Reads {
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}

// CheckTrailerCoherence validates the I2.5 required-together /
// mutually-exclusive trailer rules on an assembled trailer set.
// Returns nil when the set is coherent; returns a *CoherenceError
// naming a single rule violation otherwise.
//
// The check intentionally returns the FIRST violation encountered —
// surfacing all of them at once would force callers to display a
// list when typically one fix unblocks the rest. Standing-rule
// callers (aiwf check) re-run per commit so each commit's first
// violation surfaces.
//
// Per provenance-model.md §"Required-together and mutually-exclusive
// rules":
//
//   - on-behalf-of ↔ authorized-by: both present or both absent.
//   - principal ↔ non-human actor: required-together; principal is
//     forbidden for a human actor.
//   - on-behalf-of: forbidden for a human actor (direct human acts
//     have no on-behalf-of).
//   - force + on-behalf-of: mutually exclusive (force is human-only;
//     on-behalf-of implies an agent operator).
//   - force + non-human actor: forbidden (force is sovereign, human-
//     only).
//   - audit-only + force: mutually exclusive (force makes a transition;
//     audit-only records one that already happened — distinct intents).
//   - audit-only + non-human actor: forbidden (audit-only is sovereign,
//     same rationale as force).
//
// The (authorize, on-behalf-of) sub-agent-delegation pair is
// deliberately NOT enforced — that policy decision is reserved for
// G22 per the design doc.
func CheckTrailerCoherence(trailers []gitops.Trailer) error {
	idx := indexTrailers(trailers)

	actor := idx[gitops.TrailerActor]
	actorIsHuman := strings.HasPrefix(actor, "human/")
	actorIsNonHuman := isNonHumanActor(actor)

	_, hasPrincipal := idx[gitops.TrailerPrincipal]
	_, hasOnBehalfOf := idx[gitops.TrailerOnBehalfOf]
	_, hasAuthorizedBy := idx[gitops.TrailerAuthorizedBy]
	_, hasAuditOnly := idx[gitops.TrailerAuditOnly]

	// Required-together: on-behalf-of ↔ authorized-by.
	switch {
	case hasOnBehalfOf && !hasAuthorizedBy:
		return &CoherenceError{
			Rule:    CoherenceRuleOnBehalfOfMissingAuthorizedBy,
			Message: "aiwf-on-behalf-of requires aiwf-authorized-by (both signal scope membership)",
		}
	case hasAuthorizedBy && !hasOnBehalfOf:
		return &CoherenceError{
			Rule:    CoherenceRuleAuthorizedByMissingOnBehalfOf,
			Message: "aiwf-authorized-by requires aiwf-on-behalf-of (both signal scope membership)",
		}
	}

	// Required-together: principal ↔ non-human actor.
	if actorIsNonHuman && !hasPrincipal {
		return &CoherenceError{
			Rule:    CoherenceRulePrincipalMissingForNonHumanActor,
			Message: fmt.Sprintf("aiwf-actor %q is non-human; aiwf-principal is required", actor),
		}
	}

	// The other half of the required-together pair: a principal names who
	// a non-human actor acts for, so a principal alongside anything else —
	// a human actor, or no actor at all — describes an accountability
	// relationship with no agent in it.
	if hasPrincipal && !actorIsNonHuman {
		return &CoherenceError{
			Rule:    CoherenceRulePrincipalRequiresNonHumanActor,
			Message: fmt.Sprintf("aiwf-principal requires a non-human aiwf-actor (got actor=%q)", actor),
		}
	}

	// Mutually exclusive: on-behalf-of + human actor.
	if hasOnBehalfOf && actorIsHuman {
		return &CoherenceError{
			Rule:    CoherenceRuleOnBehalfOfForbiddenForHumanActor,
			Message: fmt.Sprintf("aiwf-on-behalf-of is forbidden when aiwf-actor is human/ (got actor=%q)", actor),
		}
	}

	// The rules predicated on a force trailer, in their own function
	// because verb.Apply enforces exactly this subset.
	if err := CheckForceTrailerCoherence(trailers); err != nil {
		return err
	}

	// Audit-only human-only.
	if hasAuditOnly && actorIsNonHuman {
		return &CoherenceError{
			Rule:    CoherenceRuleAuditOnlyNonHuman,
			Message: fmt.Sprintf("aiwf-audit-only requires a human/ actor (got actor=%q); audit-only is sovereign, like --force", actor),
		}
	}

	return nil
}

// CheckForceTrailerCoherence validates the subset of the coherence
// rules predicated on an aiwf-force trailer: a set carrying no force
// trailer is always coherent by this function's lights. Returns nil
// when the set passes; returns a *CoherenceError naming a single rule
// violation otherwise.
//
// This is what verb.Apply enforces, and the criterion for membership is
// satisfiability: a rule belongs here only if every verb that can reach
// the seam has some invocation that satisfies it. The force rules pass
// that test because a verb emitting no force trailer satisfies them
// vacuously. The principal rules fail it — the contract verbs never
// pass through the provenance-decoration layer, carry no
// aiwf-principal, and register no flag that could supply one, so
// enforcing those here is a closed door rather than a rule.
//
// Do not read the subset as "the sovereign rules". Audit-only is
// sovereign too and is deliberately not here, because it fails the same
// satisfiability test for the same reason. What retires that exclusion
// is provenance wiring on the verbs that lack it, not a change of
// principle.
//
// Adding a rule here therefore changes what the CLI refuses live, at
// the moment a verb is attempted. A rule no invocation could satisfy
// goes in CheckTrailerCoherence instead, where the history-walking
// audit reports it after the fact.
//
// force-non-human is checked before force-with-on-behalf-of because a
// non-human actor can only reach this seam through an active scope,
// whose aiwf-on-behalf-of would otherwise trip the later rule first
// and report two trailer keys the operator never typed. The rule order
// is the operator's error message, so it is chosen rather than
// inherited.
func CheckForceTrailerCoherence(trailers []gitops.Trailer) error {
	idx := indexTrailers(trailers)

	actor := idx[gitops.TrailerActor]
	actorIsNonHuman := isNonHumanActor(actor)

	_, hasOnBehalfOf := idx[gitops.TrailerOnBehalfOf]
	_, hasForce := idx[gitops.TrailerForce]
	_, hasAuditOnly := idx[gitops.TrailerAuditOnly]

	// Force human-only.
	if hasForce && actorIsNonHuman {
		return &CoherenceError{
			Rule:    CoherenceRuleForceNonHuman,
			Message: fmt.Sprintf("aiwf-force requires a human/ actor (got actor=%q); only humans wield --force", actor),
		}
	}

	// Mutually exclusive: force + on-behalf-of.
	if hasForce && hasOnBehalfOf {
		return &CoherenceError{
			Rule:    CoherenceRuleForceWithOnBehalfOf,
			Message: "aiwf-force and aiwf-on-behalf-of cannot coexist (force is human-only; on-behalf-of implies an agent)",
		}
	}

	// Mutually exclusive: audit-only + force.
	if hasAuditOnly && hasForce {
		return &CoherenceError{
			Rule:    CoherenceRuleAuditOnlyWithForce,
			Message: "aiwf-audit-only and aiwf-force cannot coexist (force makes a transition; audit-only records one that already happened)",
		}
	}

	return nil
}

// isNonHumanActor reports whether actor names an agent: present, and
// not a human/ role. An absent actor is neither human nor non-human,
// which is a distinction several rules turn on.
func isNonHumanActor(actor string) bool {
	return actor != "" && !strings.HasPrefix(actor, "human/")
}

// indexTrailers builds a key→value map. When a key appears more than
// once (e.g., aiwf-scope-ends), the last occurrence wins for the
// purpose of coherence checks — the rules in this file only care
// about presence/absence and value-shape, never about which specific
// value across repeats.
func indexTrailers(trailers []gitops.Trailer) map[string]string {
	out := make(map[string]string, len(trailers))
	for _, tr := range trailers {
		out[tr.Key] = tr.Value
	}
	return out
}

// AsCoherenceError returns the *CoherenceError if err is one (or
// wraps one), and the rule name; otherwise returns nil and "". Helper
// for callers (notably the standing rule in step 7) that switch on
// rule names rather than message text.
func AsCoherenceError(err error) (ce *CoherenceError, rule string) {
	if errors.As(err, &ce) {
		return ce, ce.Rule
	}
	return nil, ""
}
