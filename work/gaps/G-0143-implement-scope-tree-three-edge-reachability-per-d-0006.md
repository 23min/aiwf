---
id: G-0143
title: Implement scope-tree three-edge reachability per D-0006
status: addressed
discovered_in: M-0123
addressed_by:
    - M-0141
---
## What's missing

Per **D-0006** (committed in M-0123 phase 1), scope reachability is a
three-edge tree: an `aiwf authorize` scope opened on entity X grants the
authorized agent reach over X plus X's children plus X's parent — but NOT
the full reference graph. Today the spec encodes this in the `scope-reach`
predicate vocabulary subject; the impl in `internal/scope/` is not yet
reconciled against the three-edge formal model.

## Why it matters

The kernel commits to "principal × agent × scope" provenance (CLAUDE.md
§"Provenance is principal × agent × scope"). Scope reachability is the
operational arm of that commitment — the chokepoint that decides whether a
given verb invocation by an authorized agent on entity Y falls inside or
outside the open scope on X. A mismatch between the spec's three-edge model
and the impl's actual reachability check is a silent provenance bug.

## Proposed fix shape

- Audit `internal/scope/` against D-0006's three-edge formulation.
  Specifically, for each opened scope on entity X:
  * X itself is reachable (self-edge),
  * X's parent (epic→milestone or milestone→AC) is reachable (up-edge),
  * X's children (epic.milestones, milestone.acs) are reachable (down-edge),
  * **no other entity is reachable**, including cross-references via
    `relates-to`, `supersedes`, `addressed_by`, etc.
- Add finding rule or verb-time refusal for "agent verb invocation
  outside scope tree" with a structured code (TBD — coordinate with
  the legality-classifier gap).
- Test: fixture trees with cross-references; assert verb refusal for
  the agent-on-cross-ref case.

## Open question

D-0006's predicate `scope-reach` is the spec's reference; the impl
mechanism may or may not be the same shape. This gap's first step is to
read both and surface any divergence.
