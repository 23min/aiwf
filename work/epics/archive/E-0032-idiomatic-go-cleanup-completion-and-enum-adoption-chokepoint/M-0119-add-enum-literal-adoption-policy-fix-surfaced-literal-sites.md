---
id: M-0119
title: Add enum_literal_adoption policy; fix surfaced literal sites
status: done
parent: E-0032
tdd: required
acs:
    - id: AC-1
      title: Policy auto-enumerates Status constants from internal/entity/entity.go
      status: met
      tdd_phase: done
    - id: AC-2
      title: Policy fires on == or != BinaryExpr with status literal
      status: met
      tdd_phase: done
    - id: AC-3
      title: Policy fires on switch/case clauses with status literals
      status: met
      tdd_phase: done
    - id: AC-4
      title: //enums:ignore line-suffix allowlist suppresses violations
      status: met
      tdd_phase: done
    - id: AC-5
      title: Surfaced literal sites fixed and CLAUDE.md row added
      status: met
      tdd_phase: done
---
## Goal

Write `internal/policies/enum_literal_adoption.go` — an AST-based policy that enumerates `Status*` constants from [`internal/entity/entity.go`](../../../internal/entity/entity.go) and reports comparison-site string-literal use in non-test code outside the entity package. Fix the three surfaced literal sites ([`transition.go:198`](../../../internal/entity/transition.go), [`entity.go:100`](../../../internal/entity/entity.go), `:477`). Closes G-0126.

## Context

[G-0126](../../gaps/G-0126-enum-constant-adoption-has-no-mechanical-chokepoint.md) captured the missing chokepoint. Without this policy, future contributors (human or LLM) can introduce literal-vs-constant drift and pass CI — exactly the failure mode the kernel rule "framework correctness must not depend on LLM behavior" forbids. The hardcoded `ac.Status == "open"` at `transition.go:198` survived through G-0107's refactor (E-0032's broader context), proving review alone is not catching this drift class.

## Approach

New file [`internal/policies/enum_literal_adoption.go`](../../../internal/policies/) modeled on [`internal/policies/test_setup_presence.go`](../../../internal/policies/test_setup_presence.go). Uses `go/ast` and `go/parser` to:

1. **Enumerate `Status*` constants** from `internal/entity/entity.go` — build a map from literal value (e.g., `"open"`) → constant name (e.g., `entity.StatusOpen`). Done at policy-run time so adding a new status auto-extends the check with no second source of truth.
2. **Walk every `.go` file** outside `internal/entity/` and outside `*_test.go`. For each `*ast.BinaryExpr` with op `==` or `!=` and a string-literal operand matching a known value, emit a finding: `file:line: literal "open" should be entity.StatusOpen`. Same shape for `*ast.SwitchStmt`/`*ast.CaseClause` literal cases.
3. **Allowlist via `//enums:ignore <reason>`** line-suffix comments — read the file's comment positions and skip findings on annotated lines. Matches the existing `//coverage:ignore` shape.

