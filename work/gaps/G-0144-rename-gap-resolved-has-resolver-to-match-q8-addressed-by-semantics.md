---
id: G-0144
title: Rename gap-resolved-has-resolver to match Q8 addressed-by semantics
status: addressed
discovered_in: M-0123
addressed_by:
    - M-0142
---
## What's missing

The check finding code `gap-resolved-has-resolver` (impl in
`internal/check/check.go`, hint in `internal/check/hint.go`) was named when
the gap FSM used "resolved" as the addressed terminal. The current FSM
(`entity.transitions[KindGap]`) uses `addressed` and `wontfix` as terminals.
The Q8 question (`docs/pocv3/design/legal-workflows-first-principles.md`
line 475-485) ratified the addressed-by-requirement rule; the spec table's
gap-FSM cells reference the existing `gap-resolved-has-resolver` code by
its current name, but the name no longer matches the FSM vocabulary.

## Why it matters

Reader confusion is the load-bearing cost. A contributor reading
`gap-resolved-has-resolver` wonders whether the gap FSM has a `resolved`
state they missed. A reader of `aiwf check` output sees the finding fire
on a gap moving to `addressed` and has to mentally translate.

The code rename is mechanical; the spec table's `ExpectedErrorCode:
"gap-resolved-has-resolver"` line and the hint table key both need to
move atomically.

## Proposed fix shape

- Rename the constant in `internal/check/check.go` (or wherever the
  string literal lives) from `gap-resolved-has-resolver` to
  `gap-addressed-has-resolver` (or similar — to be settled with the
  Q8 rename note).
- Update the hint key in `internal/check/hint.go`.
- Update the spec cell's ExpectedErrorCode in
  `internal/workflows/spec/rules.go`.
- Update any test fixtures that string-match the old code.
- Update the renderer hint text to match.

## Pre-decision

A small ADR or D-NNNN may be warranted before the rename — older `aiwf
check` JSON output ingested by downstream tools could break on the code
change. Lean: file a D-NNNN with the rename + a one-line "downstream
tools may need to refresh" caveat, then do the rename in one commit.
