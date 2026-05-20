---
id: G-0147
title: Worktree aiwf binary discipline lacks a mechanical chokepoint
status: addressed
discovered_in: M-0130
addressed_by_commit:
    - 53bdb2261f38cc87553869e6498cbf228b158885
---
## What's missing

When a contributor — human or AI assistant — runs `aiwf check`, `aiwf doctor`, or any other `aiwf` verb during diagnostic work in a git worktree, the binary on PATH (typically `/go/bin/aiwf` from a prior `go install`) was built from a state that may pre-date the worktree's current source. Invocations look authoritative but produce results computed from stale code.

Concrete failure surfaced during M-0130 AC-1/AC-2 self-review: a phantom `fsm-history-consistent/illegal-transition` finding from the PATH binary masked the fact that the AC-1 walker design was DAG-naive. The misfire only became visible because the operator (human Q&A) prompted a deliberate verification pass against the worktree's current source, exposing the gap between what the binary reported and what the code actually does.

There is no mechanical chokepoint preventing this today. The kernel's *"framework correctness must not depend on LLM behavior"* principle is violated by silent reliance on operator recall.

## Why it matters

- Diagnostic results that look authoritative but reflect stale code are worse than no diagnostic at all — they erode the human's ability to trust `aiwf check` as the validation contract.
- The risk is highest exactly where rigor matters most: during milestone work that changes `aiwf`'s own behavior. M-0130 is one example; any future kernel-rule milestone (new check rule, new verb gate, FSM extension) has the same shape.
- Both humans and AI assistants are affected. AI assistants in particular may keep using a stale binary across many invocations without re-checking; the silent drift accumulates across a session.
- The bug pattern is fully general: worktree branch A has code X; PATH `aiwf` was built from code Y at some prior time; operator runs `aiwf check` thinking they're testing X. The mismatch is silent.

## Resolution paths

- **Doc note in CLAUDE.md (lightweight, lands with this gap).** Codifies the discipline for humans and AI assistants — when diagnosing `aiwf` behavior against worktree-current source, build a worktree-scoped binary and invoke it by path. Advisory only; depends on operator recall to remember.

- **`make diag-aiwf` target (mechanical artifact, recommended next step).** A Makefile target that builds a worktree-scoped binary (e.g., to `bin/aiwf-diag` or `.aiwf/bin/aiwf`) and prints its absolute path. The project convention becomes *"for diagnostic work against uncommitted source, run `make diag-aiwf` and invoke the printed path."* Still operator-discipline, but with a documented mechanical artifact that's easy to remember, easy to reproduce, and small enough to add as a one-commit change. This is the recommended next concrete step on top of the doc note.

- **Worktree-local shim with PATH prepend.** A `bin/aiwf` script in the worktree that delegates to `go run ./cmd/aiwf "$@"`, with the worktree's `bin/` prepended to PATH at session start. Mechanical but slow (`go run` rebuilds each invocation; ~1-2s overhead per `aiwf` call). Probably not worth the friction.

- **Pre-tool hook (heavyweight, AI-only).** A `.claude/hooks/` script that intercepts the `Bash` tool's `aiwf ...` invocations and rewrites them to use the diag binary. Affects AI assistants only; humans still need the doc note. Heavy infrastructure for an asymmetric coverage.

## Class

Operator-discipline gap. Mechanical fixes exist but require small infrastructure work; doc note + this gap entry track the discipline until the `make diag-aiwf` target (or equivalent) lands.
