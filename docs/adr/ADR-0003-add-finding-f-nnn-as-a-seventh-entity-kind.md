---
id: ADR-0003
title: Add finding (F-NNN) as a seventh entity kind
status: accepted
---
## Context

Kernel principle #1 enumerates **six entity kinds** — epic, milestone, ADR, gap, decision, contract — closed-set and hardcoded. The framework_entity_vocabulary memory and CLAUDE.md both call out the deliberate omission of `story` and `task` (execution units belong in plain GH issues, not in the framework's vocabulary).

Two emerging needs don't fit cleanly into the existing six:

- **Cycle-time findings.** A planned parallel-TDD-subagent feature returns concerns from each cycle (branch-coverage gap, weak assertion, scope leak, audit skipped, scope creep). These need a durable, audited, AC-linked representation that **blocks AC closure until a human triages**. The proximate trigger is the M-0066/AC-1 branch-coverage drift, where a long session lost track of TDD discipline; subagent isolation provides bounded context but only pays off if the finding surface is mechanical.
- **Check-time findings that need triage.** `aiwf check` already produces transient findings on every run. Some of those — a contract drift that's been in the tree for five commits, a recurring shape violation — deserve to be **escalated** into something durable, with a stable id and lifecycle, rather than re-reported every check run. Today there is no such surface.

Frontmatter arrays on the AC scale for the simple case but fail at the moments that matter: cross-AC findings (one finding affects multiple ACs), long-form repro/triage prose, stable references from gaps and decisions (`Resolves: F-007`), escalation from `aiwf check`. Each of those wants the standard kernel treatment — id, FSM, body, history, trailers.

Sibling files per AC (`AC-1.findings.yaml`) provide file-level isolation for parallel-cycle merges but introduce a new file pattern that requires `aiwf check` shape-rule extensions and still doesn't solve cross-AC scope.

The PoC is likely to graduate. Picking the storage model that future-proofs the cross-AC, escalation-from-check, and stable-reference cases now — rather than retrofitting later — is the right call. The branch's existing 66 gaps (52 addressed) demonstrate that high-volume governance kinds are realistic; findings will be at least as high-volume as gaps once cycle-time emission turns on.

## Decision

Add **`finding`** as a seventh entity kind.

### Id and storage

- Id pattern: **F-NNN** (zero-padded; same family as G-NNN, D-NNN).
- Allocated via the kernel's standard allocator. If [ADR-0001](ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md) is accepted, F-NNN inherits the inbox/mint model uniformly with the other monotonic-id kinds — no special case.
- Stored at `work/findings/F-NNN-<slug>.md`. Terminal-status entries move to `work/findings/archive/` per the companion archive ADR.

### Status FSM

`open → resolved | waived | invalid`. All three terminal. One Go function for legal transitions, hardcoded per kernel principle.

