---
id: G-0215
title: 'Kernel-wide audit: production nil-arg passes need structural chokepoint'
status: open
discovered_in: M-0159
---
## Pattern

A function in `internal/cli/` or `internal/check/` gains a new parameter (typically a `map[string]bool`, an oracle interface, or a config object). The kernel-wide chokepoint when this happens is per-AC: a policy under `internal/policies/` targets the specific signature change and asserts the call sites pass real values (e.g., `PolicyAcksHelperLift` class 4b for AC-3 named public consumers; class 4e for AC-4 leaf predicates).

The pattern that escapes per-AC chokepoints: an existing PRODUCTION call site (not a test) passes `nil` (or zero-value) for the new parameter, either because:

1. The supporting plumbing is deliberately parked at a tracked gap (e.g., `cherryPicked` at `internal/cli/check/provenance.go:67` — G-0202 parks the cherry-pick gather-side until M-0159/AC-6 lands).
2. The implementer's mass-update script applied `, nil` mechanically and a production site was caught in the sweep (the AC-4 GREEN second-reviewer probe surfaced this for `forcedUntraileredFindings`).
3. A future kernel rule's author legitimately reasoned about test sites (where nil is correct because no behavior is exercised) but not production sites (where nil silently degrades the rule).

In all three cases, every existing test passes; the rule simply silently fails to enforce its silencing arm, its oracle lookup, or its cherry-pick suppression. Behavioral evidence catches the regression only if a test specifically exercises the silencing path — most tests don't.

## Why per-AC policies don't generalize

`PolicyAcksHelperLift` covers the M-0159/AC-3 + M-0159/AC-4 surface: five named consumers (three public, two predicate-helpers) plus their predicate call sites. The policy is correct for its scope. But:

- A future kernel rule that adds a parameter to a public consumer would need its own per-AC policy entry. Drift visibility forces the author to update the policy, which is the chokepoint working as intended — but the *meta-pattern* is not policed.
- A future kernel verb that adds an interface parameter (e.g., a new oracle, a new walker) has the same exposure with no chokepoint at all.
- The `cherryPicked` nil at `provenance.go:67` is documented + tracked (G-0202), but the documentation lives in a 5-line comment block that a future reader could remove during a "cleanup" PR; the policy layer doesn't know to require a `// PARKED: G-NNNN` marker.

## Proposed chokepoint shape

A repo-wide policy under `internal/policies/` that:

1. Scans every production .go file under `internal/cli/` and `internal/check/` for CallExpr nodes.
2. For each `nil` literal argument at any position, requires EITHER:
   - A `// PARKED: G-NNNN` or `// OPTIONAL: <reason>` comment within N lines of the call site (operator-discipline marker for deliberate parks), OR
   - The receiving function's signature annotation marks that parameter position as `nilable-by-design` (a doc-comment convention to be defined).
3. Untagged production-code `nil` literals at policy-bearing positions fire with a finding pointing at the call site and naming both escape hatches.

The policy is intentionally permissive at first land — every existing site needs either a marker or the by-design annotation. The initial sweep is mechanical: each `nil` literal gets the right marker after a human-review pass.

## Known instances visible at gap-creation time

- `internal/cli/check/provenance.go:67` — `cherryPicked` parameter nil; documented as G-0202 deferred; would qualify for `// PARKED: G-0202` marker.
- Any future M-0161-class work (worktree-escape, force-push, branch rename) that adds a parameter to RunIsolationEscape would benefit from this policy.

## Out of scope for this gap

- Implementing the policy itself (that's the gap's resolution).
- Sweeping every nil-literal in the kernel and adding markers (downstream of the policy landing).
- Defining the canonical `nilable-by-design` annotation convention (an ADR or design decision precedes the policy).

## Why this matters

The kernel's "framework correctness must not depend on the LLM's behavior" principle treats mechanical chokepoints as the source of guarantees. Per-AC policies are correct but don't generalize; without a meta-chokepoint, every future kernel parameter addition introduces silent-bug exposure that depends on the implementer's vigilance and the second reviewer's sabotage instinct. AC-4 GREEN's second-reviewer probe surfaced the gap on its specific instance; a future similar gap might not get the same calibration depth.
