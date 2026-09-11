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
// also drops (G30): `--grep` matches a body line that begins
// `aiwf-entity: <id>` as readily as a real trailer, and what tells the two
// apart is git's own trailer parser, which returns a value for one and
// nothing for the other. So admission reads the parsed entity trailer —
// either aiwf-entity or aiwf-prior-entity, since the query greps both and a
// reallocate's lineage event carries only the latter. Callers skip a commit
// git cannot attribute to an entity rather than bucket a blank row.
//
// A commit carrying that trailer and nothing else is an event: a
// shipped-surface edit proves its provenance with aiwf-entity alone
// (D-0071), so there is no verb to name what it did and no separate actor
// where no verb ran.
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
	if strings.TrimSpace(idx[gitops.TrailerEntity]) == "" &&
		strings.TrimSpace(idx[gitops.TrailerPriorEntity]) == "" {
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
