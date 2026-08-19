---
id: E-0019
title: Parallel TDD subagents with finding-gated AC closure
status: proposed
---
## Status — deferred

This epic is **deferred** pending completion of the upstream substrate. Specifically:

1. The **agent-orchestration design substrate** in [`docs/archive/pocv3/agent-orchestration.md`](../../../docs/archive/pocv3/agent-orchestration.md) (landed 2026-05-08) needs to be considered fully finished — the substrate broadened the scope beyond this epic's original TDD-only framing into a general agent-orchestration model (agent registry, role-based concurrency, sub-scope provenance, per-epic pipelines, forensic bundles). When E-0019 unfreezes, its scope likely needs **rewriting** against the agent-orchestration model rather than the original four-fork framing captured in *Context* below.
2. The substrate's design must then be **implemented** — likely via one or more dedicated epics decomposed from the agent-orchestration doc once it stabilizes.
3. The blockers under *Dependencies* below. ADR-0003 is no longer among them — it was reversed by [ADR-0045](../../../docs/adr/ADR-0045-keep-six-entity-kinds-findings-live-in-entity-bodies.md), so the F-NNNN storage model and the two implementation epics that would have built it are withdrawn rather than pending. What remains open is a carrier for the gating property.

Treat the body below as the **original framing** preserved for historical reference. The actual work shape will be reassessed when the substrate is ready. Until then, this epic is a placeholder on the roadmap, not a queued execution target.

## Goal

Land **parallel TDD subagent execution with finding-gated AC closure**, so multi-AC milestones can run their cycles concurrently with mechanical guarantees against the M-0066/AC-1 branch-coverage drift class of bugs. The end state: a milestone with N independent ACs spawns N TDD-cycle subagents in worktree isolation; each runs its own red→green→refactor+audit; concerns surface as `finding` (F-NNN) entities; the human triages findings before AC closure; subagents structurally cannot waive their own findings.

## Context

The proximate trigger is **M-0066/AC-1**, where a long implementation session lost track of branch-coverage discipline mid-cycle. The TDD-cycle skill (`wf-tdd-cycle`) is advisory text — easy to drift through under the pressure of a long conversation. The fix isn't a stricter skill; it's a structural one: **bound the cycle's lifetime to a subagent invocation**, so the protocol is enforced by the runtime rather than by the LLM remembering rules.

The kernel already has the right primitives:

- `Agent({ isolation: "worktree" })` provides bounded-context execution with cheap rollback.
- `aiwf authorize` gates agent work via a typed scope FSM (active | paused | ended).
- `aiwf check` is the unified channel for "things the tree wants you to know" — and per the F-NNN ADR (filed alongside this epic), `finding` becomes the unified entity kind for those things needing human attention.
- The existing `aiwf promote` + `--force --reason` pattern (from M-0017) is the kernel's universal sovereign-act surface.

The design conversation that produced this epic resolved four forks (full synthesis at [`docs/archive/pocv3/parallel-tdd-subagents.md`](../../../docs/archive/pocv3/parallel-tdd-subagents.md)):

1. **Findings storage** → F-NNN as 7th entity kind + uniform archive convention.
2. **Subagent's edit surface** → hybrid: subagent moves AC state in worktree; parent allocates F-NNN post-merge.
3. **Resolution UX** → generic `aiwf promote` with `--force --reason`; soft check on missing fix link.
4. **Where findings get recorded** → settled by Fork 2: subagent returns JSON, parent commits findings serially in the main checkout.

Together with the AC closure chokepoint (`aiwf promote AC met` refuses on open linked findings), this gives a complete protocol where every step is enforced by the runtime or by `aiwf check`, not by skill text.

## Scope

