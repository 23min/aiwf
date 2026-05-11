---
id: ADR-0009
title: 'Orchestration substrate: substrate-vs-driver split with trailer-only events'
status: proposed
---
# ADR-0009 — Orchestration substrate: substrate-vs-driver split, trailer-only cycle events, isolation as parent-side precondition

## Context

The orchestration of LLM subagents over aiwf entities (parallel TDD cycles, builder/reviewer pipelines, code-review and security-audit roles, doc-gardening passes, etc.) needs a clear split between the *substrate* aiwf provides and the *driver* an external orchestrator runs against. The exploratory design in [`docs/pocv3/design/agent-orchestration.md`](../pocv3/design/agent-orchestration.md) names three load-bearing choices about how that split is drawn. Each has been informally settled inside the design doc but never ratified; each is cited by subsequent work ([`parallel-tdd-subagents.md`](../pocv3/design/parallel-tdd-subagents.md), the E-0019 scope) as if it were ratified; and one of them (isolation handling) has now surfaced a concrete failure mode in a real session — [G-0099](../../work/gaps/G-0099-orchestration-design-s-worktree-isolation-depends-on-agent-kwarg-honor-materialisation-should-be-a-parent-side-precondition-git-worktree-add-check-git-worktree-list-invoke-agent-with-path-so-isolation-does-not-depend-on-llm-harness-behavior.md).

This ADR records the three choices so they have a named home, are cross-linked from the design doc, and have a single page reviewers can read to see what aiwf's orchestration model commits to. The implementation epic (E-0019) consumes these decisions; the design doc continues to carry the long-form rationale and worked examples.

## Decision

aiwf's orchestration substrate is defined by three load-bearing decisions.

### 1. Substrate-vs-driver split

aiwf provides the **substrate** for orchestration:

- **Agent / capability registry** — data describing what agents exist and what role each can take.
- **Pipeline schema** — data describing cycle structure for an epic's work-shape.
- **Trailer schema for cycle events** — audit primitives, per Decision 2.
- **Verb gate for subagent contexts** — the kernel dispatcher refuses subagent-forbidden verbs (`cancel`, `reallocate`, `authorize`, any `--force` invocation) regardless of role.
- **Reconciliation check rules** — post-cycle kernel-side validation (`scope-expanded`, `cycle-trailer-incomplete`, `isolation-escape` per Decision 3) that catches subagent misbehavior mechanically.

aiwf does **not**:

- Invoke LLMs.
- Manage worktrees (the driver materialises them per Decision 3).
- Parse subagent envelopes.
- Interpret pipelines or own multi-step control flow.

These are **driver** concerns. The driver lives outside the aiwf binary — today as host-specific skills under `.claude/skills/aiwf-*`, marker-managed by `aiwf init` / `aiwf update`. New hosts get new drivers, not kernel changes.

The split keeps the kernel small, kernel-checkable, and free of host coupling. Anything the driver does that affects planning state must ride through a kernel verb (which carries trailers and emits one commit); anything the driver decides without affecting state stays outside the kernel.

### 2. Trailer-only cycle event recording

Cycle events (begin, end, finding allocation, AC promotion within a cycle) are recorded as **structured trailers on existing verbs' commits**. There are no `aiwf cycle-begin` / `aiwf cycle-end` verbs. The kernel pins a set of cycle-related trailer keys (full surface specified in [`agent-orchestration.md`](../pocv3/design/agent-orchestration.md) §9):

- `aiwf-cycle-id` — composite id like `M-NNNN/AC-N#cycle-N`.
- `aiwf-cycle-status` — `ended-success | ended-failure | ended-discarded`.
- `aiwf-cycle-role`, `aiwf-cycle-agent`, `aiwf-cycle-model`, `aiwf-cycle-host` — provenance.
- `aiwf-cycle-pipeline-step` — position in the cycle's pipeline.
- `aiwf-cycle-worktree-branch` — git ref where the cycle's commits MUST live; recorded on every cycle commit (one trailer per commit, not just at cycle-begin) and consumed by the `isolation-escape` reconciliation rule (Decision 3). The redundancy across commits is deliberate: the check is decidable from any single cycle commit without traversing to find a cycle-begin marker.
- `aiwf-cycle-scope-hint` — coarse partition declared at cycle start, consumed by the `scope-expanded` reconciliation rule.
- `aiwf-cycle-prompt-hash`, `aiwf-cycle-duration-ms`, `aiwf-cycle-files-touched`, `aiwf-cycle-lines-added`, `aiwf-cycle-lines-removed`, `aiwf-cycle-tests-added`, `aiwf-cycle-findings-count`, `aiwf-cycle-findings` — observability.

