---
id: E-0030
title: 'Branch model chokepoint: --branch flag, sequencing, isolation-escape finding'
status: proposed
---

# E-0030 — Branch model chokepoint: --branch flag, sequencing, isolation-escape finding

## Goal

Make [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s two-tier branch model mechanically enforceable: AI-actor multi-commit work cannot escape a ritual branch context, and the rituals create branches in the right sequence so kernel state-of-the-world stays visible from main throughout the cycle.

## Context

ADR-0010 (just accepted) records the branch-model decision but ships no enforcement. Without a chokepoint, the model lives in prose alone — exactly the *"correctness must not depend on the LLM's behavior"* anti-pattern the kernel has hardened against everywhere else (pre-push `aiwf check`, `internal/policies/` drift tests, the trailer-keys invariant, today's PreToolUse hook for `isolation: "worktree"`).

Concretely, three surfaces today don't yet match ADR-0010's rules:

- **`aiwf authorize`** opens an autonomous scope on an AI agent without ever asking which branch the work runs on. The scope FSM (`active | paused | ended`) is silent on branch context — the rule "AI multi-commit work requires a ritual branch" is unenforced.
- **`aiwfx-start-epic`** cuts its worktree/branch *before* running the sovereign promote + authorize commits (per [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md)), so the kernel state-of-the-world transitions land on the feature branch instead of main. Operators running `aiwf status` from main see the epic still `proposed` while it is in flight under delegation.
- **No kernel finding** catches AI-actor commits that drift outside the convention. [G-0099](../../gaps/G-0099-worktree-isolation-parent-side-precondition.md) is partial-closed by today's session-layer PreToolUse hook (denies `isolation: "worktree"` Agent kwarg), but a subagent that escapes its assigned branch by hand (via `cd`, via `git -C <other-path>`, via an off-tree merge) still goes undetected.

This epic ships the three surfaces in dependency order. [E-0019](../E-0019-parallel-tdd-subagents-with-finding-gated-ac-closure/epic.md) (parallel TDD subagents — currently deferred) builds on top of this epic's deliverables: the parallel cycles need named branches with kernel-enforced isolation, which is exactly what this work provides.

## Scope

### In scope

- **Verb-level `aiwf authorize --branch <name>` flag** with commit-trailer recording of the scope-branch coupling (a new `aiwf-branch:` trailer or equivalent on the authorize commit). Wiring through Cobra completion per CLAUDE.md's auto-completion-friendly rule.
- **AI-side preflight** in `aiwf authorize`: refuse opening a scope on `ai/<agent>` without a ritual branch context (either named via `--branch` or already checked out and recognized as a ritual-shape branch). Refusal produces an actionable error naming the ritual to use.
- **`aiwfx-start-epic` reorder** (closes G-0116): step 5 (worktree) moves after the sovereign promote (step 8) and authorize (step 9), so state-announcement commits land on main *before* the branch is cut. Cross-repo fix via the in-tree fixture pattern (CLAUDE.md § "Cross-repo plugin testing").
- **`aiwfx-start-milestone` alignment**: the symmetric rule for milestones — `aiwf promote M-NNN draft → in_progress` lands on the parent epic branch, *then* the milestone branch is cut. Same cross-repo fixture pattern.
- **Kernel finding `isolation-escape`**: at `aiwf check` (pre-push), detect AI-actor commits whose `aiwf-entity` trailer points at an entity under an active scope whose `aiwf-branch:` doesn't match the commit's actual branch. Fires only on AI-actor commits per the sovereignty principle. Closes G-0099 fully.
- **Test discipline** per CLAUDE.md: every AC under each milestone gets a Go test under `internal/policies/` or an equivalent fixture-validation; branch-coverage audit per milestone; the substring-vs-structural-assertion rule observed for the ritual-fixture milestones.

### Out of scope

- **`aiwf status` enhancement for in-flight ritual branches** (the visibility-mitigation work named in ADR-0010's Validation section). Deferred to a later epic — *"after the model has lived for a few epics"* per the ADR.
- **Substrate-vs-driver split implementation** (ADR-0009 Decisions 1 & 2). Separate concern; tracked separately.
- **Parallel TDD subagent execution** (E-0019). Builds on this epic; its own scope.
- **`CLAUDE.md § "Working in this repo"` rewrite**. Doc-only; ships as a stand-alone `wf-patch` outside this epic, ahead of or alongside the implementation.
- **Retroactive enforcement.** The kernel finding is non-retroactive — it polices commits made under active scopes after the finding lands, not historical commits.

## Constraints

- **Author sovereignty preserved.** The finding fires on AI-actor commits only; human-actor commits are never policed by this surface. `--force --reason` remains the human-only sovereign override path.
- **CLI auto-completion-friendly.** Per CLAUDE.md, the new `--branch` flag (and any new closed-set values it accepts) wires through `RegisterFlagCompletionFunc`; the completion-drift test in `cmd/aiwf/completion_drift_test.go` catches missing wiring.
- **Discoverability.** Every new flag, finding code, and trailer key is reachable via `aiwf <verb> --help`, the embedded skills under `.claude/skills/aiwf-*`, this epic's milestone specs, and CLAUDE.md cross-references.
- **One commit per mutating verb** (kernel invariant). The `aiwf authorize --branch` extension still produces exactly one authorize commit.
- **AC promotion requires mechanical evidence** (CLAUDE.md). Every AC under this epic's milestones has a Go test, finding-rule, or fixture-validation script that fails if the AC's claim breaks.
- **Cross-repo testing follows the fixture pattern.** Rituals-side milestones (M-3, M-4) author the canonical content as `internal/policies/testdata/<skill-name>/SKILL.md` fixtures; rituals-repo commits land at wrap.

## Success criteria

- Every milestone listed in *Milestones* below is `done`.
- An AI-actor `aiwf authorize <id> --to ai/<agent>` invocation that does not pass `--branch` (and is not already on a recognized ritual branch) fails with an actionable error pointing at the ritual surface to use.
- `aiwf check` reports `isolation-escape` (and only `isolation-escape`, with no false positives on author iteration) when an AI-actor's commits violate ADR-0010's branch convention; reports nothing when the convention is followed.
- Running `aiwfx-start-epic` against this repo (or a fixture project) lands the promote-to-active and authorize commits on `main` (or the parent branch) *before* the epic branch is cut.
- The same holds for `aiwfx-start-milestone` against its parent epic branch.
- [G-0099](../../gaps/G-0099-worktree-isolation-parent-side-precondition.md) and [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md) are addressed with `--by-commit` resolvers pointing at the relevant milestones in this epic.
- Author iteration on main remains unencumbered: no new pre-commit or pre-push blocker fires on human-actor commits.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Branch-context detection in preflight: parse `git branch --show-current` and pattern-match (`epic/E-*`, `milestone/M-*`, `fix/*`, etc.), require explicit `--branch`, or both? | No (M-2 design) | Decided in M-2's milestone spec before tests are written |
| Trailer key name: `aiwf-branch:` or `aiwf-scope-branch:` or both (one on `authorize`, one on every subsequent commit under the scope)? | No (M-1 design) | Decided in M-1's milestone spec; documented in `CLAUDE.md § Commit conventions` |
| Should `aiwf authorize --branch` auto-create the branch if absent, or require it to exist already (cut by the ritual)? | No (M-1 design) | Default per ADR-0010's sequencing rule (promote-then-cut) is "require the branch already exists"; auto-create deferred unless friction emerges |
| `isolation-escape` finding scope: per-commit or per-scope-lifetime? Does a single off-convention commit fire, or does the finding fire only when the *scope's* commits collectively violate? | No (M-5 design) | Decided in M-5's milestone spec; default leans per-commit for clear signal |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Rituals plugin updates lag the kernel surface, leaving operators unable to use the new flag through standard ritual flow | Medium | Each ritual-update milestone (M-3, M-4) owns both the kernel-fixture and rituals-repo commits via the cross-repo fixture pattern |
| `isolation-escape` finding produces false positives on legitimate cross-branch work (e.g., the author manually cherry-picking AI-actor commits between branches) | Medium | Author override per ADR-0010 — the finding fires on AI-actor commits *only*; the human author's manipulations are sovereign |
| Landing the chokepoint mid-stream breaks E-0029's in-flight workflow | High | E-0029 is on its own branch; kernel changes land on main and only become visible at next merge. The other session can adapt at its own pace, and `--force --reason` remains the escape hatch if needed |
| The trailer-key extension breaks existing `aiwf history` rendering | Medium | Add the new trailer key alongside existing ones; renderer falls back gracefully when the key is absent (older commits) |

## Milestones

<!-- Sequenced. M-0104 and M-0105 are siblings and can be parallelized after M-0103. M-0106 also depends only on M-0102 + M-0103, so it can run in parallel with the rituals work. -->

- [M-0102](M-0102-aiwf-authorize-branch-flag-scope-branch-trailer-coupling.md) — `aiwf authorize --branch` flag + scope-branch trailer coupling · depends on: —
- [M-0103](M-0103-ai-side-preflight-aiwf-authorize-refuses-without-ritual-branch-context.md) — AI-side preflight: refuse AI-actor scope opening without ritual branch context · depends on: M-0102
- [M-0104](M-0104-aiwfx-start-epic-sequencing-fix-closes-g-0116.md) — `aiwfx-start-epic` sequencing fix (closes G-0116) · depends on: M-0102, M-0103
- [M-0105](M-0105-aiwfx-start-milestone-sequencing-alignment.md) — `aiwfx-start-milestone` sequencing alignment · depends on: M-0102, M-0103
- [M-0106](M-0106-kernel-finding-isolation-escape-closes-g-0099.md) — Kernel finding `isolation-escape` (closes G-0099 fully) · depends on: M-0102, M-0103

## ADRs produced

This epic produces no new ADRs — it implements [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md). Any decision that surfaces during implementation (e.g., the trailer-key name, the branch-context detection heuristic) is recorded inline in the relevant milestone spec, or as a D-NNN if the decision warrants its own entity.

## References

- [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md) — Branch model: ritualized work on branches, author iteration on main (source of truth)
- [ADR-0009](../../../docs/adr/ADR-0009-orchestration-substrate-substrate-vs-driver-split-trailer-only-cycle-events-isolation-as-parent-side-precondition.md) — Orchestration substrate (Decision 3: isolation as parent-side precondition — adjacent kernel-shape decision)
- [G-0059](../../gaps/G-0059-branch-model-no-canonical-hierarchy-mapping.md) — recorded the question; now `addressed` via ADR-0010
- [G-0099](../../gaps/G-0099-worktree-isolation-parent-side-precondition.md) — worktree isolation; tier-3 partial-closed by `.claude/hooks/validate-agent-isolation.sh`; full closure depends on M-5 of this epic
- [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md) — sequencing fix; addressed by M-3 of this epic
- [E-0019](../E-0019-parallel-tdd-subagents-with-finding-gated-ac-closure/epic.md) — Parallel TDD subagents; builds on this epic's deliverables
- `docs/pocv3/design/provenance-model.md` — principal × agent × scope; sovereign override
- `CLAUDE.md § "Subagent worktree isolation"` — the precondition pattern this epic codifies in kernel form
