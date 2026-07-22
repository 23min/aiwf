---
id: M-0276
title: Gate red-first ordering via a working-tree diff-shape check
status: draft
parent: E-0070
depends_on:
    - M-0274
tdd: required
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
