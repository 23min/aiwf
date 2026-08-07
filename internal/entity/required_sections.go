package entity

import "slices"

// requiredSectionsByKind is the single definition of each kind's
// load-bearing top-level body sections, in canonical render order.
//
// It lives here, in the lowest package that any reader can reach, because
// both readers sit above it: BodyTemplate renders the scaffold `aiwf add`
// commits, and the entity-body check rule validates what a tree carries.
// Neither holds a copy — a literal beside this one is what lets a scaffold
// and the rule validating it disagree about the same kind.
var requiredSectionsByKind = map[Kind][]string{
	KindEpic:      {"Goal", "Scope", "Out of scope"},
	KindMilestone: {"Goal", "Acceptance criteria"},
	KindADR:       {"Context", "Decision", "Consequences"},
	KindGap:       {"What's missing", "Why it matters"},
	KindDecision:  {"Question", "Decision", "Reasoning"},
	KindContract:  {"Purpose", "Stability"},
}

// RequiredSections returns k's load-bearing top-level body sections in
// canonical render order, or nil for a kind carrying no set.
//
// The result is a copy. The table is package-level state read by the add
// scaffold and by every check run, so handing out the backing array would
// let one caller rewrite what all the others see.
func RequiredSections(k Kind) []string {
	return slices.Clone(requiredSectionsByKind[k])
}
