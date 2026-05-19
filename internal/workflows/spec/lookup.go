package spec

import (
	"github.com/23min/aiwf/internal/entity"
)

// LookupRules returns every Rule in Rules() that matches the (kind,
// fromState, verb) key. The plural name and slice return shape are
// load-bearing — per M-0123 phase 1's schema adjustment, the (Kind,
// FromState, Verb, Outcome) tuple keys the table, not (Kind, FromState,
// Verb). A preconditioned-pair key (e.g., epic.proposed.cancel, with a
// legal cell and a Q5/D-0003 illegal companion) returns both cells.
//
// Semantics:
//   - Hit (>=1 cell): returns a slice of every matching cell. The caller
//     resolves which cell applies by walking Preconditions.
//   - Miss (no cell): returns an empty slice (nil-equivalent on len).
//
// Per the AC-2 schema invariant (TestM0123_AC2_KeyUnique), the slice
// contains at most one cell per Outcome value — within a single key,
// legal-vs-illegal is the only distinction.
//
// LookupRules is the only public access surface for the table (AC-7); the
// Rules() slice is exported for the AC-2 / AC-5 drift policies that need
// to iterate the full table, but consumers should reach for LookupRules
// when answering "is this verb legal here?".
func LookupRules(kind entity.Kind, fromState, verb string) []Rule {
	var out []Rule
	for _, r := range Rules() {
		if r.Kind == kind && r.FromState == fromState && r.Verb == verb {
			out = append(out, r)
		}
	}
	return out
}
