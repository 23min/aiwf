---
id: M-066
title: aiwf check finding acs-body-empty
status: draft
parent: E-17
tdd: required
---

## Goal

Add an `aiwf check` finding `acs-body-empty` that fires for any AC whose body section under `### AC-N — <title>` is empty (no non-heading content between the AC's heading and the next `###` or EOF). Warning severity by default; error under `aiwf.yaml: tdd.strict: true` (sharing the same strictness field as [M-065](../E-16-tdd-policy-declaration-chokepoint-closes-g-055/M-065-aiwf-check-finding-milestone-tdd-undeclared-as-defense-in-depth.md)'s `milestone-tdd-undeclared`). This is the load-bearing chokepoint of the epic — the rule that makes the design intent mechanically enforceable.

## Approach

New rule in `internal/check/`. Extends the existing `acs-body-coherence` machinery (which already locates AC body sections by heading id) with an emptiness check on the section content. Definition of empty: between the AC's `### AC-N — <title>` heading and the next `### ` (or EOF), there is no non-whitespace content other than the heading itself. A bare heading with a blank line after it counts as empty; a heading with a single `<!-- TODO -->` HTML comment also counts as empty (operator-side intent: a comment is not the prose the design specifies).

Severity is resolved from `aiwf.yaml: tdd.strict` — the same field that gates M-065's escalation. Single source of truth: both `acs-body-empty` and `milestone-tdd-undeclared` read it; no parallel field, no second config knob.

The grandfather rule is preserved by *not* coupling this to `acs-tdd-audit` — historical milestones with met ACs and empty bodies surface as `acs-body-empty` warnings (so they're visible) but do not retroactively flunk `acs-tdd-audit`. Same pattern as M-065 / G-055.

## Acceptance criteria
