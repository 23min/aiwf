---
id: M-0167
title: Verified linter cleanups and fsm-invariants ledger annotation
status: in_progress
parent: E-0042
tdd: none
acs:
    - id: AC-1
      title: Migrate no-time-now-in-core to forbidigo, delete the bespoke policy
      status: open
    - id: AC-2
      title: Delete filepath-join-segment-by-segment; annotate fsm-invariants in ledger
      status: open
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

### AC-1 — Migrate no-time-now-in-core to forbidigo, delete the bespoke policy

**Deliverable** — Replace the bespoke `no-time-now-in-core` policy with a
`forbidigo` rule in `.golangci.yml`: default-forbid `time.Now`/`time.Since`/
`time.Until` repo-wide with `analyze-types: true`, then exclude the edge tiers
(`internal/cli`, `cmd`) and the two allowlisted core packages
(`internal/repolock`, `internal/htmlrender`). Delete the bespoke policy, its
registration, and its `grandfatherDark` entry.

**Why** — Empirically verified (D-0025, round 2): the `forbidigo` rule preserves
"no silent escape" — a new core package is forbidden by default — and
additionally catches an aliased-import (`t "time"; t.Now()`) blind spot the
bespoke AST policy misses.

**Mechanical evidence** — A structural assertion that `.golangci.yml` carries
the rule (the default-forbid `time.{Now,Since,Until}` pattern,
`analyze-types: true`, and the exclude list of edge tiers + allowlisted core),
mirroring how `race-parallel-cap` asserts the Makefile and workflows carry
`-parallel 8`. The one-time "it fires" proof is recorded in the round-2 verify.
The lint job stays green.

### AC-2 — Delete filepath-join-segment-by-segment; annotate fsm-invariants in ledger

**Deliverable** — Delete the bespoke `filepath-join-segment-by-segment` policy,
its registration, and its `grandfatherDark` entry — gocritic's `filepathJoin`
checker (active under the `diagnostic` tag) is a clean superset. In the same
edit, annotate the kept `fsm-invariants` entry in `grandfatherDark` with a note
that it routes through `mutate-hunt`, not a firing fixture: it introspects
compiled-in `entity` FSM symbols, so no fixture can reach it.

**Mechanical evidence** — A gocritic fixture test confirming `filepathJoin`
fires on a `filepath.Join` argument with an embedded separator (a
failing-if-gocritic-disabled assertion), and
`TestPolicy_FiringFixtureNoStaleAllowlist` staying green — the deleted id is
gone from both the policy corpus and the ledger, and `fsm-invariants` stays
legitimately dark with its explanatory note.

