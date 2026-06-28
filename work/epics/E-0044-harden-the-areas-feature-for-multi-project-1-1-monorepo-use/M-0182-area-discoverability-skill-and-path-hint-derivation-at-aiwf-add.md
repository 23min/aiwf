---
id: M-0182
title: Area discoverability skill and path-hint derivation at aiwf add
status: in_progress
parent: E-0044
depends_on:
    - M-0179
tdd: required
acs:
    - id: AC-1
      title: Topical aiwf-area skill exists with valid frontmatter and is discoverable
      status: open
      tdd_phase: red
    - id: AC-2
      title: 'Skill teaches the area mental model: operate-everywhere vs aiwf constraints'
      status: open
      tdd_phase: red
    - id: AC-3
      title: 'Skill teaches the area lifecycle: add, set-area, mistag, acknowledge'
      status: open
      tdd_phase: red
    - id: AC-4
      title: Single unambiguous --path-hint derives area when --area is omitted
      status: met
      tdd_phase: done
    - id: AC-5
      title: Explicit --area always wins over a conflicting --path-hint
      status: open
      tdd_phase: red
    - id: AC-6
      title: Ambiguous --path-hint sets no area, prints a suggestion, proceeds untagged
      status: open
      tdd_phase: red
    - id: AC-7
      title: Inert with no declared paths; areamatch.Derive is the SSOT primitive
      status: open
      tdd_phase: red
---
## Goal

Make the area feature **discoverable** and help operators — above all the LLM, the primary user — tag entities correctly the first time. Ship a topical `aiwf-area` skill that installs the mental model: code work happens anywhere, but once you touch `aiwf` an area is both a *guide* (which area does this work belong to?) and a *constraint* (a closed member set, the `areas.required` knob, and the mistag check you must adhere to). Back it with a deterministic `aiwf add --path-hint`, so the kernel maps a path to its area through the `areamatch` SSOT rather than the LLM eyeballing globs — driving manual tags, and mistags, toward zero.

## Context

Manual tagging is the source of mistags. The mechanical *guarantee* against a wrong tag already exists: the mistag check (M-0181) verifies at pre-push that an entity's commits landed in its area's `paths:` territory. What was missing is the **front-end** — there is no area skill, so the LLM has nowhere to learn how areas work or how to pick the right one, and no deterministic helper to derive an area from a path.

This milestone supplies that front-end. At `add` time the entity has no diff, so the only path signal is one the operator/LLM supplies explicitly (`--path-hint`); the kernel matches it against the declared `paths:` (M-0179) and fills `area` when the match is unambiguous. At wrap time the verification is already mistag — no new wrap-time code. Because the guarantee stays mechanical (mistag), the skill is legitimately advisory; its ACs are pinned by structural assertions over the embedded skill bytes plus the existing skill-coverage policy.

## Acceptance criteria

### AC-1 — Topical aiwf-area skill exists with valid frontmatter and is discoverable

A topical `aiwf-area` skill is embedded under `internal/skills/embedded/aiwf-area/SKILL.md` with valid `name:` / `description:` frontmatter, so area functionality is reachable through a single AI-discoverable channel. Evidence: the skill-coverage policy (valid frontmatter; every `aiwf <verb>` mention resolves) plus a structural assertion that the file exists with the expected `name`.

### AC-2 — Skill teaches the area mental model: operate-everywhere vs aiwf constraints

The skill explains that code work happens anywhere, but `aiwf` treats areas as both guidance and constraint: the closed member set (`aiwf.yaml: areas.members`), the `areas.required` knob, and the path oracle the checks enforce. Evidence: a structural assertion that the named mental-model section is present in the embedded bytes and names `areas.required` and the member set.

### AC-3 — Skill teaches the area lifecycle: add, set-area, mistag, acknowledge

The skill ties the lifecycle together: choose an area at `aiwf add` (explicit `--area` wins, else `--path-hint` derivation), remediate a wrong or missing tag with `aiwf set-area`, rely on the mistag check to verify, and use `aiwf acknowledge mistag` for legitimate cross-cutting. Evidence: a structural assertion that the lifecycle section names each verb, and the skill-coverage body-resolution rule that every `aiwf <verb>` mention resolves to a real verb.

### AC-4 — Single unambiguous --path-hint derives area when --area is omitted

When `--area` is omitted and `--path-hint <path>` falls under exactly one declared area's globs, `aiwf add` sets `area` to that area. Evidence: an `areamatch.Derive` unit test (single match) and an `aiwf add` dispatcher test asserting the written entity's area.

### AC-5 — Explicit --area always wins over a conflicting --path-hint

An explicit `--area` is never overwritten by derivation. When both are given and `--path-hint` would derive a *different* area, `aiwf add` honors `--area` and reports the disagreement (a cheap at-add mistag-prevention signal), never silently overriding. Evidence: dispatcher tests for the agree and disagree cases.

### AC-6 — Ambiguous --path-hint sets no area, prints a suggestion, proceeds untagged

A `--path-hint` matching zero or multiple areas does not set `area`. The verb prints a suggestion (the candidate areas, or "no area claims this path") and proceeds untagged — at which point `areas.required` (M-0178) refuses the create exactly as it does today. Evidence: dispatcher tests for the zero-match and multi-match cases.

### AC-7 — Inert with no declared paths; areamatch.Derive is the SSOT primitive

With no area declaring `paths:` (or no areas block at all), `--path-hint` performs no derivation. All path↔area matching routes through a single new primitive, `areamatch.Derive`, the SSOT this verb consumes. Evidence: an `areamatch.Derive` unit test (empty / paths-less input) and a dispatcher inert-path test.

## Constraints

- Never silently overwrite an explicit `--area`.
- The skill is advisory; the mechanical guarantee remains the mistag check (M-0181). Discoverability never becomes a guarantee the framework depends on.
- All path↔area glob matching routes through the `areamatch` SSOT — no parallel matcher.

## Out of scope

- Retroactively re-tagging existing entities in bulk — the per-entity `aiwf set-area` + mistag remediation path covers the "code moved" case.
- A new wrap-time *suggestion* finding for untagged entities (distinct from the mistag *verification*, which already exists). Filed as a gap only if friction shows.

## Dependencies

- M-0179 (`paths:` per area) — the oracle derivation reads.
- M-0181 (mistag check + `aiwf acknowledge mistag`) — the wrap-time verification the skill points at; not re-implemented here.

## References

- The `aiwf add --area` write path (E-0043 / M-0173) — extended here with `--path-hint` derivation.
- `internal/areamatch` — the glob SSOT; this milestone adds `Derive`.
- The `aiwf-acknowledge` skill (M-0181) and the `aiwf-add` skill — the area skill cross-references both.
