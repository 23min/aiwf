package verb

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// AcknowledgeMistag records a sovereign acceptance that an entity's area tag and
// its commits' landing zone legitimately disagree — the escape valve for the
// area-mistag check (M-0181/AC-6). Like AcknowledgeIllegal it is a current-day
// empty commit, but keyed per-ENTITY rather than per-SHA, carrying:
//
//	aiwf-verb: acknowledge-mistag
//	aiwf-entity: <canonical-id>
//	aiwf-actor: human/<name>
//	aiwf-reason: <free-form text>
//
// The check's WalkAcknowledgedMistags walks HEAD for these commits and exempts
// the named entities from area-mistag. The acknowledgement lives in git
// (queryable via aiwf history), aligns with the existing sovereign-act
// semantics, and does not pollute aiwf.yaml — the same rationale as
// acknowledge-illegal.
//
// Constraints (mirroring acknowledge-illegal):
//   - reason must be non-empty after trim (sovereign acts require a written
//     rationale);
//   - actor must be `human/...` (no LLM / bot ack — the cross-cutting judgment
//     is the human's);
//   - the id must resolve to a real entity (composite AC ids roll up to their
//     milestone); a typo is refused rather than recording a no-op ack that
//     silently suppresses nothing.
//
// "What verb undoes this?" — none, by deliberate design (the acknowledge-illegal
// answer): the ack is one-way; if regretted, the operator re-tags the entity
// (`aiwf set-area`) so the mistag no longer fires, or lives with the suppressed
// finding. The verb is human-sovereign by construction.
func AcknowledgeMistag(ctx context.Context, t *tree.Tree, id, actor, reason string) (*Result, error) {
	_ = ctx
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("aiwf acknowledge mistag: --reason is required (non-empty after trim)")
	}
	if !strings.HasPrefix(actor, "human/") {
		return nil, fmt.Errorf("aiwf acknowledge mistag: --actor must be human/<name> (got %q; sovereign acts trace to a named human)", actor)
	}
	canonID := entity.Canonicalize(entity.CompositeRoot(id))
	if t.ByID(canonID) == nil {
		return nil, fmt.Errorf("aiwf acknowledge mistag: unknown entity %q (it resolves to no entity in the tree)", id)
	}
	cleanedReason := strings.TrimSpace(reason)
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "acknowledge-mistag"},
		{Key: gitops.TrailerEntity, Value: canonID},
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerReason, Value: cleanedReason},
	}
	for _, tr := range trailers {
		if err := gitops.ValidateTrailer(tr.Key, tr.Value); err != nil {
			//coverage:ignore defense-in-depth mirroring acknowledge-illegal: every trailer value here is already validated upstream — canonID by t.ByID, and the actor by cliutil.ResolveActor's actorPattern at the CLI boundary (the verb's own human/ prefix check is necessary but not sufficient) — so this cannot fire via the public API; kept to catch a future malformed-trailer regression.
			return nil, fmt.Errorf("aiwf acknowledge mistag: %w", err)
		}
	}
	return plan(&Plan{
		Subject:    fmt.Sprintf("aiwf acknowledge mistag %s", canonID),
		Body:       cleanedReason,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
}
