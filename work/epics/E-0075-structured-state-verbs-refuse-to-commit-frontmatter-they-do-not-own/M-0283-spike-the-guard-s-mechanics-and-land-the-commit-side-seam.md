---
id: M-0283
title: Spike the guard's mechanics and land the commit-side seam
status: draft
parent: E-0075
depends_on:
    - M-0282
tdd: required
acs:
    - id: AC-1
      title: Unstaged HEAD-divergent content is never committed silently
      status: open
    - id: AC-2
      title: The measured priority-through-retitle laundering no longer succeeds
      status: open
    - id: AC-3
      title: A verb over a dirty disk never commits a tree identical to its parent
      status: open
    - id: AC-4
      title: The spike matrix answers every question ADR-0038 defers
      status: open
    - id: AC-5
      title: The measured nested laundering through a parent rename no longer succeeds
      status: open
    - id: AC-6
      title: edit-body bless mode still commits a working-copy edit
      status: open
---

## Goal

Settle the guard's remaining mechanics by building a throwaway prototype, then
land the commit-side guard at `verb.Apply` on what the prototype proved.

## Context

ADR-0038 settles the decisions answerable by reading the code: two seams both
inside `internal/verb`, refuse rather than warn, whole-file scope at both, no
`--force` and no repair verb, and a structural exemption for `edit-body` bless
mode. It deliberately defers the mechanics, because every defect found while
drafting it was an implementation discovery rather than a reading one — that a
comparison at `gatherCommitOps` reads the verb's own freshly-written bytes, that
a carry-along substitution duplicates subtrees and drops a milestone from a
`rewidth` commit, that a flat-file move's destination falls outside a
nested-only taxonomy. None of those is visible in prose.

So this milestone inverts the usual order: prototype first, decide from
measurement, then implement under TDD. The prototype is throwaway and never
committed; only the matrix, the answers, and the final implementation land.

## Approach

Build the guard in a scratch copy at the top of `verb.Apply` — before Phase 1,
where disk still holds the operator's state — deriving the compared path set
from `p.Ops` by prefix and the dirty set from `gitops.DirtyPaths`. Both
primitives already exist, and `DirtyPaths` is already called from
`internal/verb`.

Drive a scenario matrix over that prototype: each verb class against each tree
state, every cell a measurement. Verb classes: serializing (`set-priority`,
`promote`), move-plus-write (`retitle`), pure move (`rename`), file move
(`move`), `reallocate`, sweep (`archive`, `rewidth --apply`), config
(`rename-area`, `contract bind`), `edit-body` in both modes, and the empty-plan
verbs (`authorize`, `--audit-only`, `acknowledge`). Tree states: clean, dirty
target, dirty nested, dirty `aiwf.yaml`, untracked stray, gitignored stray,
staged, and HEAD-missing.

The matrix answers the deferred questions as data rather than argument. Then the
implementation lands test-first, and the prototype is discarded.

## Acceptance criteria

### AC-1 — Unstaged HEAD-divergent content is never committed silently

A verb run against a path whose working-copy content differs from HEAD does not
commit that difference without saying so. Whole-file, not frontmatter-only: the
measured defect covers an unblessed body edit as well as a hand-edited field.

The reproduction to pin: a gap's body rewritten in the working tree, then
`aiwf set-priority` run, after which the commit carries both the priority change
and the body rewrite while `aiwf history` shows no `edit-body` event.

### AC-2 — The measured priority-through-retitle laundering no longer succeeds

`aiwf retitle` no longer carries `-priority: high / +priority: low` into a commit
trailered `aiwf-verb: retitle`.

`retitle` earns its own criterion because it sits in both mechanisms at once — it
builds an `OpMove` and an `OpWrite`, so it is a serializing route and a
move-shaped one. A guard that covers it covers the overlap.

### AC-3 — A verb over a dirty disk never commits a tree identical to its parent

The empty-diff direction of a loaded-only comparison: asking for HEAD's value
while the working copy diverges commits a tree byte-identical to its parent — the
class M-0281 existed to eliminate.

The other direction — a false "already set; nothing to change" that drops the
operator's mutation — is claim-side and belongs to M-0284. Both are real; they
are separated because they are caught at different seams.

### AC-4 — The spike matrix answers every question ADR-0038 defers

