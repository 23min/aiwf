---
id: E-23
title: Uniform 4-digit kernel ID width (closes G-093)
status: proposed
---
# E-NEW — Uniform 4-digit kernel ID width (closes G-093)

## Goal

Land ADR-0008's policy in code and on disk. The kernel canonicalizes every id kind to 4 digits, parsers tolerate narrower legacy widths on input, the new `aiwf rewidth` verb migrates a consumer's active tree on demand, and this repo runs the verb as one of N consumers. After this epic, §07's Slice 2 ships F at canonical F-NNNN with no separate decision; downstream consumers run `aiwf rewidth` when they're ready; new consumers post-graduation are born canonical.

## Context

Three pressures converge: epic exhaustion is structural (`E-22` used; `E-NN` maxes at 99); the §07 TDD proposal already shows the policy gap (silent F-NNN → F-NNNN drift); multiple downstream consumers have already adopted aiwf with narrow-width trees and need a tested, distributed migration path.

G-093 is the discovery framing; ADR-0008 is the policy decision and `aiwf rewidth` verb shape. This epic is the implementation.

## Scope

### In scope

- **Kernel parser tolerance** — every id-parsing call site accepts both narrow and canonical widths.
- **Kernel renderer canonicalization** — every display surface emits 4-digit form.
- **Allocator default** — `canonicalPadFor(kind)` returns 4 for every kind.
- **Test-fixture sweep** — every hardcoded narrow-width id literal in `internal/**/*_test.go` and `internal/**/testdata/` updated to canonical form.
- **`aiwf rewidth` verb** per ADR-0008. `--apply` commits; default dry-run. Active-tree only; archives untouched. Idempotent. Single commit per `--apply` invocation, trailer `aiwf-verb: rewidth`. Cobra completion wiring per E-14's drift test.
- Apply `aiwf rewidth --apply` to this repo's tree as M-B's wrap deliverable.
- **Drift-check rule** `entity-id-narrow-width` — tree-state-based: silent on uniform-narrow active tree; silent on uniform-canonical; warns on narrow files in mixed-state active tree. Archive entries excluded from mixed-state computation.
- **ADR-0003 amendment** — id pattern paragraph F-NNN → F-NNNN with cross-reference to ADR-0008.
- **CLAUDE.md commitment #2 update** — collapses to a single uniform rule; mentions `aiwf rewidth` for legacy migration.
- **Embedded kernel skill content refresh** — narrow-width examples in `internal/policies/testdata/<skill>/SKILL.md` updated.
- **Rituals-plugin coordination** — the 5 enumerated files refresh; cross-repo SHA recorded in M-C's wrap.

### Out of scope

- **Renaming files inside `<kind>/archive/`** — grandfathered per ADR-0004.
- **Rewriting commit history** — old trailers keep matching via parser tolerance.
- **Width 5 or 6** — YAGNI.
- **Per-kind width tuning** — rejected in ADR-0008.
- **Folding rewidth into `aiwf update`** — rejected in ADR-0008.
- **Marker-based drift detection** — rejected in ADR-0008.
- **G-091's preventive check rule** for path-form refs — related but separate.
- **§07 TDD proposal's other findings** — tracked separately when §07 advances.

## Constraints

- **TDD: required for all three milestones.** Net-new logic in M-A (parser tolerance, allocator), M-B (verb), M-C (check rule).
- **Pure-additive parser change in M-A.** Acceptance widens; no existing valid input becomes invalid. Old trees, branches, skills must keep validating.
- **"What verb undoes this?" gate honored.** `aiwf rewidth` is one-shot per consumer; reversal is `git revert` on the migration commit, same answer as `aiwf init`. Allocator change is forward-only.
- **Verb is idempotent.** Running `aiwf rewidth --apply` on an already-canonical tree is a no-op (no commit, "no changes needed" message).
- **Verb's skill coverage decision** per ADR-0006: no per-verb skill; verb is one-shot, self-documenting via `--help`. Allowlisted in `skill_coverage.go` with rationale "one-shot migration ritual; --help is sufficient discovery surface."
- **Kernel correctness independent of LLM behavior.** The drift check rule is the chokepoint, not human review.
- **Closed-set completion wiring intact.** New verb threads through `cmd/aiwf/completion_drift_test.go`.
- **Cross-repo coordination point named.** M-C's wrap commit records the rituals-plugin commit SHA refreshing skill examples (per CLAUDE.md "Cross-repo plugin testing").

## Success criteria

