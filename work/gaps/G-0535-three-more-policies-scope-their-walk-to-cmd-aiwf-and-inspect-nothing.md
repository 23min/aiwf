---
id: G-0535
title: Three more policies scope their walk to cmd/aiwf and inspect nothing
status: open
priority: medium
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

The scope is the visible half. Measuring what each policy would examine at the
location its subject moved to shows the subject did not simply move:

- **`PolicyApplyCallersAcquireLock`** asserts that every `run*` dispatcher
  calling `verb.Apply` directly also takes the repo lock. Its trigger is dead
  while its property is intact, and the two must not be confused. Production
  holds exactly two `verb.Apply` call sites — `internal/cellcoverage/fixture.go`
  and `internal/cli/cliutil/apply.go` — so "calls Apply directly" now identifies
  no dispatcher, and a rescope to `internal/cli/` fires once on
  `FinishVerbOutcome`, which the policy's own doc comment names as an exempt
  internal helper. But the obligation it polices is alive and is per-dispatcher:
  23 files under `internal/cli/<verb>/` call `cliutil.AcquireRepoLock`, and
  `internal/verb` never calls it at all. The lock is held at the dispatcher layer
  and nowhere else, so a scan over dispatchers is the right instrument; what it
  lacks is a trigger that identifies them now that reaching Apply is not one.
- **`PolicyIntegrationTestsAssertTrailers`** keys on test functions calling
  `runBin(`. That identifier appears in one file repo-wide:
  `internal/policies/firing_fixtures_single_site_test.go`, the policy's own
  synthetic fixture. The integration tests call `runVerb(`, `runSplit(` and
  `run(`. The heuristic names an entry point that only its own test creates, so
  the path prefix is not what makes it vacuous.
- **`PolicyTestsRealCloneNotUpdateRef`** finds no `update-ref refs/remotes/`
  under `internal/cli/integration/`. Rescoping yields a clean pass that is
  indistinguishable from the vacuous one it replaces.

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

This is not one change with three instances. Each policy needs its own answer to
"what is this policy's subject now, and is the property still worth asserting at
this layer" before any prefix moves, so the options below are the shapes an
answer can take rather than a menu to pick one from.

1. **Re-aim the policy at the subject's new form.** For the lock policy that
   means asserting against the one cliutil seam that reaches `verb.Apply`, not
   against a population of dispatchers; for the trailer policy it means keying on
   the entry points the integration tests actually use. Restores the property at
   a real chokepoint, and is the most work, because the assertion has to be
   redesigned rather than relocated.
2. **Retire the policy and record why.** Warranted where the property is enforced
   elsewhere, or where the subject has genuinely dissolved rather than moved.
   The lock policy is the weakest candidate for this, not the strongest: its
   obligation is held at 23 dispatcher sites and at no other layer, so retiring
   the scan would leave the only layer that takes the lock unpoliced. A named
   chokepoint should not disappear silently, so this costs a recorded decision
   wherever it is chosen.
3. **Give each an anti-orphan assertion**, as the sovereign policy now has — a
   live-tree test asserting the scanned prefix still holds subjects. Orthogonal
   to 1 and 2 and worth doing under either, because it converts the silent
   failure into a loud one. It does not, by itself, make any of these three
   assert anything.

No lean recorded, because option 1 and option 2 are separated by the open
question below rather than by cost.

## Open questions

One per policy, and they are not the same question. The lock policy has none:
the repo lock is taken at the dispatcher layer and at no other, so there is no
redundancy to weigh — only a trigger to choose. That distinguishes it from
G-0534, where `internal/verb` really does enforce the guarantee the dispatcher
scan also claims, and the redundancy question is live.

- **Lock policy** — what identifies a mutating-verb dispatcher now that reaching
  `verb.Apply` does not? Calling `cliutil`'s finish helper and membership in the
  mutating-verb set are the two candidates. A mechanical choice, no layering
  question under it.
- **Trailer policy** — the integration tests call `runVerb(`, `runSplit(` and
  `run(` rather than the `runBin(` the heuristic names. Whether the property is
  worth re-keying to those, or whether trailer assertions are better held by the
  per-verb tests that already make them, is the open part.
- **Real-clone policy** — with no `update-ref refs/remotes/` under the
  integration tests, is the property still one this repo can violate? If the
  shape it forbids cannot occur, retiring it is honest and re-aiming it is
  invention.

## Scope

The sibling sweep G-0476's scope section asked for: "whether any other policy
scopes its walk to a path prefix that a relocation has since emptied. The failure
mode is not specific to this one." Run while closing G-0476; these are the
results. G-0476 and G-0534 carry the sovereign policy, this carries the other
four.

`PolicyCLIHelperLocations` is included for the relocation question, not as a
vacuity finding — it examines `main()` and so can still fail.
