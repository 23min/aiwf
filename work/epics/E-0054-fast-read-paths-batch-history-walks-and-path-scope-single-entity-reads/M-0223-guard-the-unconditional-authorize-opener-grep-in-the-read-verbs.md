---
id: M-0223
title: Guard the unconditional authorize-opener grep in the read verbs
status: draft
parent: E-0054
tdd: required
acs:
    - id: AC-1
      title: guard predicate returns skip for events with no scope data
      status: open
      tdd_phase: red
    - id: AC-2
      title: history and show output identical for scoped and scopeless entities
      status: open
      tdd_phase: red
    - id: AC-3
      title: measured read-verb wall-time delta recorded in Validation
      status: open
      tdd_phase: red
---
## Goal

Stop `aiwf history` **and** `aiwf show` from running a repo-wide `git log` authorize
grep on every invocation. The same waste has two near-duplicate implementations:

- `history.BuildScopeEntityMap` (`internal/cli/history/history.go`), called
  unconditionally in the `history` **text** path — a milestone with zero scopes
  measured ~2.2s text vs ~1.2s `--format=json` (which skips it): ~1.0s / ~40% waste.
- `show.readAllAuthorizeOpeners` via `LoadEntityScopeViews` (`internal/cli/show/
  scopes.go`), called **before** the `interested` set is computed, so every
  `aiwf show` pays it too — measured ~3.4s.

Guard **both**: skip the grep when the entity's *loaded events* carry no scope data
(no `AuthorizedBy`, no `aiwf-scope-ends`); when scope data is present, bound the grep
to the referenced SHAs rather than the whole history. **Consolidate** the two
implementations into one shared helper rather than fixing each separately (single
source of truth) — the render single-pass (M-0221) should reuse the same helper, not
add a third copy.

## Notes

- **Key the guard off the loaded event slice, not entity frontmatter.**
  `aiwf-scope-ends` is a commit trailer on the terminal-promote commit with no
  frontmatter counterpart; a guard reading only `authorized_by` frontmatter would
  silently drop the `[<entity> ended]` chips for a scope-ending entity. The events
  are already loaded (`ReadHistoryChain` / `ReadHistory`) before the grep runs, so the
  predicate is free.
- **Low risk, verified — not "zero".** The scope map is consumed only via
  `AuthorizedBy` and `ScopeEnds` (in `RenderScopeChips` and `LoadEntityScopeViews`'s
  `interested` set); an authorize *opener* event renders its `[scope: …]` chip from
  `e.Scope` without the map. So skipping when neither is present is output-identical —
  but only if the fixture proves it (see AC-2).
- Orthogonal to M-0221's single-pass but shares the theme; land independently. No
  `depends_on` — they touch different call sites (history/show vs resolver) — but both
  must use the one consolidated helper.

## Acceptance criteria

### AC-1 — guard predicate returns skip for events with no scope data

Unit test on the extracted predicate (not on subprocess count, which a Go test can't
observe): given an event slice with no `AuthorizedBy` and no `ScopeEnds`, the guard
returns skip; given either present, it returns run. Extracting the predicate as a
testable seam is part of the AC.

### AC-2 — history and show output identical for scoped and scopeless entities

Byte-identical `aiwf history` (text and JSON) **and** `aiwf show` output, guarded vs
unguarded, over a fixture of at least three entities: (i) a scopeless entity (guard
skips), (ii) an entity worked *under* a scope — has `AuthorizedBy` (guard must run),
and (iii) an **active-scope opener** — verb `authorize`, no `AuthorizedBy`/`ScopeEnds`
(guard skips, its `[scope:]` chip must still render). Omit (iii) and the test passes
vacuously.

### AC-3 — measured read-verb wall-time delta recorded in Validation

Structural assertion: the Validation section is present and records a before/after
wall-time measurement for `aiwf history` **and** `aiwf show` via `performance.md`'s
"How to measure" recipe. The absolute number is environment-specific, not a CI gate.
