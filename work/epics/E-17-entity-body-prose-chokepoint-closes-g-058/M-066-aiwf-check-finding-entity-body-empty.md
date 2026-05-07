---
id: M-066
title: aiwf check finding entity-body-empty
status: in_progress
parent: E-17
tdd: required
acs:
    - id: AC-1
      title: entity-body-empty (warning) when body section is empty
      status: open
      tdd_phase: done
    - id: AC-2
      title: Severity escalates to error under aiwf.yaml tdd.strict true
      status: open
      tdd_phase: red
    - id: AC-3
      title: Entities with non-empty body prose produce no finding
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

## Rescope note (per G-063, 2026-05-07)

This milestone was originally scoped AC-only as `acs-body-empty`. **Rescoped 2026-05-07** to a kind-generalized finding `entity-body-empty` covering all entity kinds whose body carries load-bearing prose. The rescope was forced by [G-063](../../gaps/G-063-no-defined-start-epic-ritual-epic-activation-is-a-deliberate-sovereign-act-with-preflight-optional-delegation-but-kernel-treats-it-as-a-one-line-fsm-flip.md): the start-epic preflight requires a "non-empty epic body" check, and the cleanest implementation is one rule parameterized by kind rather than two parallel rules. Sub-decision #4 of G-063 governs.

Title, slug, and per-AC titles have all been updated to reflect the generalized scope. The frontmatter title fields were hand-edited (operator-authorized; no `aiwf retitle` verb exists yet — see [G-065](../../gaps/G-065-no-aiwf-retitle-verb-scope-refactors-that-change-an-entity-s-or-ac-s-intent-leave-frontmatter-title-fields-permanently-misleading-only-slug-rename-is-supported.md) for the verb-mechanism gap that this rescope surfaced).

## Goal

