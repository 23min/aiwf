---
id: G-0150
title: aiwf-verb trailer values unpoliced against registered-verb closed set
status: addressed
discovered_in: M-0131
addressed_by_commit:
    - f66eb839
---
## Symptom

While committing M-0131/AC-1 from an LLM-driven session, the model
fabricated an `aiwf-verb: implement` trailer in a hand-rolled
Conventional-Commits code commit (`feat(...): ...`). Neither the
pre-commit hook (`aiwf check --shape-only`) nor the pre-push hook
(`aiwf check` full) flagged it. The bogus trailer survived
`golangci-lint`, `go test ./...`, and full kernel-tree validation.
The mistake surfaced only when the human asked "what does
`aiwf-verb: implement` mean?" — there is no aiwf verb named
`implement`.

Concrete commit (subsequently amended away): `c4719f2e` →
`204e39ae`, on `milestone/M-0131-cancel-target-state-aware`.

## Why it's a problem

- **Projection correctness.** `aiwf history <entity>` renders
  commits with `aiwf-verb: X` as kernel-verb invocations. A
  fabricated value conflates hand-rolled code commits with
  actual `aiwf` CLI runs — subtle but real misrepresentation.
- **Compounding drift.** LLM sessions and humans alike copy
  what they see. Without a chokepoint, fabricated trailer
  values propagate.
- **Invariant erosion.** The kernel principle "framework
  correctness must not depend on the LLM's behavior" assumes
  trailer values are mechanically validated. They are not.

## Why it was possible (4 reinforcing failures)

1. **No closed-set validation on `aiwf-verb` values.**
   `gitops.ValidateTrailer` validates some trailer values
   (e.g., `aiwf-force-for` must be 7-40 hex; introduced in
   M-0136). It does not validate `aiwf-verb` against the
   registered-verb set.

2. **`trailer-keys` policy is key-level, not value-level.**
   `internal/policies/trailer_keys.go` polices which keys
   appear (and that mutating verb commits carry the expected
   keys). It does not police values for closed-set keys.

3. **Two parallel sources of truth.** The Cobra command tree
   (`cmd/aiwf/` + `internal/cli/<verb>/`) is the canonical
   verb registry. The trailer-validation side does not
   reference it. Compare the completion-drift test
   (`internal/cli/integration/completion_drift_test.go`),
   which is the precedent for the right shape — but it
   guards completion wiring, not trailer values.

4. **LLM pattern-matching with no mechanical guard.** The
   model saw `aiwf-verb: promote` examples in CLAUDE.md and
   on aiwf-managed commits, pattern-matched "code commits
   should also have an `aiwf-verb` trailer," then invented
   the value `implement` when no real verb name fit. The
   kernel principle warns against exactly this dependency.

## Proposed fix

A new `aiwf check` finding rule, `trailer-verb-unknown` (or
similar). Mechanics:

- Walks git log over the merge-base-to-HEAD window (same
  shape as `fsm-history-consistent`).
- Parses each commit's trailer block.
- For commits carrying `aiwf-verb: X`, looks up X in the
  closed set of registered Cobra verbs (enumerated at runtime
  from the command tree, single source of truth).
- Fires a finding when X is unknown.

Severity: start as `warning` (advisory) so the rule can land
without retroactive breakage of any historical fabricated
trailers; promote to `error` once history is clean
(potentially by `aiwf acknowledge-illegal` over historical
strays, if any).

The rule belongs in the `aiwf check` history-walk pass — not
in `gitops.ValidateTrailer` — because hand-rolled commits
(`git commit -m ...`) bypass `gitops.ValidateTrailer`
entirely. Catching the failure mode requires reading commit
messages from git log, which is what the history-walking
rules already do.

## Generalization

The same problem class extends to every `aiwf-*` trailer key
with enum-like values that aren't currently policed:

- `aiwf-verb` → set = registered Cobra commands
- `aiwf-to` → set = the kind's allowed status set (derivable
  per `aiwf-entity` id at history-walk time)
- `aiwf-resolved-by-status` (if such a trailer exists) → similar

Free-form-value trailers (`aiwf-reason`, `aiwf-audit-only`) and
pattern-validated trailers (`aiwf-force-for`, `aiwf-actor`)
don't need closed-set checks.

A clean factoring: a single history-walk rule
`trailer-values-closed-set` that handles every enum-like
trailer key, sourced from a registry table mapping `key →
allowed-values-fn`.

## What this milestone does not do

This gap captures the design problem; the kernel-side fix is
a separate milestone (likely 1–3 ACs: introduce the
registry, wire `aiwf-verb` first, generalize to `aiwf-to`).

## Related

- M-0131 — the milestone where the failure surfaced
- M-0136 — introduced `gitops.ValidateTrailer` for
  `aiwf-force-for` (the precedent for value-level validation,
  but at the wrong layer for hand-rolled commits)
- CLAUDE.md §"Engineering principles" — *"framework
  correctness must not depend on the LLM's behavior"*
- `internal/cli/integration/completion_drift_test.go` — the
  source-of-truth pattern this gap's fix should mirror
