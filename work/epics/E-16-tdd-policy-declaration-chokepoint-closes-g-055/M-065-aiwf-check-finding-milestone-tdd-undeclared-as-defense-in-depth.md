---
id: M-065
title: aiwf check finding milestone-tdd-undeclared as defense-in-depth
status: draft
parent: E-16
tdd: required
acs:
    - id: AC-1
      title: 'milestone-tdd-undeclared (warning) when tdd: is absent'
      status: open
      tdd_phase: red
    - id: AC-2
      title: Severity escalates to error under aiwf.yaml tdd.strict true
      status: open
      tdd_phase: red
    - id: AC-3
      title: 'Milestones with tdd: set produce no finding'
      status: open
      tdd_phase: red
    - id: AC-4
      title: Finding does not retroactively engage acs-tdd-audit
      status: open
      tdd_phase: red
    - id: AC-5
      title: Finding code documented in aiwf-check skill
      status: open
      tdd_phase: red
---

## Goal

Add an `aiwf check` finding `milestone-tdd-undeclared` that fires for any milestone whose frontmatter lacks a `tdd:` field. Warning severity by default; error under `aiwf.yaml: tdd.strict: true`. This is the defense-in-depth backstop for the [M-062](M-062-tdd-flag-on-aiwf-add-milestone-with-project-default-fallback.md) creation chokepoint — it catches hand-edits that strip the field, import paths that bypass the verb, and surfaces historical (E-14-era) milestones for retro-policy review without retroactively breaking them.

## Approach

New rule in `internal/check/`. Enumerates milestones from the loaded planning tree; emits the finding when `tdd:` is absent. Severity is resolved from `aiwf.yaml: tdd.strict` (default `false` → warning, `true` → error). The closed-set `tdd:` value validation already exists in the parser (per [M-063](M-063-aiwf-yaml-tdd-default-schema-and-aiwf-init-seeding.md)); this rule only addresses the *absent* case.

The grandfather rule from [G-055](../../gaps/G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) is preserved: this rule does NOT seed `tdd_phase` retroactively, so `acs-tdd-audit` does not fire against the historical tree's already-met ACs. The rule's job is visibility, not retro-enforcement.

Test fixtures cover: a milestone with no `tdd:` (warning), a milestone with `tdd: required` (no finding), the same fixtures under `tdd.strict: true` (error). The `aiwf-check` skill gains a row in its findings table per the kernel's discoverability rule (see G-021's `PolicyFindingCodesAreDiscoverable`).

## Acceptance criteria

### AC-1 — milestone-tdd-undeclared (warning) when tdd: is absent

### AC-2 — Severity escalates to error under aiwf.yaml tdd.strict true

### AC-3 — Milestones with tdd: set produce no finding

### AC-4 — Finding does not retroactively engage acs-tdd-audit

### AC-5 — Finding code documented in aiwf-check skill

