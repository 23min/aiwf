---
id: ADR-0041
title: Classify a cross-branch reference by whether its branch is published
status: accepted
---
## Context

ADR-0030 made a reference resolvable through the cross-branch view non-blocking,
and it treats two situations identically — the id is "known to exist on some
other local branch **or** remote-tracking ref," and either way fires
`cross-branch-pending` at warning severity.

Those two are not equivalent from any observer other than the machine that
resolved them. A remote-tracking hit means the entity has been published: any
clone can fetch it, and the reference resolves for everyone eventually and for
anyone who fetches now. A local-branch hit means the entity exists on exactly
one working copy on earth. No fetch reaches it, no clone sees it, and no CI
checkout can.

Measured 2026-08-06 in this repo: `origin` carries exactly one branch. Two gaps
on mainline cite an id minted on an unpushed epic branch, so `aiwf check`
reports a warning here and CI reports an error on the same bytes — not because
the surfaces disagree about the rules, but because the id is genuinely absent
from everything CI can see. Instances of this shape run at roughly one per three
weeks with a window long enough to span a push.

The collapsed classification forces a bad choice. Blocking every cross-branch
reference at the push boundary would undo what ADR-0030 was written to enable,
and at a long-running epic's timescale would hold mainline pushes for weeks.
Tolerating the condition in CI is worse: a ref-less checkout sees only "this id
resolves to nothing," so it cannot tell an unmerged local branch from a
fabricated id, and tolerating the first tolerates the second — surrendering
`unresolved` as a blocking class everywhere it matters most.

The distinction needed to avoid that choice is already computed and then
discarded. `trunk.RefHit` carries the `Ref` each hit came from,
`trunk.ScanCrossBranch` returns local and remote hits separately, and
`cliutil.LoadTreeWithTrunk` populates both — but the check-side resolver groups
on the union and never consults which kind of ref answered.

## Decision

**Classify a cross-branch hit by the most visible ref that resolves it.**

- Resolvable from a **remote-tracking ref** — `cross-branch-pending`, warning
  severity, exactly as ADR-0030 specifies today. The entity is published; the
  reference is a validated pointer anyone can follow.
- Resolvable **only from local branch refs** — a distinct subcode at **error**
  severity. The reference is valid on this machine and nowhere else, so the tree
  is not one that can be handed to anyone.
- Resolvable at **no tier** — `unresolved`, error severity, unchanged.

Every classification above describes a caller that built the view. A surface
loading without the cross-branch scan cannot tell which of the three a missing
id belongs to, so it chooses none of them and reports a non-blocking
`unresolved-unverified` instead (G-0558). Adding a tier here therefore adds a
verdict that same rule withholds from those surfaces.

This applies symmetrically to `refs-resolve` and `body-prose-id`; both read the
same view and both carry the same exposure.

The error names the remedy, and the remedy is neither of the two ADR-0030
rejected: not editing the prose, and not waiting for the merge. It is **push the
branch that carries the id** — which is what this project's standing guidance
already asks for when it says to allocate on your working branch and push
promptly.

## Consequences

- The pre-push hook blocks a tree whose references live only on unpushed
  branches. That is the boundary at which the tree stops being local, which is
  the boundary at which the fact becomes true.
- ADR-0030's benefit survives intact for every case where the source branch has
  been published — which, under the guidance above, is the intended case.
- ADR-0030's escalation invariant extends in both directions. A classification
  must de-escalate when the branch is pushed and re-escalate when the remote
  branch is deleted, on the same live-recomputation basis that already governs
  the fall back to `unresolved` when a branch vanishes entirely.
- Linked worktrees share `refs/heads/*`, so a sibling worktree's committed but
  unpushed branch classifies as local-only. That is correct: it is unpushed.
- **CI's verdict does not converge on this decision alone.** A ref-less checkout
  still cannot resolve a published branch unless the workflow fetches remote
  refs. This decision guarantees only that every tree reaching CI has its
  references resolvable from something published; wiring CI to see them is
  G-0536's scope.
- At adoption this repo's own tree carries two references that classify
  local-only, so pushes block until the branch holding the cited id is pushed or
  merged. That is the decision working, not a migration cost to absorb.

## Validation

A fixture driving the full lifecycle, in the shape ADR-0030 already requires of
its own escalation invariant: mint an entity on a local branch and reference it
from mainline (error, local-only); push the branch (warning,
`cross-branch-pending`); delete the remote branch (error, local-only again);
delete the branch entirely (error, `unresolved`).

The read-path half — that every surface renders the same verdict on the same
bytes — is covered by the agreement invariants D-0063 records, and is
independent of this classification.

## References

- ADR-0030 — extended the cross-branch view to reference resolution; this
  refines the classification it introduced without superseding it
- ADR-0025 — built the cross-branch view, for allocation only
- G-0556 — the divergence this settles
- G-0558 — read paths raising a finding they lack the evidence for; independent,
  and fixable ahead of this
- G-0536 — no CI backstop for `aiwf check`; where CI gains the refs that make
  its verdict converge
- D-0063 — the harness direction that covers the read-path agreement half
