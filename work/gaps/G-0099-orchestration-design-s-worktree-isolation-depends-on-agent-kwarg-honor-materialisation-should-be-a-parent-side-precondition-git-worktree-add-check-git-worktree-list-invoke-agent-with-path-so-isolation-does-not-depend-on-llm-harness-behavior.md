---
id: G-0099
title: Orchestration design's worktree isolation depends on Agent kwarg honor; materialisation should be a parent-side precondition (git worktree add → check git worktree list → invoke agent with path) so isolation does not depend on LLM/harness behavior
status: open
---
## What's missing

The orchestration design ([`docs/pocv3/design/agent-orchestration.md`](../../docs/pocv3/design/agent-orchestration.md) §6.2) specifies the LLM driver as dispatching subagents "via Claude Code's `Agent` tool with `isolation: \"worktree\"`". That puts worktree materialisation on a kwarg passed into the Agent invocation — i.e., a request to the harness — with no parent-side check that the kwarg was honored.

Real-session evidence: in a recent session the operator explicitly asked for a worktree-isolated subagent; the worktree was not created; the work landed in the live tree. Failure was silent — there was no parent-side precondition that would have caught the missing isolation.

The fix is to flip the contract from **request** to **precondition**:

1. Parent calls `git worktree add <path> <branch>` *before* invoking the subagent.
2. Parent verifies via `git worktree list` (or equivalent) that the worktree exists at the expected path.
3. Parent invokes `Agent` passing the worktree path explicitly as the working directory; `isolation: "worktree"` becomes a hint, not the load-bearing mechanism.
4. On cycle return, parent verifies the subagent's commits live on the worktree branch and the diff is rooted in the worktree path (catches `cd ..` and similar escapes). Mismatch is a finding.

This generalises any "isolation didn't actually happen" failure — whether caused by a dropped kwarg, a harness bug, or an agent navigating out of its worktree — into a mechanical rule that fires regardless of why isolation broke.

## Why it matters

Same kernel principle as the pre-push `aiwf check` hook, the `internal/policies/` drift tests, the completion-drift test, and the trailer-key invariant:

> **The framework's correctness must not depend on the LLM's behavior.** Skills are advisory; the pre-push git hook and `aiwf check` are authoritative. If a guarantee depends on the LLM remembering to invoke a skill, it is not a guarantee.

Today, `isolation: "worktree"` as an Agent kwarg is exactly a remember-to-do-it contract — the same class as G-0067 ("`wf-tdd-cycle` is LLM-honor-system advisory") and the same class as every kernel surface we've already hardened. When the orchestration design ships as written, every parallel-TDD cycle that depends on worktree isolation inherits this softness; the substrate's correctness silently depends on the LLM driver having passed the right kwarg and the harness having honored it.

Surface area:

- **`docs/pocv3/design/agent-orchestration.md` §6.2** (the LLM driver) — should specify the two-step pattern (`worktree add` → check → invoke with path) and de-emphasise `isolation: "worktree"` as the mechanism.
- **`docs/pocv3/design/agent-orchestration.md` §7** (failure-mode taxonomy / quarantine) — should add an isolation-escape entry: post-cycle reconciliation verifies commits live on the worktree branch and the diff is rooted in the worktree path; mismatch fires a finding.
- **Possible kernel finding rule** — `worktree-isolation-mismatch` or similar, fired by the orchestrator's reconciliation step, AC-closure-gated like other findings.
- **ADR follow-up** — the orchestration substrate decisions (substrate-vs-driver split §6.1, trailer-only event recording §6.3, this isolation-as-precondition rule) currently live only in an exploratory design doc; lifting the load-bearing choices into a ratified ADR is the natural companion to closing this gap.
