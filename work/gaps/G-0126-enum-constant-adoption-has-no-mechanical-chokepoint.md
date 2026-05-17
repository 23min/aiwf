---
id: G-0126
title: Enum-constant adoption has no mechanical chokepoint
status: open
---
## Problem

The closed-set status/kind/phase constants in [internal/entity/entity.go](../../../internal/entity/entity.go) (`StatusOpen`, `StatusActive`, `KindEpic`, `PhaseRed`, …) are not consistently adopted at use sites. Concrete drift today:

- [internal/entity/transition.go:198](../../../internal/entity/transition.go) writes `ac.Status == "open"` instead of `ac.Status == entity.StatusOpen`.
- [internal/entity/entity.go:100](../../../internal/entity/entity.go) and `:477` enumerate the FSM-allowed values as raw string slices (`[]string{"open", "addressed", "wontfix"}`) rather than constant references.

The constants exist (`entity.go:56` and surrounding); the literals just bypass them. No mechanical check enforces adoption, so future contributors (human or LLM) can introduce the same shape and pass CI.

## Why it matters

Per the kernel rule "framework correctness must not depend on LLM behavior," closed-set adoption is the kind of invariant that needs a chokepoint rather than reviewer vigilance. The current state is exactly the failure mode the principle warns against: the convention exists, the constants exist, but enforcement is implicit. The hardcoded `"open"` at `transition.go:198` survived through the recent idiomatic-Go refactor (G-0107 steps 1 and 2) — proof that review alone isn't catching it.

## Target shape (sketch, not prescription)

A new policy under `internal/policies/enum_literal_adoption.go` that:

- Enumerates `Status*` (and later, expanded categories) constants from [internal/entity/entity.go](../../../internal/entity/entity.go) via `go/ast`, so adding a new status auto-extends the check with no second source of truth.
- Walks `.go` files outside `internal/entity/` and outside `*_test.go`, scoped to **comparison sites** (`==`, `!=`, `switch` clauses) — not raw assignments or test data.
- Reports each literal-vs-constant violation with `file:line: literal "open" should be entity.StatusOpen`.
- Allowlists via `//enums:ignore <reason>` line-suffix comments (matching the `//coverage:ignore` shape).

## Code references

- [internal/entity/entity.go](../../../internal/entity/entity.go) — `StatusOpen` and sibling constants live at `:56` and surrounding; FSM tables at `:100`, `:477`
- [internal/entity/transition.go](../../../internal/entity/transition.go) — `MilestoneCanGoDone` at `:193` uses the literal at `:198`
- [internal/policies/test_setup_presence.go](../../../internal/policies/test_setup_presence.go) — precedent AST-based policy
- [G-0107](./G-0107-reorganize-cmd-aiwf-into-idiomatic-per-verb-packages.md) — the idiomatic-Go refactor whose blind spot this gap addresses
