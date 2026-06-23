---
id: M-0173
title: aiwf add --area write path with completion and discovered-in derivation
status: in_progress
parent: E-0043
depends_on:
    - M-0171
tdd: required
acs:
    - id: AC-1
      title: --area writes area into new entity frontmatter
      status: met
      tdd_phase: done
    - id: AC-2
      title: --area rejected when value undeclared or no areas block
      status: open
      tdd_phase: red
    - id: AC-3
      title: --area is invalid for non-root kinds
      status: open
      tdd_phase: red
    - id: AC-4
      title: --area tab-completes declared members
      status: open
      tdd_phase: red
    - id: AC-5
      title: gap derives area from discovered-in when --area omitted
      status: open
      tdd_phase: red
    - id: AC-6
      title: integration seam test covers set, reject, and derive paths
      status: open
      tdd_phase: red
---
## Goal

Add the write path for `area`: an `--area <name>` flag on `aiwf add` (the five root kinds), validated and tab-completed from `aiwf.yaml: areas`, with a gap deriving its area from `discovered_in` when `--area` is omitted. Changing an entity's area reverses through the same surface.

## Context

M-0171 makes the field exist; M-0172's `area-unknown` check finding catches undeclared values at check time. This milestone gives the operator the loud, completion-assisted way to *set* the field at creation — so a carve-out workstream is tagged in the same atomic commit that creates the entity, not by a later hand-edit. The write-time validation is the verb-time twin of the `area-unknown` check, reading the same declared set through the M-0171 accessor.

## Acceptance criteria

### AC-1 — --area writes area into new entity frontmatter

`aiwf add <root-kind> --area <name> ...` (epic, ADR, gap, decision, contract) writes `area: <name>` into the new entity's frontmatter in the same atomic creating commit.

Evidence: a dispatcher test per/over the root kinds asserting the created entity's frontmatter carries the area; the seam path is also exercised in AC-6.

### AC-2 — --area rejected when value undeclared or no areas block

`--area <name>` is rejected with a usage error (exit 2, **no entity created**) when the value is not a member of the declared `aiwf.yaml: areas` set, or when no `areas` block exists. The error names the offending value and the declared set. Validation uses the M-0171 config accessor — the same declared set the `area-unknown` check reads (single source of truth, no parallel validator).

Evidence: dispatcher tests for the undeclared-value case and the no-block case, each asserting a non-zero exit, that no entity file was created, and that the message names the value and the declared members.

### AC-3 — --area is invalid for non-root kinds

`--area` is not accepted for a milestone, which derives its area from its parent epic and never stores its own. Passing `--area` to a non-root kind errors (usage error, no entity created).

Evidence: a dispatcher test asserting `aiwf add milestone --area <name> ...` errors and creates nothing.

### AC-4 — --area tab-completes declared members

`aiwf add <root-kind> --area <TAB>` completes exactly the declared `areas.members`, wired via Cobra `RegisterFlagCompletionFunc` the same way other closed-set flags are. The completion-drift policy (`cmd/aiwf/completion_drift_test.go`) stays green — the flag is registered for completion or carries an explicit opt-out entry.

Evidence: a completion test asserting the registered completion function returns the declared members for a tree with an `areas` block; the completion-drift test passes.

### AC-5 — gap derives area from discovered-in when --area omitted

`aiwf add gap --discovered-in <id>` derives the gap's `area` from the discovered-in entity's **effective** area when `--area` is omitted and that entity has one — an epic carries `area` directly; a milestone target is a two-hop derivation through its parent epic (milestones don't store `area`), via the M-0171 `ResolvedAreaByID` seam. If the discovered-in entity has no effective area, the gap is left untagged. An explicit `--area` always takes precedence over derivation.

Decision: this resolves the epic's Open Question 1 — **derive-on-omit**.

Evidence: dispatcher tests for derive-from-epic, derive-from-milestone (two-hop), no-area-source (untagged), and explicit-`--area`-overrides-derivation.

### AC-6 — integration seam test covers set, reject, and derive paths

An integration test drives the real dispatcher end-to-end (test-the-seam) so the flag wiring, config validation, and derivation are proven together, not just at the unit layer: `add --area <declared>` (set), `add --area <undeclared>` (reject), and `add gap --discovered-in <id>` (derive).

Evidence: a dispatcher/subprocess integration test in the established integration home covering the set / reject / derive paths.

## Constraints

- **Validate against config at write time** using the M-0171 accessor — no parallel validator. This is the verb-time twin of the `area-unknown` check-time finding; both read the same declared set.
- **Reversible by the same verb** — re-running with a different `--area` (or the post-create field-mutation surface, if one exists) changes the tag; no bespoke "unset area" verb invented unless real friction needs it.

## Out of scope

- The `area-unknown` check finding (M-0172, done) and read surfaces (filter/grouping milestones M-0174–M-0175).
- A bulk re-tagging verb across many entities.

## Dependencies

- M-0171 — the `area` field, `aiwf.yaml: areas` block, and config accessor (`ResolvedArea` / `ResolvedAreaByID`).

## References

- [E-0043 epic](epic.md) · [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md)
- M-0172 — the `area-unknown` check finding; this milestone is its write-time twin.
