---
id: M-0192
title: Statusline shows in-flight epics on every branch
status: done
parent: E-0047
depends_on:
    - M-0191
tdd: required
acs:
    - id: AC-1
      title: Epic HUD renders the in-flight epic list on a non-ritual branch
      status: met
      tdd_phase: done
    - id: AC-2
      title: Ritual branch shows only the current epic and its milestone
      status: met
      tdd_phase: done
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

## Work log

- **AC-1 / AC-2 — met.** The branch-contextual reshape + behavioral coverage
  landed in `f43f8e3e` (`fix(statusline): make epic HUD branch-contextual`).
  Verifying the previously-untested "show epics on every branch" code surfaced
  the current-epic-lost-to-overflow defect; the fork (ritual → current epic
  only, non-ritual → list) fixes it and simplifies the code. The fix is
  mirrored into the embedded copy `internal/skills/embedded-statusline/statusline.sh`,
  kept byte-identical per the M-0155 drift test (`TestM0155_AC1_StatuslineEmbedded`).
- G-0188's "desired behavior" was refined from "show all + secondary" to the
  branch-contextual model in the same milestone, so the gap matches the build.

## Validation

- `go test ./internal/policies/ -run 'TestStatusline_M019[12]|TestM0155_AC1'`
  — green: AC-1, AC-2 (epic-branch + milestone-branch subtests), the
  missing-`epic.md` fail-soft fallback, the full M-0191 suite, and the embed
  drift test.
- `go vet` + `golangci-lint` (policies) clean; `go build ./...` ok; full
  `internal/policies` package green (~93s, also run by the pre-commit hook).
  Full `make ci` runs at the wrap-merge into the epic branch.
- Human-verified renders: ritual `milestone/M-0192` → `▸ → E-0047/→ M-0192`
  (current epic + milestone only); non-ritual `main` →
  `○ E-0019 ○ E-0034 → E-0044 +4` (list, cap 3 + overflow, repo name correct).

## Reviewer notes

- An independent fresh-context reviewer approved with no blocking findings. It
  re-confirmed AC-2's red→green independently (stashed the script, AC-2 failed
  against the old code, passed after restore) and that the assertions are
  structural — scoped to the ` · `-delimited epic-HUD segment so the branch
  segment (which itself carries the epic/milestone id) cannot produce a false
  pass.
- Advisories addressed inline: a fail-soft test for a missing `epic.md` on a
  ritual epic branch (the `? E-NNNN` unknown-status glyph); a
  tracked-file-dependency comment on the ctx derivation; the AC-1 overflow
  assertion tightened to the exact `+1`.
- Left as documented: the ritual `m_file=""` arm is effectively unreachable (a
  milestone branch enters the ritual block only because its file was found for
  the epic-id derivation), and `epicHUDSegment`'s no-match return is a
  defensive test-helper line.
