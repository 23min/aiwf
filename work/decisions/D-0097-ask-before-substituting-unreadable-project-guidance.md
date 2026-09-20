---
id: D-0097
title: Ask before substituting unreadable project guidance
status: proposed
relates_to:
    - E-0094
    - M-0345
---
> **Date:** 2026-09-20 · **Decided by:** Peter

## Question

When a project guidance index exists but cannot be inspected, may an assistant
substitute legacy engineering guidance to keep working?

## Decision

Report the unreadable index and do not automatically fall back to legacy guidance.
Continue work independent of the missing guidance. Ask the operator before
proceeding with engineering work that depends on it.

## Reasoning

An unreadable index does not establish that project ownership is absent. Legacy
fallback could apply rules the repository deliberately replaced or excluded,
including when its installed selection is empty. Automatic fallback would keep
engineering work moving, but would silently choose policy on the operator's
behalf. Continuing independent work preserves useful progress without that choice.

## Consequences

M-0345's shared host routing must state this behavior. Fresh-session observations
must distinguish unreadable guidance from a missing index. File generation and
review alone do not prove that an assistant follows the instruction. This decision
does not change the already specified fallback for a missing or unowned index.