Trailers are kernel-pinned (drift-tested in `internal/policies/trailer_keys.go`). `aiwf history <entity>` reads them via existing infrastructure; no separate event log file is introduced.

The rationale: cycle structure is *metadata on existing mutations*, not a new mutation class. Every cycle event is already a verb invocation (`aiwf add` of a finding, `aiwf promote` of an AC, etc.); adding `cycle-begin` / `cycle-end` verbs would double the commit count per cycle and create a parallel event-log surface alongside the trailer-driven history aiwf already owns. Trailer-only keeps the kernel surface small and consistent with `aiwf history` reading `git log` (a load-bearing commitment in `design-decisions.md`).

### 3. Isolation as parent-side precondition

Subagent isolation — the guarantee that a builder subagent's writes land in an isolated git worktree, not in the operator's live tree — is a **parent-side precondition**, not a request to the agent-dispatch tool. The dispatch sequence is:

1. Parent calls `git worktree add <path> <branch>`.
2. Parent verifies via `git worktree list` (or equivalent observable check) that the worktree exists at the expected path. If absent, parent refuses dispatch.
3. Parent invokes the agent-dispatch tool, passing the worktree path explicitly as the working directory.
4. Post-cycle reconciliation is a kernel `aiwf check` rule named **`isolation-escape`**. The rule walks every commit carrying an `aiwf-cycle-id` trailer, reads the `aiwf-cycle-worktree-branch` trailer (Decision 2) from the same commit, and asserts every such commit is reachable from that branch ref. A mismatch — commit not reachable from the declared worktree branch — fires the `isolation-escape` finding and the cycle ends as `ended-failure` regardless of the subagent's envelope status.

