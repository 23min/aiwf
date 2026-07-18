---
id: G-0099
title: Worktree isolation must be a parent-side precondition, not an Agent kwarg honor
status: addressed
priority: high
addressed_by:
    - M-0106
---
## What's missing

The orchestration design ([`docs/pocv3/design/agent-orchestration.md`](../../docs/pocv3/design/agent-orchestration.md) §6.2, as originally written) specified the LLM driver as dispatching subagents "via Claude Code's `Agent` tool with `isolation: \"worktree\"`". That put worktree materialisation on a kwarg passed into the Agent invocation — i.e., a request to the harness — with no parent-side check that the kwarg was honored.

Real-session evidence: in a recent session the operator explicitly asked for a worktree-isolated subagent; the worktree was not created; the work landed in the live tree. Failure was silent — there was no parent-side precondition that would have caught the missing isolation.

The contract needs to flip from **request** to **precondition**: worktree materialisation verified before dispatch, and drift caught mechanically after the fact — regardless of whether the cause was a dropped kwarg, a harness bug, or an agent navigating out of its worktree by hand.

## Why it matters

Same kernel principle as the pre-push `aiwf check` hook, the `internal/policies/` drift tests, the completion-drift test, and the trailer-key invariant:

> **The framework's correctness must not depend on the LLM's behavior.** Skills are advisory; the pre-push git hook and `aiwf check` are authoritative. If a guarantee depends on the LLM remembering to invoke a skill, it is not a guarantee.

`isolation: "worktree"` as an Agent kwarg was exactly a remember-to-do-it contract — the same class as G-0067 ("`wf-tdd-cycle` is LLM-honor-system advisory") and the same class as every kernel surface we've already hardened. When the orchestration design shipped as originally written, every parallel-TDD cycle that depended on worktree isolation inherited this softness; the substrate's correctness silently depended on the LLM driver having passed the right kwarg and the harness having honored it.

## Resolution

Two layers close this gap, one pre-dispatch and one post-hoc mechanical.

**Pre-dispatch (session layer).** The [`.claude/hooks/validate-agent-isolation.sh`](../../.claude/hooks/validate-agent-isolation.sh) PreToolUse hook (registered in [`.claude/settings.json`](../../.claude/settings.json) with `"matcher": "Agent"`) denies any `Agent` dispatch passing `isolation: "worktree"` outright, making the unreliable kwarg unusable by construction. Contract pinned by `TestAgentIsolationHook_*` under `internal/policies/`. `CLAUDE.md` §"Subagent worktree isolation" names the operator-side complement: the parent bootstraps the worktree via `aiwf worktree add` before dispatch, verifies with `aiwf doctor --root`, and passes the worktree path explicitly to the subagent.

**Post-hoc (kernel layer).** M-0106 (under E-0030, `done`) shipped the `isolation-escape` `aiwf check` rule ([`internal/check/isolation_escape.go`](../../internal/check/isolation_escape.go), wired into `RunProvenanceCheck`) — a mechanical, pre-push-enforced rule that fires when an AI-actor's commits land on a branch other than the one bound by its active `aiwf authorize` scope (`aiwf-scope`/`aiwf-branch`/`aiwf-branch-sha` trailers, per [ADR-0010](../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s branch model). This is the load-bearing guarantee: it catches drift regardless of dispatch path — a subagent escaping its assigned worktree via `cd ..`, `git -C <other-path>`, or `git checkout main` fires the finding at the next push, independent of whether the session-layer hook was bypassed or the operator skipped a precondition step. E-0030's epic body names this rule as this gap's full closure.

## Provenance

[ADR-0009](../../docs/adr/archive/ADR-0009-orchestration-substrate-vs-driver-split.md) had proposed a different shape for the post-hoc layer — a cycle-id / `aiwf-cycle-worktree-branch` trailer scheme scoped to the (still-unstarted) E-0019 parallel-subagent orchestration substrate. It was rejected 2026-07-16: the rejection commit records that Decision 3 (the isolation-escape check) had already shipped via M-0106 using the existing provenance trailers, so the ADR no longer described what was actually built, and its other two decisions were speculative infrastructure ahead of a consumer.
