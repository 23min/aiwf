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

Build the write-scope precondition at the seam M-0282 chose, and route every
single-entity frontmatter-writing verb through it, so a hand-edited field can no
longer ride into another verb's commit unreported.

## Context

M-0282 settled where the precondition runs, what it compares, whether it refuses
or reports, and whether an escape hatch exists. This milestone is the first that
writes code against those answers.

The routing is not a one-line insertion. `planEntityWrite` is the nearest thing
to a shared single-entity write seam, and it covers seven call sites across six
files — `ac.go`, `cancel.go`, `milestone_depends_on.go`, `milestone_tdd.go` and
`retitle.go`. The rest of the routes in the epic's scope list build their plans
directly. Roughly half the work is bringing those routes to a seam that does not
yet cover them, which is why this milestone is sized above the ADR that precedes
it.

## Approach

Build the guard once, at the chosen seam, with `checkStagedConflict`'s message
shape — it already refuses this exact operator sequence for the *staged* case,
naming the overlapping path and pointing at `git restore --staged` or
`git stash`. The unstaged case is the open hole, so the operator meets one
message for one condition rather than two differently-shaped refusals.

Routes then move onto the seam one at a time, each with its own test, rather than
in one sweep — the route list is long and the failure mode of a bulk move is a
route that compiles onto the seam without ever being exercised through it.

## Acceptance criteria

### AC-1 — Unstaged HEAD-divergent content is never committed silently

A structured-state verb run against an entity whose frontmatter differs from HEAD
in the unstaged working copy does not commit that difference without saying so.
Whether the outcome is a refusal or a report is M-0282's third decision; what
this AC pins is that the silence is gone.

The reproduction E-0075 records is the shape to test: a gap's priority set to
`high` through `aiwf set-priority`, hand-edited to `low`, then a different verb
run — after which `aiwf history` still names `set-priority high` as the last
priority act while the file reads `low`.

### AC-2 — The measured priority-through-retitle laundering no longer succeeds

The specific route E-0075 measured: `aiwf retitle` no longer carries
`-priority: high / +priority: low` into a commit trailered `aiwf-verb: retitle`.

`retitle` is worth its own AC rather than being folded into AC-4 because it sits
in both mechanisms at once — it builds an `OpMove` and an `OpWrite`, so it is
both a serializing route and a move-shaped one. A guard that covers it covers the
overlap.

### AC-3 — A verb over a dirty disk never commits a tree identical to its parent

The two failure directions of a loaded-only comparison, both of which E-0075
measures and neither of which its success criteria currently assert:

- Asking for the value already on the dirty disk reports "already set; nothing to
  change" while HEAD says otherwise — false about the record, and the operator's
  requested mutation is dropped.
- Asking for HEAD's value commits a tree byte-identical to its parent — an
  empty-diff commit of exactly the class M-0281 existed to eliminate.

This AC is what makes M-0282's first decision consequential rather than academic:
a guard sited at `verb.Apply` reaches the second case but not the first, because
the same-state guard returns from the verb body before any plan exists. If the
ADR chose that seam, this AC is where the consequence has to be recorded rather
than discovered.

### AC-4 — The spike matrix answers every question ADR-0038 defers

Every route named in E-0075's *Scope* section under single-entity field writes
either passes through the precondition or carries a recorded reason for being
exempt. Stated as a reference to the epic's list rather than reproduced here, so
the two cannot drift.

The routes deliberately excluded there — `authorize`, `acknowledge illegal`,
`acknowledge mistag`, and `promote` / `cancel --audit-only` — write no files at
all, so there is nothing for a write-scope guard to compare. They are out by
construction, not by exemption.

### AC-5 — The measured nested laundering through a parent rename no longer succeeds

### AC-6 — edit-body bless mode still commits a working-copy edit

## Constraints

- `checkStagedConflict` is the precedent for the message, not just the position.
  A second, differently-shaped refusal for the unstaged case leaves the operator
  with two messages for one condition.
- The comparison degrades when there is no HEAD version. `edit-body` already
  holds both answers — bless mode refuses a never-committed entity, explicit mode
  proceeds — so the raw material sits in one file.
- Nothing here may be read as fixing the FSM walker's rename-plus-status blind
  spot. The precondition incidentally masks it; the rule stays wrong for any
  other route to such a commit, and G-0475 stays open.
- Every route moved onto the seam is exercised *through* the seam by a test.
  Compiling onto a shared helper is not evidence that the route reaches it.

## Design notes

- M-0282's four decisions are pre-locked inputs. If implementation surfaces a
  reason one of them is wrong, that is an ADR amendment, not a local judgment
  call made inside this milestone.
- The nested-path vector is explicitly *not* this milestone's problem. It belongs
  to M-0284, and the split exists so a long single-entity route list does not
  bury the vector that defeats a blocking check.

## Out of scope

- Multi-entity sweeps and the nested-path case — M-0284.
- The `internal/policies/` invariant that keeps new routes on the seam — M-0285.
- After-the-fact detection of laundering already in history (G-0480).

## Dependencies

- M-0282 — the ADR settling the seam, scope, verdict and escape hatch.

## References

- E-0075 — the parent epic, whose *Scope* section AC-4 references
- G-0466 — a verb commits frontmatter it does not own
- G-0463 — the `edit-body --body-file` instance
- M-0281 — the same-state convergence work whose empty-diff class AC-3 protects
- `internal/verb/apply.go` — `checkStagedConflict`, the precedent
- `internal/verb/common.go` — `planEntityWrite`, the partial existing seam

