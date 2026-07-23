---
id: M-0276
title: Gate red-first ordering via a working-tree diff-shape check
status: in_progress
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
      status: open
      tdd_phase: red
    - id: AC-3
      title: --phase red refuses when non-test paths are dirty or nothing is dirty
      status: open
    - id: AC-4
      title: --phase green refuses until a non-test path is dirty
      status: open
    - id: AC-5
      title: Diff-shape refusals overridable via --force --reason (human-only)
      status: open
    - id: AC-6
      title: Path universe excludes planning files and the verb's own entity write
      status: open
    - id: AC-7
      title: wf-tdd-cycle documents the red/green diff-shape gate semantics
      status: open
---

# M-0276 — Gate red-first ordering via a working-tree diff-shape check

## Goal

Close G-0252: make red-first test-then-code ordering mechanical. A
working-tree diff-shape check on the AC's TDD-phase promotes refuses
`--phase red` when implementation is already dirty and `--phase green` when no
implementation is dirty yet — proving file-touch ordering without running
tests or trusting a self-reported timeline.

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
4. `--phase green` refuses when no non-test path is dirty and succeeds once a
   non-test path is dirty — verb test, both arms.
5. Both refusals are overridable via `--force --reason` (human-only per the
   existing sovereign rule) — verb test.
6. The inspected path universe excludes the verb's own entity write and
   planning files (`work/**`/`docs/**`) so a legitimate red promote does not
   self-refuse — verb test with a planning-file-dirty fixture.
7. `wf-tdd-cycle` documents the red/green diff-shape gate semantics —
   structural policy test (skill-edit backstop).

### AC-1 — Test-path glob config surface with validation and schema registration

A config surface in `aiwf.yaml` names the glob set that classifies a path as a
*test* path for the red/green diff-shape gate. It is a first-class key with a
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

The `--phase green` promote refuses when no non-test path is dirty (no
implementation exists to have turned the test green) and succeeds once a
non-test path is dirty. The check is stateless — it inspects the current diff
only, with no red-time snapshot to compare against.

**Mechanical assertion:** a verb-level test driving `PromoteACPhase` over both
arms (no non-test path dirty → refused; a non-test path dirty → allowed).

### AC-5 — Diff-shape refusals overridable via --force --reason (human-only)

Both diff-shape refusals (red and green) are overridable with `--force
--reason "<justification>"`. `--force` is a sovereign, human-only act under the
existing provenance rule — a non-human actor is refused — so the escape hatch
cannot be exercised by an automated actor.

**Mechanical assertion:** a verb-level test asserting a would-be-refused promote
succeeds under `--force --reason`, and that `--force` from a non-human actor is
refused.

### AC-6 — Path universe excludes planning files and the verb's own entity write

The path universe the gate inspects excludes planning/entity files — `work/**`
and `docs/**` — and the verb's own frontmatter write to the milestone spec.
Without this, a legitimate red promote (which itself rewrites the AC's
frontmatter and may sit alongside dirty planning prose) would self-refuse.

**Mechanical assertion:** a verb-level test with a planning-file-dirty fixture
asserting a red promote still succeeds when only `work/**` / `docs/**` paths are
dirty.

### AC-7 — wf-tdd-cycle documents the red/green diff-shape gate semantics

The `wf-tdd-cycle` skill documents the red/green diff-shape gate: that
`--phase red` requires test-only dirtiness, `--phase green` requires
implementation dirtiness, and both are `--force`-overridable. The RED and GREEN
steps of the cycle reference the gate so an operator understands why a promote
may refuse.

**Mechanical assertion:** a structural policy test under `internal/policies/`
asserting the gate semantics appear in the named section of the embedded
`wf-tdd-cycle` SKILL.md (skill-edit structural-test backstop).

## Constraints

- Zero friction on the honest path — the check validates existing working-tree
  state; no new commit, trailer, or flag to remember.
- Stack-agnostic — classification is glob-based, never toolchain-coupled (no
  `go test -list` equivalent).
- Stateless — `--phase green` checks the current diff only; there is no
  red-time snapshot. Ordering comes from the pair of gates, not a
  "grown since red" comparison the verb cannot compute.

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
