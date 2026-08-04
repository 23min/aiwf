---
id: G-0543
title: The golangci firing harness makes claims its own tests do not pin
status: open
priority: medium
---
## Problem

`TestGolangciConfigRulesFire` proves each guarded golangci-lint config rule
fires, by running the real binary against a fixture that violates exactly that
rule. The harness exists so a dormant rule cannot pass unnoticed. Three of its
own claims are pinned by nothing.

**The call site is unpinned.** The isolation measures that keep the harness from
losing golangci-lint's start-up lock — `--allow-parallel-runners` and a private
`GOLANGCI_LINT_CACHE` — live in `golangciFixtureCmd`, and the tests that assert
them assert against that constructor. Nothing asserts the harness calls it.
Reverting the call site to a bare `exec.Command` leaves every test green and
brings the machine-contention failure straight back.

**`requireGolangci`'s fail-closed arm is undriven.** With
`AIWF_REQUIRE_GOLANGCI` set, a missing binary must fail rather than skip — that
is the contract behind CI's claim that the chokepoint cannot be silently
skipped. No test enters the branch.

**There is no positive control.** A run refused for concurrency is now
recognized and reported as a refusal rather than as a dormant rule. Two other
ways a run can finish without ever applying the config still produce the
dormant-rule message unchanged: a `--config` path that does not resolve, and a
fixture that fails to typecheck. Both were reproduced.

## Why it matters

This is the condition the harness was built against, occurring inside the
harness. A gate that reports green while proving nothing is worse than an absent
one: every commit landing before the fix takes a false pass, and a reader
auditing coverage finds a guard and stops looking.

G-0264 is the dormant-rule instance that motivated building this harness.
G-0462 then found a row of the harness itself that could not fail, for an
unrelated reason. Two independent findings on one surface is the argument for
pinning what it claims rather than fixing instances as they surface.

## Direction

The positive control is the cheapest of the three and subsumes the third
entirely: every fixture already violates revive's `package-comments`, so
requiring that finding in each row proves the config was applied, whatever the
reason it might not have been — no enumeration of failure modes needed.

The other two are ordinary seam pins. Route the harness and the isolation tests
through one helper so there is a single seam to assert on, and extract
`requireGolangci`'s decision as a pure predicate so both arms are reachable from
a table test.

Worth settling while here: whether a positive control belongs in every execution
harness of this shape, or only this one.
