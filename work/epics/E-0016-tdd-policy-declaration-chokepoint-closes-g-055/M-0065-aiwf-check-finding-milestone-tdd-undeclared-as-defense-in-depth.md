---
id: M-0065
title: aiwf check finding milestone-tdd-undeclared as defense-in-depth
status: draft
parent: E-0016
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

Add an `aiwf check` finding `milestone-tdd-undeclared` that fires for any milestone whose frontmatter lacks a `tdd:` field. Warning severity by default; error under `aiwf.yaml: tdd.strict: true`. This is the defense-in-depth backstop for the [M-0062](M-062-tdd-flag-on-aiwf-add-milestone-with-project-default-fallback.md) creation chokepoint — it catches hand-edits that strip the field, import paths that bypass the verb, and surfaces historical (E-0014-era) milestones for retro-policy review without retroactively breaking them.

## Approach

New rule in `internal/check/`. Enumerates milestones from the loaded planning tree; emits the finding when `tdd:` is absent. Severity is resolved from `aiwf.yaml: tdd.strict` (default `false` → warning, `true` → error). The closed-set `tdd:` value validation already exists in the parser (per [M-0063](M-063-aiwf-yaml-tdd-default-schema-and-aiwf-init-seeding.md)); this rule only addresses the *absent* case.

The grandfather rule from [G-0055](../../gaps/G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) is preserved: this rule does NOT seed `tdd_phase` retroactively, so `acs-tdd-audit` does not fire against the historical tree's already-met ACs. The rule's job is visibility, not retro-enforcement.

Test fixtures cover: a milestone with no `tdd:` (warning), a milestone with `tdd: required` (no finding), the same fixtures under `tdd.strict: true` (error). The `aiwf-check` skill gains a row in its findings table per the kernel's discoverability rule (see G-0021's `PolicyFindingCodesAreDiscoverable`).

## Acceptance criteria

### AC-1 — milestone-tdd-undeclared (warning) when tdd: is absent

`aiwf check` against a planning tree that contains a milestone whose frontmatter lacks the `tdd:` key emits a `milestone-tdd-undeclared` finding at warning severity for that milestone. Empty-string and explicit `null` are treated the same as absent (the field has to be present *and* set to a closed-set value to clear the finding). The finding includes the milestone id, the file path, and a hint pointing at `--tdd <required|advisory|none>` for new milestones and the `aiwf.yaml: tdd.default` field for repo-wide defaults. Implementation: a new rule in `internal/check/` enumerated alongside the existing milestone rules.

### AC-2 — Severity escalates to error under aiwf.yaml tdd.strict true

When `aiwf.yaml` contains `tdd.strict: true`, the same finding from AC-1 is emitted at error severity instead of warning. Exit code rises from 0 (or 1 if other findings) to 1 to reflect the error. The escalation lookup reads from the same loaded config struct as M-0063's `tdd.default` (no parallel reader). Tested with two fixtures sharing the same planning tree but differing only in `tdd.strict`; one produces a warning, the other an error.

### AC-3 — Milestones with tdd: set produce no finding

For any milestone whose frontmatter has `tdd:` set to one of the three closed-set values (`required`, `advisory`, `none`), the rule emits no finding regardless of `tdd.strict`. Out-of-set values (`tdd: bogus`) are M-0063's parse-time concern, not this rule's; if the tree gets that far at all, the parse will already have failed. Tested with three fixtures (one per closed-set value) and a confirmation that no finding fires.

### AC-4 — Finding does not retroactively engage acs-tdd-audit

The grandfather rule from G-0055 is preserved: for a milestone that surfaces `milestone-tdd-undeclared`, the milestone's existing ACs are *not* retroactively re-audited against `acs-tdd-audit`'s "AC `met` requires `tdd_phase: done`" rule. In practice: the historical E-0014 milestones (M-0049 through M-0055), all `met` with no `tdd_phase`, will produce one `milestone-tdd-undeclared` warning each but **zero** `acs-tdd-audit` findings. Tested with a fixture that mirrors the historical shape — every AC `status: met` and no `tdd_phase` — and asserted that only the M-tdd-undeclared finding fires.

### AC-5 — Finding code documented in aiwf-check skill

The `aiwf-check` skill's findings table (per the kernel's discoverability rule and G-0021's `PolicyFindingCodesAreDiscoverable` policy) gains a row for `milestone-tdd-undeclared`: severity (warning, escalates to error under `tdd.strict: true`), trigger (milestone frontmatter lacks `tdd:`), and remediation (set `tdd:` via `aiwf add milestone --tdd ...` or by hand-editing the frontmatter for grandfathered milestones; configure repo-wide via `aiwf.yaml: tdd.default`). The discoverability test in `internal/policies/` catches the code at CI time if the skill row is missing.

