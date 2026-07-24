---
id: M-0276
title: Gate red-first ordering via a working-tree diff-shape check
status: done
parent: E-0070
depends_on:
    - M-0274
tdd: required
acs:
    - id: AC-1
      title: Test-path glob config surface with validation and schema registration
      status: met
      tdd_phase: done
    - id: AC-2
      title: gitops helper lists working-tree paths dirty against HEAD
      status: met
      tdd_phase: done
    - id: AC-3
      title: --phase red refuses when non-test paths are dirty or nothing is dirty
      status: met
      tdd_phase: done
    - id: AC-4
      title: --phase green refuses until a non-test path is dirty
      status: cancelled
      tdd_phase: done
    - id: AC-5
      title: Diff-shape refusals overridable via --force --reason (human-only)
      status: met
      tdd_phase: done
    - id: AC-6
      title: Path universe excludes planning files and the verb's own entity write
      status: met
      tdd_phase: done
    - id: AC-7
      title: wf-tdd-cycle documents the red-first diff-shape gate semantics
      status: met
      tdd_phase: done
---

# M-0276 — Gate red-first ordering via a working-tree diff-shape check

## Goal

Close G-0252: make red-first test-then-code ordering mechanical. A
working-tree diff-shape check on the AC's `--phase red` promote refuses when
implementation is already dirty (or nothing is dirty at all) — proving
file-touch ordering without running tests or trusting a self-reported timeline.
The gate is **red-only** (D-0049); `--phase green` is not gated (see the
Decisions section for why).

## Context

Today the `--phase` promotes are metadata-only stamps; nothing checks the test
was written before the code. This milestone attaches a check to the live
`"" -> red` promote that M-0274 restores (hence the dependency). The check is a
glob-classified working-tree diff — deliberately not test execution or a SHA
trailer, both rejected in D-0047 (D-0038's cost/coupling and
"existence-not-relevance" objections). It proves ordering only; whether the
test is any good stays a review-time judgment (`wf-vacuity`, `wf-review-code`).

## Acceptance criteria

<!-- Prose shape; formalized via `aiwf add ac` at aiwfx-start-milestone.
     Each is observable behavior with a mechanical assertion. -->

1. A test-path glob config surface exists — `aiwf.yaml` key, schema, Tier-1
   validation, and completion wiring — pinned by config-parse/validation tests
   plus the completion-drift test. (The `areamatch` matcher is reused; the
   glob set itself is new — the areas `paths:` config is not the classifier.)
2. An `internal/gitops` helper returns the paths dirty in the working tree
   against HEAD — gitops-level test over a fixture repo.
3. `--phase red` refuses when any non-test path is dirty (naming the
   offending path[s]); succeeds when only test paths are dirty; refuses when
   nothing is dirty — verb test covering all three arms.
4. *(Cancelled — D-0049.)* The `--phase green` gate was dropped: it
   false-refuses test-only ACs (a test-only AC reaches green with no
   implementation change). The gate is red-only.
5. The red refusals are overridable via `--force --reason` (human-only per the
   existing sovereign rule) — verb test.
6. The inspected path universe excludes the verb's own entity write and
   planning files (`work/**`/`docs/**`) so a legitimate red promote does not
   self-refuse — verb test with a planning-file-dirty fixture.
7. `wf-tdd-cycle` documents the red-first diff-shape gate semantics —
   structural policy test (skill-edit backstop).

### AC-1 — Test-path glob config surface with validation and schema registration

A config surface in `aiwf.yaml` names the glob set that classifies a path as a
*test* path for the red-first diff-shape gate. It is a first-class key with a
typed config field (parsed and defaulted) and Tier-1 load-time validation that
rejects a malformed glob with an operator-facing error.

The `areamatch` matcher is reused to evaluate the globs; only the glob set is
new. The areas `paths:` config is deliberately *not* the classifier — a
separate key keeps test-classification independent of area assignment.

**Mechanical assertion:** config parse/validation tests over a fixture
`aiwf.yaml` (a valid set parses and round-trips; a malformed glob is rejected at
Tier-1), plus the schema anti-drift test (`TestSchema_EveryFieldHasDescription`)
forcing a registry description for the new `tdd.test_paths` key.

### AC-2 — gitops helper lists working-tree paths dirty against HEAD

