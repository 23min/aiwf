---
id: M-0180
title: Area-path dead-glob and overlap checks
status: in_progress
parent: E-0044
depends_on:
    - M-0179
    - M-0178
tdd: required
acs:
    - id: AC-1
      title: areamatch is the SSOT path-glob matcher (doublestar-backed)
      status: met
      tdd_phase: done
    - id: AC-2
      title: dead-glob fires for a glob matching no real path; escalates under required
      status: met
      tdd_phase: done
    - id: AC-3
      title: overlap fires when two areas claim one directory; escalates under required
      status: met
      tdd_phase: done
    - id: AC-4
      title: strict member decode rejects unknown keys (addresses G-0287)
      status: met
      tdd_phase: done
    - id: AC-5
      title: path-axis checks skip paths-less members (no-paths config stays inert)
      status: met
      tdd_phase: done
    - id: AC-6
      title: the new findings are AI-discoverable; paths gets a schema-doc note
      status: met
      tdd_phase: done
---
## Goal

Add the two *config-anchored* laws of the area-path matrix to `aiwf check`: **dead-glob**
(every declared area's `paths:` glob matches a real directory — no dead config) and **overlap**
(no directory is claimed by more than one area — unambiguous attribution). Introduce the shared
glob matcher (`internal/areamatch`) these checks and the Tier-2 milestones (mistag, auto-derive)
all read. This is the value layer the path oracle (M-0179) unlocks — minus the reverse coverage
law, which moves to M-0185.

## Context

M-0179 gave each area an optional `paths:` glob but consumes nothing: "glob matching is
deferred to M-0180, the first match call site." This milestone lands that call site and the two
checks that need only the *declared globs and the filesystem* — no project-directory universe:

- **dead-glob** catches a renamed / deleted / typo'd project path leaving an area pointing at
  nothing — the oracle for that area is empty.
- **overlap** catches two areas whose globs claim the same directory — the oracle is ambiguous,
  which would make mistag (M-0181) and auto-derive (M-0182) behave non-deterministically there.

The reverse law — "every project directory is claimed by some area" (the unslotted-project /
coverage catch) — is deliberately **not** here. It needs a project-directory *universe*, which
needs an explicit coverage-scope knob, bounded filesystem enumeration, noise exclusion, and a
non-monorepo activation condition — a distinct, design-heavy unit. It lands in M-0185 (see Out of
scope). The split keeps this milestone to the self-contained, ready-to-build half and delivers the
matcher its siblings depend on.

The feature's logic is a **partition + classification algebra**, not an FSM: the areas are *meant
to* partition the project directory space (within a scope) and classify entities into it. This
milestone lands **two of the three** partition laws — *no empty area column* (dead-glob) and
*directory-row disjointness* (overlap); the third, *covering* (every in-scope directory claimed by
an area), is the deferred reverse law. They are validated natively — config-load validation
(Tier 1) + `aiwf check` rules (Tier 2) + property tests (Tier 3) — never an external validator:
the areas block is downstream consumer config, so its validation must be authoritative and
in-binary.

## Acceptance criteria

<!-- Formalized via `aiwf add ac M-0180 --title "..."` at start-milestone. Each pins one
     observable behavior with named mechanical evidence. AC-1's matcher is the SSOT keystone —
     AC-2/AC-3 route their "does this path match this glob?" through it. -->

- **AC-1 — `areamatch`: SSOT path-glob matcher.** `internal/areamatch` (wrapping `doublestar/v4`)
  answers "does repo-relative path P match area glob G?" — the one definition of match dead-glob
  uses here, and mistag (M-0181) / auto-derive (M-0182) reuse. *Evidence:* table-driven unit
  test, spec-sourced glob cases (literal, `*`, `**`, multi-segment).
- **AC-2 — dead-glob finding.** A declared area whose `paths:` glob matches no real directory
  raises a finding; **warning by default, error under `areas.required`**. Reads the filesystem
  read-only, never fails on IO (per the `roadmapCaseCollision` precedent). *Evidence:* check-test —
  missing-dir glob fires, real-dir glob silent; a `required:true` row asserts error severity.
- **AC-3 — overlap finding.** A directory matched by ≥2 declared areas' globs raises a finding;
  **warning by default, error under `areas.required`**. *Evidence:* check-test — two areas, one
  shared dir fires; disjoint globs silent; a `required:true` row asserts error severity.
- **AC-4 — strict member decode (addresses G-0287).** The areas member-mapping decode rejects
  unknown keys, so a typo'd `pathz:` / `path:` is a load-time error naming the bad key, not a
  silently-dropped `paths:` that feeds dead-glob/overlap a false "no location." *Evidence:* config
  unit test — `{name, pathz: [...]}` → error naming the key.
- **AC-5 — inert without paths.** A label-only / legacy string-form config (no member declares
  `paths:`) fires neither dead-glob nor overlap. *Evidence:* fixture with a paths-less areas block
  → zero path-axis findings.
- **AC-6 — discoverability.** The new finding(s) are documented on an AI-discoverable surface
  (the `aiwf-check` finding catalog / skill), and the now-observable `paths` behavior gets a
  schema-doc note (toward G-0288, which stays open for the full-block doc). *Evidence:* structural
  content assertion on the named section.

## Constraints

- Reads the filesystem **read-only**; never writes. Composed at the CLI seam with the declared
  set sourced from config — the same seam `area-unknown` uses — so the pure `check.Run` stays
  config-agnostic (the M-0171/AC-4 boundary).
- **Severity:** dead-glob and overlap are warnings by default and escalate to error under
  `areas.required`, uniformly — consistent with how `area-unknown` escalates. Mechanically this
  **extends** `ApplyAreaRequiredStrict` (today hardcoded to bump only `area-unknown`) to cover the
  new codes, following the same CLI-composed post-pass pattern — not a verbatim reuse. No
  dead-glob/overlap severity split.
- **Does not gate the default views.** Path verification raises filter trust, not view gating.
- `area` stays single-valued. This is the path-claim (directory-column) axis, orthogonal to the
  entity-tag axis where `global` and `areas.required` live.

## Out of scope

- **The reverse coverage / unslotted-project law (M-0185)** — needs the coverage-scope knob,
  bounded single-level enumeration, a noise/`.git` exclusion rule, and the non-monorepo activation
  condition. The **scoped-coverage** model (the operator declares the root(s) that must tile;
  outside any declared root is legitimately unclaimed and silent; absence of a coverage root makes
  the law inert) is what M-0185 builds.
- **`global`-value exclusion** — moot here: this check is config↔filesystem and never reads
  entity `area` tags, so the cross-cutting sentinel never enters its domain. (That exclusion is a
  concern for the entity-touching checks: mistag and the reverse-coverage milestone.)
- **Mistag detection (M-0181)** and **auto-derive (M-0182)** — separate Tier-2 milestones; both
  consume AC-1's matcher.

## Design notes

- **The model is a partition/cardinality algebra.** Picture the area↔directory incidence as a
  matrix; dead-glob asserts no area column is empty, overlap asserts no directory row sums above
  one. Same family as M-0176's partition-totality test on `internal/areagroup` (the entity-axis
  instance), lifted to the directory axis. The property tests (Tier 3) are the algebra's oracle.
