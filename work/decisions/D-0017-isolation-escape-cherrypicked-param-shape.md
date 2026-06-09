---
id: D-0017
title: isolation-escape cherryPicked param shape
status: proposed
---
The M-0106 `isolation-escape` rule needs per-commit cherry-pick
information (the gather layer identifies cherry-picks; the rule
suppresses findings for them, per AC-6). Three plausible shapes
for plumbing the signal from gather to rule:

1. **Add fields to `scope.Commit`** — e.g. `Committer string`,
   `Body string`, or a derived `IsCherryPick bool`. Existing
   rules ignore the new fields by virtue of the zero values.

2. **Extend `BranchOracle`** — add `IsCherryPick(sha) bool` to
   the existing interface. Same construction-time concern as
   `FirstParentBranches`; same per-call shape.

3. **Add a separate parameter to `RunIsolationEscape`** — pass
   `cherryPicked map[string]bool` alongside the oracle.

Cycle 4 of M-0106 chose option 3. The M-0106 retrospective
flagged this (F-6) as worth recording deliberately rather than
leaving as wrap-body decisions text.

## Decision

Option 3 (separate `cherryPicked map[string]bool` parameter on
`RunIsolationEscape`) was chosen and shipped.

## Honest rationale (the wrap-body was not honest)

The original M-0106 wrap-body's Decisions section claimed (subsequently amended via aiwf edit-body so the verbatim quote no longer survives at a stable line; the rationalization itself remains the subject of this decision):

> *"the cherry-pick info is rule-specific; extending
> `scope.Commit` would touch every consuming rule, and
> extending `BranchOracle` couples re-author detection with
> branch reachability."*

Of those three reasons:

- **"Rule-specific" is not true.** Future rules can absolutely
  need the same signal (a hypothetical
  `provenance-trailer-incoherent` extension for cherry-picked
  AI commits, say). Each new rule would re-derive or re-thread
  the signal.

- **"Extending `scope.Commit` would touch every consuming rule"
  is misleading.** `scope.Commit` is `{SHA, Trailers}` — a
  minimal struct. Adding optional fields (`Committer string`,
  `Body string`) does not require consumers to change; existing
  rules ignore the new data. This was the path of least
  refactoring effort at the moment.

- **"Coupling re-author detection with branch reachability"
  IS the correct reason** to NOT use the BranchOracle interface.
  Re-author detection is a property of the commit's git
  metadata (committer email, body text); branch reachability is
  a property of ref topology. Coupling them in one interface
  would force every consumer to care about both.

The shape that shipped will quietly accumulate similar
parameters as more checks need per-commit derived metadata.
Today's choice does not break anything; it represents a small
future-maintenance debt that should be discharged when:

- A second check rule needs cherry-pick info → time to either
  promote `IsCherryPick` to `scope.Commit` OR introduce a
  `CommitMetadata` carrier (separate from BranchOracle) that
  bundles per-commit derived signals.

- Three or more `nil` arguments accumulate at any
  `RunIsolationEscape`-like call site → time to refactor to a
  struct param.

## When superseded

This decision is rationally superseded when any of the above
conditions arrive. Until then, the current shape stays.

## Why a D and not an ADR

This is a tactical implementation-level choice, not a
kernel-architecture commitment. ADRs name architectural choices
the kernel commits to long-term; `D-NNN` records a deliberate
implementation-time decision that may be revisited as the
codebase evolves. The choice between options 1, 2, and 3 here
is reversible by any future refactor; it does not constrain
the kernel's broader contracts.
