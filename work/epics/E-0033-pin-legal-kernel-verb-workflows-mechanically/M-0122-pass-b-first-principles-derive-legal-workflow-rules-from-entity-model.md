---
id: M-0122
title: 'Pass B first-principles: derive legal-workflow rules from entity model'
status: in_progress
parent: E-0033
depends_on:
    - M-0120
tdd: advisory
acs:
    - id: AC-1
      title: Catalog file exists at canonical path with top-level sections
      status: met
    - id: AC-2
      title: Catalog has per-kind lifecycle section for each entity kind
      status: met
    - id: AC-3
      title: R-FP-NNNN rule rows have non-empty schema fields
      status: met
    - id: AC-4
      title: R-FP-NNNN ids unique and contiguous starting at R-FP-0001
      status: met
    - id: AC-5
      title: Open questions for Pass C section present and non-empty
      status: met
---
## Goal

Derive the legal-workflow surface from first principles — the entity model, the kernel's six kinds, their lifecycles, and the cross-entity relationships — without reading M-0121's audit catalog. The output is a parallel-structure markdown catalog that Pass C (M-0123) reconciles against M-0121.

The independence from M-0121 is *load-bearing*: if first-principles derivation produces a catalog that matches existing surfaces, we have high confidence. If it diverges, those divergences become explicit decisions in Pass C.

## Methodology

For each entity kind (epic, milestone, AC, ADR, gap, decision, contract):

1. **Lifecycle** — enumerate the legal states + transitions (independent of `internal/entity/transition.go`).
2. **Birth conditions** — what `aiwf add <kind>` requires (parent exists, kind-specific flags, naming rules).
3. **Terminal states** — which states are terminal? When does the entity become archive-eligible?
4. **Cross-entity invariants** — what does this kind's state imply for sibling/parent/child entities? E.g., "an AC's lifecycle is bounded by its parent milestone's lifecycle."
5. **Verb closure** — which kernel verbs operate on this kind, and what's each verb's pre/post condition expressed against the lifecycle?

For verbs that operate across kinds (`archive`, `promote`, `add ac`, etc.):

6. **Cross-kind preconditions** — what's true about the planning tree before the verb runs?
7. **Cross-kind post-conditions** — what's true after?

## Output

A markdown file under `docs/pocv3/design/legal-workflows-first-principles.md` with the same row schema as M-0121's audit catalog:

```
| Rule id | Derivation | Scope | Statement | Severity if violated |
```

Rule ids are R-FP-001..N (separate id-space from R-AUDIT-NNN) so Pass C can reference both during reconciliation.

## Acceptance criteria

(Added via `aiwf add ac` once the catalog schema is settled.)

## Approach

- Author the catalog *without consulting* M-0121's output. Discipline matters here — if I peek, the cross-check loses its value.
- Use only:
  - The entity model from `docs/pocv3/design/design-decisions.md` (the six kinds, closed-set semantics).
  - Generic reasoning about lifecycles, ownership, and invariants.
  - ADRs that *define* the entity model (not ones that constrain workflows).
- Mark rules as *load-bearing* (this must hold or the model breaks) vs *conventional* (a sensible default but could be otherwise).

## What this milestone does *not* do

- Does not read M-0121's catalog.
- Does not reconcile (that's M-0123).
- Does not produce Go code.

### AC-1 — Catalog file exists at canonical path with top-level sections

### AC-2 — Catalog has per-kind lifecycle section for each entity kind

### AC-3 — R-FP-NNNN rule rows have non-empty schema fields

### AC-4 — R-FP-NNNN ids unique and contiguous starting at R-FP-0001

### AC-5 — Open questions for Pass C section present and non-empty

