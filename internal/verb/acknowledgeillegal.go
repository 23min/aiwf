package verb

import (
	"context"
	"errors"
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
//   - sha must resolve to a commit reachable from HEAD (M-0136/AC-4 —
//     out-of-history SHAs reject with a typed error rather than
//     silently accumulating no-op acknowledgments).
//
// Returns a Result with a Plan carrying the empty commit's trailers.
// The Apply pipeline materializes the `git commit --allow-empty` once
// the human gate clears.
//
// Stub for M-0136/AC-1 red phase. Implementation lands in green.
func AcknowledgeIllegal(ctx context.Context, root, sha, actor, reason string) (*Result, error) {
	_ = ctx
	_ = root
	_ = sha
	_ = actor
	_ = reason
	return nil, errors.New("aiwf acknowledge-illegal: verb not implemented (M-0136/AC-1 red phase)")
}