- [ ] `aiwf check` green on this repo's tree post-M-B's `aiwf rewidth --apply`. Every active entity in `work/` and `docs/adr/` uses 4-digit form.
- [ ] No `entity-id-narrow-width` warnings on this repo's tree post-M-C ship (uniform-canonical active tree).
- [ ] `aiwf-entity: E-22` in pre-migration commit trailers continues to match `aiwf history E-22` *and* `aiwf history E-0022`.
- [ ] `aiwf add epic --title "..."` allocates the next id at canonical 4-digit form (e.g., E-0023 if E-22 was previous).
- [ ] All display surfaces (`aiwf list`, `status`, `show`, `history`, `render`, JSON envelopes) emit canonical 4-digit ids regardless of on-disk filename.
- [ ] `aiwf rewidth` (no flag) runs dry-run and prints the planned moves; `--apply` performs them in one commit; second invocation on canonical tree is a no-op.
- [ ] Drift check rule fires on synthetic mixed-state fixture tree; silent on uniform-narrow and uniform-canonical fixture trees.
- [ ] CLAUDE.md commitment #2 reads as a single uniform rule.
- [ ] ADR-0003 §"Id and storage" reads `F-NNNN` with cross-reference to ADR-0008.
- [ ] All 5 rituals-plugin files refresh; SHA recorded in M-C's wrap.
- [ ] Three milestones promoted to terminal status; closing commits cite this epic.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Parser tolerance change misses an id-parsing site (hand-rolled regex in a check rule, etc.). | Medium | M-A's TDD pass starts with a `grep` audit of every id-parsing call site; table-driven test exercises both widths through each entry point. |
| `aiwf rewidth` reference-rewrite logic clobbers content inside code fences or other unintended contexts. | Medium | Verb's reference-rewrite is scoped to markdown-AST-aware patterns (id-form mentions, markdown links to active-tree paths, composite ids); code fences excluded. Synthetic fixture tests cover the edge cases. Manual diff review of M-B's apply step is the final layer. |
| Verb's idempotence breaks if rewrite logic is non-deterministic (e.g., visits files in non-stable order). | Low | Deterministic walk order pinned by test; second-run-is-no-op asserted in test suite. |
| Rituals plugin updates lag the kernel release; skill examples show narrow widths post-canonical. | Low | M-C wrap doesn't promote epic to `done` until rituals SHA is reachable from the marketplace. |
| Future kind allocates at non-canonical width because someone forgets `canonicalPadFor`. | Medium | M-C's drift check is the chokepoint; defense in depth even if the allocator regresses. |
| §07 TDD Slice 2 advances before this epic lands and CLAUDE.md commitment #2 conflicts on merge. | Low | This epic is named as Slice 2's prerequisite; conflict surface is one paragraph in CLAUDE.md, mechanically reconcilable. |

## Milestones

<!-- Bulleted list, ordered by execution sequence. Status lives in each milestone's frontmatter. Milestone ids are global (M-NNN), not epic-scoped; allocated by aiwfx-plan-milestones. -->

- M-A — Parser tolerance + renderer canonicalization + allocator default + fixture sweep · `tdd: required` · depends on: —
- M-B — Implement `aiwf rewidth` verb + apply to this repo's tree (verb's wrap PR contains both the verb code and the resulting rename diff) · `tdd: required` · depends on: M-A
- M-C — Drift-check rule (tree-state-based) + ADR-0003 amendment + CLAUDE.md commitment update + embedded skill refresh + rituals-plugin coordination · `tdd: required` · depends on: M-A, M-B

(M-A is the load-bearing kernel change. M-B implements and exercises the verb against this repo's tree. M-C locks policy in docs and prevents future drift, sequenced last so the drift check fires against an already-canonical tree.)

## ADRs produced

(None expected. ADR-0008 was filed alongside this epic and is the policy precedent. Verb shape, drift-check semantics, and migration approach are all locked in ADR-0008.)

## Dependencies

- **No upstream blockers.** ADR-0008 filed.
- **§07 TDD architecture proposal's Slice 2 (F as 7th kind) consumes this epic's outputs.** Sequencing: this epic ships before §07's Slice 2 implementation, so F is born canonical. ADR-0003 is amended by M-C before any F-related milestone allocates.
- **Compatible with ADR-0004** — archives keep their birth-width per forget-by-default; drift check excludes archives from mixed-state computation.

## References

- G-093 — surfacing gap.
- ADR-0008 — policy this epic implements.
- ADR-0003 — F kind; amended by M-C.
- ADR-0004 — archive convention; preserved in scope.
- **CLAUDE.md** "What aiwf commits to" §2 — updated by M-C.
- `internal/verb/import.go::canonicalPadFor` — current pad-policy site; relocated and broadened by M-A.
- `docs/explorations/07-tdd-architecture-proposal.md` — exploratory doc; Slice 2 consumes M-A/M-C.
- **Rituals-plugin files needing refresh in M-C** (27 narrow-width refs total):
  - `plugins/aiwf-extensions/templates/epic-spec.md` (1)
  - `plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md` (10)
  - `plugins/aiwf-extensions/skills/aiwfx-whiteboard/SKILL.md` (14)
  - `plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md` (1)
  - `plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md` (1)
