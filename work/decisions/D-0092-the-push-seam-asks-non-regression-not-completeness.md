---
id: D-0092
title: The push seam asks non-regression, not completeness
status: proposed
relates_to:
    - E-0084
    - M-0331
    - ADR-0048
    - G-0571
---
> **Date:** 2026-09-14 · **Decided by:** human/peter

## Question

ADR-0048 fixed what each write seam asks: a create must be complete, an edit
must not regress. It left the third seam open, describing the push gate only as
a gate riding the commit range the provenance audit resolves, scoped to entities
whose body content the range changed, at error severity — without saying which
question it asks of that content. Completeness and non-regression both fit that
description, and they differ in who they block.

## Decision

The push seam asks non-regression. A commit that drops a required section an
entity's body carried is refused; one that keeps an omission already present at
the range base is not. A commit that creates an entity is not judged here at
all — `aiwf add` already holds a create to completeness, and the sovereign
`--force` it offers stays in force afterwards.

## Reasoning

Completeness at the push contradicts the verb it rides behind. `aiwf edit-body`
permits an edit that keeps an existing omission, and commits it; a completeness
gate would then refuse that very commit. Neither path carries `--force`, so
there would be no way to satisfy both — two parts of one system giving opposite
answers about one edit.

It also reproduces, one seam later, the cost ADR-0048 measured and rejected at
the edit seam: refusing an author over an omission they did not introduce. That
cost is not hypothetical here. Measured 2026-09-14 against the active tree, 55
live entities omit at least one required section — 109 omissions between them,
concentrated in 24 decisions and 30 gaps — and every one of those 55 carried a
body-content change in the preceding 180 days, across 91 commits. Under
completeness each of those pushes is refused until the author writes a section
into somebody else's entity. For the born-complete kinds the remedy is not
"add the heading" either: an empty one is an error-severity `entity-body-empty`,
so it is "write the section".

Reporting an inherited omission as a warning beside the error for a newly
dropped one was the remaining alternative. It loses as a second copy of what a
tracked record already holds, delivered at the least useful moment — mid-push,
on unrelated work — and a warning nobody must act on erodes the list for the
ones that do.

A baseline ledger grandfathering the existing omissions by id would converge
everything outside it, and is the shape G-0571 names. It loses because the
ledger is a mandate with no named owner and no retirement trigger.

## Consequences

Nothing converges the bodies already committed without a required section. Every
seam reads only what is being written, so the debt is permanent rather than
merely current, and closing it would need a tree-side rule judging against a
baseline — deliberately not built. ADR-0048's Consequences names this seam as
what would change that; it does not, and that sentence is corrected to say so.
