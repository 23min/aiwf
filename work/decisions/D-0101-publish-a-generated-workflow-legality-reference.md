---
id: D-0101
title: Publish a generated workflow legality reference
status: proposed
relates_to:
    - E-0089
    - M-0322
    - D-0077
---
> **Date:** 2026-09-22 · **Decided by:** human/peter

## Question

Should E-0089 publish a generated reference for the declared legal-workflow table,
or keep the table solely as a code-side artifact? D-0077 leaves publication open;
M-0322 requires the choice before render code is committed.

## Decision

Publish a committed Markdown reference at `docs/reference/workflow-legality.md`,
generated from the workflow specification by repository development tooling.
Expose applicability and its exclusion reasons, declared transition rows with
origin, verb, target, predicates and outcome, and global restrictions in a separate
section. Preserve rejection metadata where present. Do not add a consumer CLI
command solely to regenerate this repository's documentation.

State the boundary in the generated reference: coordinate coverage requires a
representative cell, not every possible target, argument or invocation. A listed
legal transition is subject to its predicates and global restrictions; a missing
request is not permission. This reference is not an executable workflow guide,
a complete command reference, a listing of branch choreography or anti-rules,
or a substitute for the reasoning in decisions and ADRs.

Compare the committed bytes with a fresh deterministic render in a normal Go test.
Make the regeneration command discoverable beside the generated file. Keep
rendering within the spec package's existing full-table access boundary, with a
small development entry point for writing the artifact.

## Reasoning

The table's target-specific outcomes and applicability declarations are already
checked against the kernel. A generated reference makes that information readable
without requiring readers to traverse Go literals. Predicates and global
restrictions must remain visible: presenting only origin, target and a green/red
outcome would turn conditional statements into unconditional advice.

Declining publication avoids a generator and another checked artifact, but leaves
readers to inspect source or consult narrative documents that serve a different
purpose. A new consumer CLI surface would add a public command lifecycle to a
repository-documentation task; development tooling is sufficient for this scope.

The byte comparison owns freshness, not semantic truth. Existing applicability,
coordinate, drift and outcome tests continue to own the specification's claims;
renderer tests must establish faithful, deterministic representation.

## Consequences

- `docs/workflows.md` remains the narrative guide. The generated reference does
  not repair its examples or settle the documentation defects tracked by G-0560.
  Rewriting that guide stays outside M-0322 whether publication is accepted or
  declined.
- `docs/design/legal-workflows-audit.md` and
  `docs/design/legal-workflows-first-principles.md` remain source catalogs for
  reconciliation and citations, not generated current-behavior references.
  Neither is retired, rewritten or archived by M-0322; that disposition remains
  outside E-0089 whichever publication choice is taken.
- Specification changes that affect the rendered bytes require regeneration.
  The renderer's freshness test owns that obligation and retires if the generated
  reference is deliberately withdrawn or replaced.
- M-0322 does not close broader workflow-documentation gaps. A faithful render
  cannot supply semantics absent from its input tables.
