---
id: G-0394
title: promote epic to done lacks a non-terminal-children guard; archive strands it
status: addressed
discovered_in: E-0063
addressed_by_commit:
    - dcd5b012
---
## What's missing

`aiwf promote <epic> done` has no guard against the epic owning non-terminal
milestone children. `Promote` (`internal/verb/promote.go`) validates the FSM
transition (`active -> done` is legal), resolver flags, and sovereign-act rules,
then runs a projection-findings gate — but the non-terminal-children refusal
lives only in `Cancel` (`promote.go`, the `EpicCancelNonTerminalChildrenError`
branch), never on the `done` path, and no check rule flags "done epic with
non-terminal children," so the projection gate has nothing to trip on. The
epic reaches `done` with an in-progress or draft milestone underneath it.

`aiwf archive` then sweeps that terminal epic as a whole subtree: `computeArchiveMoves`
(`internal/verb/archive.go`) checks only that the epic is terminal and moves the
entire epic directory, by design — "the milestone's own status is incidental to
that move." The non-terminal milestone rides into `archive/` alongside its parent.

Now `aiwf check` fires `archived-entity-not-terminal` at error severity
(`internal/check/archive_rules.go`, `SeverityError`): the milestone lives under
`archive/` but its status is not terminal. A sequence of individually-legal verb
calls composes into a tree the kernel rejects, and no verb refused at the point
of the mistake.

## Why it matters

Two costs stack. First, the promote itself is already a semantic inconsistency:
a `done` epic that owns an in-progress milestone is incoherent the instant the
promote lands — archive only makes it mechanically detectable by relocating the
child. Second, the resulting `archived-entity-not-terminal` finding is a pre-push
error, so it blocks the next push, and it surfaces during a later bulk archive
sweep — far from the promote that caused it. Recovery is awkward because archive
has no reverse (ADR-0004 §Reversal): clearing the error means retroactively
promoting or cancelling a milestone that already sits in `archive/`, and a promote
to `done` there may be blocked by open ACs, forcing the cancel path to dispose of
something the tree already treats as shelved.

The root cause is a guard asymmetry. `cancel` gained its non-terminal-children and
open-AC guards deliberately (per D-0003 / D-0004). The symmetric precondition on
`promote`-to-terminal was never added — so `promote` is a backdoor around the
guards the dedicated verb enforces. `archive`'s "status is incidental" reasoning
is correct only *given* the invariant that a terminal epic has terminal children —
an invariant ADR-0004 assumed but nothing enforces. The `aiwfx-wrap-epic` ritual
does verify all milestones are done, but that is advisory LLM behavior, not a
kernel guarantee: "a guarantee that depends on the LLM remembering to invoke a
skill is not a guarantee."

## Direction

Fix both layers (defense-in-depth), primary first:

- **(A) Promote-time guard — the primary fix.** Mirror `cancel`'s
  non-terminal-children refusal onto the epic `-> done` transition: refuse with a
  listing of the offending milestone ids, no auto-cascade, the operator disposes
  each child first. This is the earliest and most informative catch, it is
  semantically correct independent of archive, and it makes ADR-0004's assumption
  actually hold, so archive's incidental-status reasoning becomes true rather than
  lucky. The clean framing is that every parent-to-terminal promotion (`done` and
  `cancelled` alike) should honor the same child / AC preconditions `cancel`
  already enforces — which is why this should be scoped together with G-0335
  (promote-to-cancelled bypasses the same class of guard for the milestone / AC
  case). Fixing the shared precondition once in `Promote` closes both.

- **(B) Archive-time guard — defense-in-depth.** Refuse to sweep (or skip) an epic
  whose subtree contains a non-terminal entity, with a message pointing at the
  offending child. (A) blocks the happy path but cannot make the bad state
  impossible — `--force` or a raw edit can still land a `done` epic with an open
  child — so archive should independently decline to strand one. This matches the
  project stance that the verb-time guard is the chokepoint and `--force` is a
  sovereign, audited override.

## Scope

The `Promote` epic-`done` guard (reusing `cancel`'s existing children-enumeration
and typed error), the archive-time subtree terminality check, and tests for each:
a fixture where `promote <epic> done` with a non-terminal child is refused; a
fixture where archive refuses / skips an epic whose subtree carries a non-terminal
entity; and confirmation that the `--force` bypass still reaches archive so (B) is
the layer that catches it. Coordinate with G-0335 so the shared parent-to-terminal
precondition is implemented once rather than twice.

## Provenance

Surfaced while analysing `aiwf archive`'s subtree-move behaviour during E-0063
planning. Sibling of G-0335 (open) and G-0334 (open); descends from the `cancel`
guard lineage delivered under D-0003 / D-0004.
