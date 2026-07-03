---
id: M-0223
title: Guard the unconditional authorize-opener grep in the read verbs
status: in_progress
parent: E-0054
tdd: required
acs:
    - id: AC-1
      title: guard predicate returns skip for events with no scope data
      status: met
      tdd_phase: done
    - id: AC-2
      title: history and show output identical for scoped and scopeless entities
      status: met
      tdd_phase: done
    - id: AC-3
      title: measured read-verb wall-time delta recorded in Validation
      status: met
      tdd_phase: done
---
## Goal

Stop `aiwf history` **and** `aiwf show` from running a repo-wide `git log` authorize
grep on every invocation. The same waste has two near-duplicate implementations:

- `history.BuildScopeEntityMap` (`internal/cli/history/history.go`), called
  unconditionally in the `history` **text** path — a milestone with zero scopes
  measured ~2.2s text vs ~1.2s `--format=json` (which skips it): ~1.0s / ~40% waste.
- `show.readAllAuthorizeOpeners` via `LoadEntityScopeViews` (`internal/cli/show/
  scopes.go:59`), called **before** the `interested` set is computed, so every
  `aiwf show` pays it too — measured ~3.4s.

Guard both, but the two verbs are **asymmetric** — get this right or `show` breaks:

- **`history` is safe to skip on `AuthorizedBy`/`ScopeEnds`.** `RenderScopeChips`
  (`history.go:472`) consumes the opener map *only* via `e.AuthorizedBy` and
  `e.ScopeEnds`; an authorize opener's own `[scope: …]` chip renders from `e.Scope`
  without the map (`history.go:475`). So skip `BuildScopeEntityMap` when the loaded
  events carry no `AuthorizedBy` and no `ScopeEnds`.
- **`show` must NOT skip merely on those fields.** `LoadEntityScopeViews` builds its
  `interested` set from two sources: the entity's own `AuthorizedBy` events
  (`scopes.go:66`) **and** the global opener map filtered to scopes where the entity
  *is* the scope entity (`scopes.go:70-73`). An active/paused/resumed scope opener
  has `Scope` but no `AuthorizedBy`/`ScopeEnds`, so an `AuthorizedBy`-only guard would
  drop its scope table. But source (b) doesn't need the global grep at all: a scope
  opened on `id` is authored by a commit carrying `aiwf-entity: id`, i.e. it's in
  `id`'s own history — derive it from `LoadEntityScopes(id)` directly, and run the
  global `readAllAuthorizeOpeners` grep **only** when the entity has `AuthorizedBy`
  events (source (a): resolving which scope authorized foreign work).

Consolidate into one shared helper (single source of truth): a predicate that decides
whether the global grep is needed, plus a direct-scope derivation for `show`/render —
so the render single-pass (M-0221) reuses it rather than adding a third copy.

## Notes

- **Key the predicate off the loaded event slice, not entity frontmatter.**
  `aiwf-scope-ends` and the authorize opener are commit trailers with no frontmatter
  counterpart. The events are already loaded (`ReadHistoryChain` / `ReadHistory`)
  before the grep, so the predicate is free.
- **Low risk, verified — not "zero".** History: the map is consumed only via
  `AuthorizedBy`/`ScopeEnds`. Show: direct derivation covers source (b); the grep
  covers source (a). Verified by differential diagnostic on the live tree —
  `LoadEntityScopes(id)` reproduces the current `show` scope table for openers
  (E-0032, E-0014); where foreign `AuthorizedBy` is present (E-0029) the grep is
  still used and own scopes remain a subset. Correct only if the fixture proves the
  opener case (AC-2).
- **Incidental fix (finish-in-context).** `show` currently passes the **raw** id to
  `LoadEntityScopeViews`, whose source-(b) test compares a *canonicalized* map value
  against the raw id (`scopes.go:71`, `ent == id`), so `aiwf show <narrow-id>`
  silently omits the scope table — confirmed live: `aiwf show E-14` shows none while
  `aiwf show E-0014` shows two. Deriving source (b) from the width-tolerant
  `LoadEntityScopes(id)` corrects this for free; pin it in AC-2. Not a separate gap —
  it is the exact path this milestone rewrites.
