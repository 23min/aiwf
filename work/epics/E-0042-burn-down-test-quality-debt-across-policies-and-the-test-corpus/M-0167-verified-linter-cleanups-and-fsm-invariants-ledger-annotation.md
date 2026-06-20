---
id: M-0167
title: Verified linter cleanups and fsm-invariants ledger annotation
status: in_progress
parent: E-0042
tdd: none
---
## Deliverable

The two empirically-verified policy cleanups from the E-0042 rethink (see
D-0025), plus the ledger annotation for the lone true structure-auditor.

1. **Migrate `no-time-now-in-core` to `forbidigo`.** Default-forbid
   `time.Now`/`time.Since`/`time.Until` repo-wide, then exclude the edge tiers
   (`internal/cli`, `cmd`) and the two allowlisted core packages (`repolock`,
   `htmlrender`), with `analyze-types: true`. Verified to preserve "no silent
   escape" (a new core package is forbidden by default) and to additionally
   catch the aliased-import blind spot the bespoke AST policy misses. Delete the
   bespoke policy once the `forbidigo` rule is proven to fire on a fixture and
   CI stays green.

2. **Delete `filepath-join-segment-by-segment`.** gocritic's `filepathJoin`
   checker is active and a clean superset; the repo has zero first-argument
   violations. Removing the bespoke policy and its `grandfatherDark` entry loses
   nothing.

3. **Annotate `fsm-invariants` in `grandfatherDark`.** It is the one policy no
   fixture can reach — it introspects compiled-in `entity` FSM symbols and
   discards `root`. Keep its ledger entry with a note that it routes through
   `mutate-hunt`, not a firing fixture.

## Why this milestone shrank

Its original premise — "the ~8 structure-auditors" — was wrong: there is exactly
one (`fsm-invariants`). The broader `depguard`/`ruleguard` migrations it might
have carried were evaluated and rejected on empirical grounds (D-0025); only
these two cleanups survived.

## Mechanical evidence

- `forbidigo` migration: a `golangci-lint` run over a fixture proving the rule
  fires on a new core package and stays silent on the edge/allowlisted packages;
  deleting the bespoke policy leaves CI green.
- `filepath-join` delete: gocritic `filepathJoin` firing on the same fixture
  shape; CI green after deletion.
- `fsm-invariants`: the annotation present in `grandfatherDark`;
  `TestPolicy_FiringFixtureNoStaleAllowlist` still green (it stays legitimately
  dark).

## Acceptance criteria

Pinned when the milestone starts.