Add an `aiwf check` finding `entity-body-empty` that fires for any entity whose load-bearing body section is empty (no non-heading content between the section heading and the next, or EOF; HTML comments do not satisfy non-empty). Warning severity by default; error under `aiwf.yaml: tdd.strict: true` (sharing the same strictness field as [M-065](../E-16-tdd-policy-declaration-chokepoint-closes-g-055/M-065-aiwf-check-finding-milestone-tdd-undeclared-as-defense-in-depth.md)'s `milestone-tdd-undeclared`). This is the load-bearing chokepoint that makes per-kind body-prose intent mechanically enforceable.

## Approach

New rule in `internal/check/`. Per-kind dispatch: each entity kind has a hardcoded list of load-bearing body sections; the rule walks the body, locates each named section by heading, and asserts non-empty content between that heading and the next.

| Kind | Required non-empty body sections |
|---|---|
| epic | `Goal`, `Scope`, `Out of scope` |
| milestone | `Goal`, `Approach`, `Acceptance criteria` |
| AC (sub-element of milestone) | `### AC-N — <title>` body |
| gap | `What's missing`, `Why it matters` |
| adr | `Context`, `Decision`, `Consequences` |
| decision | `Question`, `Decision`, `Reasoning` |
| contract | `Purpose`, `Stability` |

Definition of empty: between the section's heading and the next heading (or EOF), there is no non-whitespace content other than the heading itself. Multiple consecutive blank lines, leading/trailing whitespace, and Windows line endings all count as empty. HTML comments are stripped before the emptiness check (operator intent to defer is not the prose the design specifies).

For ACs, the rule shares the heading-locator from the existing `acs-body-coherence` rule rather than re-parsing the markdown. For top-level kinds, a similar locator scans the body's `## ` headings.

Severity is resolved from `aiwf.yaml: tdd.strict` — the same field that gates M-065's escalation. Single source of truth: both `entity-body-empty` and `milestone-tdd-undeclared` read it; no parallel field, no second config knob.

The grandfather rule is preserved by *not* coupling this to `acs-tdd-audit`: historical entities with empty bodies surface as `entity-body-empty` warnings (so they're visible) but do not retroactively flunk other audits.

## Acceptance criteria

### AC-1 — entity-body-empty (warning) when body section is empty

`aiwf check` against a planning tree containing an entity with at least one empty load-bearing body section emits an `entity-body-empty` finding at warning severity. The rule fires for each entity kind in the per-kind table above. AC bodies use the existing heading locator (`### AC-N — <title>`); top-level kinds scan `## <section>` headings. Definition of empty: between the section heading and the next heading (or EOF), no non-heading non-whitespace content. Multiple blank lines, leading/trailing whitespace, and Windows line endings all count as empty. The finding includes the entity id (composite for ACs), kind, missing section name, file path, and a hint pointing at `aiwf add ac --body-file` (M-067, AC-only) for ACs and to a follow-up gap for the non-AC `--body-file` flags. Implementation: a new rule in `internal/check/`, with per-kind body-section dispatch sharing the heading-locator from the existing `acs-body-coherence` rule.

### AC-2 — Severity escalates to error under aiwf.yaml tdd.strict true

When `aiwf.yaml` contains `tdd.strict: true`, the `entity-body-empty` finding is emitted at error severity instead of warning, regardless of kind. The escalation reads from the same `tdd.strict` field that M-065's `milestone-tdd-undeclared` reads — single source of truth for the project's strictness posture, no parallel field. Tested with two fixtures sharing the same planning tree but differing only in `tdd.strict`; one produces a warning, the other an error. Exit code rises to 1 in the strict case.

### AC-3 — Entities with non-empty body prose produce no finding

For any entity whose load-bearing body sections each contain at least one non-heading line of non-whitespace content, the rule emits no finding. The check is permissive about *what* the prose is — a one-line paragraph, a bullet list, a code block, a single sentence, or rich multi-paragraph detail all clear the rule. The kernel principle "prose is not parsed" applies (per `acs-and-tdd-plan.md:197`); the rule asserts presence, not structure. Tested with several positive fixtures spanning kinds (epic, milestone, AC, gap).

### AC-4 — Bare HTML comments do not satisfy the non-empty requirement

An entity whose load-bearing body section contains only HTML comments (e.g. `<!-- TODO: write this -->` or `<!-- placeholder -->`) is treated as empty — the comment is operator intent to defer, not the prose the design specifies. The rule strips HTML comment blocks before the emptiness check; if nothing non-whitespace remains, the finding fires. Edge case: a single HTML comment followed by real prose passes (the prose is what counts); a single HTML comment with nothing else does not. Tested with both shapes across at least two kinds.

### AC-5 — Finding does not retroactively engage acs-tdd-audit

The grandfather rule from G-055 / G-058 is preserved: for an AC that surfaces `entity-body-empty`, the AC's status / phase fields are not retroactively re-audited against `acs-tdd-audit`. In practice: the historical E-14 milestones (M-049 through M-055), all `met` with empty bodies, will produce `entity-body-empty` warnings per AC but **zero** new `acs-tdd-audit` findings. Same pattern as M-065 / G-055. Top-level kinds do not have an analogous retroactive-audit coupling, so this AC remains AC-scoped in its concern; the assertion is "no new `acs-tdd-audit` findings introduced when adding `entity-body-empty`," which is independent of how many non-AC kinds the rule covers.

### AC-6 — Finding code documented in aiwf-check skill

The `aiwf-check` skill's findings table gains a row for `entity-body-empty`: severity (warning, escalates to error under `tdd.strict: true`), trigger (any load-bearing body section is empty per the per-kind list above), and remediation (write prose for the named section; for ACs, use `aiwf add ac --body-file` from M-067; for other kinds, edit body and run `aiwf edit-body`, until the follow-up gap delivers `--body-file` for those verbs). The discoverability test in `internal/policies/` (per G-021's `PolicyFindingCodesAreDiscoverable`) catches the code at CI time if the row is missing.

