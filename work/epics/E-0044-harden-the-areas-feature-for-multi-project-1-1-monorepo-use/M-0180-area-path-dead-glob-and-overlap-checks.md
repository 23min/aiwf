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
      status: open
      tdd_phase: green
    - id: AC-5
      title: path-axis checks skip paths-less members (no-paths config stays inert)
      status: open
      tdd_phase: red
    - id: AC-6
      title: the new findings are AI-discoverable; paths gets a schema-doc note
      status: open
      tdd_phase: red
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

### AC-2 — dead-glob fires for a glob matching no real path; escalates under required

### AC-3 — overlap fires when two areas claim one directory; escalates under required

### AC-4 — strict member decode rejects unknown keys (addresses G-0287)

### AC-5 — path-axis checks skip paths-less members (no-paths config stays inert)

### AC-6 — the new findings are AI-discoverable; paths gets a schema-doc note

