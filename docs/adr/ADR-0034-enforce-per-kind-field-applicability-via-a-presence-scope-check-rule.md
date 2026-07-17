---
id: ADR-0034
title: Enforce per-kind field applicability via a presence-scope check rule
status: proposed
---
# ADR-0034 — Enforce per-kind field applicability via a presence-scope check rule

> **Date:** 2026-07-17 · **Decided by:** human/peter

## Status vocabulary (aiwf)

aiwf's ADR statuses are: `proposed | accepted | superseded | rejected`.

## Context

The `area` feature (ADR-0009 era design; see `internal/check/area_unknown.go` / `area_required.go`) established the pattern for an optional, per-kind, closed-set frontmatter field carried on the shared `Entity` struct rather than a per-kind struct. But `area`'s check rules only ever gate the *value* — is a declared area value legal, is a required area present — never whether the field's mere *presence* is legal for the entity's kind. Nothing in the kernel rejected `area:` (or any optional field) showing up on a kind that was never meant to carry it; the type system doesn't model per-kind field legality (the shared `Entity` struct exposes every optional field to every kind), so nothing did.

E-0066 added `priority` as a second such field, legal only on gap and decision. Unlike `area`, this scope restriction had to be a mechanical, enforced fact (per the epic's own constraint: "Gap and decision only" must be an enforced fact, not prose") — an epic, milestone, ADR, or contract carrying a stray `priority:` needed a real finding, not silent tolerance. No existing check rule shape covered "this field is present on a kind that shouldn't have it," so `priority-not-applicable` (`internal/check/priority_not_applicable.go`) is the first rule of a new class: presence-scope enforcement, as opposed to `area`'s value-scope enforcement.

## Decision

Per-kind applicability of an optional shared-struct field is enforced by a dedicated presence-scope check rule — not the type system, and not folded into the field's own value-validation rule. The rule:

- Consults a `CarriesOwnPriority`-style predicate (`entity.CarriesOwnPriority(kind) bool`) — one function naming the closed set of kinds allowed to carry the field — rather than hardcoding the kind list inline in the check rule itself. The predicate is the single source of truth; both the check rule and any writer-side gate (`aiwf add --priority`, `aiwf set-priority`) consult the same function.
- Fires as its own named finding (`priority-not-applicable`), structurally mirroring `area_unknown.go`'s shape, at `SeverityWarning` — consistent with `area_unknown`'s advisory posture, and because a stray field on the wrong kind is a hygiene issue the operator should fix, not a load-bearing correctness violation that should block a push.
- Stays separate from the field's own closed-set value-validation rule (`priority_valid.go`) — presence-scope and value-scope are two different questions ("should this kind have the field at all" vs. "is the value legal given it's present"), and conflating them into one rule would make either question harder to test or explain in isolation.

This is a *general mechanism*, not a `priority`-specific one-off: the next optional shared-struct field that is legal on a strict kind subset follows the same shape — a `CarriesOwn<Field>`-style predicate plus a dedicated presence-scope check rule paired with a firing fixture (per `firing_fixture_presence.go`'s existing requirement) — rather than re-deriving the enforcement approach from scratch.

## Consequences

- **Positive:** future per-kind optional fields have a named, precedented pattern to follow, cutting the design cost of the next one down to "name the predicate, write the rule, mirror the fixture." The kernel's check-rule taxonomy now has two distinct field-enforcement classes (value-scope, presence-scope) instead of conflating them under `area`'s original value-only shape.
- **Negative / follow-up:** the type system still does not model per-kind field legality — the shared `Entity` struct exposes every optional field to every kind at the Go level, so a hand-edited or programmatically-constructed `Entity` value can still set a field on the wrong kind; only `aiwf check` (a load-time validation pass, not a compile-time guarantee) catches it. This is a deliberate trade-off consistent with the kernel's broader "errors are findings, not parse failures" principle (CLAUDE.md), not an oversight, but it means presence-scope enforcement is only as strong as the check's coverage — a construction path that never runs through `aiwf check` (e.g., a script writing frontmatter directly and never invoking the CLI) bypasses it entirely, same as any other check-rule.
- No migration cost — this ADR describes machinery `priority` already ships with; no existing entities need to change.

## References

- `internal/check/priority_not_applicable.go` — the rule this ADR generalizes from.
- `internal/check/area_unknown.go` / `area_required.go` — the value-scope precedent this rule deliberately does not extend.
- E-0066 — the epic that introduced `priority` and this rule.
- G-0078 — the ratified design decisions E-0066 executes.
