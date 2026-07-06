---
id: E-0060
title: Resolve cross-branch entity references at check and read time
status: proposed
---

## Goal

Let a branch, worktree, or session validly reference an entity minted on a
different local branch or worktree — in `aiwf check` and in `aiwf show`/`aiwf
list` — without waiting for a merge and without copying the entity anywhere.

## Context

An entity discovered mid-flight — typically a gap, filed on whatever branch
the operator happened to be working on (an epic worktree, a patch branch) —
is physically absent from every other local branch until that branch merges.
`aiwf check`'s reference resolution has no way to distinguish "this id is
real, just not here yet" from "this id was never allocated": both
`refs-resolve` (structured fields) and `body-prose-id` (prose mentions)
resolve purely against the currently loaded tree and fire the same
`unresolved` finding either way. `ADR-0030` records the decision this epic
implements: reuse the cross-branch view the allocator already computes as a
second input, to reference resolution and reads, never to mutation.

The cross-branch view itself already exists and needs no new git-scanning
mechanism. `E-0052` (done) built it for the allocator: `Tree.LocalRefIDs`
(M-0212) unions ids from every local `refs/heads/*`; `Tree.RemoteRefIDs`
(M-0214) does the same for `refs/remotes/*`; both feed `AllocationIDs()`
alongside `Tree.TrunkIDs`. `ADR-0025` scoped that view to allocation
(prevention) only, deliberately keeping `ids-unique` trunk-anchored.

There is also a shipped precedent for exactly the resolver shape this epic
needs: `G-0241` (addressed) added a second-tier resolver to
`classifyBodyToken` (`internal/check/body_prose_id.go`) that falls back to
`Tree.TrunkIDs` on a local-tree miss. That tier is deliberately **silent** —
a trunk-known id resolves with no finding at all, correct because trunk is
authoritative. A sibling local or remote-tracking branch is provisional —
it can be rebased, renamed, or abandoned before it ever merges — so it needs
a **visible, non-blocking, escalatable** tier instead of a silent one. This
epic adds that tier; it does not change `TrunkIDs`' existing silent
behavior.

Cherry-picking the source commit onto the referencing branch, and a
dedicated "pull"/"materialize" verb that creates a fresh commit with correct
trailers, were both considered and rejected — see `ADR-0030`'s Context
section for the full analysis. Both duplicate the entity's physical
presence and require later reconciliation; this epic's approach never
copies anything, so there is nothing to reconcile.

## Scope

### In scope

- Extend `refs-resolve` (structured fields, e.g. `depends_on`) and
  `body-prose-id` (prose mentions) so that a miss against the local tree
  consults `Tree.LocalRefIDs`/`Tree.RemoteRefIDs` before firing
  `unresolved`. A hit there classifies as a new, distinct, non-blocking
  finding instead.
- The escalation invariant as a named, mechanically-tested acceptance
  criterion: an id classified cross-branch-pending while its source branch
  exists must re-escalate to a hard `unresolved` once that branch
  disappears from the cross-branch view too (deleted, abandoned, never
  merged).
- Extend `aiwf show`/`aiwf list` to resolve and render an entity's content
  from another local branch or remote-tracking ref when it is
  cross-branch-known but locally absent — read-only, no working-tree/index/
  ref writes — and to visibly label the result as sourced from elsewhere,
  never presented indistinguishably from a locally-resolved entity.
- Whatever tracking the read-side milestone needs to know *where* (which
  ref, what path) a cross-branch-known id lives, since `LocalRefIDs`/
  `RemoteRefIDs` today are bare id sets with no path/ref info (unlike
  `TrunkIDs`, which is `[]trunk.ID` and already carries path).

### Out of scope

- Any mutating verb (`promote`, `edit-body`, `cancel`, `reallocate`, and so
  on) operating against a cross-branch-pending target. An entity stays
  singly-owned and mutable only from the branch that physically holds its
  file — unchanged by this epic.
- Changes to `ids-unique`'s existing trunk-anchored, working-tree-vs-trunk
  basis, or to `TrunkIDs`' existing silent resolution tier (`G-0241`) — this
  epic adds a separate tier for local/remote refs alongside it.
