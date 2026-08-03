---
id: G-0490
title: tdd.require_test_metrics is retroactive over all history and unadoptable
status: addressed
priority: medium
addressed_by_commit:
    - dda7ab043
---
## What's missing

The `tdd.require_test_metrics` knob gates the `acs-tdd-tests-missing` warning:
when true, every AC at `tdd_phase: done` under a `tdd: required` milestone must
carry an `aiwf-tests:` trailer on some commit in its history, or the check warns
(`internal/cli/check/tests_metrics.go:57`; documented at
`internal/config/config.go:503-509`). It defaults to false.

The rule walks an AC's **entire** history and has no forward-scoping — no
enable-time watermark, no date or commit boundary, no grandfathering. So the cost
of turning it on is proportional to a repo's accumulated history rather than to
its future discipline.

Measured on this tree: **783 ACs sit at `tdd_phase: done`, against 275 commits in
all history carrying an `aiwf-tests:` trailer.** Even taking every one of those
commits as a distinct AC, enabling the knob fires on the order of 500 warnings.

## Why it matters

Those warnings are not actionable. The trailer is written by
`aiwf promote <id>/AC-N --phase ... --tests "pass=N fail=N skip=N"` at the moment
the phase advances; an AC that reached `done` without it cannot acquire one now
except by back-stamping — which is precisely the dishonesty the rituals warn
against for the phase ladder itself, where a batch-stamped timeline is
indistinguishable from one written after the fact.

So the knob is adoptable only by a repo with no history. Every repo that has been
running aiwf long enough to want stricter governance is the one that cannot turn
it on. That inverts the intent: this is opt-in governance for consumers who want
more rigor, and the rigor is gated behind a debt they have no honest way to
clear.

aiwf's own tree is the demonstration — `aiwf.yaml` carries no `tdd:` block at
all, so the rule has never run here.

The severity is bounded: nothing is wrong today, the default is off, and no
behaviour is incorrect. What is lost is a shipped feature that no mature consumer
can use.

## Options

1. **Forward-scope the rule at enable time.** Record a watermark when the knob is
   switched on and only evaluate ACs that reach `done` after it. Preserves the
   rule's intent exactly, costs a stored boundary and the decision of where it
   lives.
2. **Grandfather by walking once at enable time**, exempting the ACs already
   terminal. Same effect, no stored state, but the exemption set has to be
   recomputed or persisted, and a later-added historical AC would slip through.
3. **Document the limitation and leave the mechanism alone.** Cheapest; makes the
   constraint honest rather than surprising, and lets a greenfield consumer adopt
   the knob deliberately. Does nothing for an existing tree.
4. **Withdraw the knob.** Fewest moving parts if the forward-scoped version is
   not worth building, and it removes a surface that reads as available but is
   not.

Option 1 is the one that keeps the feature's value, and option 3 is a legitimate
stopping point if the feature is judged marginal — it converts a trap into a
documented boundary for one line of prose. Option 2's recomputation edge makes it
strictly worse than option 1 for the same work.

Note that any option touching the evaluation boundary sits adjacent to the TDD
phase machinery, which is tightly coupled and deliberately conservative — the
change should be scoped so it cannot perturb `acs-tdd-audit` or the red-first
gate.

## Related

- Discovered while measuring whether the knob could be enabled here as a cheap
  mechanical strengthening of review-finding discipline. It cannot; the
  convergence work proceeds without it.
