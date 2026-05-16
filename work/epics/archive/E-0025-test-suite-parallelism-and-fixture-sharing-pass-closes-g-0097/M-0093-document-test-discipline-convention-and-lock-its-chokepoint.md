---
id: M-0093
title: Document test-discipline convention and lock its chokepoint
status: done
parent: E-0025
depends_on:
    - M-0091
    - M-0092
tdd: none
acs:
    - id: AC-1
      title: CLAUDE.md gains a Test discipline section under Go conventions
      status: met
    - id: AC-2
      title: policy test asserts each internal/* test pkg has setup_test.go and TestMain
      status: met
    - id: AC-3
      title: G-0097 promoted to addressed and the closing trailer cites E-0025
      status: met
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

### AC-1 — CLAUDE.md `### Test discipline` section (in the wrap commit, `8eb05a7`)

New `### Test discipline` section under `## Go conventions` covers the five load-bearing rules: setup_test.go per package, `t.Parallel()` first-line on parallelizable tests, serial skip-list documented in `setup_test.go`'s comment block, `sync.Once` for shared expensive fixtures, and the `-race -parallel 8` cap. Each rule names its mechanical chokepoint (or the package-local skip-list convention for serial tests). The section cross-references `internal/policies/test_setup_presence.go` and `internal/policies/race_parallel_cap.go` so a reader of the doc lands on the enforcement code.

Two rows added to the *What's enforced and where* table at the bottom of *Go conventions*: the setup-presence chokepoint that lands in AC-2 of this milestone, and the race-cap policy that landed in M-0091/AC-1 (was previously documented in the section body but not surfaced in the chokepoint table). Both rows mark "Blocking via CI test."

Structural-assertion evidence: `internal/policies/claude_md_test_discipline.go` walks the heading hierarchy of CLAUDE.md and asserts `### Test discipline` exists under `## Go conventions`. A future edit that moves, renames, or accidentally deletes the section fires the policy at the next CI run.

### AC-2 — `internal/policies/test_setup_presence.go` (in the wrap commit)

AST-level walk via `go/parser` of every directory under `internal/` containing at least one `*_test.go` file (testdata/ subtrees skipped). For each such directory, the policy asserts that `setup_test.go` exists and that the file contains a top-level `func TestMain(m *testing.M)` declaration. The TestMain match anchors on: name `TestMain`, no receiver, exactly one parameter, parameter type `*testing.M` (selector-expr `testing.M` wrapped in a star expr). Substring-grep alternatives were rejected per CLAUDE.md *Substring assertions are not structural assertions* — a flat grep for `func TestMain(` would match a function inside a build-tagged file or a misnamed helper.

Scope is `internal/*` per spec. `cmd/aiwf/` has a per-file audit shape (M-0092's `setup_test.go` skip-list captures captureStdout/Stderr/Run-caller serialization as a finer-grained discipline than a presence check could express). Extending the policy to `cmd/aiwf/` is a future gap if real friction surfaces.

Helper: `hasTestMainDecl(*ast.File) bool` for the TestMain signature check; isolating it from the walk keeps the AST inspection unit-testable if a future regression motivates explicit cases.

### AC-3 — G-0097 promoted to addressed

`aiwf promote G-0097 addressed --by E-0025 --reason "Closed via E-0025 — M-0091 (internal/* parallelism, ~2.2x speedup) + M-0092 (cmd/aiwf/* parallelism, 47% wall-time reduction; AC-4 deferred to G-0125) + M-0093 (test-discipline convention in CLAUDE.md + setup_test.go presence chokepoint in internal/policies/)."`. Landed as a separate verb commit on this branch (kernel rule: gap-promote-to-terminal requires `--by <entity-id>` or `--by-commit <sha>` to satisfy `gap-resolved-has-resolver`).

## Decisions made during implementation

- **Belt-and-suspenders for AC-1's doc-shape assertion.** AC-1 is doc-shape; the spec body implies the AC-2 policy alone is the mechanical chokepoint (it enforces what AC-1 documents). Per CLAUDE.md *AC promotion requires mechanical evidence*, doc-shape ACs benefit from a structural assertion on the named section. Added `PolicyClaudeMdTestDisciplineSection` as a small heading-hierarchy walk. Catches the inverse failure mode: the section is moved/renamed/deleted while AC-2's chokepoint stays intact. Trivial cost, useful belt.
- **G-0097 closed via `--by E-0025`, inside M-0093.** The spec body's AC-3 hinted that the promote could land at epic wrap (`aiwfx-wrap-epic`). Doing it inside M-0093 keeps AC-3's terminal state local — the milestone wraps `done` without needing AC-3 to stay `open` pending the epic wrap. The closing trailer references the full E-0025 arc (M-0091, M-0092 with AC-4 deferred, M-0093); future readers of `aiwf history G-0097` see the resolution chain.

## Validation

### Build + lint + check

- `go build ./cmd/aiwf` — green.
- `golangci-lint run` — 0 issues.
- `aiwf check` — 0 errors (warning: G-0097 awaits archive sweep, expected — see Reviewer notes).
- `go test ./...` — all packages pass.

### Policy tests (the AC-2 + AC-1 chokepoints, in `internal/policies/`)

- `TestPolicy_TestSetupPresence` — passes (every `internal/*` test-bearing package has `setup_test.go` with `TestMain`).
- `TestPolicy_ClaudeMdTestDisciplineSection` — passes (`### Test discipline` section exists under `## Go conventions` in CLAUDE.md).
- Both run as part of `go test ./internal/policies/`; the discovered packages cover everything M-0091 + M-0092 landed.

## Deferrals

- (none)

## Reviewer notes

- **AC-3 done in-milestone, not at epic wrap.** The spec hinted the G-0097 promote could land at `aiwfx-wrap-epic` time; this milestone took it in-flight so AC-3 closes inside M-0093 (avoids leaving AC-3 `open` while M-0093 is `done`). Per the kernel's `gap-resolved-has-resolver` rule, the promote required `--by E-0025`; the trailer references the epic and the rationale text names all three milestones.
- **G-0097 archive sweep is now pending.** `aiwf check` warns `terminal-entity-not-archived` for G-0097. Expected — the gap is terminal, the sweep is opt-in. Either run `aiwf archive --apply` before/after E-0025 wraps, or let the next routine sweep pick it up.
- **`cmd/aiwf/` not scoped.** Per spec the policy is `internal/*`-only. The cmd-side audit shape (captureStdout/Stderr/Run-caller serialization, integration_g37 file-level skip) is finer-grained than a presence check can express; that audit lives in `cmd/aiwf/setup_test.go`'s comment block as reviewer-enforced discipline. If a future contributor adds a test file in `cmd/aiwf/` that omits the setup_test.go convention, M-0092's skip-list won't catch it — but the test suite would fail under -race for a different reason (concurrent stdout capture), making the omission self-correcting. Worth a follow-up gap if it ever surfaces in the field.
- **Race-cap policy retroactively added to the chokepoint table.** `internal/policies/race_parallel_cap.go` landed in M-0091/AC-1 but wasn't called out in the *What's enforced and where* table. Adding it now alongside the new setup-presence policy keeps the table comprehensive.
- **M-0091's spike branch is unrecoverable.** The original `spike/test-parallel` branch (referenced in M-0091's spec) was deleted before this epic started. M-0091 reconstructed the pattern from prose. Future ritual lesson: don't delete a spike branch until the wrap-merge confirms.

### AC-1 — CLAUDE.md gains a Test discipline section under Go conventions

The new section documents:
- `TestMain` per package, `os.Setenv` for the GIT identity (immutable; never `t.Setenv` under `t.Parallel`).
- `t.Parallel()` first-line on every test that does not legitimately need serial execution; the skip-list rationale lives as a comment in the package's `setup_test.go`.
- `sync.Once` for shared expensive fixtures (the live-repo `*Tree` is the canonical example); `// do not mutate` comment at the loader site.
- `-race -parallel 8` cap in `Makefile` and the GitHub workflows; rationale (macOS git-fan-out flake at higher parallelism); the cap lives in three files and changes uniformly.
- `setup_test.go` is the filename. Conventional, mechanical, AI-discoverable.

A contributor reading this section can write a new test file in the right shape without prior knowledge of the spike or M-0091/M-0092. The section cross-references the policy test by code-path (`internal/policies/test_setup_presence.go` or equivalent).

### AC-2 — policy test asserts each internal/* test pkg has setup_test.go and TestMain

The policy is a Go test under `internal/policies/` that:

1. Walks every directory under `internal/` containing at least one `*_test.go` file.
2. For each such directory, parses files with `go/parser` and asserts that `setup_test.go` exists and contains a top-level `func TestMain(m *testing.M)` declaration.
3. AST-level check (per the epic's design decisions table — "presence of `setup_test.go` is a reasonable proxy"). Substring assertions over flat source are explicitly rejected per CLAUDE.md *Substring assertions are not structural assertions*.
4. Fails CI with a finding pointing at the offending package directory and citing the `## Test discipline` section.

Scope is `internal/*`; `cmd/aiwf/` is intentionally excluded because the per-file audit shape there is different (single package, mixed parallelism). If future evidence shows `cmd/aiwf/` should also be guarded, that's a follow-up gap; this milestone leaves it alone.

### AC-3 — G-0097 promoted to addressed and the closing trailer cites E-0025

After AC-1 and AC-2 land and the epic's success criteria are met at wrap, `aiwf promote G-0097 addressed` runs and the trailer references E-0025. This AC is mechanical — the promote happens as part of the wrap ritual for the epic, not this milestone — but recording it here keeps the closing thread visible.