- Orthogonal to M-0221's single-pass; land independently. No `depends_on` — different
  call sites — but both must use the one consolidated helper.

## Acceptance criteria

### AC-1 — guard predicate returns skip for events with no scope data

Unit test on the extracted predicate (not subprocess count, which a Go test can't
observe). For the **history** map: given events with no `AuthorizedBy` and no
`ScopeEnds`, return skip; given either present, return run. For **show**: given an
entity that is a scope opener (an `authorize` event in its own stream) but has no
`AuthorizedBy`, the global grep is still skipped **and** the direct-scope derivation
returns its scopes. Extracting the predicate/derivation as a testable seam is part of
the AC.

### AC-2 — history and show output identical for scoped and scopeless entities

Two-part oracle. **(a) Equivalence** — `aiwf history` (text and JSON) **and**
`aiwf show` output byte-identical guarded-vs-unguarded for **canonical-id**
invocations, over a fixture of at least four entities: (i) a scopeless entity
(skips); (ii) an entity worked *under* a scope — has `AuthorizedBy` (grep runs);
(iii) an **active direct-scope opener** — `authorize` verb, no `AuthorizedBy`/
`ScopeEnds` — whose `show` scope **table** must be non-empty (history renders its
chip from `e.Scope`); omit (iii) and the test passes vacuously while `show` silently
loses direct scopes; (iv) a scope-ended entity (`ScopeEnds` present). **(b) Width
fix** — a **narrow-width opener** (an `E-NN` legacy id) queried by its narrow id:
`show`'s scope table must render. This is a deliberate correction of the raw-id
omission, *not* asserted as guarded==unguarded (the unguarded raw-id path is the
bug).

### AC-3 — measured read-verb wall-time delta recorded in Validation

Structural assertion: the Validation section is present and records a before/after
wall-time measurement for `aiwf history` **and** `aiwf show` via `performance.md`'s
"How to measure" recipe. The absolute number is environment-specific, not a CI gate.

## Validation

