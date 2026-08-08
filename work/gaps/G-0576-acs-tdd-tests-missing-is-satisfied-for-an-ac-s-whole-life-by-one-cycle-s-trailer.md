---
id: G-0576
title: acs-tdd-tests-missing is satisfied for an AC's whole life by one cycle's trailer
status: open
priority: medium
discovered_in: M-0300
---
## What's missing

`acs-tdd-tests-missing` asks whether an acceptance criterion at `tdd_phase: done`
carries an `aiwf-tests` trailer anywhere in its history. It reads the criterion's
*entire* history and looks for any such trailer, so the question it answers is
"was evidence ever attached to this AC", not "was evidence attached to the cycle
that produced its current `done`".

An AC therefore satisfies the rule for the rest of its life once any single
cycle attaches a trailer. Measured with `tdd.require_test_metrics: true`: walk an
AC through red, green and done with `--tests "pass=N fail=0 skip=0"`, demote it
to `open`, then walk a second cycle attaching nothing at all and promote it back
to `met` — the AC is absent from the finding list, while two ACs that never
carried tests are reported.

The demote is reachable today: `aiwf promote <m>/AC-N open --force --reason ...`
is a sanctioned sovereign act, and it is what a refuted review finding calls for.

## Why it matters

The rule exists so a `done` phase means a test run stood behind it. After a
rework cycle the field still reads `done` and the check still passes, but the
evidence it rests on belongs to the cycle a reviewer just refuted. The guard
goes quiet at exactly the moment the evidence bar has been shown to be too low —
the same shape as G-0569, which names the neighbouring case where the *phase*
survives a revert it should not.

The two compound. G-0569's fix (reset the phase on a demote) restores
`acs-tdd-audit`, and leaves this rule satisfied by the first cycle regardless,
so repairing one without the other moves the soft spot rather than closing it.

The fix is to scope the history walk to the current cycle — the commits after
the most recent transition into `open` on that composite id — rather than the
AC's whole history. That boundary does not exist as a concept in the rule today.
