---
id: G-0281
title: 'Opt-in gaps inbox: file gaps via plumbing onto a never-checked-out ref'
status: open
prior_ids:
    - G-0280
priority: low
---
## Problem

Filing a gap mid-flight carries "collision fear": on a feature/epic worktree (or
a second session) `aiwf add gap` allocates `max(working tree ∪ origin/main)+1`,
blind to sibling worktrees and to unpushed trunk. Two contexts that both observe
the same `max` while neither has published produce the same id; it surfaces later
as an `ids-unique` finding cured by `aiwf reallocate`. The cure is cheap, but the
*fear* is constant — the operator files gaps frequently and feels the risk every
time. This gap was itself filed through that exact trap: a sibling worktree had
independently allocated and reallocated the next id, unpushed and invisible to
this tree's allocator.

## Decision

Build a gaps-inbox path as the prioritized response, with these constraints:

- **Opt-in, default-off.** A config knob (e.g. `aiwf.yaml: gaps.inbox`), no CLI
  default change. Dogfood in this repo before enabling elsewhere.
- **Reversible.** Flipping the knob off returns to today's behaviour; no
  irreversible state, no migration.
- **Architecturally significant.** The mechanism departs from two kernel norms
  (commit on the current branch; push as its own gate), so the decision earns an
  ADR and the work an epic when planned.

## Design direction

File the gap onto a dedicated **never-checked-out** gaps ref via git plumbing,
without touching the operator's working tree, index, or HEAD:

1. fetch the gaps ref,
2. allocate the id against it,
3. build the new tree from the parent's **full** tree plus one new blob,
   `commit-tree` it, `update-ref` with a compare-and-swap guard,
4. push (opt-in).

"Never checked out" is load-bearing: it removes the worktree-desync hazard of
plumbing `update-ref` on a ref that is checked out elsewhere, and (notes-style,
on a `refs/aiwf/*`-class ref) makes accidental porcelain writes hard by
construction. The fetch-CAS-push turns the residual cross-machine race into an
immediate non-fast-forward rejection at file time, not a latent merge-time
finding.

The pattern is proven in headless contexts (GitHub's `createCommitOnBranch` with
`expectedHeadOid`, `ghp-import` onto gh-pages, `git notes`, `git stash`, every
server-side receive). The novel cut is exposing it as a local developer CLI.

## Reconciliation

Filing writes only to the inbox ref — nothing lands in `work/gaps/` on any
checked-out branch until a separate, explicit step moves it there. That step
isn't decided yet and must be before M-0187 gets ACs:

- **Materialization is an explicit verb, not automatic.** `aiwf gaps import`
  (naming TBD) fetches the inbox ref, diffs its `work/gaps/` tree against the
  current branch's, and lands any new files as one ordinary commit via the
  M-0186 primitive (parent = current HEAD, blob writes = the new files). No
  auto-triggering — not on `add`, not on push, not from a hook — the operator
  decides when to reconcile. Of the headless precedents cited above, only one
  actually matches this problem's shape: GitHub's `createCommitOnBranch` and
  ordinary server-side `receive-pack` write straight to the real target
  branch (no worktree exists server-side to desync, so no side ref and no
  reconciliation step are needed at all); `ghp-import`/`gh-pages` and `git
  notes` are deliberately *permanent* side channels never meant to rejoin the
  primary tree, so nothing reconciles because reconciliation was never the
  goal. `git stash` is the one built for exactly this: a temporary parking
  ref with a deliberate, named move-back command (`stash pop`/`apply`).
  `aiwf gaps import` is that same shape, applied to `refs/aiwf/gaps` instead
  of `refs/stash`. Because import lands as an ordinary verb-commit, it needs
  no new collision handling — the existing `ids-unique` / `aiwf reallocate`
  cure already covers it once it lands.
- **Read-only visibility can go further than materialization, cheaply.**
  `aiwf status`, `aiwf show`, and `aiwf render --format=html` can each
  optionally peek at the inbox ref (fetch + list, no write) and surface
  pending entries clearly marked "inbox — not yet imported," without
  touching the mutating-verb surface at all: nothing mutable lives there
  until import, so `promote`/`edit-body`/`cancel` never need to understand a
  third storage location. This is a narrower, read-only cousin of "the
  loader unions the inbox," which was considered and rejected for the write
  path — rejected there because it would make every mutating verb reason
  about three possible entity locations instead of two.
- **Fetch-cost tradeoff for peek surfaces.** A live "N pending" count needs a
  fresh `git fetch` of the inbox ref — real network cost on `status`, which
  is otherwise fast and local. Follow the existing `aiwf add --fetch`
  convention: local-ref state by default, an explicit `--fetch` flag to
  force a refresh.

