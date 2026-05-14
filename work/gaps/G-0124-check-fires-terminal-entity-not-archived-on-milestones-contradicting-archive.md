---
id: G-0124
title: check fires terminal-entity-not-archived on milestones, contradicting archive
status: addressed
addressed_by_commit:
    - 7d75c78
---
## What's missing

The `aiwf check` rule `terminal-entity-not-archived` ([`internal/check/archive_rules.go:27-53`](../../internal/check/archive_rules.go)) fires for any entity whose status is terminal and whose file is not under an `archive/` path — including milestones. Per ADR-0004 §"Storage — per-kind layout" (and the verbatim copy in the `aiwf archive` verb's docstring at [`internal/verb/archive.go:40`](../../internal/verb/archive.go)), **milestones do not archive independently — they ride with their parent epic**. The `aiwf archive` verb correctly skips them.

This produces a standing false-positive warning on every check run whenever an epic carries done milestones but isn't itself terminal. Reproduced today: M-0091 (status `done`, parent E-0025 status `active`) triggers `terminal-entity-not-archived` while `aiwf archive` (dry-run) reports `no terminal-status entities awaiting sweep (tree is converged)`. The contradiction is on screen at the same `aiwf check` / `aiwf archive` invocation pair.

The aggregate `archive-sweep-pending` count inherits the bug — it counts the milestone false-positive and inflates the "N entities awaiting sweep" message by one for every done-milestone-under-active-epic case.

## Why it matters

Standing warnings that can't be acted on erode the trust signal of the check chokepoint. The rule's emitted message — `"awaiting aiwf archive --apply sweep"` — is actively misleading: running the verb does nothing for the cited entity. An operator (human or LLM) acting on the message learns the system lies about its own remediation paths, and starts ignoring the rule output broadly. That's exactly the failure mode the kernel's "framework correctness must not depend on LLM behavior" principle exists to prevent: when the chokepoint emits noise, both the LLM and the human train themselves to skip the chokepoint.

A second cost: the `archive-sweep-pending` aggregate is supposed to escalate to blocking past the configured `archive.sweep_threshold` knob. With milestones counted, the threshold trips early on healthy trees that just have done milestones under in-flight epics — making the escalation knob unreliable as a forcing function.

## Resolution shape

One-line skip in `terminalEntityNotArchived`:

```go
if e.Kind == entity.KindMilestone {
    continue // milestones ride with their parent epic per ADR-0004
}
```

Regression test under `internal/check/archive_rules_test.go`: add a fixture with a terminal milestone under an active epic; assert the rule does **not** fire on the milestone, but still fires on co-located terminal entities of other kinds (gap, decision) that would legitimately await sweep. The test locks the kernel/verb agreement: same fixture, both surfaces (`aiwf check` and `aiwf archive --dry-run`) report consistent answers.

The fix is one of M-0108's canonical examples of the cross-verb consistency rule it should encode — two surfaces (`check` and `archive`) carrying parallel definitions of "what's sweep-eligible" drifted, and only one was right. Worth mentioning as evidence-in-flight under E-0031 once landed, if the lesson generalizes.
