---
id: G-073
title: depends_on is restricted to milestone→milestone edges; cross-kind blocking lives in body prose only; subsumes G-072 in scope
status: open
discovered_in: E-21
---
## What's missing

The `depends_on` field has structural support (universal struct field, validated by `aiwf check`) but is **scoped narrowly via milestone-only schema**. Six call sites today, all milestone-shaped:

| Concern | Location | Scope |
|---|---|---|
| Struct field on Entity | `internal/entity/entity.go:361` (`DependsOn []string`) | universal (any entity could carry it in YAML) |
| Schema declaration | `internal/entity/entity.go:457` (OptionalFields includes `depends_on`) | milestone schema only |
| Schema rule | `internal/entity/entity.go:460` (`{Name: "depends_on", AllowedKinds: []Kind{KindMilestone}}`) | milestone referents only |
| Forward-ref enumeration | `internal/entity/refs.go:38–39` | milestone source only |
| Cycle detection | `internal/check/check.go:487–512` (`no-cycles/depends_on`) | milestone DAG only |
| Render | `cmd/aiwf/render_resolver.go:108` | milestone reads only |

No other kind's schema declares `depends_on` as an optional field; no kind allows non-`KindMilestone` referents. Result: milestone→milestone edges are first-class (writer verb pending per G-072), but **every other blocking relationship lives in body prose**.

Concrete cross-kind cases the kernel can't represent today:

- **Epic depends on ADR ratification.** E-19's *Dependencies* prose lists ADR-0003 and ADR-0004 as required. The kernel doesn't know.
- **ADR depends on ADR.** ADR-0003 inherits ADR-0001's id allocation model (cited in body prose). Prose-only.
- **Contract depends on ADR.** `linked_adr` captures motivation but not blocking — the contract can stay in `proposed` regardless of the ADR's status.
- **Cross-epic dependencies.** "Epic X needs epic Y done first" — purely tribal; the only structural cross-epic data is `parent` for milestone-in-epic ownership.
- **Implementation-epic chains.** Once ADR-0003 is ratified, an implementation epic for `finding` is filed — that epic depends on ADR-0001's implementation epic and ADR-0004's implementation epic. All prose.

## Why it matters

The asymmetry violates two kernel principles:

1. **"Framework correctness must not depend on LLM behaviour."** Prose-only blocking *is* LLM-dependent. An LLM reading *"this epic depends on ADR-0003"* in body prose has to interpret it consistently; `aiwf check` has no way to validate or enforce it. Promoting an entity to `active` despite an unsatisfied prose-mentioned dependency would succeed silently — the gate is in the LLM's head, not the kernel.
2. **"Kernel functionality must be AI-discoverable."** Applies to data shape, not just verb help. An AI assistant trying to understand *"what's blocking what?"* has structured data only for milestone-DAG edges. Every other blocking case requires body-prose interpretation — the heuristic surface that the discoverability principle is designed to push against.

Practical costs that surface today:

- **Synthesis skills (E-21) lose determinism.** Tier classification falls back to grep heuristics for cross-kind blocking. Output is reproducible-ish, not deterministic. The synthesis skill's value to the operator scales with how much of its reasoning is structured-data-grounded versus prose-mention-grounded.
- **`aiwf promote` is silent on cross-kind blockers.** An operator promotes an entity to `active` even when its prose-mentioned dependencies aren't ratified or done. The kernel has no mechanical check; the discipline lives in the operator's head.
- **Render misses sequencing.** Mermaid graphs and roadmap show `parent`/child relationships but not blocking. The actual sequencing the operator cares about is invisible to render.
- **Repeated grep dance.** Multi-epic planning sessions re-derive the same blocking graph from prose every time. The synthesis we just did manually for E-20 is exactly this — the kernel had no help to give.

## Fix shape

**Generalise the existing `depends_on` field rather than introducing a parallel `blocked_by`.** The two would carry identical data — `depends_on` and `blocked_by` are the same edge from opposite ends. Adding `blocked_by` would be data duplication for naming preference.

Five coupled changes:

1. **Schema relaxation.** Add `depends_on` to `OptionalFields` on every kind's schema (epic, ADR, contract, decision, gap, plus future finding). Struct already supports it (`Entity.DependsOn` is universal).
2. **Referent kinds widening.** Per-kind `AllowedKinds` set for each schema's `depends_on` rule. Default: any kind can be referenced, with narrowing where it makes sense (e.g., a contract's `depends_on` probably only points at ADRs and other contracts; that's a per-kind tuning at design time).
3. **Cycle detection generalisation.** Extend `internal/check/check.go:noCycles` from milestone DAG to global DAG over all entities. Algorithm unchanged; node set wider. New finding subcode `no-cycles/depends_on` already exists; coverage broadens.
4. **Per-kind dependency-satisfaction predicate.** New helper in `internal/entity/transition.go`: `SatisfiesDependency(kind, status) bool`. **Distinct** from `IsTerminal` (the helper E-20/M-072 introduces) — rejects negative terminals (`cancelled`, `rejected`, `wontfix`) and accepts only positive ones (`done`, `accepted`, `addressed`, `resolved`). Mapping:

   | Referent kind | Satisfies |
   |---|---|
   | Epic | `done` |
   | Milestone | `done` |
   | ADR | `accepted` |
   | Decision | `accepted` |
   | Contract | `accepted` (or further: `deprecated` / `retired` could also count, depending on whether dependents need the contract still active or merely existed-and-decided) |
   | Gap | `addressed` |
   | Finding (future) | `resolved` |

5. **Status-aware FSM gating.** `aiwf promote` refuses status transitions when any `depends_on` entry isn't in a satisfied state per the predicate above. Override via `--force --reason` for sovereign acts. New finding subcode like `depends-on-unsatisfied` for the unified `aiwf check` report.

Follow-on work that falls out:

- **Reverse query.** `aiwf list --depended-on-by <id>` traverses the same `depends_on` data backwards. UI feature on existing data; ships once `aiwf list` exists (E-20).
- **Render integration.** Mermaid graphs in `aiwf render` and `aiwf status --format=md` gain blocking edges. Falls out of generalised cycle detection's data structures.
- **Writer verb** (G-072's original scope). Folds into this work — a writer that only handles milestone referents would be incomplete once the schema allows cross-kind. The verb shape (`aiwf milestone depends-on M-NNN --on M-MMM` or `--depends-on` flag on `aiwf add milestone`) generalises naturally.

## Relationship to G-072

This gap **supersedes G-072 in scope**. G-072 was the trigger: discovered when planning E-20 produced no clean way to set `depends_on` on M-073/M-074 (the verb that would write it doesn't exist; `aiwf edit-body` refuses frontmatter changes). G-072 remains accurate as the narrow writer-verb observation.

But fixing only G-072 (a writer verb for milestone `depends_on`) ships a **half-feature**: the field would be writable but still scoped milestone-only, leaving cross-kind blocking in prose forever. The synthesis skill (E-21) and the FSM gating both want the same generalisation.

The two should land together as one epic when the friction is paid for. G-072 stays open as the discovery record; this gap is the design lens. The implementation epic, when filed, lists both.

## Surfaced

Discovered during E-20 planning when E-21's design conversation asked *"what blocks what, deterministically?"* The first proposal was a parallel `blocked_by` field; the operator pushed back ("doesn't depends_on imply blocked_by? Do we really need a new field?"), exposing that `depends_on` already carries the semantic — it's just narrowly scoped. Captured here so the future implementation epic has the full design context ready.
