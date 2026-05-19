---
id: M-0130
title: Implement fsm-history-consistent check rule for FSM tree-invariant
status: draft
prior_ids:
    - M-0126
parent: E-0033
depends_on:
    - M-0123
tdd: required
acs:
    - id: AC-1
      title: fsm-history-consistent check rule registered; walks git log per entity
      status: open
      tdd_phase: red
    - id: AC-2
      title: 'manual-edit subcode: status-change commit lacks aiwf-verb trailer entirely'
      status: open
      tdd_phase: red
    - id: AC-3
      title: Hint table entry for fsm-history-consistent and each subcode
      status: open
      tdd_phase: red
---
## Goal

Implement the `fsm-history-consistent` check rule that makes the per-entity status FSM a **tree-invariant** rather than just a verb-precondition. Closes gap **G-0132** (filed during M-0121's audit). Addresses the catalog's R-RULE-149 placeholder.

Without this check rule, R-RULE-019 and R-RULE-001..018 (the entity FSM rules in §10.1 of `legal-workflows-audit.md`) have a "hard-reject" severity that is **aspirational for manual-edit cases** — the verb-time chokepoint catches verb-mediated illegal transitions, but a direct markdown edit that flips a status without going through `aiwf promote` bypasses the FSM entirely. The closest existing chokepoint is `provenance-untrailered-entity-commit` (warning) which catches the *trailer absence* but not the specific FSM violation.

This milestone makes the chokepoint real.

## Inserted between M-0123 and M-0124

M-0124 (positive cell coverage) and M-0125 (negative cell coverage) depend on this milestone — their tests assert the full hard-reject behavior R-RULE-001..018 promises, including the manual-edit cases.

## Acceptance criteria

(Added via `aiwf add ac` once M-0123's spec schema is settled — the check rule's exact emission shape will be informed by the canonical `Rule` table.)

## Approach

A new check rule under `internal/check/`, e.g. `fsm_history_consistent.go`:

1. **Walk git log per entity.** For each entity in the loaded tree, walk `git log --follow -- <entity-path>` to recover the status-change history. (The loader resolves entity paths across active and archive per ADR-0004; the walk follows file moves automatically.)
2. **Validate each status-change commit:**
   - Parse the prior status (from the parent commit's blob via `git show <parent>:<path>`).
   - Parse the new status (from this commit's blob).
   - If they differ, verify `(prior, new) ∈ entity.AllowedTransitions(kind, prior)` OR the commit carries a non-empty `aiwf-force:` trailer.
3. **Emit `fsm-history-consistent` findings** (severity `error`) per violation:
   - Subcode `illegal-transition` — change is not in the FSM and no force trailer
   - Subcode `forced-untrailered` — change matches a sovereign-act shape (e.g., epic `proposed → active`) but lacks the force trailer
   - Subcode `manual-edit` — change has no `aiwf-verb:` trailer at all (overlaps with `provenance-untrailered-entity-commit` but with FSM-specific framing)
4. **Hint entry** in `internal/check/hint.go` for the new code, per policies/finding_hints.go.
5. **Test fixtures** under `internal/check/testdata/fsm-history-consistent/` with at least one case per subcode plus a positive (clean) baseline.
6. **Update the audit catalog** (`docs/pocv3/design/legal-workflows-audit.md`) to remove R-RULE-149's "currently unimplemented" qualifier and remove the "pending G-0132" note from §10.1's enforcement-status legend.

## Severity escalation (optional follow-up)

Once `fsm-history-consistent` lands, the existing `provenance-untrailered-entity-commit` (warning) could be **strengthened to error** in the FSM-touching subset, since the new finding gives the auditable record. Whether to do that flip here or in a follow-up is a design call best made during implementation. Default: defer.

## What this milestone does *not* do

- Does not refactor the existing `provenance-untrailered-entity-commit` (separate work).
- Does not rewrite `aiwf history` to surface the new finding (history already prints all findings via `aiwf check` output).
- Does not produce migration tooling for repos with pre-existing FSM-illegal commits — the finding fires, the operator resolves with verbs or `--force` audits.

## At wrap

Promote G-0132 to `addressed`:

```
aiwf promote G-0132 addressed
```

(Per the gap FSM: `open → addressed | wontfix`.) Add `addressed_by: [M-0130]` to G-0132's frontmatter in the same wrap commit.

## Related

- **G-0132** — the gap this milestone closes
- **R-RULE-149** in `legal-workflows-audit.md` — the spec entry
- **ADR-0011** — methodology committing FSM-as-tree-invariant
- **CLAUDE.md §Engineering principles** — *"framework correctness must not depend on the LLM choosing to enforce"*

### AC-1 — fsm-history-consistent check rule registered; walks git log per entity

### AC-2 — manual-edit subcode: status-change commit lacks aiwf-verb trailer entirely

### AC-3 — Hint table entry for fsm-history-consistent and each subcode

