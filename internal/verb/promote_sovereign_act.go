package verb

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
)

// requireHumanActorForSovereignAct enforces the kernel rule that
// sovereign-act-shape transitions are human-only by default. The
// closed-set list of sovereign-act-shape transitions lives in the
// entity package (`entity.IsSovereignActShape` /
// `sovereignActShapes` in `internal/entity/sovereign.go`); this gate
// consults it directly rather than carrying a parallel hardcoded
// copy.
//
// M-0095 was the first such rule (epic proposed → active, motivated
// by G-0063). The predicate was hardcoded inline at this site until
// M-0130's audit consolidated it into the kernel property at the
// entity layer. Future ADRs that ratify new sovereign-act-shape
// transitions update the list in `internal/entity/sovereign.go`;
// this gate fires on them automatically with no verb-layer change.
//
// Caller has already verified !force; this helper does not re-check
// that. --force is the explicit override, and it reaches a guard of its
// own: verb.Apply runs CheckForceTrailerCoherence over the assembled
// trailer set ahead of any filesystem work, and refuses a force trailer
// from a non-human actor. So the two routes into a sovereign act are
// each closed — this helper covers the unforced one, the apply seam
// covers the forced one (ADR-0040).
// verbName is the verb the operator actually invoked, and it is a
// parameter rather than a constant because more than one verb reaches
// a sovereign edge: ADR-0047 puts both epic cancel edges in the closed
// set, and `aiwf cancel` is how a human spells them. A message naming
// promote there would send the reader to a different command than the
// one that refused them.
func requireHumanActorForSovereignAct(verbName string, kind entity.Kind, from, to entity.Status, actor string) error {
	if !entity.IsSovereignActShape(kind, from, to) {
		return nil
	}
	if strings.HasPrefix(actor, "human/") {
		return nil
	}
	// The remedy names only the human-run path. Offering --force here
	// would send this actor at a gate that refuses it: the coherence
	// guard at verb.Apply rejects a force trailer from a non-human
	// actor, and this message is reachable only for a non-human actor,
	// so that advice would be wrong every time it was shown.
	//
	// Both states are named, not just the destination: with several
	// edges into a terminal epic status now sovereign, the destination
	// alone does not identify which edge was refused.
	return fmt.Errorf("aiwf %s %s %s → %s: sovereign act requires a human/ actor (got %q); have a human run the verb",
		verbName, kind, from, to, actor)
}
