---
id: M-0119
title: Add enum_literal_adoption policy; fix surfaced literal sites
status: in_progress
parent: E-0032
tdd: required
acs:
    - id: AC-1
      title: Policy auto-enumerates Status constants from internal/entity/entity.go
      status: open
      tdd_phase: red
    - id: AC-2
      title: Policy fires on == or != BinaryExpr with status literal
      status: open
      tdd_phase: red
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

## Surfaces touched

- `internal/policies/enum_literal_adoption.go` — new
- `internal/entity/transition.go` — literal fix at `:198`
- `internal/entity/entity.go` — FSM table cleanup at `:100`, `:477`
- `CLAUDE.md` — new table row

## Out of scope

- Broadening the denylist beyond `Status*` (e.g., to `Kind*`, `Phase*`, trailer keys, scope events) — seed is `Status*`; expansion via later gaps as drift surfaces.
- Removing the `//enums:ignore` allowlist mechanism — matches existing `//coverage:ignore` pattern.
- Reorganizing existing policy files in `internal/policies/`.

## Dependencies

- None (orthogonal to verb-move work).

### AC-1 — Policy auto-enumerates Status constants from internal/entity/entity.go

### AC-2 — Policy fires on == or != BinaryExpr with status literal