The `isolation: "worktree"` kwarg supported by some agent-dispatch tools (e.g., Claude Code's `Agent`) is a hint, not the load-bearing mechanism. Materialisation that depends on the kwarg being honored is structurally the same LLM-honor-system shape the framework's "correctness must not depend on LLM behavior" principle is designed to refuse — and the failure mode G-0099 documents (real session, isolation silently degraded to "we asked nicely") is the concrete evidence.

**Why kernel-side over driver-side.** A driver-side check ("orchestrator runs reconciliation post-cycle") would be one more layer of LLM-honor-system shape: a different driver implementation could skip the check, ship it as a non-blocking warning, or refactor it away — and the substrate's isolation guarantee would silently weaken to "whatever the current driver enforces." Putting the check in `aiwf check` makes isolation a kernel invariant that every driver (current Claude Code skill, hypothetical `aiwfdo` sidekick per §6.4 of the design doc, third-party drivers) inherits for free, validated at pre-push and in CI the same way every other `aiwf check` rule is. The check is decidable from `git log` alone — no filesystem inspection at validation time — so it composes cleanly with the rest of the kernel's tolerant-by-design loader.

**Driver responsibility under this rule.** The driver still owns steps 1-3 (materialise the worktree, verify presence via `git worktree list`, dispatch with the worktree path); this is parent-side by definition because it must happen before the agent is invoked. The driver also records `aiwf-cycle-worktree-branch` on every cycle commit so the kernel rule has the data it needs. The driver does **not** own the reconciliation decision; the kernel does.

### Sovereign override

The substrate's three decisions create chokepoints with no agent-side bypass:

- Decision 1's verb gate refuses subagent invocations of `cancel`, `reallocate`, `authorize`, `--force` regardless of prompt — sovereign acts trace to a named human, per the existing provenance model.
- Decision 2's trailers are kernel-pinned; missing or malformed trailers fire `cycle-trailer-incomplete`.
- Decision 3's reconciliation is mechanical; an isolation escape is a finding, gated by the F-NNN AC-closure rule (per [ADR-0003](ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md)).

Human operators retain the sovereign override surface that already exists — `aiwf promote F-NNNN waived --force --reason "..."` for findings that need waiving — but the override is human-only and carries the same provenance trail as any other sovereign act.

## Consequences

### Positive

- **The substrate-driver split is named.** Reviewers and future authors of orchestration-related skills, agents, and milestones have one ratified page that says "this is what aiwf provides; that is the driver's job." No more re-litigating §6.1 of the design doc every time orchestration comes up.
- **Cycle history rides existing infrastructure.** `aiwf history <entity>` already reads trailers; cycle events ride that pipe; no event log file, no graph projection — consistent with what aiwf commits to in `design-decisions.md`.
- **Isolation is structurally enforceable.** "Worktree didn't materialise" is observable before dispatch (precondition check) and after merge (reconciliation). The substrate's correctness for this surface does not depend on the LLM driver remembering to pass a kwarg.
- **Closes G-0099.** Decision 3 is the resolution shape G-0099 names.
- **Aligns with ADR-0001's mint-at-trunk model.** Under ADR-0001's eventual ratification, the post-merge `ids-unique` collision class for parallel cycles vanishes structurally, complementing this ADR's isolation guarantees.

### Negative

- **The driver layer is host-specific by design.** Different LLM hosts get different drivers. Today only Claude Code is in scope; a second host means a second skill set. The trade-off is intentional (kernel is host-agnostic, driver is host-specific) but it ships an unavoidable per-host cost.
- **Trailer surface grows.** ~17 cycle-related trailer keys (Decision 2, including `aiwf-cycle-worktree-branch`) are pinned in `internal/policies/trailer_keys.go`. Drift-test cost is small but real; trailer-schema migrations are a kernel concern.
- **Reconciliation check rules are kernel work.** The substrate-vs-driver split puts these on aiwf, which means new `aiwf check` rules (`scope-expanded`, `isolation-escape`, `cycle-trailer-incomplete`) ship as kernel code. Worth the cost — the alternative (driver-side checks) re-introduces the LLM-honor-system shape this ADR exists to refuse — but worth naming.
- **Kernel grows a worktree-aware check rule.** `isolation-escape` is the first `aiwf check` rule that reads git branch structure (via `git log --branches` / equivalent) rather than just the planning tree. Modest expansion of what `aiwf check` does; bounded to one rule.

### Implementation

- **E-0019** (Parallel TDD subagents with finding-gated AC closure) is the implementing epic; its milestones decompose the three decisions into kernel and driver-side work. Kernel-side work includes the `isolation-escape` `aiwf check` rule (Decision 3), the `aiwf-cycle-worktree-branch` trailer addition (Decision 2), and the existing-rule wiring for `scope-expanded` and `cycle-trailer-incomplete`. Driver-side work includes the precondition dispatch sequence (steps 1-3) and the trailer-recording at cycle-begin/end.
- **G-0099** closes when the `isolation-escape` kernel rule lands plus the precondition-pattern lands in the driver-side dispatch skill.
- **Design doc** [`agent-orchestration.md`](../pocv3/design/agent-orchestration.md) carries the long-form rationale, sequence diagrams, and worked examples; this ADR carries the load-bearing claims. The two are intended to be read together — design doc for "why and how," ADR for "what we committed to."

## References

- [`docs/pocv3/design/agent-orchestration.md`](../pocv3/design/agent-orchestration.md) — long-form orchestration design; sections §6.1, §6.2, §6.3, §7, §8, §9 are the source material for this ADR's three decisions.
- [`docs/pocv3/design/parallel-tdd-subagents.md`](../pocv3/design/parallel-tdd-subagents.md) — TDD-specific application of the substrate; consumer of these decisions.
- [ADR-0001](ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md) — eliminates the post-merge `ids-unique` collision class for parallel cycles; complementary to Decision 1's parallel-cycle safety story.
- [ADR-0003](ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) — F-NNN findings are the AC-closure gate this ADR's reconciliation rules feed into.
- G-0099 — orchestration design's worktree isolation is LLM-honor-system; Decision 3 is its resolution shape.
- E-0019 — implementing epic.
- [CLAUDE.md](../../CLAUDE.md) *Engineering principles* §"Framework's correctness must not depend on the LLM's behavior" — informs every decision in this ADR, especially Decision 3.
- [CLAUDE.md](../../CLAUDE.md) *Working with the user* / *Authoring an ADR* — informs the `proposed`-status discipline this ADR is drafted under (decision recorded; not yet ratified; iterated via `aiwf edit-body`).
