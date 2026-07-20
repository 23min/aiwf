---
id: D-0044
title: Add ErrInternal marker to FinishVerbOutcome's err contract
status: proposed
relates_to:
    - M-0271
---

# D-0044 — Add ErrInternal marker to FinishVerbOutcome's err contract

> **Date:** 2026-07-20 · **Decided by:** human/peter

## Question

`cliutil.FinishVerbOutcome`'s `err` parameter, as landed under M-0271/AC-1,
only distinguished a `Coded` error (→ `ExitFindings`) from everything else
(→ `ExitUsage`). While migrating `archive`/`rewidth`/`import` onto it for
AC-2, `import`'s `LoadTreeWithTrunk` failure surfaced a real, actively-tested
case (`TestRun_LoadTreeWithTrunkFailure`) that must report `ExitInternal`
with a specific message, not `ExitUsage` — a code `FinishVerb` itself already
supported for its own nil-outcome/no-plan/apply-failure branches, but the
`err`-parameter path couldn't express for a caller-supplied error. Forcing
this case through the existing two-way split would have silently regressed
a tested exit-code contract.

## Decision

Add a small unexported `internalError` type plus an exported
`cliutil.ErrInternal(msg string) error` constructor. `FinishVerbOutcome`'s
err branch checks for it via `errors.As` (after the `Coded` check, before
the default) and reports `ExitInternal` instead of `ExitUsage`. Callers wrap
a message with `cliutil.ErrInternal(...)` wherever the failure is the
caller's own infrastructure breaking (a config/tree load failure, a domain
verb call erroring outright) rather than a usage mistake.

## Reasoning

Two alternatives were rejected:

- **A bespoke local envelope-emission helper surviving per verb**, just for
  this one case. Defeats AC-2's actual goal (deleting the `failX`/
  `emitXEnvelope`/`withCommitSHA` triads) for the sake of one branch.
- **Accepting the `ExitUsage` drift** on `import`'s `LoadTreeWithTrunk`
  failure. Rejected outright — it is a real, tested exit code an operator or
  script may already depend on, not an edge case safe to reshape quietly.

Applied consistently once introduced: `import`'s `LoadTreeWithTrunk` failure
keeps its exact original message and `ExitInternal` code. `archive`'s
`verb.Archive`-call failure and `rewidth`'s `verb.Rewidth`-call failure
(both previously `ExitInternal`, both `//coverage:ignore`d/untested
filesystem-failure edge cases) also route through `ErrInternal`, recovering
their exact original exit code rather than drifting to `ExitUsage` — no
reason to leave two of the three migrated verbs with a narrower, weaker
guarantee than the one that happened to have a test.

## Consequences

`FinishVerbOutcome`'s exit-code contract now has three error classes
(`Coded` → `ExitFindings`, `ErrInternal`-wrapped → `ExitInternal`, else →
`ExitUsage`) instead of two. Any future verb migrated onto `FinishVerb`/
`FinishVerbOutcome` with an early "my own infrastructure broke" failure
should reach for `cliutil.ErrInternal(...)` rather than reintroducing a
local envelope helper.
