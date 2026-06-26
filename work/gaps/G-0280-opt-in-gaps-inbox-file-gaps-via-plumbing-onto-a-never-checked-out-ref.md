---
id: G-0280
title: 'Opt-in gaps inbox: file gaps via plumbing onto a never-checked-out ref'
status: open
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

## Provenance

Emerged from a design discussion (2026-06-26) prioritizing the operator's
frequent "collision fear" when filing gaps. Sibling to the `G-0272` / `G-0273` /
`G-0274` collision cluster; this is the gaps-specific structural option those
three alternatives left open. Filed via add + reallocate through a live
sibling-worktree collision — itself evidence for the priority.
