// Package verb — I2.5 trailer-coherence rules.
package verb

import (
	"errors"
	"fmt"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/gitops"
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

// Coherence rule names. These are referenced by `aiwf check`'s
// provenance findings (step 7) — keep them stable.
const (
	CoherenceRuleOnBehalfOfMissingAuthorizedBy    = "on-behalf-of-missing-authorized-by"
	CoherenceRuleAuthorizedByMissingOnBehalfOf    = "authorized-by-missing-on-behalf-of"
	CoherenceRulePrincipalMissingForNonHumanActor = "principal-missing-for-non-human-actor"
	CoherenceRulePrincipalForbiddenForHumanActor  = "principal-forbidden-for-human-actor"
	CoherenceRuleOnBehalfOfForbiddenForHumanActor = "on-behalf-of-forbidden-for-human-actor"
	CoherenceRuleForceWithOnBehalfOf              = "force-with-on-behalf-of"
	CoherenceRuleForceNonHuman                    = "force-non-human"
	CoherenceRuleAuditOnlyWithForce               = "audit-only-with-force"
	CoherenceRuleAuditOnlyNonHuman                = "audit-only-non-human"
)

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
	actorIsNonHuman := actor != "" && !actorIsHuman

	_, hasPrincipal := idx[gitops.TrailerPrincipal]
	_, hasOnBehalfOf := idx[gitops.TrailerOnBehalfOf]
	_, hasAuthorizedBy := idx[gitops.TrailerAuthorizedBy]
	_, hasForce := idx[gitops.TrailerForce]
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

	// Mutually exclusive: principal + human actor.
	if hasPrincipal && actorIsHuman {
		return &CoherenceError{
			Rule:    CoherenceRulePrincipalForbiddenForHumanActor,
			Message: fmt.Sprintf("aiwf-principal is forbidden when aiwf-actor is human/ (got actor=%q)", actor),
		}
	}

	// Mutually exclusive: on-behalf-of + human actor.
	if hasOnBehalfOf && actorIsHuman {
		return &CoherenceError{
			Rule:    CoherenceRuleOnBehalfOfForbiddenForHumanActor,
			Message: fmt.Sprintf("aiwf-on-behalf-of is forbidden when aiwf-actor is human/ (got actor=%q)", actor),
		}
	}

	// Mutually exclusive: force + on-behalf-of.
	if hasForce && hasOnBehalfOf {
		return &CoherenceError{
			Rule:    CoherenceRuleForceWithOnBehalfOf,
			Message: "aiwf-force and aiwf-on-behalf-of cannot coexist (force is human-only; on-behalf-of implies an agent)",
		}
	}

	// Force human-only.
	if hasForce && actorIsNonHuman {
		return &CoherenceError{
			Rule:    CoherenceRuleForceNonHuman,
			Message: fmt.Sprintf("aiwf-force requires a human/ actor (got actor=%q); only humans wield --force", actor),
		}
	}

	// Mutually exclusive: audit-only + force.
	if hasAuditOnly && hasForce {
		return &CoherenceError{
			Rule:    CoherenceRuleAuditOnlyWithForce,
			Message: "aiwf-audit-only and aiwf-force cannot coexist (force makes a transition; audit-only records one that already happened)",
		}
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
