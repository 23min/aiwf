---
id: M-0283
title: Spike the guard's mechanics and land the commit-side seam
status: in_progress
parent: E-0075
depends_on:
    - M-0282
tdd: required
acs:
    - id: AC-1
      title: Unstaged HEAD-divergent content is never committed silently
      status: open
      tdd_phase: green
    - id: AC-2
      title: The measured priority-through-retitle laundering no longer succeeds
      status: open
      tdd_phase: green
    - id: AC-3
      title: A verb over a dirty disk never commits a tree identical to its parent
      status: open
      tdd_phase: red
    - id: AC-4
      title: Every verb entry point has a stated guard decision or a reasoned exemption
      status: open
    - id: AC-5
      title: The measured nested laundering through a parent rename no longer succeeds
      status: open
      tdd_phase: red
    - id: AC-6
      title: edit-body bless mode still commits a working-copy edit
      status: open
      tdd_phase: red
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

**The prototype leads.** Every defect found while settling this design was
visible only by running code, and none was found by classifying it. The
classification below exists to drive the prototype — a checklist of what to try —
not to discover anything on its own.

Build the guard in a scratch copy at the top of `verb.Apply`, before Phase 1,
where disk still holds the operator's state. Two inputs decide each path.

**One bit, because that is all the instrument yields.** The guard's dirty set
comes from `gitops.DirtyPaths` — `git diff --name-only HEAD` unioned with
`git ls-files --others --exclude-standard` — and staged paths are already refused
by `checkStagedConflict` earlier in the same function. Between them those two
queries distinguish three classes of path, not the eight a HEAD/index/disk
tri-state would suggest, so the decidable question per path is simply: *is it
dirty?* Anything finer is a distinction the tooling erases.

**Five roles, because that is what actually discriminates.** What separates the
measured defects is not a path's git state but its role in the plan:

| role | derived from | the decision |
|---|---|---|
| named write | `OpWrite.Path` | refuse if dirty, unless the op declares it adopts the working copy |
| move source | `OpMove.Path` | refuse if dirty |
| move destination | `OpMove.NewPath` | absent by construction, except a flat-file move onto an existing path |
| nested under a move | prefix of `OpMove.Path` / `NewPath` | the open question — no verb named it |
| not in the plan | — | nothing; `Apply` never touches it |

Roles crossed with one bit is five to eight real decisions, all reachable. The
two-axis alternative was measured at roughly one part in seven load-bearing, and
it could not express the nested-milestone defect as a decision distinct from the
hand-edited-body one, because both land in the same cell with different answers.

**Then the second layer, which is where the unsettled questions live.** Per-path
verdicts have to compose into one plan-level outcome — refuse if any path is
dirty, or only for paths whose content the verb computed. That is a rule, not a
table, and carry-along substitution is one candidate answer to it.

Drive the prototype across the role/bit grid for each verb class, record what
happens, decide from the results, then implement test-first and discard it.

**Three limits are known before the prototype starts, and the guard must state
them rather than imply completeness.**

- **A dirty path can be invisible.** `assume-unchanged`, `skip-worktree` and
  sparse checkout make both of `DirtyPaths`' queries answer "clean" for a path
  whose disk content differs. Measured, a nested milestone's `tdd:` laundered
  through an epic rename that way. G-0487.
- **A clean path can already be corrupt, and the guard then makes it worse.** A
  directory move flattens symlinks and forces mode `100644`, after which those
  paths read as modified forever — so this guard would refuse every later verb on
  that directory with no aiwf-side recovery, while a bare `chmod +x` would be
  refused although it can launder nothing. G-0486.
- **Some unowned writes involve no divergence at all.** The loader normalizes as
  it reads, and the next write-verb commits that normalization under its own
  trailer with disk and HEAD in agreement throughout. G-0488.

Those three also correct the framing this milestone inherited. The root cause is
not that verbs compare against a projection of disk; it is that a verb rewrites a
**whole file re-serialized from a lossy in-memory model** rather than editing the
fields it owns. HEAD-divergence is one way that goes wrong. The guard addresses
that one, and the surgical-commit approach ADR-0038 defers is the shape that
would address the rest.

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

### AC-4 — Every verb entry point has a stated guard decision or a reasoned exemption

The mechanical half is coverage of the axis that drifts. Every exported
`(*Result, error)` entry point under `internal/verb`, plus the unexported
composite branches an AST scan of exported functions cannot see, either has a
recorded decision about how the guard treats it or a reviewed allowlist entry
giving its specific reason. The assertion is derived from the source rather than
from a hand-authored list, in the shape `verb_result_noop_invariant.go` already
uses — so adding a verb without deciding its treatment fails, which is the only
drift a policy can actually catch.

State its reach honestly: it proves every route is *named*. It cannot prove an
answer is correct, or that it was measured rather than reasoned. Those are read
at the wrap review, which is where this project puts judgment a check cannot
carry. Claiming more would repeat the mistake M-0282 recorded — a chokepoint that
reads as enforcing and does not.

The non-mechanical half is the milestone's actual output: each question ADR-0038
defers is answered from the prototype rather than by argument — how the compared
path set is derived, how per-path verdicts compose into a plan-level outcome,
whether carry-along substitution is adopted and under what corrections, what the
`ExitUsage` change costs across `cliutil`, and which existing tests break. An
answer with no measurement behind it is not an answer, and the wrap review is
where that is judged.

Two answers the prototype is expected to produce, flagged so they are not
mistaken for new scope. The sweeps' claim scope: `archive` and `rewidth` return
their NoOp precisely when the selected set is *empty*, so scoping their guard to
that selection would give it nothing to look at. And `import`: its NoOp is
constructed outside `internal/verb` on a different type, so no site inventory
contains it, yet it is in E-0075's scope and needs a decision like any other
route.

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
