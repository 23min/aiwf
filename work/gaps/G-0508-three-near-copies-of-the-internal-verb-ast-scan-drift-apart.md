---
id: G-0508
title: Three near-copies of the internal/verb AST scan drift apart
status: open
discovered_in: M-0285
---
## What's missing

Four policies scan `internal/verb` for package-level function declarations, and each
carries its own copy of the walk: `WalkGoFiles`, filter on the `internal/verb/`
prefix, parse, iterate `Decls`, skip non-functions and bodyless declarations.

- `internal/policies/projection_findings_presence.go`
- `internal/policies/verb_write_guard_coverage.go`
- `internal/policies/noop_claim_scope.go`
- `internal/policies/claim_guard_presence.go`

They have already drifted. Three filter `fn.Recv != nil`; `noop_claim_scope.go` does
not, and that asymmetry is load-bearing — it is what makes a converging method
demand a ledger row the sibling policy can report as unvouchable. It is now pinned
by a test and a comment, but the pin defends one instance of a general condition.

`noop_claim_scope.go` also filters `_test.go` paths that `WalkGoFiles(root, true)`
has already excluded — a copy defending against something the shared helper handles,
which is what drift looks like before it becomes a defect.

## Why it matters

Each copy is a place where a plausible edit changes what a policy sees. The
method-filter asymmetry is the concrete case: adding the filter to the fourth copy
reads as consistency cleanup and would leave two policies green over an unguarded
converging method. A test now catches that specific edit. Nothing catches the next
one.

## Scope

Extract one scanner returning every package-level declaration under `internal/verb`
with the fields the four consumers need — path, line, called identifiers, and
whether the declaration has a receiver — and route all four through it. Each policy
then states its own receiver filter at its own site, so today's asymmetry is visible
code in two files rather than an unwritten dependency between two copies.

Prefer a field on the record over an `includeMethods` parameter; a flag hides the
same fact one layer down.

## References

- M-0285 — where the fourth copy landed and the asymmetry was pinned
