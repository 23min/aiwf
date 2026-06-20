---
id: M-0167
title: Verified linter cleanups and fsm-invariants ledger annotation
status: in_progress
parent: E-0042
tdd: none
acs:
    - id: AC-1
      title: Migrate no-time-now-in-core to forbidigo, delete the bespoke policy
      status: cancelled
    - id: AC-2
      title: Delete filepath-join-segment-by-segment; annotate fsm-invariants in ledger
      status: met
---
## Deliverable

M-0167 carries one verified cleanup plus a ledger annotation. The `forbidigo`
migration originally planned as AC-1 was **reversed** after an independent
`wf-rethink` (config cannot satisfy the no-time-now obligations; the bespoke
policy is kept) — see D-0025 and G-0264.

1. **Delete `filepath-join-segment-by-segment`.** gocritic's `filepathJoin`
   checker is active and a clean superset; the repo has zero first-argument
   violations. Removing the bespoke policy and its `grandfatherDark` entry loses
   nothing.
2. **Annotate `fsm-invariants` in `grandfatherDark`.** It is the one policy no
   fixture can reach — it introspects compiled-in `entity` FSM symbols and
   discards `root`. Keep its ledger entry with a note that it routes through
   `mutate-hunt`, not a firing fixture.

## Why this milestone shrank

Its original premise — "the ~8 structure-auditors" — was wrong: there is exactly
one (`fsm-invariants`), and the broader `depguard`/`ruleguard` migrations were
rejected on empirical grounds (D-0025). The one surviving migration
(`no-time-now-in-core` → `forbidigo`) was then **reversed** by an independent
`wf-rethink` during this milestone: config path-scoping is per-linter (it would
silence the sibling `panic`/`os.Exit` rules) and a YAML rule supplies no
mechanical firing evidence. So M-0167 reduced to the single `filepath-join`
delete plus the annotation — and the rethink's testing surfaced a separate live
vacuous chokepoint (the dormant `panic`/`os.Exit` forbidigo rules), filed as
G-0264 and addressed by its own milestone.

## Mechanical evidence

- `filepath-join` delete: `TestGocriticFilepathJoinConfigured` — a structural
  guard that parses `.golangci.yml` and asserts `gocritic` is enabled, the
  `diagnostic` tag is on, and `filepathJoin` is not in `disabled-checks` (the
  three conditions under which `filepathJoin` fires, verified empirically against
  golangci-lint v2.12.2). It fails if the config drops the checker; CI green
  after deletion; `TestPolicy_FiringFixtureNoStaleAllowlist` green (the deleted
  id is gone from both the policy corpus and the ledger). Execution-firing —
  running `golangci-lint` against a fixture — is the structural guard's residual
  gap and is deferred to M-0170.
- `fsm-invariants`: the annotation present in `grandfatherDark`;
  `TestPolicy_FiringFixtureNoStaleAllowlist` still green (it stays legitimately
  dark).

## Acceptance criteria

### AC-1 — Migrate no-time-now-in-core to forbidigo, delete the bespoke policy

**[Cancelled — D-0025: the forbidigo migration was reversed after an independent
`wf-rethink`; `no-time-now-in-core` is kept bespoke. The dormant-forbidigo
finding the rethink surfaced is tracked as G-0264.]**

**Deliverable** — Replace the bespoke `no-time-now-in-core` policy with a
`forbidigo` rule in `.golangci.yml`: default-forbid `time.Now`/`time.Since`/
`time.Until` repo-wide with `analyze-types: true`, then exclude the edge tiers
(`internal/cli`, `cmd`) and the two allowlisted core packages
(`internal/repolock`, `internal/htmlrender`). Delete the bespoke policy, its
registration, and its firing-fixture test.

**Why reversed** — An independent `wf-rethink` reconstructed the enforcement from
intent and tested the integration end-to-end: forbidigo's per-linter path
scoping would silence the sibling `panic`/`os.Exit` rules on the edge; a YAML
rule has no `Violation` construction line, so the firing-fixture meta-gate has no
evidence it can fire; and the scope would decouple from `layerTier`, losing the
"no silent escape" guarantee. Verdict: keep the bespoke policy.

### AC-2 — Delete filepath-join-segment-by-segment; annotate fsm-invariants in ledger

**Deliverable** — Delete the bespoke `filepath-join-segment-by-segment` policy,
its registration, and its `grandfatherDark` entry — gocritic's `filepathJoin`
checker (active under the `diagnostic` tag) is a clean superset. In the same
edit, annotate the kept `fsm-invariants` entry in `grandfatherDark` with a note
that it routes through `mutate-hunt`, not a firing fixture: it introspects
compiled-in `entity` FSM symbols, so no fixture can reach it.

**Mechanical evidence** — `TestGocriticFilepathJoinConfigured`, a structural
guard that parses `.golangci.yml` and asserts `gocritic` is enabled, the
`diagnostic` tag is on, and `filepathJoin` is not in `disabled-checks` — the
three conditions under which `filepathJoin` fires (verified empirically against
golangci-lint v2.12.2). All three branches fail on a broken config, so the delete
cannot silently lose coverage. `TestPolicy_FiringFixtureNoStaleAllowlist` stays
green — the deleted id is gone from both the policy corpus and the ledger, and
`fsm-invariants` stays legitimately dark with its explanatory note. This is the
**structural** guard chosen at AC implementation (E-0042 planning); execution
firing — running `golangci-lint` against a fixture — is the residual gap M-0170
closes, generalizing it across `gocritic` and the dormant `forbidigo` rules.
