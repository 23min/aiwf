package check

import (
	"fmt"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// CodeTrailerVerbUnknown fires when a commit carries
// `aiwf-verb: <value>` whose value is not in the closed set of
// verb paths registered in the running binary's Cobra command
// tree.
//
// The kernel principle "framework correctness must not depend on
// the LLM's behavior" assumes trailer values are mechanically
// validated. Before G-0150 they were not — an LLM-driven session
// fabricated `aiwf-verb: implement` on a hand-rolled Conventional-
// Commits code commit; every gate (pre-commit `aiwf check
// --shape-only`, pre-push `aiwf check`, golangci-lint, `go test`)
// passed. The fabricated trailer would have polluted `aiwf history`
// projections by misrepresenting a hand-rolled code commit as a
// kernel-verb invocation.
//
// Severity is warning at landing time so the rule introduces
// without retroactive breakage of existing fabricated trailers in
// history. Promotion to error is contingent on cleaning history
// first (potentially via `aiwf acknowledge-illegal` for the few
// intentional historical strays, if any).
//
// Closes G-0150.
const CodeTrailerVerbUnknown = "trailer-verb-unknown"

// RunTrailerVerbUnknown returns one finding per commit in commits
// whose `aiwf-verb:` trailer value is neither in registeredVerbs (the
// kernel Cobra command tree) nor in ritualVerbs (the non-kernel verbs
// stamped by embedded ritual skills). Commits without an `aiwf-verb:`
// trailer, with an empty value, or whose value resolves are silent.
//
// An empty registeredVerbs set short-circuits to no findings —
// the verb enumeration runs at RunE time and could in principle
// return empty (cobra tree wiring failure); we'd rather skip than
// flood every commit as "unknown."
//
// Per G-0190 the caller is expected to derive ritualVerbs from the
// embedded ritual snapshot (typically via skills.RitualTrailerVerbs)
// so the allowlist stays in lock-step with what the rituals actually
// stamp. A nil ritualVerbs is treated as the empty set; the kernel
// `add`/`promote`/etc. verbs still resolve via registeredVerbs.
//
// M-0159/AC-3: ackedSHAs carries the set of commit SHAs that have
// been retroactively acknowledged via `aiwf acknowledge-illegal`.
// The CLI gather layer computes the map once per check invocation
// (via WalkAcknowledgedSHAs in acks.go) and passes it here so
// historical stray commits with `aiwf-verb: <fabricated>` trailers
// can be quieted without rewriting history. Per-SHA closed-set
// scoping; nil or empty map is "no acknowledgments."
//
// Closes G-0150.
func RunTrailerVerbUnknown(commits []scope.Commit, registeredVerbs, ritualVerbs map[string]struct{}, ackedSHAs map[string]bool) []Finding {
	if len(commits) == 0 || len(registeredVerbs) == 0 {
		return nil
	}
	var out []Finding
	for _, c := range commits {
		if ackedSHAs[c.SHA] {
			// M-0159/AC-3 — retroactive acknowledgment exempts this
			// commit. Same per-SHA closed-set semantics as the other
			// two ack-consuming rules.
			continue
		}
		for _, tr := range c.Trailers {
			if tr.Key != gitops.TrailerVerb {
				continue
			}
			if tr.Value == "" {
				continue
			}
			if _, ok := registeredVerbs[tr.Value]; ok {
				continue
			}
			if _, ok := ritualVerbs[tr.Value]; ok {
				continue
			}
			out = append(out, Finding{
				Code:     CodeTrailerVerbUnknown,
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"commit %s carries aiwf-verb: %q which is not a registered top-level verb or subverb (closed set sourced from the running binary's Cobra command tree)",
					shortHash(c.SHA), tr.Value),
			})
		}
	}
	return out
}
