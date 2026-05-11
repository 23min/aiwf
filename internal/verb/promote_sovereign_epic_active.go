package verb

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
)

// requireHumanActorForEpicActivation enforces M-0095: the `epic /
// proposed → active` edge is a sovereign act per G-0063. Non-human
// actors are refused. The error names the rule explicitly (so a
// reader landing on the message understands *why* it's blocked) and
// points at the `--force --reason "..."` override path.
//
// The rule is scoped to two conditions, intentionally narrow:
//
//   - kind == epic (other kinds' active/accepted/in_progress edges are
//     a separate open question, deferred at planning time)
//   - newStatus == active (other epic transitions — proposed → cancelled,
//     active → done, etc. — are not sovereign acts under this rule)
//
// Caller has already verified !force; this helper does not re-check
// that. --force is the explicit override, and the existing provenance
// coherence rule (`aiwf-force requires a human/ actor`) ensures
// non-human + --force still fails at the coherence chokepoint, so the
// override path is human-only by construction.
func requireHumanActorForEpicActivation(kind entity.Kind, newStatus, actor string) error {
	if kind != entity.KindEpic || newStatus != entity.StatusActive {
		return nil
	}
	if strings.HasPrefix(actor, "human/") {
		return nil
	}
	return fmt.Errorf("aiwf promote epic active: sovereign act requires a human/ actor (got %q); have a human run the verb, or use `--force --reason \"...\"` to override", actor)
}
