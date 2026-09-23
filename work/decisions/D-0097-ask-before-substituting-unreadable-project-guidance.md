---
id: D-0097
title: Ask before substituting unreadable project guidance
status: accepted
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
Warn that the project guidance could not be loaded. Clearly labelled provisional
advice based on general knowledge may continue, provided the assistant states that
it may conflict with the unreadable project rules and does not claim to follow
them. Recommendations that depend on a project-specific rule require readable
guidance or operator direction.

## Reasoning

An unreadable index does not establish that project ownership is absent. Legacy
fallback could apply rules the repository deliberately replaced or excluded,
including when its installed selection is empty. Automatic fallback would keep
engineering work moving, but would silently choose policy on the operator's
behalf. A warning lets the operator distinguish general advice from advice grounded
in the project's selected guidance, preserving useful progress without silently
choosing a replacement policy.

## Consequences

M-0345's shared host routing must state this behavior. Fresh-session observations
must distinguish unreadable guidance from a missing index. File generation and
review alone do not prove that an assistant follows the instruction. This decision
does not change the already specified fallback for a missing or unowned index.
