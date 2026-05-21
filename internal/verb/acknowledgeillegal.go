package verb

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/gitops"
)

// AcknowledgeIllegal records a retroactive sovereign override for a
// historical FSM-illegal commit that fsm-history-consistent flags. The
// acknowledgment lives as a current-day empty commit carrying:
//
//	aiwf-verb: acknowledge-illegal
//	aiwf-force-for: <sha>
//	aiwf-actor: human/<name>
//	aiwf-reason: <free-form text>
//
// The fsm-history-consistent rule (M-0136/AC-2) walks HEAD's reachable
// history for `aiwf-force-for` trailers and exempts illegal-transition
// findings whose offending commit appears as a target.
//
// Constraints (M-0136/AC-1):
//   - reason must be non-empty after trim (sovereign acts require a
//     written rationale).
//   - actor must be `human/...` (sovereign acts trace to a named
//     human; no LLM / bot ack).
//   - sha must match the 7-40-hex SHA pattern (the trailer's value
//     constraint, enforced via gitops.ValidateTrailer).
//
// AC-4 will extend this with a reachability check (sha must resolve
// to a commit reachable from HEAD); that gate is deferred to keep
// AC-1 focused on commit shape.
//
// Returns a Result with a Plan carrying the empty commit's trailers.
// The Apply pipeline materializes the `git commit --allow-empty` once
// the human gate clears.
func AcknowledgeIllegal(ctx context.Context, root, sha, actor, reason string) (*Result, error) {
	_ = ctx
	_ = root
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("aiwf acknowledge-illegal: --reason is required (non-empty after trim)")
	}
	if !strings.HasPrefix(actor, "human/") {
		return nil, fmt.Errorf("aiwf acknowledge-illegal: --actor must be human/<name> (got %q; sovereign acts trace to a named human)", actor)
	}
	cleanedReason := strings.TrimSpace(reason)
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "acknowledge-illegal"},
		{Key: gitops.TrailerForceFor, Value: sha},
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerReason, Value: cleanedReason},
	}
	for _, tr := range trailers {
		if err := gitops.ValidateTrailer(tr.Key, tr.Value); err != nil {
			return nil, fmt.Errorf("aiwf acknowledge-illegal: %w", err)
		}
	}
	short := sha
	if len(short) > 8 {
		short = short[:8]
	}
	return plan(&Plan{
		Subject:    fmt.Sprintf("aiwf acknowledge-illegal %s", short),
		Body:       cleanedReason,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
}
