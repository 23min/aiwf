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

### AC-1 — `CLAUDE.md` gains a `## Test discipline` section under *Go conventions*

The new section documents:
- `TestMain` per package, `os.Setenv` for the GIT identity (immutable; never `t.Setenv` under `t.Parallel`).
- `t.Parallel()` first-line on every test that does not legitimately need serial execution; the skip-list rationale lives as a comment in the package's `setup_test.go`.
- `sync.Once` for shared expensive fixtures (the live-repo `*Tree` is the canonical example); `// do not mutate` comment at the loader site.
- `-race -parallel 8` cap in `Makefile` and the GitHub workflows; rationale (macOS git-fan-out flake at higher parallelism); the cap lives in three files and changes uniformly.
- `setup_test.go` is the filename. Conventional, mechanical, AI-discoverable.

A contributor reading this section can write a new test file in the right shape without prior knowledge of the spike or M-0091/M-0092. The section cross-references the policy test by code-path (`internal/policies/test_setup_presence.go` or equivalent).

### AC-2 — `PolicyTestSetupPresent` (or equivalently named) in `internal/policies/` asserts every `internal/*` test-bearing package has a `setup_test.go` with a `TestMain` declaration

The policy is a Go test under `internal/policies/` that:
1. Walks every directory under `internal/` containing at least one `*_test.go` file.
2. For each such directory, parses files with `go/parser` and asserts that `setup_test.go` exists and contains a top-level `func TestMain(m *testing.M)` declaration.
3. AST-level check (per the epic's design decisions table — "presence of `setup_test.go` is a reasonable proxy"). Substring assertions over flat source are explicitly rejected per CLAUDE.md *Substring assertions are not structural assertions*.
4. Fails CI with a finding pointing at the offending package directory and citing the `## Test discipline` section.

Scope is `internal/*`; `cmd/aiwf/` is intentionally excluded because the per-file audit shape there is different (single package, mixed parallelism). If future evidence shows `cmd/aiwf/` should also be guarded, that's a follow-up gap; this milestone leaves it alone.

### AC-3 — G-0097 promoted to `addressed` and the closing trailer cites E-0025

After AC-1 and AC-2 land and the epic's success criteria are met at wrap, `aiwf promote G-0097 addressed` runs and the trailer references E-0025. This AC is mechanical — the promote happens as part of the wrap ritual for the epic, not this milestone — but recording it here keeps the closing thread visible.

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
