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
// that. --force is the explicit override, and the existing provenance
// coherence rule (`aiwf-force requires a human/ actor`) ensures
// non-human + --force still fails at the coherence chokepoint, so the
// override path is human-only by construction.
func requireHumanActorForSovereignAct(kind entity.Kind, from, to, actor string) error {
	if !entity.IsSovereignActShape(kind, from, to) {
		return nil
	}
	if strings.HasPrefix(actor, "human/") {
		return nil
	}
	return fmt.Errorf("aiwf promote %s %s: sovereign act requires a human/ actor (got %q); have a human run the verb, or use `--force --reason \"...\"` to override", kind, to, actor)
}
