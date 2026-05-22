---
id: G-0156
title: cellcoverage.SatisfyPredicate falls through silently on unknown Op
status: open
discovered_in: M-0124
---
## Problem

`cellcoverage.SatisfyPredicate` (`internal/cellcoverage/fixture.go:366-429`) is a nested `switch` over `p.Subject` (outer) and `p.Op` (inner). For each Subject case, only the documented Ops are handled; other Ops fall through with no mutation. The function's silent-drift guard (`spec.EvaluatePredicate` re-check) at line 440 catches this — but the failure message is "fixture does not satisfy ..." rather than the more direct "unsupported Op for Subject ..."

Surfaced by the M-0124 reviewer agent's audit (post-wrap, pre-merge). The reviewer judged this *"robust-but-worth-tightening"* — the guard works, but the failure path is indirect.

## Why it matters

When a future predicate widens an Op for an existing Subject (e.g., adding `parent.tdd ∈ {required, advisory}`), the inner switch falls through silently and the silent-drift guard surfaces a misleading message. A developer reading the failure has to grep `SatisfyPredicate` and figure out the inner switch missed the new Op — extra cognitive cost. Explicit `default:` arms in each inner switch would surface the exact widening point.

## Fix outline

For each inner `switch p.Op` in `SatisfyPredicate`, add:

```go
default:
    t.Fatalf("SatisfyPredicate: Subject=%q Op=%q not implemented", p.Subject, p.Op)
```

(or equivalent — the message should include both Subject and Op so the gap is unambiguous).

Affected Subject cases: `self.addressed_by`, `self.superseded_by`, `self.tdd_phase`, `parent.tdd`, `any-child.status`, `any-child-ac.status`, `all-children-acs.status`. About 7 small inserts.

The existing silent-drift guard stays — it's the chokepoint that catches "the case ran but produced no mutation" for *correct* Ops where the mutation logic itself has a bug.

## Discovered in

M-0124 (reviewer-agent audit pre-merge).

## Status

`open`.
