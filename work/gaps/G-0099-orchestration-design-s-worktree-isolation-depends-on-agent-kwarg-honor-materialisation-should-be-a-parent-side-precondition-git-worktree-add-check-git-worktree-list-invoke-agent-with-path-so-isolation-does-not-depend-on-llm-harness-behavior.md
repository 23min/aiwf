---
id: G-0099
title: Orchestration design's worktree isolation depends on Agent kwarg honor; materialisation should be a parent-side precondition (git worktree add → check git worktree list → invoke agent with path) so isolation does not depend on LLM/harness behavior
status: open
---
## What's missing

The orchestration design ([`docs/pocv3/design/agent-orchestration.md`](../../docs/pocv3/design/agent-orchestration.md) §6.2, as originally written) specified the LLM driver as dispatching subagents "via Claude Code's `Agent` tool with `isolation: \"worktree\"`". That put worktree materialisation on a kwarg passed into the Agent invocation — i.e., a request to the harness — with no parent-side check that the kwarg was honored.

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

`isolation: "worktree"` as an Agent kwarg was exactly a remember-to-do-it contract — the same class as G-0067 ("`wf-tdd-cycle` is LLM-honor-system advisory") and the same class as every kernel surface we've already hardened. When the orchestration design shipped as originally written, every parallel-TDD cycle that depended on worktree isolation inherited this softness; the substrate's correctness silently depended on the LLM driver having passed the right kwarg and the harness having honored it.

## Resolution shape

[ADR-0009](../../docs/adr/ADR-0009-orchestration-substrate-substrate-vs-driver-split-trailer-only-cycle-events-isolation-as-parent-side-precondition.md) **Decision 3** captures the isolation-as-precondition rule in `proposed` form. The design doc has been amended accordingly:

- **`docs/pocv3/design/agent-orchestration.md` §6.2** — now specifies the precondition pattern (`git worktree add` → `git worktree list` presence check → invoke agent with path); explicitly demotes `isolation: "worktree"` to a hint.
- **`docs/pocv3/design/agent-orchestration.md` §7.7** — new section "Isolation as parent-side precondition (closes G-0099)" with the post-cycle reconciliation rule and the `isolation-escape` finding.
- **`docs/pocv3/design/agent-orchestration.md` §8** — adds an L4 row to the scope-enforcement summary for isolation reconciliation.

This gap closes when:

1. ADR-0009 ratifies (`proposed → accepted`), AND
2. The implementing milestone under E-0019 lands the kernel/driver code for the precondition + reconciliation pair (kernel finding `isolation-escape`, driver-side dispatch sequence per §6.2 steps 3 and 4 and 7, cycle-id trailer schema sufficient for kernel-checkable reconciliation per Decision 3's check-site question).

The exact check site for step 4 of the resolution above — kernel `aiwf check` rule, orchestrator-side code, or both — is still being shaped under ADR-0009's `proposed` window and will be pinned when that ADR ratifies.
