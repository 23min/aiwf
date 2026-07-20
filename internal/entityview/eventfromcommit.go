package entityview

import (
	"strings"

	"github.com/23min/aiwf/internal/gitops"
)

// eventfromcommit.go — the pure HistoryEvent constructor render's single
// pass (E-0054 / M-0221) uses to reproduce ReadHistoryChain's per-record
// output from the shared HEAD walk, without re-grepping per entity.
//
// ReadHistoryChain stays the authoritative oracle: it is untouched here,
// and the AC-3 differential (render bucket == ReadHistory) fails if this
// constructor drifts from it. The field mapping below is kept in lockstep
// with ReadHistoryChain's parse loop.

// EventFromCommit builds a HistoryEvent from one commit's raw fields.
// sha is the full hash (shortened here to match ReadHistoryChain);
// authorDate is git's %aI; subject is %s; body is %B (the full raw
// message, from which the prose body %b is derived); trailers is the
// parsed trailer block.
//
// Returns ok=false for the prose-mention false-positive ReadHistoryChain
// also drops (G30): a commit whose aiwf-entity trailer matched a grep but
// which carries neither aiwf-verb nor aiwf-actor is not a real entity
// event. Callers skip such commits rather than bucket a blank row.
func EventFromCommit(sha, authorDate, subject, body string, trailers []gitops.Trailer) (HistoryEvent, bool) {
	// Single-value trailers collapse to a last-value map (matching git's
	// per-key extraction for the one-occurrence aiwf trailers); aiwf-scope-ends
	// repeats, so collect every value in trailer order — the same shape
	// ReadHistoryChain's SplitMultiValueTrailer produces.
	idx := make(map[string]string, len(trailers))
	var scopeEnds []string
	for _, tr := range trailers {
		if tr.Key == gitops.TrailerScopeEnds {
			if v := strings.TrimSpace(tr.Value); v != "" {
				scopeEnds = append(scopeEnds, v)
			}
			continue
		}
		idx[tr.Key] = tr.Value
	}

	verb := strings.TrimSpace(idx[gitops.TrailerVerb])
	actor := strings.TrimSpace(idx[gitops.TrailerActor])
	if verb == "" && actor == "" {
		return HistoryEvent{}, false
	}

	ev := HistoryEvent{
		Commit:       ShortHash(sha),
		Date:         authorDate,
		Detail:       strings.TrimSpace(subject),
		Verb:         verb,
		Actor:        actor,
		To:           strings.TrimSpace(idx[gitops.TrailerTo]),
		Force:        strings.TrimSpace(idx[gitops.TrailerForce]),
		AuditOnly:    strings.TrimSpace(idx[gitops.TrailerAuditOnly]),
		Principal:    strings.TrimSpace(idx[gitops.TrailerPrincipal]),
		OnBehalfOf:   strings.TrimSpace(idx[gitops.TrailerOnBehalfOf]),
		AuthorizedBy: strings.TrimSpace(idx[gitops.TrailerAuthorizedBy]),
		Scope:        strings.TrimSpace(idx[gitops.TrailerScope]),
		ScopeEnds:    scopeEnds,
		Reason:       strings.TrimSpace(idx[gitops.TrailerReason]),
		Body:         StripTrailers(strings.TrimSpace(bodyAfterSubject(body))),
	}
	if metrics, ok := gitops.ParseTestMetrics(idx[gitops.TrailerTests]); ok {
		m := metrics
		ev.Tests = &m
	}
	return ev, true
}

// bodyAfterSubject returns git's %b (the message body) given %B (the full
// raw message): everything after the first blank line that separates the
// subject from the body. A message with no blank line has no body, so it
// returns "". This mirrors git's own subject/body split, so
// EventFromCommit's StripTrailers input matches ReadHistoryChain's %b.
func bodyAfterSubject(fullBody string) string {
	if i := strings.Index(fullBody, "\n\n"); i >= 0 {
		return fullBody[i+2:]
	}
	return ""
}
