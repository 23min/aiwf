package spec

import (
	"github.com/23min/aiwf/internal/entity"
)

// LookupRules returns every Rule in Rules() that matches the (kind,
// fromState, verb) query. Results span targets, outcomes and preconditions;
// this query is deliberately broader than a rule's full identity.
//
// Semantics:
//   - Hit (>=1 cell): returns a slice of every matching cell. The caller
//     resolves which cell applies by target and Preconditions.
//   - Miss (no cell): returns an empty slice (nil-equivalent on len).
//
// Results obey the same uniqueness invariant as Rules():
// (Kind, FromState, Verb, ToState, Outcome, Preconditions).
//
// LookupRules is the only public access surface for the table (AC-7); the
// Rules() slice is exported for the AC-2 / AC-5 drift policies that need
// to iterate the full table, but consumers should reach for LookupRules
// when answering "is this verb legal here?".
func LookupRules(kind entity.Kind, fromState, verb string) []Rule {
	var out []Rule
	rules := Rules()
	for i := range rules {
		r := &rules[i]
		if r.Kind == kind && r.FromState == fromState && r.Verb == verb {
			out = append(out, *r)
		}
	}
	return out
}
