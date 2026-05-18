---
id: G-0130
title: 'fsm-history-consistent: enforce FSM as tree-invariant via git-log walk'
status: open
prior_ids:
    - G-0128
discovered_in: M-0121
---
## What's missing

The kernel's commitment "framework correctness must not depend on the LLM choosing to enforce" (CLAUDE.md §Engineering principles, design-decisions.md §Cross-cutting) requires that the per-entity status FSM be a **tree-invariant** — every status change visible in the working tree must trace to an FSM-legal transition in git history. Today, the FSM is enforced only as a **verb-precondition** by `aiwf promote` and `aiwf cancel`. A direct markdown edit that flips a status without going through a verb bypasses the FSM entirely, leaving a state with no FSM-legal predecessor — and aiwf's checks would not flag it.

The closest existing chokepoint is `provenance-untrailered-entity-commit`, which fires when an entity is mutated without `aiwf-verb:` / `aiwf-entity:` trailers. This is a **warning**, not an error, and it does not validate the specific status transition implied by the diff. A malicious or careless commit could:

- Skip TDD's `red` entry by setting `tdd_phase: green` directly
- Undo a `cancelled` by editing frontmatter back to `active`
- Move an ADR from `superseded` back to `accepted`
- Promote an epic to `active` without the `--force --reason` sovereign-act

None of these would be caught at pre-push.

## Why it matters

This was surfaced as **review finding #3** during M-0121's audit catalog work (2026-05-18). The audit catalog committed to interpretation (b) — FSM-as-tree-invariant — but the implementation only supports (a) — FSM-as-verb-precondition. Without this check, R-RULE-001..018's "hard-reject" framing in `docs/pocv3/design/legal-workflows-audit.md` §10.1 is aspirational rather than mechanical.

Resolving this gap completes the chokepoint that makes the FSM tree-invariant in practice, not just in design.

## Implementation outline

A new check rule in `internal/check/`, e.g. `fsm_history_consistent.go`:

1. For each entity in the loaded tree, walk `git log --follow -- <entity-path>` to recover the status history.
2. For each commit that touched the entity's frontmatter status field:
   - Parse the prior status (from the parent commit's blob).
   - Parse the new status (from this commit's blob).
   - Verify `(prior, new) ∈ entity.AllowedTransitions(kind, prior)`, OR the commit carries an `aiwf-force:` trailer with a non-empty reason.
3. Emit `fsm-history-consistent` finding (severity `error`) for any status change that fails either check:
   - Subcode `illegal-transition` — change is not in the FSM and no force trailer
   - Subcode `forced-untrailered` — change matches a sovereign-act shape (e.g., epic `proposed → active`) but lacks the force trailer
   - Subcode `manual-edit` — change has no `aiwf-verb:` trailer at all (overlaps with `provenance-untrailered-entity-commit` but with FSM-specific framing)

## Severity escalation

This finding's existence allows us to *strengthen* the existing `provenance-untrailered-entity-commit` from warning to error in the FSM-touching subset, since the new check provides the auditable record of which transitions are legal.

## Implementation cost

Estimated 1 milestone:
- New check rule file with the git-log walk + FSM validation
- Test fixtures: legal-transition cases, illegal-transition cases, forced cases, manual-edit cases
- Update `internal/policies/` to document the new finding code in the discoverability rule
- Update the audit catalog's R-RULE-149 to remove the "(unimplemented)" qualifier

## Related

- Audit catalog R-RULE-019, R-RULE-149 reference this gap
- ADR-0011 (Legal-workflow spec methodology) §Cell-coverage commitment
- design-decisions.md §Cross-cutting "Enforcement does not depend on the LLM choosing to enforce"