- **Native validation, three tiers — no CUE.** The areas block is *downstream consumer* config;
  its validation must be authoritative and dependency-free. Tier 1: `config.Areas.validate()` at
  load (hard error; AC-4 lives here). Tier 2: `aiwf check` rules over the live tree (dead-glob,
  overlap; AC-2/3). Tier 3: property tests pin the cardinality laws. The aiwf contract-binding /
  CUE mechanism is a *consumer-supplies-the-validator, unavailable-is-a-warning* surface — the
  wrong layer for a kernel-owned guarantee, so it is deliberately not used.
- **Matcher home + dependency.** `internal/areamatch` wraps `doublestar/v4`. Justification:
  stdlib `filepath.Match` cannot evaluate `**`, which every M-0179 `paths:` example uses; and the
  matcher is needed epic-wide, so the first-sequenced consumer (M-0180) introduces it as the SSOT
  rather than letting M-0181/M-0182 each roll their own "what does a glob mean."
- **`depends_on: [M-0179, M-0178]`.** M-0179 is the paths oracle (hard). M-0178 supplies
  `areas.required` and the `ApplyAreaRequiredStrict` post-pass pattern the severity contract
  extends (it bumps only `area-unknown` today). **Not M-0184** — this milestone never reads entity
  `area` tags, so the `global` predicate is not a dependency.
- **Gaps:** AC-4 addresses G-0287 (mistyped member key silently drops paths). AC-6 nods toward
  G-0288 (full areas-schema doc), which stays open for the broader block-level doc.

## Dependencies

- **M-0179** (`paths:` per area) — the oracle dead-glob/overlap read. Hard.
- **M-0178** (`areas.required`) — the strictness knob and escalation seam the severity contract
  reuses.

