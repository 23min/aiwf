---
id: M-0192
title: Statusline shows in-flight epics on every branch
status: in_progress
parent: E-0047
depends_on:
    - M-0191
tdd: required
acs:
    - id: AC-1
      title: Epic HUD renders the in-flight epic list on a non-ritual branch
      status: open
      tdd_phase: red
    - id: AC-2
      title: Ritual branch shows only the current epic and its milestone
      status: open
      tdd_phase: red
---
## Deliverable

The epic HUD (`.claude/statusline.sh`, G-0188) becomes **branch-contextual** and
behaviorally tested against the M1 harness:

- **Ritual branch** (`epic/E-*`, `milestone/M-*`): show **only** the current
  epic — the one the branch belongs to — with its status glyph/color, plus its
  milestone inline on a milestone branch. No other in-flight epics; no `+N`
  overflow.
- **Main / non-ritual**: show the in-flight epic list — all non-terminal epics
  with canonical glyph/color (`→` active, `○` proposed/draft), terminal
  (`done` / `cancelled`) filtered out, capped at ~3 with `+N` overflow.

## Scope: verify + reshape (lightened)

The "show epics on every branch" code shipped earlier untested, so this
milestone is primarily verification — give the HUD behavioral coverage and
close G-0188. Verification surfaced a real defect: under the original
"show-all + accentuate-current" shape, the current epic is **lost to `+N`
overflow** whenever it sorts past the cap (≥4 in-flight epics on the latest
epic's branch — observed live on `milestone/M-0192`, where E-0047 fell into
`+2`). The branch-contextual reshape (per the branch-contextual HUD-scope
choice made for this epic) both fixes that and simplifies the code: the ritual
branch renders only the current epic, so there is no list to overflow.

## Why this milestone (per the epic)

M2 of E-0047. Builds directly on the M1 harness — every assertion runs the
real script against fixtures rather than grepping its source.

### AC-1 — Epic HUD renders the in-flight epic list on a non-ritual branch

On a non-ritual branch (e.g. `main`), the epic HUD renders the in-flight list.
`TestStatusline_M0192_AC1_NonRitualRendersEpicList` extends the M1 harness with
an epic-fixture scaffold (`work/epics/<id>-*/epic.md` with a chosen status),
runs `statusline.sh` with the repo as CWD on a non-ritual branch, strips ANSI,
and asserts: non-terminal epics appear with the canonical glyph/color (`→`
active, `○` proposed/draft); terminal epics (`done` / `cancelled`) are absent;
and with more than the cap (3) in-flight, a `+N` overflow marker appears.
Characterization of already-shipped behavior — the evidence is the new test,
which fails if the list path regresses (empty HUD, wrong glyph, missing
overflow).

### AC-2 — Ritual branch shows only the current epic and its milestone

On a ritual branch the HUD shows only the current epic and its milestone —
nothing else. `TestStatusline_M0192_AC2_RitualShowsOnlyCurrentEpic` scaffolds
four or more in-flight epics, checks out an `epic/E-*` (and a `milestone/M-*`)
branch whose epic sorts last, runs `statusline.sh`, and asserts: the current
epic id appears (with its glyph), the milestone id appears inline on a
milestone branch, **no other epic id appears**, and there is **no `+N`
overflow marker**. The test fails against the pre-reshape "show-all +
accentuate" code (where the current epic is swallowed into `+N` overflow) and
passes after the branch-contextual rewrite. This is the genuine red→green of
the milestone.