- Any new git-scanning mechanism. This epic is a pure consumer of the
  cross-branch view `E-0052`/`ADR-0025` already compute.
- Cross-machine visibility beyond what `git fetch` already brings into
  `refs/remotes/*` locally. No new networking, no new remote protocol.

## Constraints

- Reuse `Tree.LocalRefIDs`/`Tree.RemoteRefIDs`/`AllocationIDs()` as-is where
  possible; any widening of these fields (e.g. adding path/ref info) is
  additive and must not change the allocator's existing consumption of
  them.
- The escalation path is proven by a fixture-driven test — branch created,
  id classifies pending, branch deleted, `aiwf check` re-run, id classifies
  `unresolved` — not asserted only in prose or code comments.
- No entity content is ever copied, cached, or materialized into the
  working tree, the index, or a new ref by any part of this epic.
  Resolution is always a live read against the other ref at the point of
  use.

## Success criteria

- [ ] A reference (structured field or prose) to an id that exists only on
      another local branch or remote-tracking ref classifies as a distinct,
      non-blocking finding, not `unresolved`.
- [ ] A reference to an id that exists nowhere — not locally, not
      cross-branch — still hard-fails as `unresolved`, unchanged from
      today.
- [ ] The escalation fixture test described in Constraints exists, passes
      in CI, and is named as an acceptance criterion of the milestone that
      delivers the check-side change.
- [ ] `aiwf show`/`aiwf list` render an entity's content sourced from
      another local branch, visibly labeled as such.
- [ ] No mutating verb accepts a cross-branch-pending target as its
      operand.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Do `LocalRefIDs`/`RemoteRefIDs` need to widen to carry path + ref (mirroring `trunk.ID`), or can the read-side milestone do a fresh lookup at render time without changing the allocator-facing fields? | Yes, for the read-side milestone | Resolve during that milestone's planning, informed by `internal/trunk`'s existing per-id path tracking for `TrunkIDs` |
| What's the exact subcode/finding-code shape for the new pending tier — a new subcode on the existing `refs-resolve`/`body-prose-id` codes, or a new finding code entirely? | Yes, for the check-side milestone | Resolve during that milestone's planning; either shape satisfies `ADR-0030`'s decision, this is an implementation-detail choice |
| Should `aiwf status`/`aiwf render --format=html` also surface cross-branch-pending references, or is `check` + `show` + `list` sufficient for v1? | No | Defer; candidate follow-on gap if it turns out to matter |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The escalation path silently regresses in a future refactor of the cross-branch view (e.g. narrowing what counts as "local") | High — a dangling reference could masquerade as valid forever | The escalation fixture test is a named, CI-gated acceptance criterion, not just documented behavior (`ADR-0030` Validation section) |
| Read-side `git show <ref>:<path>` lookups add per-invocation git subprocess cost to `aiwf show`/`aiwf list` | Low/Medium | Scope the lookup to fire only on a local-tree miss; never on the common case where the entity already resolves locally |

## Milestones

Not yet allocated — candidates to sequence via `aiwfx-plan-milestones`:

- Check-side: add the cross-branch-pending tier to `refs-resolve` and
  `body-prose-id`, sourced from `Tree.LocalRefIDs`/`Tree.RemoteRefIDs`,
  with the escalation invariant as a named, mechanically-tested acceptance
  criterion.
- Read-side: extend `aiwf show`/`aiwf list` to resolve and label an
  entity's content from another local ref when cross-branch-known but
  locally absent; resolves the path/ref-tracking open question above.

## ADRs produced

- ADR-0030 — Extend cross-branch view to reference resolution and reads

## References

- ADR-0030 — Extend cross-branch view to reference resolution and reads
- ADR-0025 — Allocator's cross-branch view spans all refs, fed to
  allocation only
- E-0052 — Broaden the id allocator's cross-branch view to cut collisions
  (prior epic this builds on)
- G-0241 — BodyProseIDIndex skips TrunkIDs; trunk-only ids appear
  unresolved (precedent second-tier resolver pattern)
- G-0272, G-0281 — sibling collision/visibility cluster
