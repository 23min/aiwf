---
id: ADR-0030
title: Extend cross-branch view to reference resolution and reads
status: accepted
---
# ADR-0030 — Extend cross-branch view to reference resolution and reads

> **Date:** 2026-07-06 · **Decided by:** human/peter

## Context

ADR-0025 built a cross-branch view for the id allocator — the working tree, every
local `refs/heads/*`, every remote-tracking `refs/remotes/*`, and the configured
trunk ref — so the allocator sees every id already burned anywhere locally
knowable. That ADR was explicit that the view feeds **allocation (prevention)
only**: `ids-unique` stays trunk-anchored, and no other surface consumes the view,
specifically to avoid false-flagging an entity that is legitimately present on
two branches as a duplicate.

That narrow scope leaves a second, distinct friction unaddressed. An entity —
typically a gap, discovered mid-flight and filed on whatever branch the operator
happened to be working on (an epic worktree, a patch branch) — is physically
absent from every other branch, including `main`, until that branch merges. When
a *different* piece of work (a new epic, a new patch, planned on `main`) needs to
refer to that id — in a structured field or in prose ("addresses `G-0400`") —
`aiwf check`'s reference resolution has no way to tell "this id is real, just not
here yet" apart from "this id was never allocated." Both `refs-resolve` (structured
fields) and `body-prose-id` (prose mentions) resolve purely against the currently
loaded tree's entity index; a miss fires the same `unresolved` finding a
fabricated id would. The friction is structural, not cosmetic: the operator either
waits for the source branch to merge before referencing the entity, or reaches for
something outside the kernel to make it visible sooner.

**Alternative considered and rejected: cherry-picking the entity's add-commit onto
the referencing branch.** Mechanically viable but rejected on its merits:

- It bypasses the verb layer entirely — no `aiwf`-shaped commit, no trailer
  validation, no `aiwf check` involvement at the point of landing. Undiscoverable,
  not completion-friendly, and outside every guarantee the kernel otherwise makes
  about how a mutation lands.
- It preserves the original commit's author and full message verbatim (cherry-pick
  semantics) while changing only the committer — the landed commit misattributes
  `aiwf-actor`/`aiwf-principal` to whoever authored it on the source branch, not
  whoever actually placed it on the destination branch.
- Once the source branch eventually merges, the same logical "entity added" event
  exists as two independent commits reachable from `main` — `aiwf history <id>`
  then renders a permanent, confusing duplicate "added" entry for a single event.

**Alternative considered and rejected: a dedicated "pull"/"materialize" verb** that
creates a fresh, correctly-trailered commit on the destination branch by copying
the entity's content. This fixes the actor-fidelity problem but not the structural
one — it still produces a second physical copy of the entity, still duplicates the
add-event once the branches converge, and still raises an ownership question (which
copy is allowed to be promoted/edited going forward). Rejected because the chosen
decision below achieves the same practical outcome — the entity is usable from the
referencing branch — without ever copying anything.

**Alternative considered and rejected: do nothing; wait for the source branch to
merge before authoring the reference.** Sound, but reintroduces exactly the
friction this ADR exists to remove, and scales poorly whenever the source branch
is a long-lived epic.

## Decision

We reuse the exact cross-branch view ADR-0025 already computes — no new git
scanning mechanism — as an input to two additional, read-only surfaces:

1. **Check-side.** When `refs-resolve` (structured fields, e.g. `depends_on`) or
   `body-prose-id` (id-shaped tokens in markdown prose) misses the currently
   loaded tree, consult the cross-branch view before firing `unresolved`. If the
   id is found there — known to exist on some other local branch or
   remote-tracking ref — fire a distinct, non-blocking subcode,
   `cross-branch-pending`, instead. An id absent from *both* the local tree and
   the cross-branch view is unchanged: it still hard-fails as `unresolved`, exactly
   as today, because that case is still indistinguishable from a fabricated id.