An `internal/gitops` helper returns the set of paths that differ in the working
tree relative to `HEAD` — the raw material the phase gate classifies. It reports
both staged and unstaged changes so a `git add`-ed edit is not invisible to the
gate.

**Mechanical assertion:** a gitops-level test over a fixture git repo — seed a
dirty working tree (staged and unstaged edits), assert the helper returns
exactly the expected path set; assert the empty set on a clean tree.

### AC-3 — --phase red refuses when non-test paths are dirty or nothing is dirty

The `--phase red` promote consults the working-tree diff (AC-2) classified by
the test-path globs (AC-1) and:

- **refuses** when any non-test path is dirty, naming the offending path(s) —
  code-before-test violates red-first;
- **succeeds** when only test paths are dirty — the honest red state;
- **refuses** when nothing is dirty — a red phase with no test written is
  vacuous.

**Mechanical assertion:** a verb-level test driving `PromoteACPhase` over a
fixture repo across all three arms (non-test path dirty → refused, offending
path named; test-only dirty → allowed; clean → refused).

### AC-4 — --phase green refuses until a non-test path is dirty

**Cancelled (D-0049).** The green gate false-refuses test-only ACs: a regression
or characterization test reaches green with no implementation change, so
`--phase green` could never pass without `--force`. The gate is red-only;
`--phase green` is not gated. The original contract, no longer implemented, is
preserved below for the record.

The `--phase green` promote refuses when no non-test path is dirty (no
implementation exists to have turned the test green) and succeeds once a
non-test path is dirty. The check is stateless — it inspects the current diff
only, with no red-time snapshot to compare against.

**Mechanical assertion:** a verb-level test driving `PromoteACPhase` over both
arms (no non-test path dirty → refused; a non-test path dirty → allowed).

### AC-5 — Diff-shape refusals overridable via --force --reason (human-only)

The red diff-shape refusals (a non-test path already dirty, or nothing dirty)
are overridable with `--force --reason "<justification>"`. `--force` is a
sovereign, human-only act under the existing provenance rule — a non-human actor
is refused — so the escape hatch cannot be exercised by an automated actor.

**Mechanical assertion:** a verb-level test that a would-be-refused `--phase red`
promote succeeds under `force=true` (the gate runs only under `if !force`). The
human-only property is enforced at the provenance-decoration layer by the
existing coherence rule (`CoherenceRuleForceNonHuman`) and independently pinned
by the coherence tests; the escape hatch inherits it rather than re-checking it
in the verb.

### AC-6 — Path universe excludes planning files and the verb's own entity write

The path universe the gate inspects excludes planning/entity files — `work/**`
and `docs/**` — and the verb's own frontmatter write to the milestone spec.
Without this, a legitimate red promote (which itself rewrites the AC's
frontmatter and may sit alongside dirty planning prose) would self-refuse.

**Mechanical assertion:** a verb-level test with a planning-file-dirty fixture
asserting a red promote still succeeds when only `work/**` / `docs/**` paths are
dirty.

### AC-7 — wf-tdd-cycle documents the red-first diff-shape gate semantics

The `wf-tdd-cycle` skill documents the red-first diff-shape gate: that
`--phase red` requires test-only dirtiness, that `--phase green` is deliberately
not gated (a test-only AC reaches green with no implementation change), and that
the red refusal is `--force`-overridable. The RED step of the cycle references
the gate, and the section resolves the new-symbol / compile-stub case, so an
operator understands when a promote may refuse.

**Mechanical assertion:** a structural policy test under `internal/policies/`
asserting the gate semantics appear in the named section of the embedded
`wf-tdd-cycle` SKILL.md (skill-edit structural-test backstop).

## Constraints

- Zero friction on the honest path — the check validates existing working-tree
  state; no new commit, trailer, or flag to remember.
- Stack-agnostic — classification is glob-based, never toolchain-coupled (no
  `go test -list` equivalent).
- Stateless — the `--phase red` gate checks the current diff only; there is no
  red-time snapshot. Ordering comes from the red gate alone (red-only per
  D-0049), not a "grown since red" comparison the verb cannot compute.

## Design notes

- Implements D-0047 point 1; closes G-0252. Attaches to the live `"" -> red`
  promote M-0274 restores — hence `depends_on: M-0274`.
