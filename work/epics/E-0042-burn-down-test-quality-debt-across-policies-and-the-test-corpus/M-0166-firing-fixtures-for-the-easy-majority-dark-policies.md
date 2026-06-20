---
id: M-0166
title: Firing fixtures for the easy-majority dark policies
status: done
parent: E-0042
tdd: none
acs:
    - id: AC-1
      title: Firing fixtures for the single-site dark policies
      status: met
    - id: AC-2
      title: Firing fixtures for the multi-site dark policies (every dark site)
      status: met
    - id: AC-3
      title: Firing fixtures for acks-helper-lift; ledger reduced to fsm-invariants
      status: met
---
## Deliverable

Burn down the `grandfatherDark` ledger in `internal/policies/firing_fixture_presence.go`
from its current 43 entries to exactly one (`fsm-invariants`), by adding a firing
fixture for every dark policy that can take one. A "firing fixture" is a Go test
that constructs a synthetic violating input, calls the policy, and asserts it
returns ≥1 `Violation` — thereby covering the policy's `Policy: "<id>"`
construction line so the firing-fixture-presence meta-gate has evidence the
policy can actually fire (G-0262, the burn-down of the G-0259 audit ledger).

The authoritative work-list (from the coverage profile): **43 dark policy ids /
85 dark construction sites.** `fsm-invariants` (7 sites) is the lone
structure-auditor — it introspects compiled-in `entity` FSM tables and discards
`root`, so no fixture reaches it; it **stays grandfathered permanently**. The
other 42 policies (~78 sites) get fixtures.

## Approach

Darkness is **per construction site**, so a policy leaves the ledger only when
**every** one of its dark `Policy:` lines is covered — multi-site policies need a
fixture per dark site (or one fixture that trips several). Fixtures are
table-driven by pattern cluster, reusing the established synthetic-root shape
(`mustWrite` a crafted file under a `t.TempDir()` root, call the policy, assert
the expected `Violation.Policy` id):

- **substring / func-body scan** — a single crafted `.go` under the temp root
  carrying the forbidden token or a func body missing the required guard.
- **missing-fixed-file** — the m0132 / devcontainer / CLAUDE.md family: an empty
  or malformed temp root trips the "missing or non-conformant file" line.
- **doc / finding-code** — a crafted `internal/check/` source plus minimal doc
  channels.

`trailer-order-matches-constants` and `trailer-keys-via-constants` already have
positive controls for their main lines; only their **defensive** drift lines
(file-not-found / parse-error / empty-set) remain dark and need targeted
fixtures — the entries are **not** stale.

## Mechanical evidence

After each AC, the covered policies' `grandfatherDark` entries are deleted and
both env-gated gate tests stay green under the coverage profile
(`make coverage-gate`):

- `TestPolicy_FiringFixturePresence` — no dark-and-not-grandfathered policy
  (proves the new fixtures actually lit the sites).
- `TestPolicy_FiringFixtureNoStaleAllowlist` — no grandfathered-but-lit entry
  (proves each removal was earned; the ledger cannot rot).

End state: `grandfatherDark = {fsm-invariants}`, both gates green.

## Acceptance criteria

### AC-1 — Firing fixtures for the single-site dark policies

**Deliverable** — Add firing fixtures for the ~25 single-dark-site policies
(`apply-callers-acquire-lock`, `authorized-by-via-allow`, `no-history-rewrites`,
`no-signature-bypass`, `verbs-validate-then-write`, … — the substring/func-body
one-liners) and delete their `grandfatherDark` entries.

**Mechanical evidence** — `make coverage-gate` green: `TestPolicy_FiringFixturePresence`
and `TestPolicy_FiringFixtureNoStaleAllowlist` both pass with the removed
entries; each new fixture asserts its policy returns ≥1 `Violation`.

### AC-2 — Firing fixtures for the multi-site dark policies (every dark site)

**Deliverable** — Add fixtures covering **every** dark site of the ~16
multi-site policies (the m0132/devcontainer/CLAUDE.md missing-file family,
`design-doc-anchors-valid`, `finding-codes-have-tests`, `race-parallel-cap`,
`read-only-verbs-do-not-mutate`, `test-setup-presence`, `m0137-ac3-batched-walker`,
`trailer-order-matches-constants` defensive lines, `trailer-keys-via-constants`
defensive line, `capture-stdout-singleton`, `embedded-rituals-no-retired-tracking-doc`,
`integration-tests-assert-trailers`, …) and delete their entries.

**Mechanical evidence** — `make coverage-gate` green with those entries removed;
the coverage profile shows zero remaining dark sites for each.

### AC-3 — Firing fixtures for acks-helper-lift; ledger reduced to fsm-invariants

**Deliverable** — Light all 11 dark sites of `acks-helper-lift` (the heaviest
policy: ~10 violation classes over its check-helper-lift audit) and delete its
entry, leaving `grandfatherDark = {fsm-invariants}`.

**Mechanical evidence** — `make coverage-gate` green; `grandfatherDark` contains
exactly `fsm-invariants`; `TestPolicy_FiringFixtureNoStaleAllowlist` confirms it
is the only legitimately-dark policy remaining.