2. **Read-side.** `aiwf show` and `aiwf list` may resolve and render an entity's
   content by reading its blob directly from the other branch or ref (e.g. `git
   show <ref>:<path>`) — strictly read-only, nothing is written to the working
   tree, the index, or any ref.

No entity content is ever copied or duplicated by this decision. At every point in
time there is exactly one physical copy of the entity, on whichever branch it was
originally minted on, until that branch naturally merges. A reference resolved via
the cross-branch view is a validated pointer to that one copy, not a fork of it —
which is precisely why this decision avoids every failure mode that ruled out the
cherry-pick and pull-verb alternatives: no duplicate commits, no duplicate `aiwf
history` entries, no actor/trailer misattribution, and no ownership ambiguity over
which copy is allowed to be mutated next.

Mutating verbs (`promote`, `edit-body`, `cancel`, and so on) are explicitly **not**
extended to operate cross-branch. An entity stays singly-owned and mutable only
from the branch that actually holds its file on disk. This decision touches
reference resolution and reads only.

**Escalation is a hard, mechanically-tested invariant, not a documented
convention.** A `cross-branch-pending` classification must never become a
permanent, silent softening of what is actually a dangling reference. If an id
previously classified `cross-branch-pending` later disappears from the
cross-branch view too — the source branch is deleted, or abandoned and never
merged — the next `aiwf check` run must re-escalate the finding to a hard
`unresolved`. This requires a fixture-driven test that exercises the full
lifecycle: reference validated as `cross-branch-pending` while the source branch
exists, source branch deleted, `aiwf check` re-run, finding asserted to now report
`unresolved`. A check whose "softened" classification can silently persist forever
after its justification vanishes is not a passing implementation of this decision.

## Consequences

**Positive:**

- The dominant "reference an entity minted on another branch" friction is closed
  without introducing any new commit, ref, or file-copy machinery — it is a pure
  consumer of data ADR-0025 already computes for a different purpose.
- No new collision surface, no new ownership question, no new reconciliation step
  at merge time: because nothing is ever duplicated, there is nothing to
  reconcile. The reference resolves for real, automatically, the moment the source
  branch's content becomes reachable from the referencing branch.
- `aiwf show`/`aiwf list` become genuinely useful across a multi-worktree session
  without requiring the operator to check out, merge, or otherwise touch the other
  branch.

**Negative:**

- `refs-resolve` and `body-prose-id` both gain a third classification tier
  (resolved / cross-branch-pending / unresolved) where today there are two. This
  is a direct, deliberate widening of ADR-0025's scope, which held the cross-branch
  view to allocation only — this ADR is the recorded decision to widen it to a
  second consumer.
- The escalation path is new surface area that must be actively maintained: any
  future change to how the cross-branch view is computed (e.g. narrowing what
  counts as a "local" ref) must be checked against whether it silently breaks
  escalation for entities currently classified `cross-branch-pending`.
- A `cross-branch-pending` reference cannot be mutated (promoted, edited) from the
  referencing branch — by design, per the ownership rule above — so this decision
  only ever helps the "reference it" case, never the "also resolve/close it from
  here" case. That remains gated on the entity's physical presence, unchanged.
- The read-side lookup in `aiwf show`/`aiwf list` adds a second data path (local
  tree vs. cross-branch blob read) that render/list code must keep distinguishable
  to the operator — a result sourced from another branch must be labeled as such,
  not presented indistinguishably from a locally-resolved entity.

## Validation (optional)

The decision holds as long as: (1) a fixture-driven test proves the escalation
path fires (pending → unresolved) whenever the source branch disappears from the
cross-branch view, run on every CI pass, and (2) no mutating verb is ever extended
to accept a `cross-branch-pending` target — if one ever needs to, that is a new
decision, not a quiet extension of this one.

## References

- Related ADRs: `ADR-0025` (the cross-branch view this decision extends),
  `ADR-0001` (mint-at-trunk-integration; proposed, orthogonal — that ADR concerns
  id-minting timing, this one concerns referencing already-stably-minted ids)
