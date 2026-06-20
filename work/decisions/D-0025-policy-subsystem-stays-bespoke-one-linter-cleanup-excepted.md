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
meta-tests — is over-built. Three passes ran: an architecture rethink (proposing
~9 linter migrations, retiring the meta-gate, several deletes), an adversarial
empirical verify that ran `golangci-lint` against fixtures, and — during M-0167
— an independent `wf-rethink` of the one surviving migration (no-time-now →
forbidigo), reconstructed from intent per the ritual's §Independence and tested
end-to-end.

## Decision

The policy subsystem **stays bespoke.** The proposed migrations and the
meta-gate retirement are **rejected.** **One** cleanup is adopted (the
`filepath-join` delete); a second was initially adopted (no-time-now →
forbidigo) and then **reversed** after the deeper rethink.

**Rejected (empirically refuted):**

- `depguard` ↔ `layering-direction`: no tier abstraction (an O(n²)
  hand-maintained deny matrix) and silently passes a new untiered package — the
  "unplaced package must be placed" self-check is lost.
- Retiring the firing-fixture meta-gate for a per-policy table-test: darkness is
  per-construction-site (~106 sites / 51 policies, 19 multi-site); a per-policy
  table-test can't see a dark site inside a multi-site policy, and loses the
  `-coverpkg` fail-closed guard. The meta-gate is strictly stronger.
- `ruleguard` ↔ the G-0235 write-discipline trio: not faithfully expressible
  (read-only `os.OpenFile` false positives, or an unbounded statement-shape
  matrix); no payoff churning days-old, AST-correct code.
- Deleting the frozen-snapshot policies (`m0137-ac3-batched-walker`,
  `acks-helper-lift`): both guard live structural properties no other test
  carries.

**Reversed after the M-0167 `wf-rethink` (was initially adopted):**

- `no-time-now-in-core` → `forbidigo`. The round-2 verify confirmed forbidigo
  *fires* on `time.Now` in isolation, but an independent `wf-rethink` during
  M-0167 — reconstructing the enforcement from intent and testing the
  integration end-to-end — refuted it on the obligations:
  - Config path-scoping is per-*linter*, not per-pattern: excluding the edge for
    the time rule would also silence the sibling `panic`/`os.Exit` forbidigo
    rules (no clean coexistence without a brittle message-marker hack).
  - A YAML rule has no `Violation` construction line, so the firing-fixture
    meta-gate has zero evidence it can fire — failing the mechanical-evidence
    obligation the migration was meant to satisfy.
  - The scope would become a hand-maintained edge/exemption path list, decoupled
    from `layerTier` — losing the "no silent escape" (tier-derived) guarantee.

  The bespoke AST policy *is* the from-scratch design; the rethink verdict is
  **keep**. The migration is net-neutral churn that weakens two obligations.

**Adopted (empirically confirmed):**

- Delete `filepath-join-segment-by-segment`: gocritic's `filepathJoin` checker
  is active and a clean superset; the repo has zero first-argument violations.

## Rationale

The bespoke complexity is largely load-bearing: the kept set is cross-file /
cross-language / per-construction-site checks no single-file linter expresses.
"Verify by measuring, not reasoning" decided every reversal here — and the
forbidigo reversal is the sharpest instance: the isolated round-2 verify missed
the integration seam (existing rules, mechanical-evidence, tier-derived scope),
which the independent rethink + end-to-end testing caught.

## Consequence

- E-0042 / M-0166 burns down `grandfatherDark` with firing fixtures that assert
  ≥1 violation, keeping the meta-gate; scope is construction sites, not policies.
- E-0042 / M-0167 now carries only the `filepath-join` delete + the
  `fsm-invariants` annotation; AC-1 (the forbidigo migration) is cancelled.
- The rethink's empirical testing surfaced that the existing `panic`/`os.Exit`
  forbidigo patterns are **dormant** (match nothing under golangci-lint v2.12.2)
  — a live vacuous chokepoint, tracked as G-0264 and addressed by a new
  milestone.
- This decision is the durable record so the depguard / ruleguard /
  meta-gate-retirement **and** the no-time-now → forbidigo migration are not
  re-litigated. Relates to G-0259 (the audit that seeded the ledger), G-0235
  (the write-discipline trio), and G-0264 (the dormant-forbidigo finding).
