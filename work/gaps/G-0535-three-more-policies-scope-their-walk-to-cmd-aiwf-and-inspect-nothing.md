---
id: G-0535
title: Three more policies scope their walk to cmd/aiwf and inspect nothing
status: open
priority: medium
---
## What's missing

Policies scope their AST walk to `cmd/aiwf/`. That directory holds one Go file —
`main.go`, declaring `main()` — and no test files at all, because the per-verb
dispatchers moved to `internal/cli/<verb>/` and the dispatcher integration tests
to `internal/cli/integration/`.

Four were found that way. `PolicyApplyCallersAcquireLock` is resolved and is
recorded here only for what its resolution establishes; the rest remain:

| policy | file:line | additional filter | functions examined |
| --- | --- | --- | --- |
| `PolicyIntegrationTestsAssertTrailers` | `integration_tests_assert_trailers.go:79` | test funcs calling `runBin(` | 0 |
| `PolicyTestsRealCloneNotUpdateRef` | `tests_real_clone.go:42` | `_test.go` files | 0 |
| `PolicyCLIHelperLocations` | `cli_helper_locations.go:52` | helper-name set | 1 (`main`) |

The first two are vacuous: no input reaches the predicate. The third is a
negative guard that still examines something, so it is degenerate rather than
vacuous — it asserts helpers have not appeared in a directory where nothing
would put them.

`PolicyApplyCallersAcquireLock` now walks `internal/cli/` with the `cliutil`
helper layer excluded by path, selecting dispatchers by package rather than by
name, and a live-tree test fails if the prefix stops holding dispatchers of
either naming shape. Its resolution is the reference case for the three below,
and it establishes two things they inherit: the trigger can be as dead as the
scope — its `run` name filter was case-sensitive and dropped every verb's `Run`
while matching each subverb's `run<Sub>` — and re-aiming is worth measuring
before it is scheduled, because the property turned out intact and only its
selector was lost.

The scope is the visible half. Measuring what each policy would examine at the
location its subject moved to shows the subject did not simply move:

- **`PolicyIntegrationTestsAssertTrailers`** keys on test functions calling
  `runBin(`. That identifier appears in one file repo-wide:
  `internal/policies/firing_fixtures_single_site_test.go`, the policy's own
  synthetic fixture. The integration tests call `runVerb(`, `runSplit(` and
  `run(`. The heuristic names an entry point that only its own test creates, so
  the path prefix is not what makes it vacuous.
- **`PolicyTestsRealCloneNotUpdateRef`** finds no `update-ref refs/remotes/`
  under `internal/cli/integration/`. Rescoping yields a clean pass that is
  indistinguishable from the vacuous one it replaces.

`docs/design/legal-workflows-audit.md` R-AUDIT-0051 tracked the lock policy and
moved with it, so the normative spec now describes the `internal/cli/` scope and
the helper exclusion. No row states a stale scope for the three below; each will
need its own when it resolves, since correcting a row ahead of its policy would
desync the two.

## Why it matters

Two named chokepoints appear in the enforced set, run green on every push, and
inspect nothing; a third examines only `main()`. The properties they assert are
real ones — trailer assertions in dispatcher integration tests, real clones
rather than `git update-ref` in tests — and a reader auditing whether either is
covered finds a policy named for it and stops.

The rule they break is not "the walk is scoped wrongly" but that a path
prefix is an unchecked assumption about layout. A relocation invalidates it
silently, in the one direction no test reports: a policy that examines nothing
returns no violations, which is indistinguishable from a policy that examines
everything and finds nothing wrong.

## Options

This is not one change with three instances. Each policy needs its own answer to
"what is this policy's subject now, and is the property still worth asserting at
this layer" before any prefix moves, so the options below are the shapes an
answer can take rather than a menu to pick one from.

1. **Re-aim the policy at the subject's new form.** For the trailer policy that
   means keying on the entry points the integration tests actually use. Restores
   the property at a real chokepoint, and is the most work, because the
   assertion has to be redesigned rather than relocated. This is how the lock
   policy resolved.
2. **Retire the policy and record why.** Warranted where the property is enforced
   elsewhere, or where the subject has genuinely dissolved rather than moved —
   the real-clone policy is the strongest candidate, since the shape it forbids
   no longer occurs anywhere the scan could reach. A named chokepoint should not
   disappear silently, so this costs a recorded decision wherever it is chosen.
   The lock policy is the caution against reaching for this first: its
   obligation looked dissolved because its selector had broken, and measuring
   the property's live sites is what separated the two.
3. **Give each an anti-orphan assertion**, as the sovereign policy now has — a
   live-tree test asserting the scanned prefix still holds subjects. Orthogonal
   to 1 and 2 and worth doing under either, because it converts the silent
   failure into a loud one. It does not, by itself, make any of these three
   assert anything.

No lean recorded, because option 1 and option 2 are separated by the open
question below rather than by cost.

## Open questions

One per policy, and they are not the same question. The lock policy's was
whether anything still identified a dispatcher once reaching `verb.Apply` no
longer did; something did, and it resolved mechanically, with no layering
question under it. That is what separates these from G-0534, where
`internal/verb` really does enforce the guarantee the dispatcher scan also
claims, so the redundancy question there is live.

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
results. G-0476 and G-0534 carry the sovereign policy; this carries the four the
sweep found, of which the lock policy has since resolved.

`PolicyCLIHelperLocations` is included for the relocation question, not as a
vacuity finding — it examines `main()` and so can still fail.