- `resolved` — the underlying issue was fixed. The resolving commit references the F-NNN via the standard `aiwf-entity:` trailer. A soft check warns when a `resolved` transition has no associated fix commit nearby.
- `waived` — sovereign accept. Requires `--force` and `--reason` (kernel's existing pattern from M-0017). The existing `--force` rule means human-actor only; subagents structurally cannot waive their own findings.
- `invalid` — false positive (subagent was wrong, rule fired incorrectly). Requires `--reason`. Human-actor only by the same convention applied to other consequential transitions.

### Frontmatter

Minimal load-bearing fields:

```yaml
---
id: F-007
title: Branch coverage gap on empty-string input
status: open
code: branch-coverage-gap
linked_acs: [M-066/AC-1]
linked_entities: []
recorded_by: ai/claude
---
```

- `code` — stable finding code from a kernel-pinned set. Initial set: `branch-coverage-gap`, `weak-assertion`, `scope-leak`, `audit-skipped`, `convention-violation`, `discovery-gap`, `discovery-decision`, `ac-split-suggested`. Extensible per-rule when escalated from `aiwf check` (each elevated check finding has a corresponding F-NNN code). The full enumeration is finalized in the implementation epic.
- `linked_acs` — composite AC ids that this finding blocks. Multi-AC findings list multiple. Empty for findings not tied to an AC (e.g., escalated check-time findings on a milestone or gap).
- `linked_entities` — any other entity ids the finding pertains to (gaps, ADRs, decisions, contracts, milestones).
- `recorded_by` — provenance string (`ai/claude`, `framework/aiwf-check`, `human/<email>`). Aligns with the existing principal × agent × scope model in CLAUDE.md.

### Body

Free-form prose. What was found, why it matters, repro context. Optional `## Resolution` and `## Waiver` sections written at terminal promotion.

Body-section validation (analogous to M-0066's `entity-body-empty` for milestones) is **deferred** until that pattern generalizes. The initial soft check on `resolved` transitions covers the most common discipline gap.

### Verb surface

**No new verb family.** Reuse the kernel's universal verbs:

- `aiwf add finding --code <code> --linked-acs <ac-ids> --title "..." --body-file <path>`
- `aiwf promote F-007 resolved [--reason "..."]`
- `aiwf promote F-007 waived --force --reason "..."`
- `aiwf promote F-007 invalid --reason "..."`
- `aiwf show F-007`, `aiwf history F-007` — work via the kernel's generic dispatch.

Per Fork 3 of the design conversation, dedicated `aiwf finding {resolve,waive,invalidate}` verbs were rejected: they duplicate `aiwf promote` and set a precedent for verb-family-per-kind that violates the kernel's minimum-verb-surface posture. Resolution metadata (links to fix commits, follow-up gaps) flows through trailers and body sections, not through verb arguments.

### AC closure chokepoint

`aiwf promote M-NNN/AC-N met` reads the finding tree, refuses with `findings-block-met` if any `open` finding has the AC in its `linked_acs`. Override is `--force` (human-only by existing rule). A new `aiwf check` finding code `ac-has-open-findings` lifts this into the unified report so the block surfaces during routine validation, not only at promotion-time.

### Principle #1 amendment

> **Six entity kinds** → **Seven entity kinds** — epic, milestone, ADR, gap, decision, contract, **finding** — each with a closed status set and one Go function for legal transitions. Hardcoded; not driven by external YAML.

The closure-by-vocabulary remains intact; the set grows by one with explicit ADR-level rationale, setting the precedent for how future kinds get added. The `story`/`task` exclusion documented in CLAUDE.md and the framework_entity_vocabulary memory continues to hold — finding is a *governance* artifact, not an execution unit.

## Consequences

**Positive:**

- Single discipline for "things that need human attention" — TDD-cycle findings and `aiwf check` escalations live in the same surface, queried the same way (`aiwf show F-NNN`, `aiwf history F-NNN`, `aiwf status` aggregation).
- Cross-AC and cross-milestone findings become first-class via `linked_acs` / `linked_entities`. The reframe path (finding triages into a follow-up gap, decision, or ADR) works through standard cross-references, no new mechanism.
- Generic `aiwf promote` handles all status transitions; `aiwf history F-NNN` works for free; `aiwf check` ID-uniqueness, slug-rules, and shape rules apply to findings without per-kind special-casing.
- Sovereignty is preserved by reusing the existing `--force` rule. Subagents structurally cannot waive their own findings.
- Compatible with [ADR-0001](ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md): F-NNN inherits whatever id-allocation model the framework adopts. Parallel subagent worktrees filing findings under the inbox model become structurally collision-free.

**Negative:**

- Every code path that switches on kind (rendering, status aggregation, `aiwf check` shape rules, schema validation, FSM dispatch, completion enumeration) gains a `case finding:` branch. The audit cost is real but bounded — each branch is straightforward by the kernel's existing patterns.
- The body-section validator (M-0066's `entity-body-empty` rule) needs `finding` added to its kind list when the validator generalizes; deferred for now but not free.
- New finding-code surface (`branch-coverage-gap`, `weak-assertion`, etc.) is a vocabulary that needs review and stability discipline. Codes are added as their underlying rules ship; not all need to exist on day one. Stability becomes a kernel concern once the surface stabilizes.
- Every `aiwf check` invocation now scans the findings tree for `ac-has-open-findings`. Cost is small (the tree is small); not a measurement-driven decision yet.
- Operationally, the discipline of "subagent emits finding → parent records F-NNN → human triages" requires the parent orchestrator (epic-level work) to be implemented before findings have a routine producer. F-NNN entities are useful even without the parallel-subagent feature (escalation from `aiwf check` works standalone).

## Alternatives considered

- **Frontmatter array on the AC.** Simple, no new file machinery, matches "smaller PoC" instincts. Fails the cross-AC, escalated-from-check, and stable-reference requirements. Right answer for a smaller PoC; wrong answer once graduation is in view.
- **Sibling file per AC (`AC-1.findings.yaml`).** File-level isolation for parallel-cycle merges. Requires `aiwf check` shape-rule extensions; doesn't solve cross-AC scope; inconsistent with how the kernel treats other auxiliary state.
- **Per-kind verb family (`aiwf finding resolve|waive|invalidate`).** Guided UX. Duplicates the kernel's universal `aiwf promote`. Sets a precedent for verb-family-per-kind that violates the kernel's minimum-verb-surface posture.
- **Reservation-range id allocation for parallel subagents.** Pre-allocate F-NNN ranges per subagent to avoid collisions inside parallel worktrees. Sparse ids hurt readability; the inbox/mint model from ADR-0001 already solves this more elegantly. Rejected.

## References

- Companion ADR: uniform archive convention for terminal-status entities (filed alongside this one).
- [ADR-0001](ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md) — proposed inbox/mint model for id allocation; F-NNN inherits.
- Design synthesis: `docs/pocv3/design/parallel-tdd-subagents.md` (companion design doc; full four-fork resolution and end-to-end flow).
- CLAUDE.md "What the PoC commits to" §1 (six entity kinds — amended by this ADR).
- Framework_entity_vocabulary memory: deliberate omission of `story`/`task`; finding is governance, not execution.
- M-0066/AC-1 wrap context — the proximate trigger for the cycle-time-findings need.
