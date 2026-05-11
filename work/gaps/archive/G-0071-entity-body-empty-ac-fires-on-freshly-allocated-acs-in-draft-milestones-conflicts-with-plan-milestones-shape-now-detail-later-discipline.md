---
id: G-0071
title: entity-body-empty/ac fires on freshly-allocated ACs in draft milestones; conflicts with plan-milestones 'shape now, detail later' discipline
status: addressed
discovered_in: E-0020
addressed_by:
    - M-0075
---
## What's missing

`internal/check/entity_body.go`'s `entity-body-empty` rule is **lifecycle-blind**: it fires on every entity whose load-bearing body section is empty, regardless of where the entity sits in its lifecycle. Two distinct cases hit:

### Case 1 — pre-implementation (draft milestones with freshly-allocated ACs)

The `entity-body-empty/ac` subcode fires on every AC whose body under `### AC-N` is whitespace-only, regardless of the parent milestone's status or the AC's TDD phase. The check is shape-aware (heading must be present, in canonical form) but lifecycle-blind: an AC allocated 30 seconds ago via `aiwf add ac` produces a warning the moment the next `aiwf check` runs.

Concretely, allocating M-0072..M-0074 for E-0020 via `aiwfx-plan-milestones` produced 24 `entity-body-empty/ac` warnings on a freshly-rebuilt binary — one per allocated AC. The plan-milestones skill explicitly mandates this state: *"Does not draft individual milestone specs in deep detail — that happens just-in-time when each milestone is started."* Detail under each `### AC-N` is supposed to fill at `aiwfx-start-milestone`, not at allocation. The skill's discipline and the kernel's check rule are at odds.

### Case 2 — post-implementation (terminal-status entities with empty load-bearing sections)

The same shape recurs for terminal-status entities. The kernel rule is "removal means flipping status to a terminal value, not deleting the file" (CLAUDE.md kernel commitment #2) — but the body-empty rule still fires on those preserved files. Concrete example: `ADR-0002` (`docs/adr/ADR-0002-test-dry-run-delete-me.md`) is terminal-status `rejected`, was rejected on 2026-05-07, and still surfaces three warnings — `entity-body-empty/adr` on `## Context`, `## Decision`, `## Consequences` — because the file has no real prose under any of them. There is no path to silence those warnings without either:

1. Filling the body with fabricated prose (lying about the file's nature; it was test debris from a dry-run validation of `aiwf cancel`), or
2. Deleting the file (against kernel discipline #2), or
3. Fixing the rule.

The same trap applies to any future `addressed`/`wontfix` gap, `superseded`/`rejected` ADR or decision, `cancelled` epic or milestone, `retired` contract — i.e., every kind has terminal states that produce preserved-but-stale entity files. A long-running repo would accumulate persistent warnings against terminal artifacts, none of which can be silenced without violating one of the rules above.

ADR-0004 (proposed) addresses the *storage* side (terminal-status entities move to `work/<kind>/archive/`), but doesn't address the *validation* side — the body-empty rule still fires on archived files. The two layers need to land together for the noise to actually go away.

## Why it matters

The rule was scoped at M-0066/AC-1 to catch *shipped* empty bodies — entities promoted to terminal status without prose. The current implementation generalizes that intent to every lifecycle state, which means it fires both *before* the body is supposed to be filled (Case 1) and *after* the file's purpose has terminated (Case 2). The intended target — *active* entities with intentionally-empty load-bearing sections — is sandwiched between two false-positive bands.

Three downstream costs across both cases:

1. **Plan-milestones output is dirty by design.** Every epic broken into N milestones with M ACs each surfaces N×M warnings on the next `aiwf check`. For E-0020 that's 24; a 5-milestone epic with 5 ACs each yields 25. The right baseline after planning is "tree clean except provenance"; the rule makes that unattainable without filling stub prose. Compounded by Case 2: every rejected/superseded/addressed entity adds permanent warnings.
2. **The "fix" defeats planning hygiene.** Stubbing one-line prose under each AC heading at allocation time produces exactly the rotting AC bodies the plan-milestones anti-pattern warns against (*"AC definitions written 6 weeks before the work starts are usually wrong"*). Operators are nudged toward writing throwaway placeholder prose to silence warnings — including, for Case 2, prose that misrepresents test debris as architectural decisions.
3. **Strict mode amplifies the cost.** `tdd.strict` escalates `entity-body-empty` to error via `ApplyTDDStrict` in `entity_body.go`. On a strict-mode repo, planning a multi-milestone epic would block `aiwf check` until every AC body is filled — at exactly the moment the operator should be deferring detail. Same blocking behavior for any preserved test artifact or terminated entity.

## Fix shapes

The unifying lens: the rule should fire only on entities in **active** states — neither pre-implementation drafts nor post-implementation terminal artifacts. Two complementary mechanisms cover both cases:

- **Status gating (covers both cases via one predicate).** Skip the rule when the entity is in a non-active lifecycle state. Concretely: skip when entity status is `draft` (pre-impl) OR when `entity.IsTerminal(kind, status) == true` (post-impl, helper landing in E-0020/M-0072 per ADR-0004). For ACs, gate on the parent milestone's status. The result: `aiwf check` warnings surface only on active entities with empty load-bearing sections — exactly the population the rule was designed for. Single predicate, kind-uniform.
- **Phase gating (refines Case 1 only).** When the parent milestone is `tdd: required` and an AC has `tdd_phase: red` with `status: open`, treat the empty AC body as expected. Fire once the AC is `tdd_phase: green` (implementation has begun) or the milestone is `in_progress`. Phase data is already in the AC's frontmatter; the rule just needs to consult it. Doesn't help Case 2.

Status gating is the simpler primitive and addresses both cases. Phase gating is more precise for Case 1 specifically. Both could coexist (status-gated by default, phase-aware refinement when `tdd: required`).

This fix lives in `internal/check/entity_body.go`. The `requiredSectionsByKind` map and the AC sub-element walk both need to consult entity (or parent-milestone) status before emitting findings. ADR-0004's `entity.IsTerminal(kind, status)` helper (introduced by E-0020/M-0072) is the right primitive; phase gating uses the existing AC `tdd_phase` field already on the struct.

Surfaced during E-0020 planning when `aiwfx-plan-milestones` produced 24 warnings against the just-allocated M-0072/M-0073/M-0074 (Case 1), and during the post-push `aiwf check` when ADR-0002 surfaced 3 persistent warnings (Case 2). Documented as a follow-up rather than scoped into E-0020 because the fix is a kernel-discipline concern in `internal/check/entity_body.go`, not in the list verb being added — but it would cleanly close the warning baseline E-0020 currently leaves behind, plus the standing ADR-0002 noise.
