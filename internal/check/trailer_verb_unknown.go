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
// Severity was warning at G-0150 landing time so the rule introduced
// without retroactive breakage of existing fabricated trailers in
// history. G-0218 Patch 1 closed the composition-time gap with a
// `commit-msg` git hook (HookInstallSHA below); G-0218 Patch 2 (this
// file) tightens severity to error for commits whose ancestry
// includes the hook-install SHA. Pre-hook history stays at warning —
// rewriting those SHAs would invalidate addressed_by_commit refs.
//
// Closes G-0150.
const CodeTrailerVerbUnknown = "trailer-verb-unknown"

// HookInstallSHA is the full SHA of the commit that installed the
// commit-msg hook materialization in internal/initrepo (G-0218
// Patch 1). Commits descending from this SHA either pre-date the
// hook in the operator's local clone (acceptable — operator hadn't
// run `aiwf update` yet) or bypassed the hook via `--no-verify` /
// git plumbing (the policy violation this rule's post-cutoff
// severity tightening targets).
//
// The CLI gather layer at internal/cli/check/provenance.go walks
// `git rev-list HookInstallSHA..HEAD` once per check invocation to
// build the post-cutoff SHA set. When HookInstallSHA is unreachable
// from HEAD (shallow clone, fork that diverged before the hook
// landed), the walk yields an empty set and every finding stays at
// warning — the fallback preserves the G-0150 baseline so divergent
// histories aren't retroactively broken.
//
// G-0218 Patch 2.
const HookInstallSHA = "0baed90b951f3d6e755a44ca427b7e01e90c2f5c"

// RunTrailerVerbUnknown returns one finding per commit in commits
// whose `aiwf-verb:` trailer value is neither in registeredVerbs (the
// kernel Cobra command tree) nor in ritualVerbs (the non-kernel verbs
// stamped by embedded ritual skills). Commits without an `aiwf-verb:`
// trailer, with an empty value, or whose value resolves to either
// closed set are silent.
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
// been retroactively acknowledged via `aiwf acknowledge illegal`.
// The CLI gather layer computes the map once per check invocation
// (via WalkAcknowledgedSHAs in acks.go) and passes it here so
// historical stray commits with `aiwf-verb: <fabricated>` trailers
// can be quieted without rewriting history. Per-SHA closed-set
// scoping; nil or empty map is "no acknowledgments." Ack is checked
// before the cutoff decision so an explicit acknowledgment overrides
// every severity transition.
//
// G-0218 Patch 2: postCutoffSHAs carries the set of commit SHAs that
// descend from HookInstallSHA (computed once per check invocation by
// the CLI gather layer via `git rev-list HookInstallSHA..HEAD`).
// Findings on these commits emit at SeverityError with a remediation
// hint — the commit-msg hook would have refused them at composition
// time, so a post-cutoff fabricated trailer means the commit
// bypassed the hook (`--no-verify` or git plumbing). Pre-cutoff
// commits (SHA absent from the map) stay at SeverityWarning per the
// G-0150 baseline so existing trunk history isn't retroactively
// broken. nil or empty postCutoffSHAs degrades to "all warning" —
// the safe fallback for shallow clones, forks that diverged before
// the hook landed, or any future state where the cutoff SHA is
// unreachable from HEAD.
//
// Closes G-0150; G-0218 Patch 2 tightens severity for post-cutoff.
func RunTrailerVerbUnknown(commits []scope.Commit, registeredVerbs, ritualVerbs map[string]struct{}, ackedSHAs, postCutoffSHAs map[string]bool) []Finding {
	if len(commits) == 0 || len(registeredVerbs) == 0 {
		return nil
	}
	var out []Finding
	for _, c := range commits {
		if ackedSHAs[c.SHA] {
			// M-0159/AC-3 — retroactive acknowledgment exempts this
			// commit. Same per-SHA closed-set semantics as the other
			// two ack-consuming rules. Checked before the cutoff
			// decision so an ack always wins.
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
			severity := SeverityWarning
			hint := ""
			if postCutoffSHAs[c.SHA] {
				severity = SeverityError
				hint = "the commit-msg hook installed by `aiwf init` / `aiwf update` (G-0218) refuses values outside the registered verb set ∪ ritualVerbs allowlist at composition time. This commit descends from the hook-install SHA, so it bypassed the hook via `--no-verify` or git plumbing. Fix the trailer value, or reword without an aiwf-verb trailer if the commit carries no kernel-meaningful intent."
			}
			out = append(out, Finding{
				Code:     CodeTrailerVerbUnknown,
				Severity: severity,
				Message: fmt.Sprintf(
					"commit %s carries aiwf-verb: %q which is not a registered top-level verb or subverb (closed set sourced from the running binary's Cobra command tree)",
					shortHash(c.SHA), tr.Value),
				Hint: hint,
			})
		}
	}
	return out
}
