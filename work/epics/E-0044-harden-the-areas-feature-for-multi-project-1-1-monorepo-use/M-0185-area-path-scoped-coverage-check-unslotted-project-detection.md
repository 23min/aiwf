---
id: M-0185
title: Area-path scoped-coverage check (unslotted-project detection)
status: done
parent: E-0044
depends_on:
    - M-0179
    - M-0180
    - M-0178
tdd: required
acs:
    - id: AC-1
      title: coverage_roots parses and validates at config load (Tier-1)
      status: met
      tdd_phase: done
    - id: AC-2
      title: areas block rejects unknown top-level keys at load
      status: met
      tdd_phase: done
    - id: AC-3
      title: unslotted child directory under a coverage root raises area-unslotted
      status: met
      tdd_phase: done
    - id: AC-4
      title: coverage law is inert without a declared coverage root
      status: met
      tdd_phase: done
    - id: AC-5
      title: area-unslotted is warning by default, error under areas.required
      status: met
      tdd_phase: done
    - id: AC-6
      title: coverage enumeration is single-level, bounded, and IO-safe
      status: met
      tdd_phase: done
    - id: AC-7
      title: area-unslotted is AI-discoverable and coverage_roots is schema-documented
      status: met
      tdd_phase: done
    - id: AC-8
      title: a dead coverage root or coverage-with-no-paths raises a warning
      status: met
      tdd_phase: done
---
## Goal

Add the *covering* law of the area-path matrix to `aiwf check`: within an operator-declared
**coverage scope**, every project directory is claimed by some area — an unclaimed directory
raises an "unslotted project" finding. This is the monorepo-specific catch for a newly-added
project that nobody slotted into an area, and the third partition law M-0180 deliberately
deferred.

## Context

M-0180 delivered the two *config-anchored* laws (dead-glob, overlap) and the shared `areamatch`
matcher. The remaining law — coverage — is structurally different: it needs a definition of
*which directories are projects* (a universe), which the declared globs alone don't supply. The
forward laws only ever look at what the config *declares*; coverage must look at what the
filesystem *contains* and ask "did any area claim it?"

The universe problem is why this is its own milestone. A single project monorepo with, say, an
`infra` area and an app area plus a legitimate uncovered remainder (`README`, `docs/`, top-level
config) must **not** be flagged wholesale — total partition is the wrong assertion outside a true
multi-project layout. The model is therefore **scoped, opt-in coverage**, not blanket coverage.

## Scoped-coverage model

- The operator optionally declares one or more **coverage roots** in `aiwf.yaml` (the directory
  subtrees whose children are projects expected to be slotted).
- Within a coverage root, every immediate child directory must be claimed by some area's
  `paths:`; an unclaimed child raises the unslotted-project finding.
- Directories **outside** any declared coverage root are unscoped and never flagged (the `infra`
  area, top-level files, `docs/` — all legitimately silent).
- **No coverage root declared → the law is inert.** The knob's presence is also the
  "this is a multi-project monorepo" activation signal, so a semantic-section / single-project
  repo that merely declares `paths:` never trips coverage.

Two distinct "rests" the model keeps separate (settled in design): the *filesystem* remainder is
just unscoped-and-fine; the *entity* remainder (cross-cutting ADRs/decisions) is tagged
`area: global` on the entity axis. `global` is an entity-tag, not a directory claim, so it never
enters the directory-coverage domain.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- A new `aiwf.yaml` coverage-root knob parses and validates (Tier-1 schema; reuses the M-0179
  dual-form decode discipline; a malformed value is a load-time error).
- Within a declared coverage root, a child directory claimed by no area's glob raises an
  unslotted-project finding; a fully-slotted root is silent.
- Inert when no coverage root is declared (the activation signal); coverage-declared-but-no-`paths:`
  is surfaced (not silently inert) per the M-0185 review — see AC-8.
- Severity: warning by default, escalating to error under `areas.required` (consistent with
  dead-glob/overlap and `area-unknown`).
- Bounded, read-only enumeration: single-level `os.ReadDir` per declared root, never fails on IO
  (per the `roadmapCaseCollision` precedent); enumerating only *declared* roots sidesteps the
  `.git` / `node_modules` / build-output noise a blanket walk would pick up.
- Reuses M-0180's `areamatch` matcher for "is this directory claimed by an area's glob" — no
  second matcher.

## Constraints

- Reads the filesystem read-only; never writes. Composed at the CLI seam with the declared set
  from config, like `area-unknown` and the M-0180 checks.
- Does not gate the default views (raises filter trust, not view gating).
- `area` stays single-valued. Coverage is the *covering* half of the directory-partition; it does
  not read entity `area` tags (the `global` sentinel is irrelevant here).

