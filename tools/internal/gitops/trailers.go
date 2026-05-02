package gitops

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Trailer key constants. Verbs and tests should reference these
// rather than literal strings so a future rename or audit lands in
// one place. Pre-I2.5 keys (Verb…Tests) preserve their existing
// semantics; I2.5 keys (Principal…Reason) are added by the
// provenance model — see docs/pocv3/design/provenance-model.md.
const (
	TrailerVerb        = "aiwf-verb"
	TrailerEntity      = "aiwf-entity"
	TrailerActor       = "aiwf-actor"
	TrailerTo          = "aiwf-to"
	TrailerForce       = "aiwf-force"
	TrailerPriorEntity = "aiwf-prior-entity"
	TrailerPriorParent = "aiwf-prior-parent"
	TrailerTests       = "aiwf-tests"

	// I2.5 provenance trailers.
	TrailerPrincipal    = "aiwf-principal"
	TrailerOnBehalfOf   = "aiwf-on-behalf-of"
	TrailerAuthorizedBy = "aiwf-authorized-by"
	TrailerScope        = "aiwf-scope"
	TrailerScopeEnds    = "aiwf-scope-ends"
	TrailerReason       = "aiwf-reason"

	// I2.5 audit-only recovery (G24, plan step 5b).
	TrailerAuditOnly = "aiwf-audit-only"
)

// trailerOrder is the canonical write order for known trailers. The
// existing pre-I2.5 trailers come first (matching what verbs already
// emit by hand), followed by the I2.5 keys in the order documented in
// provenance-model.md §"Trailer set". Unknown keys sort to the end
// alphabetically so future-trailer round-trips are stable.
var trailerOrder = []string{
	TrailerVerb,
	TrailerEntity,
	TrailerActor,
	TrailerTo,
	TrailerForce,
	TrailerPriorEntity,
	TrailerPriorParent,
	TrailerTests,
	TrailerPrincipal,
	TrailerOnBehalfOf,
	TrailerAuthorizedBy,
	TrailerScope,
	TrailerScopeEnds,
	TrailerReason,
	TrailerAuditOnly,
}

// trailerOrderIndex maps each known key to its position in the
// canonical write order. Built once at package init for O(1) lookup.
var trailerOrderIndex = func() map[string]int {
	m := make(map[string]int, len(trailerOrder))
	for i, k := range trailerOrder {
		m[k] = i
	}
	return m
}()

// SortedTrailers returns a copy of trailers in canonical write order.
// Known keys come first in trailerOrder sequence; unknown keys come
// last in lexicographic order. Repeated keys (e.g. multiple
// aiwf-scope-ends entries on one commit) preserve their input order
// among themselves so callers can rely on stable per-key emission.
func SortedTrailers(trailers []Trailer) []Trailer {
	out := make([]Trailer, len(trailers))
	copy(out, trailers)
	sort.SliceStable(out, func(i, j int) bool {
		ki, oi := out[i].Key, len(trailerOrder)
		kj, oj := out[j].Key, len(trailerOrder)
		if idx, ok := trailerOrderIndex[ki]; ok {
			oi = idx
		}
		if idx, ok := trailerOrderIndex[kj]; ok {
			oj = idx
		}
		if oi != oj {
			return oi < oj
		}
		// Same canonical position: known-vs-known of the same key
		// (preserves input order via stable sort) or both unknown
		// (lex by key).
		if oi == len(trailerOrder) {
			return ki < kj
		}
		return false
	})
	return out
}

// roleIDPattern matches the `<role>/<id>` shape used by aiwf-actor,
// aiwf-principal, and aiwf-on-behalf-of: exactly one '/', no
// whitespace, neither side empty, slash-free either side.
var roleIDPattern = regexp.MustCompile(`^[^\s/]+/[^\s/]+$`)

// shaPattern matches a 7–40 hex string — the shape `git rev-parse`
// produces and what aiwf-authorized-by / aiwf-scope-ends reference.
var shaPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// scopeEvents is the closed set of values aiwf-scope may carry on
// authorize commits. opened/paused/resumed map to the scope FSM
// transitions; there is no "ended" — scope termination is recorded
// separately by aiwf-scope-ends on the terminal-promote commit.
var scopeEvents = map[string]struct{}{
	"opened":  {},
	"paused":  {},
	"resumed": {},
}

// ValidateTrailer enforces I2.5 write-time shape rules per known key.
// Returns nil for unknown keys (forward compatibility — future
// trailers don't break old binaries) and for keys whose semantic
// shape is "any non-empty string" (verb, entity, to, prior-entity,
// tests are all loose strings).
//
// Identity-bearing trailers must match `<role>/<id>`; principal and
// on-behalf-of additionally require a `human/` role (per the
// "principal is always human" kernel rule). SHA-shaped trailers
// validate as 7–40 hex. Scope is a closed-set enum. Reason and force
// require a non-empty value after trim. Aiwf-audit-only follows the
// reason shape (non-empty, free text).
//
// SHA-points-to-a-real-authorize-commit is verified at READ time, not
// here — write-time checks against historical SHAs would race with
// rebases and force-pushes. See provenance-model.md §"Trailer set".
func ValidateTrailer(key, value string) error {
	switch key {
	case TrailerActor:
		if !roleIDPattern.MatchString(value) {
			return fmt.Errorf("%s: %q must match <role>/<id>", key, value)
		}
	case TrailerPrincipal, TrailerOnBehalfOf:
		if !roleIDPattern.MatchString(value) {
			return fmt.Errorf("%s: %q must match <role>/<id>", key, value)
		}
		if !strings.HasPrefix(value, "human/") {
			return fmt.Errorf("%s: role must be human/ (got %q)", key, value)
		}
	case TrailerAuthorizedBy, TrailerScopeEnds:
		if !shaPattern.MatchString(value) {
			return fmt.Errorf("%s: %q must be 7–40 hex", key, value)
		}
	case TrailerScope:
		if _, ok := scopeEvents[value]; !ok {
			return fmt.Errorf("%s: %q must be one of opened|paused|resumed", key, value)
		}
	case TrailerReason, TrailerForce, TrailerAuditOnly:
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s: value must be non-empty after trim", key)
		}
	}
	return nil
}
