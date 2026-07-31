---
id: M-0284
title: Land the claim-side precondition and record each sweep's call
status: draft
parent: E-0075
depends_on:
    - M-0283
tdd: required
acs:
    - id: AC-1
      title: A no-change claim is never made against HEAD-divergent state
      status: open
    - id: AC-2
      title: Every NoOp site compares at the scope its own claim asserts
      status: open
    - id: AC-3
      title: Each multi-entity sweep carries a recorded in-or-out call the guard matches
      status: open
---

## Goal

Close the nested-path vector — the one that defeats a blocking check — and give
each multi-entity sweep an explicit in-or-out call rather than letting it inherit
the single-entity answer.

## Context

M-0283 put the precondition on the single-entity routes. Two classes remain, and
both are ones a guard keyed on the verb's named target cannot see.

The nested case is the worse of the two. `gatherCommitOps` walks a moved
directory's destination recursively and commits whatever is on disk for every
file inside, so moving an epic directory commits every nested entity's on-disk
bytes. No verb names those entities. E-0075 records the reproduction: `tdd: none`
hand-edited to `tdd: required` on a milestone — the field that decides whether
`acs-tdd-audit` fires at all — then `aiwf rename` run on the parent epic. The
change landed in a commit trailered for the epic, and `aiwf history` on the
milestone shows no event for it.

The sweeps are the second class. Each writes frontmatter across many entities in
one commit, and each has a different reason it might be exempt, so none of them
inherits the single-entity answer.

## Approach

The nested case first, since it is the vector that lets an illegal FSM transition
through. `checkStagedConflict` already solves the staged half by prefix-matching
staged paths against an `OpMove`'s source and destination rather than walking the
filesystem — the unstaged guard follows that shape rather than inventing a second
one.

Then each sweep in turn, each getting a recorded call. Under the
committed-path scope M-0282 may have chosen, a sweep's paths *are* the commit's
paths and the call is "in by construction, here is the test that proves it";
under entity scope each is genuine enumeration work. The AC is phrased to hold
either way.

## Acceptance criteria

### AC-1 — A no-change claim is never made against HEAD-divergent state

The reproduction E-0075 records no longer succeeds silently: `tdd:` hand-edited
on a milestone, then `aiwf rename` on the parent epic, no longer produces a
commit that attributes the change to the epic while `aiwf history` on the
milestone shows no event for it.

This is the epic's worst vector because it defeats two rules at once. The
provenance audit skips any commit carrying a non-empty `aiwf-verb:` trailer and
reads changed *paths* rather than frontmatter, so which fields moved is invisible
to it. And the FSM history walker deliberately skips a commit that both renames
and changes status, on the stated reasoning that pure renames don't change
status — a premise this defect falsifies.

### AC-2 — Every NoOp site compares at the scope its own claim asserts

The vector belongs to any `OpMove` whose source is a directory — that is, any
epic or contract entity moved by `rename`, `retitle`, `reallocate`, `archive` or
`rewidth --apply`. All five emit `OpMove` and all five reach the same recursive
walk in `gatherCommitOps`.

E-0075's *Scope* names four of these under nested paths and omits `retitle`,
though its *Context* separately notes that `retitle` builds both an `OpMove` and
an `OpWrite`. `retitle` on an epic moves the epic's directory, so it carries the
vector identically. Coverage here follows the code rather than the epic's list.

### AC-3 — Each multi-entity sweep carries a recorded in-or-out call the guard matches

Each of `rename-area`, `rewidth --apply`, `import --on-collision update` and
`archive` carries an explicit recorded decision about whether the precondition
applies to it, and the guard's actual behavior matches that decision.

Phrased as "a recorded call the guard matches" rather than "the sweeps are
covered" because the correct answer differs per sweep and depends on M-0282's
scope decision. What this AC forbids is a sweep whose treatment is accidental —
covered because it happened to route through the seam, or exempt because nobody
looked.

The reasons differ and are worth stating: `rename-area` writes `area:` across
every tagged entity plus `aiwf.yaml`; `rewidth --apply` rewrites `id:` tree-wide;
`import --on-collision update` rewrites existing entities from outside the tree;
`archive` moves files without changing their content. The last is the one most
likely to be exempt and the one whose exemption most needs writing down, since it
also emits directory moves and so appears in AC-2.

## Constraints

- A guard keyed on the verb's named target cannot see nested entities. Whatever
  lands has to reach paths no verb names.
- Coverage follows the code, not E-0075's route list, wherever the two disagree.
  The `retitle` omission above is one known instance; a second found during
  implementation is handled the same way and noted.
- No sweep inherits the single-entity answer by default. "Exempt because nobody
  looked" fails AC-3 exactly as an unconsidered inclusion does.
- The precondition incidentally masks the FSM walker's blind spot. That is not a
  fix, and G-0475 stays open on its own terms.

## Design notes

- M-0282's scope decision determines whether AC-3 is discovery or confirmation.
  Under committed-path scope a sweep's paths are the commit's paths, so each call
  is "in, by construction" plus a test; under entity scope each sweep needs
  individual enumeration. The AC holds under both, and the milestone is sized for
  the more expensive branch.
- `rewidth` appears in both AC-2 and AC-3, and its retirement is proposed under
  G-0481. Nothing is decided there, so it is treated as live here; if it is
  retired before this milestone runs, its in-or-out call is answered by
  retirement rather than by the guard, and AC-3 records that as the reason.
- `archive` is the interesting sweep: it moves files without rewriting their
  content, so the laundering it can carry is whatever was already hand-edited on
  disk rather than anything the verb itself computes. Whether that counts as the
  verb committing frontmatter it does not own is a genuine call, not a formality.

## Out of scope

- The single-entity routes — M-0283.
- The `internal/policies/` invariant — M-0285.
- After-the-fact detection of laundering already in history (G-0480), which is
  the only backstop for this vector if M-0282 chose entity scope.

## Dependencies

- M-0283 — the precondition must exist before these routes reach it.
- M-0282 — its scope decision determines the shape of AC-3.

## References

- E-0075 — the parent epic; its *Scope* section is the route list AC-2 corrects
- G-0466 — a verb commits frontmatter it does not own
- G-0475 — the FSM walker's rename-plus-status blind spot
- G-0480 — after-the-fact detection; the backstop if the guard is entity-scoped
- `internal/verb/apply.go` — `gatherCommitOps`, the recursive walk; `stagedPathConflicts`, the prefix-matching precedent
- `internal/check/fsm_history_walker.go` — the walker whose premise the reproduction falsifies

