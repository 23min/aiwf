---
id: G-0605
title: Investigate type-aware static analysis for aiwf's self-validation
status: open
discovered_in: M-0313
---
## What's missing

Every Go-source check aiwf runs is a pattern match over a parsed syntax tree.
Syntax matches spellings; the rules are written about meaning. Where the two
agree a check works, and where a rename, an import alias, or an intermediate
binding separates them it silently does not.

Two concrete instances, read from the code rather than inferred:

- `no-time-now-in-core` compares the package identifier against the literal
  `"time"`, so an aliased import walks past it.
- `atomic-write-chokepoint` matches the same way, on package identifier plus
  selector name.

The scope is bounded and worth stating precisely: `go/ast` is imported by 66
files in `internal/policies` and three guard tests elsewhere. Every Go-source
analysis aiwf performs is self-validation, and none of it ships to a consumer.
So a type-aware layer would change how this repo checks itself and nothing
about the product.

What is missing is the evidence to decide whether that is worth doing —
specifically the four things the exploration names as conditions: a candidate
list of checks that are demonstrably spelling-bound and the drift each misses;
a measured load time for this module under `packages.NeedTypes|NeedSyntax|NeedDeps`;
a view on whether type-aware checks belong in CI rather than the inner loop;
and a shape for the layer that is shared rather than per-policy.

Analysis, including the advantages and disadvantages in full, is written up in
[`docs/explorations/10-type-aware-static-analysis.md`](../../docs/explorations/10-type-aware-static-analysis.md).

## Why it matters

The blind spot is shared by roughly sixty-six checks, so the arithmetic differs
sharply from fixing one rule: a layer built once amortizes, and a second bespoke
analyzer with its own loading and its own taint model is the present situation
with extra steps.

It also bounds what this repo can express. A rule of the form "this value must
not reach there" is a dataflow question, and today it can only be approximated
by naming conventions — which is how a check ends up asserting a spelling and
reporting health it does not have.

The counter-argument is real and is why this is aspirational rather than
planned. A type-checked, SSA-backed analysis is *more* code than what it
replaces, not less; it adds a large dependency; and it moves the policy suite
from seconds to tens of seconds on every run. It also would not close the
category, since "is this expression an assertion?" stays a judgment no type
information settles.

Nothing is blocked on this. It is recorded so the trade can be decided on
evidence when a second rule wants the capability, rather than re-derived from
scratch each time a syntax-only check is defeated.
