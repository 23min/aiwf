---
id: D-0025
title: Policy subsystem stays bespoke; one linter cleanup excepted
status: proposed
relates_to:
    - E-0042
    - M-0166
    - M-0167
    - G-0259
---
## Context

E-0042 planning asked whether `internal/policies/` — aiwf's bespoke CI
meta-tests — is over-built: whether a large subset reinvents `golangci-lint`
linters, and whether the firing-fixture meta-gate (`firing_fixture_presence`
plus the `grandfatherDark` ledger) is YAGNI. Two independent rethinks ran: an
architecture pass (proposing ~9 linter migrations, retiring the meta-gate, and
several deletes) and an adversarial empirical verify that wrote the candidate
configs and ran `golangci-lint` against fixtures.

## Decision

The policy subsystem **stays bespoke.** The proposed migrations and the
meta-gate retirement are **rejected**, with two cleanups adopted.

**Rejected (empirically refuted):**

- `depguard` ↔ `layering-direction`: `depguard` has no tier abstraction (it
  needs an O(n²) hand-maintained deny matrix) and silently passes a new
  untiered package — the "unplaced package must be placed" self-check is lost.
- Retiring the firing-fixture meta-gate for a per-policy table-test: darkness is
  **per-construction-site** (measured ~106 sites across 51 policies, 19
  multi-site). A per-policy table-test cannot see a dark site inside a
  multi-site policy, and loses the `-coverpkg` fail-closed guard. The meta-gate
  is strictly stronger.
- `ruleguard` ↔ the G-0235 write-discipline trio: not faithfully expressible
  (false positives on read-only `os.OpenFile`, or an unbounded per-statement
  shape matrix); no payoff churning days-old, AST-correct code.
- Deleting the frozen-snapshot policies (`m0137-ac3-batched-walker`,
  `acks-helper-lift`): both guard live structural properties no other test
  carries (the m0137 perf test is a 10-second budget, not a structural
  assertion; the acks sibling-tests only check that signatures compile).

**Adopted (empirically confirmed):**

- `no-time-now-in-core` → `forbidigo`: "no silent escape" survives (a new core
  package is forbidden by default) and `forbidigo` additionally catches an
  aliased-import blind spot the bespoke AST policy misses. Config:
  default-forbid + exclude-edge + `analyze-types: true`.
- Delete `filepath-join-segment-by-segment`: gocritic's `filepathJoin` checker
  is active and a clean superset; the repo has zero first-argument violations.

## Rationale

The bespoke complexity is largely load-bearing: the kept set is cross-file /
cross-language / per-construction-site checks (finding-code ↔ doc ↔ test
cross-references, the Cobra ↔ skill bijection, JSON-lockfile + shell-vs-YAML
agreement, markdown anchors) that no single-file linter expresses. The
empirical verify overturned the architecture pass's confident "expresses
cleanly" claims — a live instance of the "verify by measuring, not reasoning"
principle the policy subsystem itself exists to enforce.

## Consequence

- E-0042 / M-0166 burns down `grandfatherDark` by writing firing fixtures that
  assert ≥1 violation, **keeping the meta-gate**; scope is construction sites,
  not policies.
- E-0042 / M-0167 carries the two adopted cleanups plus the ledger annotation
  for `fsm-invariants` (the lone structure-auditor, unreachable by fixture).
- This decision is the durable record so the depguard / ruleguard /
  retire-the-meta-gate questions are not re-litigated. Relates to G-0259 (the
  audit that seeded the ledger) and G-0235 (the write-discipline trio).