## Out of scope

- The forward laws (dead-glob, overlap) and the `areamatch` matcher — delivered in M-0180.
- Mistag detection (M-0181) and auto-derive (M-0182).
- A static glob-intersection reading of coverage — the law is defined over the *enumerated
  directories within declared roots*, not over abstract glob-set algebra.
- A declared path-bearing "catch-all / remainder" area — the unscoped complement needs no area
  (YAGNI; revisit only if a real case demands it).

## Design notes

- This is the **covering** law of the area↔directory matrix: dead-glob is the *no-empty-column*
  property (every area locates something), overlap is *row-disjointness* (no directory claimed
  twice), and coverage is *covering* (every in-scope directory is claimed). Same cardinality-algebra
  family as M-0176's entity-axis partition test, lifted to the directory axis — though the three do
  not jointly partition one set: dead-glob/overlap range over glob matches repo-wide, while coverage
  ranges only over the immediate children of declared coverage roots.
- The universe = the immediate children of the declared coverage root(s) — Option A: literal roots,
  single-level, with depth handled by declaring multiple roots at any depth. The knob is the single
  source of truth for that universe; deriving it from area glob anchors was rejected as brittle
  (multi-root, anchorless, variable-depth) for a check meant to be trustworthy. **Option B** —
  coverage entries as globs matching project dirs directly at any depth — is deferred as a
  backward-compatible dual-form evolution (a bare path stays a literal root; a glob entry is the new
  form). When B is picked up, pin the disambiguation rule (a glob metacharacter ⇒ glob entry) and
  the caveat that `fs.ValidPath` permits `*` / `[` / `{` in a path segment, so a literal directory
  named e.g. `app[1]` would otherwise be misread as a glob (M-0185 design review, Obs 5).
- Native validation, in-binary: Tier-1 config-load validation for the knob, Tier-2 `aiwf check`
  rule for the law, Tier-3 property test for covering. No external validator (downstream config).
- `depends_on: M-0179` (paths oracle), `M-0180` (the `areamatch` matcher + the forward laws this
  completes), `M-0178` (the `areas.required` escalation seam).
- **Areas-block strict-key guard (from the M-0208 review).** When `coverage_roots` becomes a
  modeled key, also add an areas-block-level strict-key guard mirroring G-0287's member-level
  `unknownMemberKey` — so a typo'd areas key (e.g. `requried:`) is rejected at load rather than
  silently ignored. This reframes forward-compat as *explicit schema evolution*: the M-0208
  surgical writer's byte-preservation survives untouched; only the decode side tightens.
- **Review outcome (the M-0185 two-lens review).** The fresh-context code + design reviews
  converged on the `"."`-root noise (`.git` / `.claude` flagged as unslotted, contradicting the
  "sidesteps noise" claim) → fixed by skipping dot-prefixed children. Two opted-in-but-silent
  diagnostics were surfaced as AC-8: a dead coverage root (`area-coverage-root-missing`) and
  coverage-declared-without-`paths:` (`area-coverage-no-paths`). Deferred follow-up: the
  areas-block strict-key guard is asymmetric — only the `areas:` block rejects unknown keys; the
  top-level `aiwf.yaml` decode stays non-strict (tracked as a gap, mirrored in Deferrals at wrap).

## Dependencies

- M-0179 (`paths:` per area) — the oracle.
- M-0180 (dead-glob/overlap + `areamatch`) — the matcher and the forward laws this completes.
- M-0178 (`areas.required`) — the escalation seam for the severity contract.
- M-0208 (rename-area writer fix) — must land first: it makes the `coverage_roots` knob survive
  `aiwf rename-area`, which previously regenerated the `areas:` block and silently dropped every
  sibling key (`required`, and any future key) on rewrite.

## References

- M-0180 — the forward laws + `internal/areamatch` matcher this reuses.
- `internal/check/check.go` (`roadmapCaseCollision`) — the read-only, never-fail-on-IO
  directory-read precedent.
- `internal/config/config.go` — `Areas`; the coverage-root knob extends the schema here.
- `internal/areagroup/areagroup.go` — the entity-axis partition (M-0176); coverage is the
  directory-axis covering law.

### AC-1 — coverage_roots parses and validates at config load (Tier-1)

A new `areas.coverage_roots` (`[]string`) field on `config.Areas` decodes through
the existing custom `Areas.UnmarshalYAML`, with an explicit `coverage_roots: []`
normalized to nil so empty equals absent. `Areas.validate()` rejects any entry
that is empty, whitespace-padded, or not a valid repo-relative path —
`fs.ValidPath` (no leading slash, no `..` segments; `.` permitted as the
repo-root scope) — as a hard load error naming the bad value, the Tier-1 gate.
*Evidence:* `TestConfig_CoverageRoots_ParsesAndValidates` (`internal/config/area_test.go`),
8 cases at the real `config.Load` seam.

