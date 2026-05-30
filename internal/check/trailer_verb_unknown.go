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

// ritualVerbs is the closed set of *ritual* verbs that the
// aiwf-extensions rituals legitimately stamp as `aiwf-verb:` values
// even though they are not kernel Cobra verbs. They name ritual
// lifecycle acts (a trailered epic-wrap merge, etc.) that are
// meaningful in `aiwf history` but have no kernel command. Recognizing
// them here keeps `trailer-verb-unknown` from firing on every ritual
// wrap commit while still catching genuinely-fabricated verbs (G-0180).
//
// Source: the `aiwf-verb:` trailers stamped by the embedded rituals
// under internal/skills/embedded-rituals (today only `wrap-epic` is
// emitted; `wrap-milestone` is its sibling and is included so the same
// finding does not recur if a wrap-milestone commit starts carrying
// the trailer). If a future ritual stamps a new non-kernel verb, extend
// this set (and ideally derive it from the embedded snapshot — tracked
// alongside G-0180).
var ritualVerbs = map[string]struct{}{
	"wrap-epic":      {},
	"wrap-milestone": {},
}

// RunTrailerVerbUnknown returns one finding per commit in commits
// whose `aiwf-verb:` trailer value is neither in registeredVerbs (the
// kernel Cobra command tree) nor in the ritualVerbs allowlist (ritual
// lifecycle verbs like `wrap-epic`). Commits without an `aiwf-verb:`
// trailer, with an empty value, or whose value resolves are silent.
//
// An empty registeredVerbs set short-circuits to no findings —
// the verb enumeration runs at RunE time and could in principle
// return empty (cobra tree wiring failure); we'd rather skip than
// flood every commit as "unknown."
//
// Closes G-0150.
func RunTrailerVerbUnknown(commits []scope.Commit, registeredVerbs map[string]struct{}) []Finding {
	if len(commits) == 0 || len(registeredVerbs) == 0 {
		return nil
	}
	var out []Finding
	for _, c := range commits {
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
