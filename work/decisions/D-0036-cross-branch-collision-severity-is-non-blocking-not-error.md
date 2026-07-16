---
id: D-0036
title: cross-branch-collision severity is non-blocking, not error
status: accepted
relates_to:
    - E-0060
    - M-0259
---
# D-0036 — Cross-branch-collision severity is non-blocking, not error

> **Date:** 2026-07-16 · **Decided by:** human/peter

## Question

M-0259/AC-3 detects a cross-branch reference whose id carries divergent
blob content across two or more refs (via `trunk.DetectCollisions`,
blob-SHA comparison) and needed to decide what severity to give the
resulting `cross-branch-collision` finding: should it block `aiwf
check`/the pre-push gate (`SeverityError`), or merely surface visibly
without blocking (`SeverityWarning`)? ADR-0030's text calls for "a
distinct subcode... instead of being silently classified as the soft
tier," but does not specify severity, and the milestone's first
implementation defaulted to `SeverityError`, matching the intuition
that a "collision" should block.

An independent code-quality review of M-0259 (dispatched at wrap) flagged
that blob-SHA divergence cannot, on its own, distinguish two distinct
situations: (a) a genuine duplicate-mint collision — two refs
independently allocated different entities under the same id — and (b)
an ordinary same-entity edit landed on one of several refs that share
the id's history, still unmerged. Both produce identical evidence (a
differing blob SHA at the same path). Because aiwf's own recommended
workflow is to work across multiple worktrees of one repo (per
CLAUDE.md's worktree-default guidance), and linked worktrees share
local branch refs, case (b) is not a rare adversarial edge case — it
is provoked by routine, same-person, same-repo multi-worktree usage:
one worktree runs `aiwf edit-body` on an entity while another worktree
holds an unrelated reference to that same entity from a branch that
never had it locally.

## Decision

`refs-resolve/cross-branch-collision` and `body-prose-id/cross-branch-collision`
are `SeverityWarning` (non-blocking), not `SeverityError`. The subcode
stays distinct and visible — it is never silently folded back into
`cross-branch-pending` — satisfying ADR-0030's stated requirement. Only
the blocking behavior changes.

## Reasoning

The base rate favors "this is an edit" over "this is a genuine
collision": the id allocator already consults this same cross-branch
view specifically to prevent duplicate-mint collisions at allocation
time (E-0052/ADR-0025), so a genuine duplicate mint should be rare by
construction, while same-entity edits across unmerged branches are
ordinary git usage — more so given aiwf's own recommended multi-worktree
workflow. A blocking check that is usually noise trains operators to
route around it (`--no-verify`, or eroded trust in the gate generally),
which is a worse outcome than under-detecting the rarer genuine case a
little later.

Critically, the genuine duplicate-mint case is not left undetected by
this decision — it is still caught, just later: the pre-existing,
already-blocking `ids-unique`/`trunk-collision` check fires the moment
both copies land in a single shared working tree (at merge time),
independent of whether this cross-branch-collision finding blocked
anything earlier. AC-3's early-detection value is preserved (the
distinct subcode still surfaces the divergence immediately, for a human
to inspect and manually reconcile if warranted); only its blocking
enforcement is deferred to the existing, more precise mechanism.

Alternatives considered:

- **Keep it blocking (`SeverityError`), document as an accepted v1
  limitation.** Rejected: the exposure is not a rare corner case here —
  it is provoked by aiwf's own recommended workflow — so shipping it as
  blocking risks routine, disruptive false positives from day one, not
  a theoretical edge case worth merely footnoting.
- **Disambiguate via git lineage** (e.g. `git merge-base` between the
  diverging refs, confirming whether they share a common ancestor blob
  for the entity) before escalating. This is the only fully correct
  fix, but is real new scope — a second git primitive integration, new
  edge cases (renames across the merge-base, 3+ diverging refs), its
  own TDD cycle. Properly a follow-on piece of work, not a wrap-time
  patch to M-0259. Tracked as a gap for future consideration.
- **A cheap heuristic** (e.g. "only escalate if none of the diverging
  copies matches trunk, so a lone outlier is treated as an edit").
  Rejected: it does not actually resolve the ambiguity — two different
  branches independently editing the same entity for unrelated reasons
  produces the same "no consensus" shape as a genuine collision, so the
  heuristic just relabels the ambiguity rather than resolving it, while
  adding real complexity and false confidence.

## Consequences

- The `aiwf reallocate` remedy in the hint table/skill doc for both
  `cross-branch-collision` subcodes is reframed to acknowledge the
  ambiguity — compare manually first; `aiwf reallocate` only applies
  once a genuine duplicate mint is confirmed, not for the common
  in-flight-edit case.
- `G-0416` tracks git-lineage-based disambiguation as future, scoped
  work, should the coarse v1 severity prove insufficient in practice.