- **Dedicated `tdd-cycle` agent definition** under `.claude/agents/tdd-cycle.md` (or the host's equivalent). System prompt encodes the red→green→refactor+audit protocol and required output artifacts (diff, test output, branch-coverage audit, findings JSON). The interface itself is the discipline — the agent literally cannot return without producing the artifacts.
- **Parent orchestrator** logic in the existing milestone-implementation skill (`aiwfx-start-milestone` or a successor): identify independent ACs (disjoint filesets), spawn subagents in parallel via `Agent({ subagent_type: "tdd-cycle", isolation: "worktree" })`, merge worktrees serially to milestone branch, run `aiwf check` after each merge.
- **Bounded edit-scope guard.** Subagent declares its allowed paths at spawn; parent audits the worktree diff post-cycle, treats out-of-scope changes as a `scope-leak` finding (and may refuse the merge).
- **Findings-recording flow.** Parent walks each subagent's findings JSON, calls `aiwf add finding --code <code> --linked-acs <ac-ids> --title "..." --body-file <path>` per finding (one commit each per kernel rule).
- **AC closure chokepoint.** `aiwf promote M-NNN/AC-N met` reads the finding tree, refuses with `findings-block-met` when any `open` finding has the AC in its `linked_acs`. Override via `--force` (human-only by existing rule).
- **`aiwf check` integration.** New finding code `ac-has-open-findings` lifts AC findings into the unified report.
- **Initial finding-code set** finalized: `branch-coverage-gap`, `weak-assertion`, `scope-leak`, `audit-skipped`, `convention-violation`, `discovery-gap`, `discovery-decision`, `ac-split-suggested`. Each code emits to a known finding shape (linked AC, severity, recommended remediation).
- **Dogfooding.** The kernel's own ongoing milestones become the canonical fixture; once landed, `M-NNN` milestones with multi-AC structure start using the parallel flow.

## Dependencies

1. **Uniform archive convention** ([ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md)) — accepted, and the convention has shipped. Met.
2. **The F-NNNN storage model** ([ADR-0003](../../../docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md)) — **withdrawn.** aiwf carries six entity kinds and `finding` is not one; see [ADR-0045](../../../docs/adr/ADR-0045-keep-six-entity-kinds-findings-live-in-entity-bodies.md). The two implementation epics this list anticipated — one for the kind, one for findings-gated AC closure — go with it, and neither was filed.
3. **A carrier for the gating property** — open, and the only real blocker this list still holds. The property is worth keeping: a subagent must be refused by the kernel, not by skill text, when it tries to close an AC over an open concern. ADR-0045 says where such a concern lives instead — the body of the entity whose lifetime it shares — but nothing yet gates an AC promote on one.

Optional but compatible:

- **[ADR-0001](../../../docs/adr/ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md)** — proposed inbox/mint id allocation.

## Out of scope

- **Body-section validator generalization for findings** (analogous to M-0066's `entity-body-empty` for milestones). F-NNN bodies will eventually require structured `## Resolution` / `## Waiver` sections on terminal promotion, enforced by `aiwf check`. Wait for M-0066's pattern to settle and generalize first; soft check on missing fix link covers the immediate need.
- **`aiwf reframe F-007 --as-gap` verb.** "This finding really wants to be a gap" — would resolve F-0007 and pre-fill a `G-NNN` with linked context. Nice-to-have. Filed as a follow-up gap if friction shows up; cross-references between F-NNN and `G-NNN` already work via `linked_entities` without a dedicated verb.
- **Multi-host adapter for the `tdd-cycle` agent.** PoC targets Claude Code only. If a non-Claude consumer adopts the framework, the agent definition gets ported then.
- **Heuristic auto-detection of independent ACs.** The parent orchestrator initially relies on the milestone spec declaring AC filesets explicitly (or the conservative default of "serialize unless declared disjoint"). Inferring independence from the AC body prose is a future optimization once usage shows what shapes are common.
- **Subagent observability beyond the JSON return.** When a subagent does the wrong thing, the parent currently sees the result, not the reasoning. Richer introspection (subagent transcripts surfaced into `aiwf history`) is deferred until real friction shows up.
- **Cross-cycle findings.** A finding that pertains to no specific AC but to the milestone or epic as a whole. The data model supports this via empty `linked_acs` + non-empty `linked_entities`, but no current verb produces such findings; dogfooding will surface what's needed.
- **`aiwf record-finding` as a separate verb.** `aiwf add finding` already records the finding atomically with all needed fields; a parallel record-only verb is unnecessary surface.

## Success criteria

- A milestone with two or more independent ACs runs its TDD cycles in parallel via `Agent` subagents, with worktree isolation, and produces a clean fast-forward merge to the milestone branch.
- A subagent that drifts from branch-coverage discipline (e.g., adds an `if` arm without a covering test) returns a `branch-coverage-gap` finding rather than silently shipping the gap.
- A human triaging findings can close all of them via `aiwf promote F-NNN <terminal>` and then `aiwf promote AC met` succeeds. With any open finding, the AC promote refuses cleanly.
- A subagent attempts to `aiwf promote AC met` while findings are open and is refused by the kernel chokepoint, not by skill text.
- The cycle protocol is **AI-discoverable** end-to-end: every verb, finding code, JSON envelope field is reachable via `aiwf <verb> --help`, the agent definition's documented contract, or the design synthesis doc.

## Notes

- This epic deliberately depends on three implementation epics (3, 4, 5 in the dependency list) that need to land in order. Trying to ship the parallel-subagent flow before F-NNN exists or before the AC chokepoint is wired produces a half-finished feature; the kernel principle "no half-finished implementations" requires the full stack.
- Once landed, this epic becomes the **canonical reference for "how AI assistants execute milestone work"** — the upstream of every future milestone-implementation flow on the kernel itself.
