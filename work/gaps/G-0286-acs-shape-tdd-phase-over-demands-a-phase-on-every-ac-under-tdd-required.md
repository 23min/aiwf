---
id: G-0286
title: acs-shape tdd-phase over-demands a phase on every AC under tdd required
status: open
---
## Problem

`internal/check/acs.go` (~line 153, `ac.TDDPhase == "" && tddRequired`) raises
`acs-shape/tdd-phase` on **every** acceptance criterion that lacks a `tdd_phase`
when its milestone is `tdd: required` — regardless of the AC's status. The phase
enum is `red|green|refactor|done`, with no "not started" member, so an AC that has
been added but not yet worked has no honest value to carry. Strengthening a
milestone `advisory → required` therefore reddens the tree for every pre-existing
AC, and the only way to clear it is to seed untouched ACs to `red` — which
misrepresents them (`red` means "a failing test exists").

## Why it is a bug, not just friction

The design commits (CLAUDE.md #8 / design-decisions §ACs) only to **"AC `met`
requires `tdd_phase: done`" when the milestone is `tdd: required`** — not to
"every AC carries a phase." So the rule is *stricter than the commitment*: it
enforces phase-presence on ACs the commitment never required it for. The
load-bearing integrity gate is met → done; a phase on an unstarted AC adds no
integrity, only friction.

## Proposed fix (one design fork to settle first)

Relax `acs-shape/tdd-phase` so an **absent** phase is legal until the AC reaches
`met` (absent = "not started"); keep the existing "present-but-not-in-the-closed-
set" error and the met → done audit untouched. Net: `advisory → required` becomes
non-disruptive, and the companion set-policy verb (tracked in G-0168) no longer
needs an auto-seed-vs-refuse decision.

The fork to settle: does `tdd: required` mean **"every AC is phase-tracked from
creation"** (the strict reading — start each AC at `red` by writing the failing
test first) or **"every AC reaches done before met"** (the committed reading)?
Only the strict reading justifies the current behaviour; the design doc commits to
the latter. If the strict reading is kept deliberately, this gap closes as
"working as intended" and the burden moves entirely to the verb (refuse-with-hint,
never auto-seed).

## Relationship

- **G-0168** holds the missing set-policy verb (the `tdd:` row plus the
  2026-06-26 re-discovery). This gap is the *check-layer* half: independently
  actionable — relaxing the rule helps even before any verb ships — and a
  kernel-rule change, so it is tracked separately rather than buried in the verb
  umbrella.

## Provenance

Surfaced 2026-06-26 while strengthening a milestone `advisory → required` and
finding the upgrade reddened the tree via this rule. Split from the G-0168 fold as
the architecturally-distinct check-layer concern.