### AC-2 — areas block rejects unknown top-level keys at load

`unknownAreasKey` / `knownAreasKeys` reject any key outside
`{members, default, required, coverage_roots}` in the `areas:` mapping at
config-load, naming the bad key — the areas-block-level analogue of G-0287's
member-level guard, closing the silent-drop where yaml.v3's non-strict
`value.Decode` would discard a typo'd key. Skipped for a non-mapping `areas:`
value (which `value.Decode` rejects). *Evidence:*
`TestConfig_AreasBlock_RejectsUnknownKey` (`internal/config/area_test.go`).

### AC-3 — unslotted child directory under a coverage root raises area-unslotted

`check.AreaCoverage` + `CodeAreaUnslotted`: within each declared coverage root,
every immediate child directory is tested against the declared area globs via the
`areamatch` SSOT (`claimedByAnyArea` → `areamatch.Match`), and an unclaimed child
fires `area-unslotted` naming the directory and the root. A whole-project `**`
glob claims the bare project directory, so the fully-slotted root is silent — no
second matcher. Composed at the CLI seam (`internal/cli/check`) with the declared
areas + coverage roots from `aiwf.yaml`. *Evidence:* `TestAreaCoverage`
(`internal/check/area_coverage_test.go`) + the dispatcher-seam test
`TestRunCheck_AreaUnslottedSurfacesViaDispatcher` (`internal/cli/integration`).

### AC-4 — coverage law is inert without a declared coverage root

With no `areas.coverage_roots` declared, `AreaCoverage` returns no findings even
when unclaimed directories exist — the knob's presence is the activation signal,
so a single-project / semantic-section repo (which declares no coverage root) is
never flagged wholesale. *Evidence:* the "no coverage root declared is inert"
case in `TestAreaCoverage`. (The complementary opted-in-but-undeliverable cases —
a declared root with no `paths:`, or a dead root — are surfaced, not silent; see
AC-8.)

### AC-5 — area-unslotted is warning by default, error under areas.required

`area-unslotted` is emitted at `SeverityWarning`; the CLI-composed
`ApplyAreaRequiredStrict` post-pass bumps it (and the AC-8 coverage findings) to
`SeverityError` under `aiwf.yaml: areas.required: true`, uniformly with
`area-unknown` / `area-dead-glob` / `area-overlap`. *Evidence:*
`TestApplyAreaRequiredStrict_EscalatesCoverageFindings` asserts both severities
and that an unrelated control code passes through unchanged.

### AC-6 — coverage enumeration is single-level, bounded, and IO-safe

Enumeration is one `os.ReadDir` per declared root (immediate children only — a
grandchild two levels deep is never flagged), reads the filesystem read-only, and
never fails on a transient/permission IO error (the `roadmapCaseCollision`
precedent). Only declared roots are enumerated — never a blanket walk — and
hidden (dot-prefixed) children (`.git` / `.github` / `.claude`) are skipped (the
Unix dotfile convention; hidden dirs are tooling/VCS artifacts, never projects).
The skip covers only *hidden* dirs — a `.` root still enumerates non-hidden
top-level dirs (`docs/`, `node_modules/`), so point coverage at a dedicated
project-parent root (`projects/`, `apps/`) rather than `.` unless every
non-hidden top-level dir is genuinely a project. *Evidence:* the single-level
(grandchild), non-dir-skip, hidden-dir-skip, empty-root, and
indeterminate-stat-error cases in `TestAreaCoverage`.

### AC-7 — area-unslotted is AI-discoverable and coverage_roots is schema-documented

Each emitted coverage finding (`area-unslotted`, `area-coverage-root-missing`,
`area-coverage-no-paths`) carries a `hintTable` entry and a table row in the
`aiwf-check` skill's `## Findings (warnings)` section, and `areas.coverage_roots`
has an "Areas `coverage_roots` schema" note (toward G-0288) documenting the
opt-in model, the dot-dir skip, and the dual-form deferral. *Evidence:*
`TestAreaCoverageFinding_StructurallyDocumented` (`internal/policies`), a
structural assertion scoped to the warnings section (with the markdownSection
self-guard against vacuity), plus the `finding-codes-have-hints` policy.

### AC-8 — a dead coverage root or coverage-with-no-paths raises a warning

