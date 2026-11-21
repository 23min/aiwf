---
id: M-0366
title: Model verb commit output in a ratified spine
status: draft
parent: E-0100
depends_on:
    - M-0364
    - M-0365
tdd: none
---
## Goal

A ratified spine for what a verb commits under ADR-0058, an amended commit or a new one, through
a push, a push whose reply is lost and the `pull --rebase` after it, a branch cut and a merge,
with what `aiwf check` reads after each; and a record of where today's `aiwf` departs from it.

## Closes

- (none)

## Context

ADR-0058's Decision states when a verb amends and what the amended commit records. The
transactions model (`docs/research/transactions/model/RESULTS-amend.md`) explored amend designs,
and the amend prototype's scenarios (`docs/research/fewer-planning-commits/amend-prototype/`)
played the git cases against a patched build. Today's `aiwf` never amends, so the spine and the
binary differ wherever the spine amends. E-0100 records the risk that amend does not fit in one
spine.

## Acceptance criteria

## Constraints

- An amended commit carries the scope ids and operation identities the spines of
  M-0364 and M-0365 decide; this spine does not decide them again.

## Out of scope

- Any change to `aiwf`.

## Dependencies

- M-0364 and M-0365.

## References

- ADR-0058, ADR-0057 and ADR-0056.
- ADR-0022 and ADR-0036, which ADR-0058 builds on.