## References

- `internal/check/area_unknown.go` — the config-sourced CLI composition seam this follows, and
  `ApplyAreaRequiredStrict`, the escalation seam AC-2/AC-3 extend.
- `internal/check/check.go` (`roadmapCaseCollision`) — the read-only, never-fail-on-IO
  filesystem-read precedent.
- `internal/config/config.go` — `Areas` / `Member` / `validate()`; AC-4 hardens the member decode.
- `internal/areagroup/areagroup.go` — the entity-axis partition (M-0176); the directory-axis
  laws here are the same algebra.
- [ADR-0020](../../../docs/adr/ADR-0020-dual-form-areas-members-schema-backward-compatible-label-location-evolution.md),
  [ADR-0021](../../../docs/adr/ADR-0021-sanctioned-global-area-value-for-inherently-cross-cutting-entities.md).
- M-0181 (mistag) and M-0182 (auto-derive) — the Tier-2 consumers of the `areamatch` matcher.

### AC-1 — areamatch is the SSOT path-glob matcher (doublestar-backed)

`internal/areamatch` wraps `doublestar/v4` (new direct dependency — stdlib
`filepath.Match` cannot evaluate `**`) as the single source of glob-match
semantics for the whole epic. It exposes `Match` (pure repo-relative-path
predicate), `MatchFS` (eager filesystem walk, the list primitive overlap
needs), `MatchesAny` (early-terminating boolean-any via `GlobWalk`, the
primitive dead-glob needs), and `Validate` (Tier-1 syntax gate). Pinned by a
spec-sourced table test over the doublestar grammar (literal / `*` / `**` /
multi-segment + the `ErrBadPattern` wrap); registered at layering tier 7.
100% coverage.

### AC-2 — dead-glob fires for a glob matching no real path; escalates under required

`check.AreaDeadGlob` + `CodeAreaDeadGlob`, fed the declared areas through the
config-agnostic `AreaPaths` projection at the CLI seam (the M-0171/AC-4
boundary — `check.Run` never reads `aiwf.yaml`). Per-glob: each declared glob
must locate ≥1 real path via `areamatch.MatchesAny`, else a finding naming the
member + glob. Reads the filesystem read-only and never fails on IO (the
`roadmapCaseCollision` `os.Stat` guard). Warning by default, escalated to error
under `areas.required` by the extended `ApplyAreaRequiredStrict`. The design
review surfaced that the "malformed globs are owned by Tier-1" claim was
aspirational, so this AC also wired `areamatch.Validate` into
`config.Areas.validate()` (a malformed glob is now a hard load error). Unit +
dispatcher-seam tests; 100% coverage.

### AC-3 — overlap fires when two areas claim one directory; escalates under required

`check.AreaOverlap` + `firstSharedPath` + `CodeAreaOverlap`. Per overlapping
area-*pair* (not per-directory — per-directory would explode to thousands of
findings on a `**` overlap), naming both areas and the lexically-smallest
shared path (deterministic). Uses `areamatch.MatchFS` to materialize and
intersect each area's matched-path set — the genuine list-consumer. IO-safe and
escalation-wired like dead-glob. Independently design-reviewed: per-pair
confirmed as the better model; the eager set-comparison accepted as intrinsic
to overlap and self-limiting at scale. Unit + dispatcher-seam tests; a
multi-candidate case behaviorally pins the lexically-smallest determinism
(mutation-verified). 100% coverage.

### AC-4 — strict member decode rejects unknown keys (addresses G-0287)

`unknownMemberKey` / `knownMemberKeys` reject any key outside `{name, paths}` in
a mapping member at config-load, naming the bad key — closing G-0287's silent
drop where a typo'd `pathz:` vanished through yaml.v3's non-strict
`Node.Decode` and fed the path-axis checks a false "no paths". The `Member`
doc-comment lockstep note now lists `knownMemberKeys` so a future field
addition updates the keyset. Tested at the real `Load` seam (`pathz` typo +
an unrelated key + the malformed-glob row from AC-2). 100% coverage.

### AC-5 — path-axis checks skip paths-less members (no-paths config stays inert)

Both path-axis checks fire nothing for a label-only / legacy bare-string
config (no member declares `paths:`) — the E-0043 backward-compat guarantee.
A cross-cutting unit test asserts both checks inert in one place; an
integration test exercises the legacy `members: [a, b]` form end-to-end through
the real `config.Load` → `AreaPaths` projection → checks seam the unit test
cannot reach. Non-vacuous (fails if either check fires without paths).