Measured on the kernel tree in this devcontainer (Docker/linuxkit, ~5,500 commits)
with the `performance.md` "How to measure" recipe: `strace -f -e trace=execve` for
the git-subprocess count, a byte-diff of each verb's output before/after, and
best-of-7 wall-time. The **before** binary is the unguarded read verbs (this
milestone's source changes stashed); the **after** binary is the guarded build.
The target is a scopeless entity — one whose loaded events carry no
`aiwf-authorized-by`, no `aiwf-scope-ends`, and no own authorize-opener — the
common case, where both guards skip their walk.

| verb | before | after | saved | git subprocesses (before → after) |
|---|---|---|---|---|
| `aiwf history <scopeless>` | 2.33s | 1.30s | 1.03s (~44%) | 5 → 3 |
| `aiwf show <scopeless>` | 3.61s | 2.44s | 1.16s (~32%) | 7 → 6 |

Output is **byte-identical** before/after for both verbs on the scopeless target —
the guard changes performance, not results. `aiwf history` drops the two
subprocesses of the guarded `BuildScopeEntityMap` (its own `HasCommits` probe plus
the authorize grep); `aiwf show` drops the global authorize grep and — because the
entity has no own opener — the per-entity `LoadEntityScopes` walk too, so its cost
falls to the same order as `aiwf history`. Absolute numbers are devcontainer-specific
and not a CI gate; the load-bearing mechanical evidence for this milestone is the
guard predicates, the guarded-vs-unguarded equivalence, and the width fix under
`internal/cli/`.

## Work log

### AC-1 — guard predicate returns skip for events with no scope data
Extracted `HasScopeData`, `HasAuthorizedBy`, and `HasOwnScope`
(`internal/cli/history/scopeguard.go`) as testable seams keyed off the already-loaded
event slice, not entity frontmatter. Pinned by `TestHasScopeData` / `TestHasAuthorizedBy`
/ `TestHasOwnScope` (predicate tables, both directions) plus
`TestScopeGuard_ShowDerivationForActiveOpener` (the `show` source-(b) derivation for an
active opener carrying no `aiwf-authorized-by`).

### AC-2 — history and show output identical for scoped and scopeless entities
Consolidated the two byte-identical greps into `cliutil.AuthorizeOpeners`; guarded
`aiwf history`'s chip map behind `ScopeMapFor`, and rewrote `show.LoadEntityScopeViews`
— source (b) from the width-tolerant `cliutil.LoadEntityScopes` gated by `HasOwnScope`,
source (a) via the guarded global grep gated by `HasAuthorizedBy`, with an `ent != id`
guard against self-scope double-counting. Equivalence pinned by
`TestScopeGuard_ShowViewsEquivalence` (guarded vs an unguarded oracle over a five-entity
fixture — scopeless, worked-under-scope, active-opener, scope-ended, self-scope) and
`TestScopeGuard_HistoryChipsEquivalence`; the incidental narrow-id width fix by
`TestScopeGuard_ShowWidthFix`. The removed `BuildScopeEntityMap` test migrated to
`TestAuthorizeOpeners_NonRepoReturnsEmpty`.

### AC-3 — measured read-verb wall-time delta recorded in Validation
Measured before/after on the kernel tree (see `## Validation`) with the `performance.md`
recipe; structural evidence `TestM0223_AC3_ValidationRecordsReadVerbMeasurement`
(section-scoped via `extractMarkdownSection`; asserts both verbs, a before/after axis,
and ≥4 wall-time values).

The authoritative per-AC phase/status timeline lives in `aiwf history M-0223/AC-N`.

## Deferrals

None originate in this milestone. The path-scoped-history sibling was cancelled pre-start
and its constraints captured in G-0340. The `performance.md` ground-doc refresh — renaming
the removed private impls to `cliutil.AuthorizeOpeners` and marking the grep-guard lever
shipped — is owned by the E-0054 epic wrap, deferred so it lands together with M-0221's
render lever; an epic-wrap doc task, not gap-worthy milestone work.

## Reviewer notes

Independent fresh-context review (code-quality lens, `wf-review-code`): **APPROVE**. The
reviewer ran its own five mutation probes, a merged-profile coverage sweep, and
live-binary checks (a narrow-id `aiwf show` query rendered the same scope table as its
canonical-width form); every load-bearing claim held. Design lens (`wf-rethink`) on the
guard module: keep, no rewrite — KISS, single source of truth for the grep, layering
`history → cliutil` and `show → history + cliutil`.

Three non-blocking findings, dispositions:

- **Comment tense** (`cliutil.AuthorizeOpeners`): reworded — render (M-0221) *will* reuse
  the helper; it does not call it yet.
- **Error-detail trade-off** (`cliutil.AuthorizeOpeners`): the consolidated helper wraps a
  `git log` failure without the `exec.ExitError` stderr extraction the removed
  `readAllAuthorizeOpeners` surfaced. Deliberate KISS choice — the failure path is
  unreachable after `HasCommits` on a valid repo (it carries `//coverage:ignore`), and
  collapsing it to one return avoids three untestable defensive branches. Kept.
- **History equivalence granularity**: the committed differential asserts guarded ==
  unguarded at the `RenderScopeChips` chip level — the precise seam, since the guard's
  only effect is the map handed to that function — plus a binary wiring check;
  `--format=json` never consumed the map, so is vacuously preserved. Full-output
  byte-identity was additionally confirmed by the `## Validation` measurement's `diff`
  step. Kept.

`make coverage-gate` was run against a throwaway commit of this implementation and passed
(diff-scoped: every changed statement tested or `//coverage:ignore`'d); it re-runs
authoritatively on the wrap commit at CI-on-push.
