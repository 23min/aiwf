---
id: G-0535
title: Three more policies scope their walk to cmd/aiwf and inspect nothing
status: open
---
## What's missing

Four policies scope their AST walk to `cmd/aiwf/`. That directory holds one Go
file — `main.go`, declaring `main()` — and no test files at all, because the
per-verb dispatchers moved to `internal/cli/<verb>/` and the dispatcher
integration tests to `internal/cli/integration/`.

Measured against the live tree, each returns zero violations, and three of the
four have nothing to return them from:

| policy | file:line | additional filter | functions examined |
| --- | --- | --- | --- |
| `PolicyApplyCallersAcquireLock` | `apply_callers_lock.go:28` | name prefix `run` | 0 |
| `PolicyIntegrationTestsAssertTrailers` | `integration_tests_assert_trailers.go:79` | test funcs calling `runBin(` | 0 |
| `PolicyTestsRealCloneNotUpdateRef` | `tests_real_clone.go:42` | `_test.go` files | 0 |
| `PolicyCLIHelperLocations` | `cli_helper_locations.go:52` | helper-name set | 1 (`main`) |

The first three are vacuous: no input reaches the predicate. The fourth is a
negative guard that still examines something, so it is degenerate rather than
vacuous — it asserts helpers have not appeared in a directory where nothing
would put them.

`docs/design/legal-workflows-audit.md` R-AUDIT-0051 carries the matching claim in
the normative spec: "Every `run*` dispatcher in `cmd/aiwf/` that calls
`verb.Apply` directly must also call `cliutil.AcquireRepoLock`". Doc and code
agree, and both name a scope with no dispatchers in it — so correcting the doc
alone would desync it from the policy. The row moves when the policy does.

## Why it matters

Three named chokepoints appear in the enforced set, run green on every push, and
inspect nothing. The properties they assert are real ones — the repo lock before
`verb.Apply`, trailer assertions in dispatcher integration tests, real clones
rather than `git update-ref` in tests — and a reader auditing whether any is
covered finds a policy named for it and stops.

The rule the four break is not "the walk is scoped wrongly" but that a path
prefix is an unchecked assumption about layout. A relocation invalidates it
silently, in the one direction no test reports: a policy that examines nothing
returns no violations, which is indistinguishable from a policy that examines
everything and finds nothing wrong.

## Options

1. **Rescope each to where its subject now lives, then resolve what fires.**
   `internal/cli/` for the lock policy, `internal/cli/integration/` for the two
   test-shaped ones. Precedent says budget for what the rescope reveals rather
   than for the one-line change: rescoping the sovereign policy the same way
   surfaced a second defect in its guard predicate, filed as G-0534.
2. **Give each an anti-orphan assertion**, as the sovereign policy now has — a
   test over the live tree asserting the scanned prefix still holds subjects.
   This does not fix a scope; it makes the next relocation fail loudly. Cheap,
   and it is what stops the class from recurring in the policies not yet
   rescoped.
3. **Delete the ones whose property is enforced elsewhere.** The repo lock is
   also held by `cliutil.AcquireRepoLock`'s own call sites and by the concurrency
   scenarios in `internal/stresstest`; whether that makes the policy redundant is
   a decision, not an observation.

Options 1 and 2 together are the lean, 2 first — an assertion that can fail is
worth more than a scope that happens to be right today, and it is what would have
caught all of these.

## Scope

The sibling sweep G-0476's scope section asked for: "whether any other policy
scopes its walk to a path prefix that a relocation has since emptied. The failure
mode is not specific to this one." Run while closing G-0476; these are the
results. G-0476 and G-0534 carry the sovereign policy, this carries the other
four.

`PolicyCLIHelperLocations` is included for the relocation question, not as a
vacuity finding — it examines `main()` and so can still fail.
