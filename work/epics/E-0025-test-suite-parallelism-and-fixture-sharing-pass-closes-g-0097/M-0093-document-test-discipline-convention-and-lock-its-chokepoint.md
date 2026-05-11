---
id: M-0093
title: Document test-discipline convention and lock its chokepoint
status: draft
parent: E-0025
depends_on:
    - M-0091
    - M-0092
tdd: none
---

# M-0093 — Document test-discipline convention and lock its chokepoint

## Goal

Capture the parallel-by-default + shared-fixture convention in `CLAUDE.md` as a new `## Test discipline` section, and ship a `setup_test.go`-presence policy test under `internal/policies/` in the same commit so the rule and its mechanical enforcement land together. After this milestone, a contributor or AI assistant authoring a new test file picks up the convention from the playbook, and any future package that omits `setup_test.go` fails CI.

## Context

M-0091 and M-0092 establish the pattern across the existing codebase. The pattern's value compounds only if future test files adopt it by default; that needs a written convention and a mechanical chokepoint. Per the kernel's "framework correctness must not depend on the LLM's behavior" rule, the policy test is the chokepoint — `CLAUDE.md` is the human-facing rule that the chokepoint enforces.

This milestone is single-commit: the documentation change and the policy test ship together. The reason is symmetric — a `CLAUDE.md` rule without enforcement is advisory drift waiting to happen; a policy test without a `CLAUDE.md` entry leaves a contributor seeing a CI failure with no explanation.

## Acceptance criteria

(ACs allocated via `aiwf add ac`; bodies follow below.)

## Constraints

- **One commit for the milestone**, carrying `aiwf-verb`, `aiwf-entity: M-0093`, and `aiwf-actor` trailers. `CLAUDE.md` change and policy test ship in the same commit.
- **Policy test is AST-level, not substring.** Per CLAUDE.md, a substring grep for `func TestMain(` would pass even if the function lived under a `// +build ignore` line or in a misnamed file. `go/parser` is the right primitive.
- **Scope is `internal/*`, not the whole module.** The chokepoint is targeted; widening is a future decision.
- **No per-function `t.Parallel()` audit.** Per the epic's design decisions table — too much friction for marginal value. Presence of `setup_test.go` is the chokepoint; per-function discipline lives in `CLAUDE.md` and review.

## Design notes

- The policy lives in a new file under `internal/policies/` (e.g., `test_setup_presence.go` plus `test_setup_presence_test.go`), following the package's existing pattern (one policy per file).
- The AST walk uses `go/parser` with `parser.PackageClauseOnly | parser.ParseComments` initially, switching to `parser.AllErrors` only if needed for discovery of edge cases.
- If a future contributor wants to opt a package out (e.g., a future `internal/foo/` that has tests but legitimately needs no `TestMain` — none anticipated today), the policy gains an allowlist with a one-line rationale per the kernel's "rule-with-allowlist" pattern.

## Surfaces touched

- `CLAUDE.md` — new `## Test discipline` subsection under *Go conventions*
- `internal/policies/test_setup_presence.go` (or equivalent) — policy implementation
- `internal/policies/test_setup_presence_test.go` (or equivalent) — the running policy test

## Out of scope

- Extending the policy to `cmd/aiwf/`. Future gap if real friction surfaces.
- A per-function `t.Parallel()` audit.
- Shipping the convention to downstream consumers via a `wf-rituals` skill. Consumers copy the `CLAUDE.md` section into their own playbook or wait for an opt-in skill in a follow-up.

## Dependencies

- **M-0091** — internal/* rollout. Without it, the policy test fails CI on the first run.
- **M-0092** — cmd/aiwf/ rollout. Not a hard dep for the policy (scoped to `internal/*`), but the convention's wording in `CLAUDE.md` references the cmd/aiwf-specific audit shape, which M-0092 finalizes.

## References

- E-0025 epic spec.
- M-0091 and M-0092 — the convention this milestone documents and pins.
- CLAUDE.md *What's enforced and where* — this milestone adds a new row to the chokepoint table.

## Work log

(filled during implementation)

## Decisions made during implementation

- (none yet)

## Validation

(pasted at wrap: `aiwf check` clean; `go test ./internal/policies/...` shows the new policy test passing; the convention section in `CLAUDE.md` reviewed for AI-discoverability)

## Deferrals

- (none yet)

## Reviewer notes

- (none yet)