Each question the ADR records as deferred has an answer backed by a matrix cell
rather than by argument: how the compared path set is derived, whether
carry-along substitution is adopted and under what corrections, what the
`ExitUsage` change costs across `cliutil`, and which existing tests break.

The matrix itself is the deliverable — every verb class against every tree state,
with the observed behaviour recorded. A question answered without a cell behind
it does not satisfy this criterion.

Where an answer contradicts an ADR-0038 decision rather than refining it, the ADR
is superseded rather than quietly rewritten.

### AC-5 — The measured nested laundering through a parent rename no longer succeeds

`tdd:` hand-edited on a milestone, then `aiwf rename` on the parent epic, no
longer produces a commit that attributes the change to the epic while
`aiwf history` on the milestone shows no event for it.

The nested vector is commit-side, which is why it sits here rather than with the
claim-side work. It is also the vector that defeats a blocking check, since the
FSM history walker skips a commit that both renames and changes status.

Coverage follows the code rather than any prose route list: the vector belongs to
every `OpMove` whose source is a directory, and to a file move's own destination.
A taxonomy scoped to "nested under a move" was measured to leave flat-file
renames laundering freely.

### AC-6 — edit-body bless mode still commits a working-copy edit

Bless mode's precondition is that the working copy diverges from HEAD, so a guard
refusing divergence would block the one verb whose job is to commit it — and
would make the recovery ADR-0038 recommends unreachable.

The exemption is verified, not merely declared: the guard asserts the exempted
write's content equals the bytes on disk, so it can smuggle nothing else. Bless
mode already refuses any frontmatter divergence of its own accord, which is what
keeps the exemption from becoming a laundering route.

## Constraints

- The prototype is throwaway. It is never committed, and no implementation commit
  may depend on it existing.
- The matrix is measured, not predicted. A cell filled by reasoning is not a
  measurement, and the numbers this milestone reports are its own.
- `checkStagedConflict` is the precedent for the message and the position, so the
  operator meets one message shape for one condition rather than two.
- Every route the guard covers is exercised *through* the guard by a test.
  Compiling onto a shared helper is not evidence that a route reaches it.
- Nothing here may be read as fixing the FSM walker's rename-plus-status blind
  spot. The precondition incidentally masks it; G-0475 stays open on its own
  terms.

## Design notes

- The comparison must name the *pre-mutation* working copy. By the time
  `gatherCommitOps` runs, Phase 1 moves and Phase 2 writes have already mutated
  disk, so a comparison there reads the verb's own bytes — measured to refuse
  every content-writing verb on a clean tree while skipping every move-shaped
  one.
- A pre-mutation guard prefix-matches rather than predicts. `stagedPathConflicts`
  already derives the nested set that way for the staged twin, with a comment
  recording why it is equivalent to walking the filesystem.
- Carry-along substitution — taking HEAD's blob for a path no verb named — is a
  candidate refinement, not a settled decision. Prototyped, it duplicated
  subtrees under `retitle` / `move` / `reallocate`, dropped a milestone from a
  `rewidth --apply` commit, and left flat-file renames uncovered; it also
  interacts with link-rewrite `OpWrite`s in a way that makes two sibling
  milestones behave oppositely. Adopt it only with those resolved, and record the
  reasoning either way.
- `ExitUsage` for the precondition needs a change in `internal/cli/cliutil`,
  which maps every `Apply` error to `ExitInternal`. That sits outside
  `internal/verb` and has to be scoped rather than assumed.

## Out of scope

- The claim-side precondition and the 22 NoOp sites — M-0284.
- Each multi-entity sweep's recorded in-or-out call — M-0284.
- The `internal/policies/` invariant keeping new routes on the seam — M-0285.
- After-the-fact detection of laundering already in history (G-0480).

## Dependencies

- M-0282 — ADR-0038, which settles the decisions this milestone implements and
  names the questions it answers.

## References

- E-0075 — the parent epic
- ADR-0038 — the decisions settled, and the mechanics deferred here
- G-0466 — a verb commits frontmatter it does not own
- G-0463 — the `edit-body --body-file` instance
- G-0475 — the FSM walker blind spot this must not be read as fixing
- M-0281 — the same-state convergence work whose empty-diff class AC-3 protects
- `internal/verb/apply.go` — `checkStagedConflict` and `stagedPathConflicts`
- `internal/gitops/gitops.go` — `DirtyPaths`, the dirty-set primitive