- Sizing watch: this spans config + gitops + verb + skill layers and sits at
  the upper edge of one milestone. If it reads as more than ~3 days once
  specced at start, split it — infrastructure (config surface + diff helper,
  ACs 1-2) then the guards + skill text (ACs 3-7). Kept whole for now per the
  ritual's spec-just-in-time preference.
- Edge cases to pin at start (each a potential false-positive on a hot path):
  staged vs unstaged changes, intermediate commits between red and green,
  renames, and shared fixtures under non-test paths.

## Surfaces touched

- `internal/verb/ac.go` (`PromoteACPhase`)
- `internal/gitops/` (working-tree-vs-HEAD dirty-path helper)
- `internal/config/` (test-path glob config surface) + `aiwf.yaml`
- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md`

## Out of scope

- The seeding fix (M-0274, prerequisite) and plan-time AC content (M-0275).
- Running the test suite or verifying the test actually fails (D-0047 out of
  scope — that judgment stays at review).
- Any content-immutability check on the test file between red and green.

## Dependencies

- M-0274 — restores the live `"" -> red` promote this gate attaches to.
  Without it the gate has no live event to guard.

## References

- G-0252 — the gap this milestone closes.
- D-0047 — Contract-first AC timing and red-first ordering enforcement.
- D-0038 — the boundary between mechanizable structural claims and review-time
  judgment this gate respects.

## Work log

### AC-1 — Test-path glob config surface

Added `tdd.test_paths` ([]string) with Tier-1 glob validation routed through the
`areamatch` SSOT (empty, whitespace-dirty, and malformed globs are hard load
errors naming the entry) and a schema field-description registry entry; the key
is documented in the `design-decisions.md` config table for
discoverability. · commit 9de44c4f · tests:
`TestConfig_TDDTestPaths_ParsesAndValidates` (5 cases) +
`TestSchema_IncludesTDDTestPaths`.

### AC-2 — gitops dirty-path helper

Added `DirtyPaths(ctx, workdir)` in `internal/gitops`: the union of tracked
changes vs HEAD (staged + unstaged) and untracked non-ignored files, sorted and
deduplicated. Untracked files are deliberately included so a newly-written,
not-yet-added test file registers as dirty. · commit daebc9f0 · tests:
`TestDirtyPaths` (clean / unstaged-modify / staged-new / untracked-new /
ignored-excluded) + `TestDirtyPaths_NonRepoErrors`.

### AC-3 — --phase red diff-shape guard

Wired an opt-in red-first gate into `PromoteACPhase`
(`requireDiffShapeForPhasePromote` in `internal/verb/promote_phase_gate.go`):
when `tdd.test_paths` is configured, an unforced `--phase red` classifies the
working-tree dirty paths and refuses on any non-test dirtiness (naming the
paths) or on a wholly-clean tree; inactive when unconfigured, so existing
callers and stress scenarios are untouched. · commit 99c694c4 · tests:
`TestPromoteACPhase_RedGate_DiffShape` (test-only pass / non-test refuse+name /
nothing-dirty refuse / unconfigured-inactive).

### AC-4 — --phase green diff-shape guard

Extended `requireDiffShapeForPhasePromote` to gate `--phase green` alongside red:
green refuses when no non-test (implementation) path is dirty and succeeds once
one is. Stateless — the current diff only, no red-time snapshot. · commit
978cb02f · tests: `TestPromoteACPhase_GreenGate_DiffShape` (no-impl refuse /
impl-present succeed).

### AC-5 — --force bypasses the gate

Pinned that `force=true` skips the diff-shape gate (it runs only under
`if !force`): a red with a dirty non-test path and a green with no non-test path
each land under `--force`. Test-only — the force-bypass already existed, so the
RED was mutation-confirmed (flipping `!force`→`true` fails both arms); the
human-only property is the existing `CoherenceRuleForceNonHuman`. · commit
99bd380e · tests: `TestPromoteACPhase_ForceBypassesDiffShapeGate` (red / green).

### AC-6 — planning-file exclusion

Added `isPlanningPath` and a `continue` in the classify loop so `work/**` and
`docs/**` paths (planning/entity + docs, including the verb's own frontmatter
write) are excluded from the dirty universe — a legitimate red promote beside
dirty planning prose no longer self-refuses. · commit 77b1a604 · tests:
`TestPromoteACPhase_RedGate_ExcludesPlanningPaths` (docs-path / work-path).

### AC-7 — wf-tdd-cycle documents the red-first diff-shape gate semantics

Added a named "The red/green diff-shape gate" section to the embedded
`wf-tdd-cycle` skill (`--phase red` wants test-only dirtiness, `--phase green`
wants an implementation path, both `--force`-overridable, opt-in) with
references from the RED and GREEN steps. Shipped surface — canonical
placeholders, no real ids. · commit 411664ac · tests:
`TestM0276_TddCycleDocumentsDiffShapeGate` (structural, section-scoped;
skill-edit backstop).

### Wrap review — red-only rework

The independent wrap design review found the green gate false-refuses test-only
ACs. Reworked to a **red-only** gate: recorded D-0049, removed the green branch
from the verb and its tests, de-greened the `wf-tdd-cycle` skill, cancelled
AC-4, and narrowed AC-5/AC-7 to the red gate. From the same review, reconciled
the skill's strict-red guidance with the gate for the new-symbol / compile-stub
case. · commits fb134633 (skill: ordering-red vs semantic-red) · 6e0c657b
(verb: red-only) · c9e07335 (skill: red-only) · D-0049.

## Decisions made during implementation

### Opt-in gate activation (empty `tdd.test_paths` → inactive)

The gate is opt-in: with no test-path globs configured it is a no-op. Forced by
D-0047's constraints — a baked-in default glob set would be language-specific
(violating stack-agnosticism) and would start refusing every existing consumer's
`--phase` promotes (violating zero-friction). A project activates the gate by
declaring its own test globs. This repo ships the gate but does **not** activate
it (no `tdd.test_paths` in its own `aiwf.yaml`); dogfooding is a separate
operator decision, deferred to epic wrap.

### D-0049 — the gate is red-only (green gate dropped)

Implementation and the wrap design review found the green half unsound: it
false-refuses test-only ACs (a regression/characterization test reaches green
with no implementation change, so `--phase green` could never pass without
`--force`), and its only unique catch is indistinguishable from a legitimate
test-only AC. The red gate carries the whole ordering guarantee. Recorded as
D-0049; AC-4 cancelled; AC-5/AC-7 narrowed to red.

### The compile-stub tension (documented, not gated)

For a new symbol in a typed language the failing test needs a minimal stub to
reach its assertion — a non-test change the red gate would reject. Rather than
force `--force`, `wf-tdd-cycle` now distinguishes ordering-red (the test exists,
the code does not — a compile failure counts, per the Three Laws of TDD) from
semantic-red (the assertion fails once it compiles): promote `--phase red` at
the compile-failure moment, before the stub. No mechanical fix exists —
distinguishing an honest stub from real implementation is the
existence-not-relevance wall D-0047 / D-0038 refuse to climb.

## Validation

- `go test -race ./...` — green (71 packages) at the wrap review.
- `make check-fast` — exit 0 (vet + lint + full test suite), re-run green after
  the red-only rework.
- `make coverage-gate` — exit 0; every changed line covered or `//coverage:ignore`'d.
- `aiwf check` — 0 error-severity findings on M-0276.
- Independent two-lens review (code-quality + design-quality, fresh-context) —
  both approved; the design review's green-gate finding drove D-0049.

## Deferrals

- **G-0445** — the gate hardcodes a `docs/` exclusion that may be real
  implementation in a consumer repo (a false-pass at `--phase red`); make the
  excluded set configurable or scope it to `work/` only.

## Reviewer notes

- The `DirtyPaths`-error branch in `requireDiffShapeForPhasePromote` is
  `//coverage:ignore`'d — genuinely unreachable (promoting an AC requires a
  committed milestone, so HEAD always exists; the gitops error path itself is
  tested at the gitops layer).
- `pathMatchesAnyGlob` swallows `areamatch.Match`'s error — safe because
  `tdd.test_paths` globs are Tier-1 validated at config load, so a pattern error
  cannot reach it.
- The gate assumes the aiwf project root equals the git repo root (consistent
  with all of aiwf's gitops helpers); a nested-project layout would drift — a
  whole-design property, out of scope here.
- D-0047's "structural fact about which paths changed and when" prose slightly
  over-claims: committed changes are invisible to the gate, so the guarantee is
  "working-tree-dirty ordering at the red-promote instant," not full
  touch-ordering. Honest overall (the what-it-doesn't-prove boundary is stated);
  noted for a future D-0047 prose tidy.
