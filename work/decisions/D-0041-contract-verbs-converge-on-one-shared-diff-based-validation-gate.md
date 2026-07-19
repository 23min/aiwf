---
id: D-0041
title: Contract verbs converge on one shared diff-based validation gate
status: proposed
---
> **Date:** 2026-07-19 · **Decided by:** human/peter

## Question

The four contract-mutating verbs gate their writes in three unrelated styles:
an entity-id-filtered post-mutation check for bind, idempotency/`--force`
checks only for recipe install, a manual referential-integrity scan for recipe
remove, and no gate on unbind. Should they converge on one shared validation
concept, or is per-verb divergence justified by differing blast radii? The
answer was non-obvious because no defect has ever been observed here, the one
shareable helper is already shared, and bind's id-keyed scoping doesn't
transfer to verbs that mutate the validators map.

## Decision

Converge: build one shared validation gate that computes findings introduced
by a mutation as a before/after diff of contract-check findings on the
projected config, and route all four contract-mutating verbs through it.

## Reasoning

- The alternative (accept divergence, document it, revisit on a trigger) was
  the YAGNI-conservative option and was seriously weighed. Convergence won on
  uniformity grounds: one gate concept mirrors how the entity-tree verbs all
  converge on the projection check, a future contract verb inherits the gate
  instead of inventing a fourth style, and diff-based scoping is *more*
  correct than the current id-filter (which the audit noted is not a true
  before/after diff).
- The diff-based mechanism is the only shape that generalizes: id-filtering
  cannot scope findings for verbs that have no entity id (the recipe verbs).
- Cost is bounded and lands while the contract subsystem's context is warm.

## Consequences

- Built in E-0069's converged-gate milestone; per-verb bespoke checks are
  replaced, and remove keeps its precise referencing-ids error message on top
  of the shared gate.
- Verbs whose mutations cannot structurally introduce findings (unbind,
  remove) get the gate as a safety net; a diff that surfaces anything there is
  itself a signal of a contract-check regression.
