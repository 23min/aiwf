---
id: M-0192
title: Statusline shows in-flight epics on every branch
status: in_progress
parent: E-0047
depends_on:
    - M-0191
tdd: required
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
