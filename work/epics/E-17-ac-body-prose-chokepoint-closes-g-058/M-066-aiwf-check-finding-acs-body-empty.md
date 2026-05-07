---
id: M-066
title: aiwf check finding acs-body-empty
status: draft
parent: E-17
tdd: required
acs:
    - id: AC-1
      title: acs-body-empty (warning) when body section is empty
      status: open
      tdd_phase: red
    - id: AC-2
      title: Severity escalates to error under aiwf.yaml tdd.strict true
      status: open
      tdd_phase: red
    - id: AC-3
      title: ACs with non-empty body prose produce no finding
      status: open
      tdd_phase: red
    - id: AC-4
      title: Bare HTML comments do not satisfy the non-empty requirement
      status: open
      tdd_phase: red
    - id: AC-5
      title: Finding does not retroactively engage acs-tdd-audit
      status: open
      tdd_phase: red
    - id: AC-6
      title: Finding code documented in aiwf-check skill
      status: open
      tdd_phase: red
---

## Goal

Add an `aiwf check` finding `acs-body-empty` that fires for any AC whose body section under `### AC-N — <title>` is empty (no non-heading content between the AC's heading and the next `###` or EOF). Warning severity by default; error under `aiwf.yaml: tdd.strict: true` (sharing the same strictness field as [M-065](../E-16-tdd-policy-declaration-chokepoint-closes-g-055/M-065-aiwf-check-finding-milestone-tdd-undeclared-as-defense-in-depth.md)'s `milestone-tdd-undeclared`). This is the load-bearing chokepoint of the epic — the rule that makes the design intent mechanically enforceable.

## Approach

New rule in `internal/check/`. Extends the existing `acs-body-coherence` machinery (which already locates AC body sections by heading id) with an emptiness check on the section content. Definition of empty: between the AC's `### AC-N — <title>` heading and the next `### ` (or EOF), there is no non-whitespace content other than the heading itself. A bare heading with a blank line after it counts as empty; a heading with a single `<!-- TODO -->` HTML comment also counts as empty (operator-side intent: a comment is not the prose the design specifies).

Severity is resolved from `aiwf.yaml: tdd.strict` — the same field that gates M-065's escalation. Single source of truth: both `acs-body-empty` and `milestone-tdd-undeclared` read it; no parallel field, no second config knob.

The grandfather rule is preserved by *not* coupling this to `acs-tdd-audit` — historical milestones with met ACs and empty bodies surface as `acs-body-empty` warnings (so they're visible) but do not retroactively flunk `acs-tdd-audit`. Same pattern as M-065 / G-055.

## Acceptance criteria

### AC-1 — acs-body-empty (warning) when body section is empty

`aiwf check` against a planning tree that contains a milestone with at least one AC whose body section is empty emits an `acs-body-empty` finding at warning severity for that AC. Definition of empty: between the AC's `### AC-N — <title>` heading and the next `### ` heading (or EOF), there is no non-whitespace content other than the heading line itself. Multiple consecutive blank lines, leading/trailing whitespace, and Windows line endings all count as empty. The finding includes the milestone id, the AC composite id (`M-NNN/AC-N`), the file path, and a hint pointing at the `--body-file` flag (M-067) and the design-intent citation. Implementation: a new rule in `internal/check/`, sharing the heading-locator from the existing `acs-body-coherence` rule rather than re-parsing the markdown.

### AC-2 — Severity escalates to error under aiwf.yaml tdd.strict true

When `aiwf.yaml` contains `tdd.strict: true`, the `acs-body-empty` finding is emitted at error severity instead of warning. The escalation reads from the same `tdd.strict` field that M-065's `milestone-tdd-undeclared` reads — single source of truth for the project's TDD strictness posture, no parallel field. Tested with two fixtures sharing the same planning tree but differing only in `tdd.strict`; one produces a warning, the other an error. Exit code rises to 1 in the strict case.

### AC-3 — ACs with non-empty body prose produce no finding

For any AC whose body section contains at least one non-heading line of non-whitespace content, the rule emits no finding. The check is permissive about *what* the prose is — a one-line paragraph, a bullet list, a code block, a single sentence, or rich multi-paragraph detail all clear the rule. The kernel principle "prose is not parsed" applies (per `acs-and-tdd-plan.md:197`); the rule asserts presence, not structure. Tested with several positive fixtures covering the range of acceptable shapes.

### AC-4 — Bare HTML comments do not satisfy the non-empty requirement

An AC whose body contains only HTML comments (e.g. `<!-- TODO: write this -->` or `<!-- placeholder -->`) is treated as empty — the comment is operator intent to defer, not the prose the design specifies. The rule strips HTML comment blocks before the emptiness check; if nothing non-whitespace remains, the finding fires. Edge case: a single HTML comment followed by real prose passes (the prose is what counts); a single HTML comment with nothing else does not. Tested with both shapes.

### AC-5 — Finding does not retroactively engage acs-tdd-audit

The grandfather rule from G-055 / G-058 is preserved: for an AC that surfaces `acs-body-empty`, the AC's status / phase fields are not retroactively re-audited against `acs-tdd-audit`. In practice: the historical E-14 milestones (M-049 through M-055), all `met` with empty bodies, will produce one `acs-body-empty` warning per AC but **zero** new `acs-tdd-audit` findings. Tested with a fixture mirroring the historical shape (every AC `status: met`, empty body) and asserted that only `acs-body-empty` fires.

### AC-6 — Finding code documented in aiwf-check skill

The `aiwf-check` skill's findings table gains a row for `acs-body-empty`: severity (warning, escalates to error under `tdd.strict: true`), trigger (AC body section under `### AC-N — <title>` is empty), and remediation (write a paragraph naming pass criteria / edge cases / code references; or use `aiwf add ac --body-file` from M-067 for in-verb scaffolding). The discoverability test in `internal/policies/` (per G-021's `PolicyFindingCodesAreDiscoverable`) catches the code at CI time if the row is missing.

