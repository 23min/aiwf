---
id: G-0239
title: PolicyAcksHelperLift omits WalkAcknowledgedSHAEntities
status: addressed
addressed_by_commit:
    - 0228c5b1b8d3d52d190b9e91e95c166bde9a45dd
---
## What's missing

`internal/policies/acks_helper_lift.go` polices the single-compute / one-call-site contract for `WalkAcknowledgedSHAs` but does NOT police the new `WalkAcknowledgedSHAEntities` helper added by G-0231 item 3.

Specifically, the policy walks production Go source for calls to `WalkAcknowledgedSHAs` and refuses anything outside the canonical CLI gather layer (`internal/cli/check/check.go::Run`). Rule-internal recomputes are forbidden (violation class 3c). The new walker `WalkAcknowledgedSHAEntities` is called exactly once at the gather layer today (`internal/cli/check/check.go:122`), but no policy enforces this.

## Why it matters

Per CLAUDE.md "framework correctness must not depend on the LLM's behavior." The single-compute contract for the per-(SHA, entity) ack walker is currently policed only by code review and ordinary unit tests. Two regression paths the policy would catch:

1. A future rule that needs per-(SHA, entity) acks duplicates the walker inside its own file (instead of consuming from the gather-layer-passed parameter). Silent perf regression + drift risk.
2. A refactor accidentally drops the call at the gather layer; the rule degrades to "nil map → no acks suppress findings" and historical findings re-fire as errors, silently breaking the suppression contract.

`PolicyAcksHelperLift` is the kernel's chokepoint for both regressions for the legacy walker; extending it to the new walker is the parallel guarantee.

## How to fix

~30 lines of mechanical AST work parallel to the existing structure. Specifically: add `WalkAcknowledgedSHAEntities` to the policy's allowlist of names it polices; assert exactly one production-source call site (the gather layer); refuse rule-internal call sites. Mirror the existing test discipline at `internal/policies/acks_helper_lift_test.go`.

## Source

G-0231 reviewer pass, N6 finding ("extend PolicyAcksHelperLift to police WalkAcknowledgedSHAEntities with the same single-compute / provenance / one-consumer contract").
