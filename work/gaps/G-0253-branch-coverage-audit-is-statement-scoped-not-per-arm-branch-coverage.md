---
id: G-0253
title: branch-coverage-audit is statement-scoped, not per-arm branch coverage
status: open
discovered_in: M-0066
---
## What's missing

The diff-scoped coverage gate landed by G-0067 (sub-goal a,
`internal/policies/branch_coverage_audit.go`) enforces *statement* coverage on
changed lines, not *branch* coverage. Go's `-cover` instrumentation records,
per basic block, how many times the block executed — not which arm of an
`if`/`switch`/`select` was taken. So a changed conditional whose block runs at
least once reads as "covered" even if one arm is never exercised.

The gate's v1 is therefore "diff-scoped statement coverage with the
`//coverage:ignore` escape." True per-arm branch correlation — asserting that
both the taken and the untaken arms of a conditional on a changed line are
exercised — is a strictly stronger property this version does not provide.

## Why it matters

The `wf-tdd-cycle` branch-coverage HARD RULE is specifically about branches:
its named failure mode is an untested *defensive branch* that ships subtly
wrong. Statement coverage catches a wholly-unexercised block but not a
half-exercised conditional, so the strongest form of the rule remains
honor-system even after G-0067's statement-scoped gate.

Candidate mechanisms:

- Go 1.20+ binary coverage (`GOCOVERDIR`) records per-block counts but still
  not per-arm; investigate whether the toolchain exposes branch-level data.
- AST-level arm enumeration: for each changed conditional, enumerate its arms
  and correlate each with a distinct covered block, flagging any arm with no
  covering block. More involved; G-0067's feasibility note flagged this as the
  hard part.

This is the deepening follow-up to G-0067's statement-scoped gate, not a
regression in it.