### AC-6 — the new findings are AI-discoverable; paths gets a schema-doc note

A structural content assertion (`TestAreaPathFindings_StructurallyDocumented`)
that `area-dead-glob` and `area-overlap` are *table rows in the `## Findings
(warnings)` section* of the `aiwf-check` skill — the structural upgrade over
the substring-level `finding-codes-are-discoverable` policy — with a self-guard
that fails if the section-scoping ever over-extends (so the structural claim
can't go vacuous). Plus a `paths` schema note documenting
`areas.members[].paths` toward the full areas-schema reference (G-0288). The
per-code hints + skill rows landed with AC-2/AC-3 (forced by the finding-codes
chokepoints at code introduction).

## Work log

- **AC-1** — `areamatch` SSOT (Match/MatchFS/MatchesAny/Validate) · commit `1d1f9931` (MatchesAny/Validate added in `eece9fa4` per review) · areamatch 100%.
- **AC-2** — dead-glob check + Tier-1 `Validate` wiring · commit `eece9fa4` · 100%.
- **AC-3** — overlap check (per-pair) · commit `9f1e040a` · 100%.
- **AC-4** — strict member-key decode (G-0287) · commit `d944b07d` · 100%.
- **AC-5** — path-axis inertness pinning · commit `60de424b`.
- **AC-6** — structural discoverability + `paths` schema note · commit `9967025c`.
- **Wrap-review nits** — seam-test exit-code comments + overlap determinism case · commit `ca68037b`.

Phase timelines are in `aiwf history M-0180/AC-<N>`; not duplicated here.

## Decisions made during implementation

These were lightweight implementation choices (no ADR / `D-NNN` warranted — the
area architecture is set by ADR-0020 / ADR-0021):

- **Two walk primitives in the SSOT.** `MatchesAny` (early-terminating) for
  dead-glob's existence question; `MatchFS` (eager list) for overlap's
  set-intersection. The AC-1 design review flagged that a single eager
  `MatchFS` would enumerate a `**`-glob's whole subtree just to test
  emptiness.
- **Tier-1 glob-syntax validation landed in AC-2**, not deferred to AC-4 —
  closes a malformed-glob hole the review surfaced (the check's claim was
  unfulfilled). AC-4 stayed scoped to unknown *keys*, a distinct concern.
- **Overlap finding model: per-pair**, not per-directory — avoids the N-way
  finding explosion; design-reviewed and confirmed.
- **Overlap eager set-comparison: accepted, not optimized.** A syntactic
  disjointness pre-filter was considered and declined — YAGNI (aiwf's own tree
  is small, the check runs at push-time, and the pathological case is exactly
  the misconfiguration the check flags). Decided explicitly with the operator;
  no gap filed (the design rationale lives at the call site).

## Validation

- All six ACs `met` / `tdd_phase: done`.
- `go build ./...` green; full test suite green (`go test ./...`).
- `aiwf check` (worktree binary): **0 errors**; the new dead-glob/overlap rules
  produce no false positives against the live aiwf tree.
- Branch-coverage audit clean, scoped to the M-0180 diff (base `3827bee9`).
- Full-module `golangci-lint run`: 0 issues.

## Deferrals

None requiring a gap. The overlap eager-enumeration perf hardening was
considered and **accepted** (see Decisions), not deferred. G-0288 (the full
areas-schema doc surface) is a pre-existing M-0179 gap that AC-6 advanced with
a note; it stays open as before — not a new M-0180 deferral.

## Reviewer notes

- **Three independent fresh-context reviews.** (1) `areamatch` SSOT API
  (AC-1, `sound-with-recommendations`) → adopted `Validate` (closing the
  malformed-glob hole) and `MatchesAny` (early-terminate). (2) `AreaOverlap`
  design + perf (AC-3, `sound-with-recommendations`) → per-pair confirmed,
  eager set-comparison accepted, and it *debunked* a proposed "sorted-Glob
  early-stop" optimization (doublestar.Glob is pre-order, not lexically
  sorted — the change would have silently altered which path findings name).
  (3) Full change-set code-quality review at wrap (`approve-with-nits`) → both
  nits fixed in `ca68037b`.
- **The check CLI swallows config-load errors** (`if cfgErr == nil`), but a
  malformed glob still aborts loudly upstream via `LoadTreeWithTrunk` (exit 3,
  naming the bad glob) — verified by the code-quality reviewer running the
  binary, so the Tier-1 gate is not silently bypassed by `aiwf check`.

