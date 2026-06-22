---
id: M-0173
title: aiwf add --area write path with completion and discovered-in derivation
status: draft
parent: E-0043
depends_on:
    - M-0171
tdd: required
---
## Goal

Add the write path for `area`: an `--area <name>` flag on `aiwf add` (the five root kinds), validated and tab-completed from `aiwf.yaml: areas`, with a gap deriving its area from `discovered_in` when `--area` is omitted. Changing an entity's area reverses through the same surface.

## Context

M-0171 makes the field exist; the area-unknown check-finding milestone catches undeclared values at check time. This milestone gives the operator the loud, completion-assisted way to *set* the field at creation — so a carve-out workstream is tagged in the same atomic commit that creates the entity, not by a later hand-edit.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac` against this milestone.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — `aiwf add <root-kind> --area <name> ...` writes `area: <name>` into the new entity's frontmatter in the creating commit.
- **AC-2 candidate** — `--area` is rejected (usage error, no entity created) when the value is not in the declared `areas` set, or when no `areas` block exists; the error names the value and the declared set. (Single source of truth: the same accessor M-0171/M-0172 use.)
- **AC-3 candidate** — `--area` is invalid for the non-root kinds (milestone derives from parent); passing it there errors.
- **AC-4 candidate** — `--area <TAB>` tab-completes exactly the declared `areas` members, wired the same way other closed-set flags are (Cobra `RegisterFlagCompletionFunc`); the completion-drift policy passes.
- **AC-5 candidate** — `aiwf add gap --discovered-in <id>` derives the gap's `area` from the discovered-in entity's **effective** area when `--area` is omitted (an epic carries `area` directly; a milestone target is a two-hop derivation through its parent epic, since milestones don't store `area`). Open Question 1 in the epic; lean: derive-on-omit.
- **AC-6 candidate** — Subprocess integration test covers set / reject / derive paths (test-the-seam).

## Constraints

- **Validate against config at write time** using the M-0171 accessor — no parallel validator. (This is the verb-time twin of the area-unknown check-time finding; both read the same declared set.)
- **Reversible by the same verb** — re-running with a different `--area` (or the post-create field mutation surface, if one exists) changes the tag; no bespoke "unset area" verb invented unless needed.

## Out of scope

- The area-unknown check finding and read surfaces (filter/grouping milestones).
- A bulk re-tagging verb across many entities.

## Dependencies

- M-0171 — the `area` field, `aiwf.yaml: areas` block, and config accessor.

## References

- [E-0043 epic](epic.md) · [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md)