## Risks (to weigh at the ADR / epic)

- **Tree-construction footgun:** build the new tree from the parent's full tree,
  not from the single file, or the commit deletes the rest of the repo; isolate
  from the real index so staged work isn't swept up.
- **Hook bypass:** plumbing commits skip pre-commit / commit-msg; the verb must
  carry the validation `aiwf check` would have (it already constructs shape-valid
  commits), and pre-push still fires on push.
- **Push-inside-a-verb:** network on an otherwise-local verb and a step on the
  "push is its own gate" norm — must be explicit opt-in, never silent.
- **Operator confusion / signing:** a commit appears on a ref with no working-tree
  action; honour signed-commit policy via `commit-tree -S` where required.
- **Retry-on-reject is cheap only while the id stays a leaf.** A losing push
  is an ordinary non-fast-forward rejection, not data loss, and the verb
  should auto-retry (fetch → re-allocate → rebuild → push) a bounded number
  of times before giving up. But "re-allocate" is a pure rename only if
  nothing yet references the provisional id. The moment local work builds on
  it before reconciliation lands — an epic or milestone citing the gap,
  another gap's Kindred concerns section — renumbering also means rewriting
  every one of those references: structurally the same operation `aiwf
  reallocate` already performs for today's merge-time-discovered collision.
  The retry loop should invoke that existing reallocate machinery, not a
  bespoke rename, and should not be sold as free.
- **Deferred push compounds both odds and blast radius.** Allocation reads
  whatever was last fetched; because push is opt-in and not automatic, the
  gap between "I allocated" and "I actually pushed" can be arbitrarily long.
  The longer it is, the more likely someone else pushed a colliding
  allocation in the interim (raising the odds of a collision) *and* the more
  local work has had time to reference the provisional id (raising the cost
  of fixing one, per the point above). Prompt, frequent pushes of the inbox
  ref keep both risks small — an operator-discipline argument for pushing
  often even though the flag makes it optional.

## Relationship to the collision cluster

- This is **gaps-only**. `G-0272` (union allocator with worktree HEADs),
  `G-0273` (fetch-before-allocate), and `G-0274` (batch reallocate) stay — they
  are **kind-general** (milestones, epics, ADRs collide by the same mechanism)
  and remain the floor.
- Collisions are **friction, not correctness**: `aiwf check` catches every
  collision at the push chokepoint regardless of allocation path. So this whole
  effort optimizes file-time *friction*, not a correctness hole. That reframes
  misuse-prevention too: a gaps inbox is an **ergonomic funnel, not a security
  boundary** — a hand-filed gap that is wrong is caught by `check` exactly as
  anywhere, so enforcing verb-only writes would duplicate the existing chokepoint.
  Make the verb the easy path; let `check` stay the authority.

## Why a git ref, not a real allocator service

The fetch → allocate → CAS-update → push cycle is a **compare-and-swap
sequence generator** — the same primitive as `UPDATE seq SET value = value +
1 WHERE value = expected_old_value`, retry on zero-rows-affected — built on
git's own ref semantics instead of a database row. A non-force push already
*is* CAS on a pointer: "update this ref only if it's still at the sha I last
read, else reject." Nothing new is invented here; it's the same primitive
GitHub's `createCommitOnBranch`/`expectedHeadOid` exposes as a first-class
API feature, and the same guarantee every ordinary git push already relies
on. Building it out of a git ref rather than a real external database, a
Redis `INCR`, or a lock service (Zookeeper/etcd) avoids a new infrastructure
dependency, a new credential/access-control surface, and keeps everything
offline until the push step — origin already is the coordination point every
git workflow depends on; this just narrows what's coordinated to one small
dedicated ref instead of a whole branch.

This also explains, not just justifies, why the feature stays **gaps-only
and opt-in, default-off** rather than becoming the allocation mechanism for
every entity kind on day one: every other aiwf verb is fully offline —
commit, promote, edit-body all work with zero network reachability. Routing
gap-filing through the inbox introduces a dependency on reaching the remote
(at least at push time) that nothing else in the kernel has. That's a
deliberate, bounded departure from aiwf's otherwise fully offline, git-native
design; confining it to one entity kind behind a config flag is how the
pattern gets tried without committing the whole kernel to it. Whether it
later generalizes to other entity kinds is a separate decision this gap does
not make.

## Provenance

Emerged from a design discussion (2026-06-26) prioritizing the operator's
frequent "collision fear" when filing gaps. Sibling to the `G-0272` / `G-0273` /
`G-0274` collision cluster; this is the gaps-specific structural option those
three alternatives left open. Filed via add + reallocate through a live
sibling-worktree collision — itself evidence for the priority.