**Fix work in the same commit set:**
- `internal/entity/transition.go:198` → `entity.StatusOpen` (file is inside `internal/entity/` so the policy doesn't fire on it, but the literal still gets fixed for codebase consistency).
- `internal/entity/entity.go:100,:477` → use constants for the FSM tables.

**CLAUDE.md update:** new row in the "What's enforced and where" table: `Closed-set string constants are used at comparison sites` / chokepoint `internal/policies/enum_literal_adoption.go` / blocking via CI test.

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac <M-id> --title "..."`. -->

## Work log

### AC-1 — Policy auto-enumerates Status constants from internal/entity/entity.go

`enumerateEntityStatusConstants(root)` AST-parses `internal/entity/entity.go` at policy-run time and builds a `map[stringValue]constName` for every top-level `const` whose name starts with `Status`. Adding a new status constant auto-extends the rule with no second source of truth. The test `TestEnumerateEntityStatusConstants_LiveTree` spot-checks the canonical entries (Open / Active / Done / Cancelled / Draft / Met / Deferred / Addressed) against the real entity.go.

Commit `39c96ff9` (combined with AC-2/3/4) · 1 test passes

### AC-2 — Policy fires on `==`/`!=` BinaryExpr with status literal

`ast.Inspect` walks `*ast.BinaryExpr` nodes. When the op is `token.EQL` or `token.NEQ` and either operand is a `*ast.BasicLit` of `token.STRING` kind whose unquoted value is in the enumerated map, a Violation is emitted pointing at the literal's position. Synthetic-input tests drive a tempdir tree with a synthetic `internal/entity/entity.go` (carrying Status constants for the enumerator to read) plus a drift file under `internal/cli/<pkg>/drift.go`; both the `==` and `!=` branches are covered.

Commit `39c96ff9` (combined) · 2 tests pass

### AC-3 — Policy fires on switch/case clauses with status literals

Same `ast.Inspect` walk hits `*ast.CaseClause`. Each case expression is checked the same way as the BinaryExpr operands. Synthetic-input test pins the fire branch.

Commit `39c96ff9` (combined) · 1 test passes

### AC-4 — `//enums:ignore` line-suffix allowlist suppresses violations

`collectIgnoredLines(f, fset)` reads `f.Comments` (populated by `parser.ParseComments`) and builds a `map[line]bool` of lines carrying a trailing `//enums:ignore` comment. The Violation emitter checks this set before appending. Matches the `//coverage:ignore` convention. The reason after the directive is required prose so the suppression carries audit context (the parser doesn't enforce the reason — that's reviewer discipline, matching `//coverage:ignore`'s shape). Two synthetic-input tests cover the BinaryExpr and switch/case suppression paths plus a no-false-positive test on `s == entity.StatusOpen`.

Commit `39c96ff9` (combined) · 3 tests pass

### AC-5 — Surfaced literal sites fixed and CLAUDE.md row added

22 sites updated across 4 files:

- **internal/entity/transition.go:198** — the G-0126-named drift: `ac.Status == "open"` → `ac.Status == StatusOpen`.
- **internal/entity/entity.go** — 6 FSM-table literal lists (lines 100, 449, 455, 466, 477, 492, 503). The spec named :100 and :477; the other four schema tables carried the same drift shape and were fixed for codebase consistency. The policy doesn't fire on internal/entity/ files (they own the constants), but the literal-vs-constant choice is the same hygienic concern.
- **internal/render/glyph.go** — 4 switch/case lines (the `StatusGlyph` palette: ✓ / → / ○ / ✗). New `import "github.com/23min/aiwf/internal/entity"`.
- **internal/verb/auditonly.go** — 2 case lines. `isKnownACStatus` uses `entity.StatusOpen/Met/Deferred/Cancelled`. `isKnownTDDPhase` uses `entity.TDDPhaseRed/Green/Refactor/Done` — TDDPhase is out of the M-0119 denylist scope but using the constants here side-steps a Status* false-positive on the literal `"done"` (StatusDone and TDDPhaseDone share the value).

CI wiring: `internal/policies/policies_test.go` gains `TestPolicy_EnumLiteralAdoption`, which calls the policy against the live tree. With all 22 sites fixed, this passes — and any future drift fails CI.

CLAUDE.md: the "What's enforced and where" table gains a row naming the policy as the chokepoint for closed-set Status* constant adoption at comparison sites.

Commit `34175e75` · 0 policy violations on the live tree; `make test` green; G-0126 closed.

## Decisions made during implementation

- **Scope: Status* only.** Per spec — TDDPhase*, Kind*, trailer keys, scope events are out of scope. The seed denylist is exactly what's named in the M-0119 spec's *Out of scope* note. Drift in those other categories will surface their own gaps when it bites.

- **Skip both internal/entity/ AND internal/policies/.** Skipping `internal/entity/` was in the spec (the package owns the constants). Skipping `internal/policies/` is additional — this policy file's own docstring carries example literals as content ("Concretely: `ac.Status == \"open\"` violates..."), and the synthetic-input test fixtures intentionally contain status literals. Both are content, not drift.

- **TDDPhase constants used at auditonly.go's `isKnownTDDPhase` even though they're out of scope.** Sidesteps the StatusDone == "done" == TDDPhaseDone aliasing that would otherwise have made the policy flag that one line.

- **Six FSM tables in entity.go fixed, not just the spec-named two.** The spec named entity.go:100 and :477. The other four (epics, milestones, ADRs, decisions, contracts schema entries) carried the same drift. Fixing all six in the same commit is the consistency choice — leaving four with raw literals while fixing two would have looked like negligence.

- **AC-1+2+3+4 land in one commit, AC-5 in a separate commit.** The first commit creates the policy code + synthetic-input tests but does NOT wire into `policies_test.go` (which would fail CI with 22 unfixed live-tree violations). The second commit fixes the sites AND wires the policy. Two commits keeps the diff readable: the policy + its tests are one logical unit, the cleanup of existing drift is another.

## Validation

- `make test`: **all green** (~7 min on macOS with `-parallel 8`).
- `aiwf check`: **0 errors**, 20 warnings (all pre-existing advisory — same set as M-0118 wrap).
- `PolicyEnumLiteralAdoption` on the live tree: **0 violations**.
- All 5 ACs at `status: met`, `tdd_phase: done`.
- 2 commits comprise the milestone delivery: `39c96ff9` (policy + tests), `34175e75` (live-tree fixes + CI wiring + docs).
- G-0126 closed (the gap entity will be transitioned to `addressed` after this milestone wraps, per the kernel's separate-promote convention).

## Deferrals

- **TDDPhase*, Kind*, trailer-key, scope-event constant adoption** — same chokepoint pattern but with broader scope. Filed mentally; expansion via later gaps as drift surfaces. The policy's structure makes adding new categories a 5-minute change (extend the enumerator's prefix filter, add the new sites to the live tree).
- **Raw assignments (`s := "open"`)** — deliberately not flagged. Common in test fixtures, YAML decoding scaffolding, and the kernel's add-verb body where the AC's initial status is set via constant ("open" is already a constant in entity.go; the assignment site is currently uses the constant directly). If drift in this shape surfaces, broaden the policy via a later gap.

## Reviewer notes

- The policy file's docstring carries example literals as content ("Concretely: `ac.Status == \"open\"` violates..."). My skip-internal/policies-from-scanning logic handles this; a reviewer wondering why the policy doesn't fire on its own file will find the answer in the policy body's scope filter.

- The TDDPhase fix at `auditonly.go:279` is out of strict M-0119 scope per the spec, but in scope of "fix things in the same neighborhood while you're there." If a future gap formalizes TDDPhase adoption, the policy's `Status` prefix becomes a configurable allow-list parameter; this pre-emptive fix saves one site of churn then.

- M-0119's two commits are small (437 + 22 line deltas). The audit trail reads cleanly: commit 1 is the policy machinery, commit 2 is the codebase update + the CI wiring. Either commit reverts cleanly if a regression surfaces.

- This is the last milestone in E-0032. After wrap, `aiwfx-wrap-epic E-0032` is the next step.