The two opted-in-but-undeliverable misconfigurations are surfaced rather than
silently skipped (from the M-0185 review, Obs 1 + Obs 3): a declared root that
resolves to no directory — non-existent, or naming a file — fires
`area-coverage-root-missing` (dead config, the coverage analogue of
`area-dead-glob`, via an `os.Stat` guard that distinguishes "resolves to no
directory" from a transient/permission IO error); and `coverage_roots` declared
with no area `paths:` fires a single `area-coverage-no-paths` (the path oracle is
dormant) instead of degenerating into a per-child storm. Both warn by default and
escalate under `areas.required` (AC-5). *Evidence:* the dead-root, file-root,
indeterminate-stat, and no-paths cases in `TestAreaCoverage`.

## Work log

- **AC-1 / AC-2** — `coverage_roots` parse+validate and the areas-block strict-key guard · commit `96620ee9`.
- **AC-3 / AC-4 / AC-5 / AC-6 / AC-7** — `area-unslotted` check + CLI seam + severity + IO-safe single-level enumeration + discoverability · commit `229b1c90`.
- **AC-8** — `area-coverage-root-missing` + `area-coverage-no-paths` diagnostics + the hidden-dir skip · commit `1f5f8f7e`.
- **AC bodies + design notes** · commit `776cee6b`; **wrap-review corrections** (honest dot-dir rationale + required-escalation e2e test; AC-6 reword) · commits `b2b984a6` / `e8da91b7`.

Phase timelines are in `aiwf history M-0185/AC-<N>`; not duplicated here.

## Decisions made during implementation

Lightweight implementation choices (no ADR / `D-NNN` — the area architecture is set by ADR-0020 / ADR-0021):

- **Universe = Option A** (literal coverage roots, single-level immediate children, multiple roots for mixed depth). Option B (coverage entries as globs) is deferred as a backward-compatible dual-form evolution; the disambiguation rule and the `app[1]` pathological caveat are pre-pinned in Design notes.
- **Opted-in-but-undeliverable is surfaced, not silent** (review Obs 1 + Obs 3): a dead coverage root (`area-coverage-root-missing`) and coverage-declared-without-`paths:` (`area-coverage-no-paths`) warn rather than no-op. Folded into AC-8.
- **Hidden-dir skip** in the enumeration (the Unix dotfile convention), resolving the `.`-root false positive both review lenses converged on.

## Validation

- All 8 ACs `met` / `tdd_phase: done`.
- `go build ./...` green; `make ci` green (vet + lint + race test-cov + selfcheck); `make coverage-gate` green (diff-scoped coverage + firing-fixture meta-gate).
- `aiwf check` (worktree binary): **0 errors** (5 benign inherited warnings — `acs-tdd-audit` ×3 on M-0176, `epic-active-no-drafted-milestones`, `provenance-untrailered-scope-undefined`).
- `internal/check` coverage: `area_coverage.go` 100% modulo one `//coverage:ignore`'d `os.ReadDir`-error branch (unreproducible as the test user); `golangci-lint` 0 issues.

## Deferrals

- **G-0305** — top-level `aiwf.yaml` decode stays non-strict (only the `areas:` block rejects unknown keys). The whole-config strict-decode follow-up; the gap records the legacy-key (`actor:` / `aiwf_version:`) read-tolerance constraint the fix must preserve.

## Reviewer notes

- **Two independent fresh-context two-lens reviews** (code-quality `wf-review-code` + design `wf-rethink`).
  - **Round 1** (pre-wrap; code `REQUEST-CHANGES`, design `SOUND-WITH-RECOMMENDATIONS`): both lenses converged on a `.`-root false positive (`.git` / `.claude` flagged as unslotted, contradicting the "sidesteps noise" claim) → fixed via the hidden-dir skip; 7 empty AC bodies → filled; Obs 1 (dead root silent) + Obs 3 (no-paths silent) → surfaced as AC-8.
  - **Round 2** (wrap-stage re-review; code `APPROVE-WITH-NITS`, design `SOUND-WITH-RECOMMENDATIONS`): prior findings confirmed resolved. Load-bearing doc fix — the dot-dir rationale overclaimed "sidesteps `node_modules`/build noise" (false: those are not hidden) → reworded to the hidden-dirs principle, with `.` root framed as a knowingly-opted edge. Nits addressed: the ENOTDIR path-traversal edge commented; an end-to-end `areas.required` escalation test added.
- **Inherited trunk-collision** (not M-0185 substance): the M-0208-era gap at its old id collided with a different gap that had since landed on trunk → resolved via `aiwf reallocate` (`G-0302` → `G-0306`).
- **M-0208 latent fix** surfaced during preflight: M-0208's body referenced its reallocate-history ids un-backticked, firing `body-prose-id` in a fresh checkout → backticked (commit `5bc71d6b`).

