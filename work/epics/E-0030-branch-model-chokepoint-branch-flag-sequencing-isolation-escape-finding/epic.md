---
id: E-0030
title: 'Branch model chokepoint: --branch flag, sequencing, isolation-escape finding'
status: proposed
---
# E-0030 — Branch model chokepoint: --branch flag, sequencing, isolation-escape finding

## Goal

Make [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s two-tier branch model mechanically enforceable end-to-end: AI-actor multi-commit work cannot escape a ritual branch context, the rituals create branches in the right sequence so kernel state-of-the-world stays visible from main throughout the cycle, the finding rule that catches drift is itself a cell in the layer-4 branch-choreography spec ([ADR-0011](../../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md) §"Scope"), and the human override path is gated by a typed sovereign-trailer signature.

## Context

ADR-0010 (accepted 2026-05-12) records the branch-model decision but ships no enforcement. ADR-0011 (accepted 2026-05-18) ratified the spec-cell methodology for layers 1–3 (FSM, per-verb pre/post, cross-verb sequence) and **explicitly carved out layer 4 — branch choreography — as this epic's scope**. Without the chokepoint, the model lives in prose alone — exactly the *"correctness must not depend on the LLM's behavior"* anti-pattern the kernel has hardened against everywhere else (pre-push `aiwf check`, `internal/policies/` drift tests, the trailer-keys invariant, today's PreToolUse hook for `isolation: "worktree"`).

Concretely, three surfaces today don't yet match ADR-0010's rules:

- **`aiwf authorize`** opens an autonomous scope on an AI agent without ever asking which branch the work runs on. The scope FSM (`active | paused | ended`) is silent on branch context — the rule "AI multi-commit work requires a ritual branch" is unenforced.
- **`aiwfx-start-epic`** cuts its worktree/branch *before* running the sovereign promote + authorize commits (per [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md)), so the kernel state-of-the-world transitions land on the feature branch instead of main. Operators running `aiwf status` from main see the epic still `proposed` while it is in flight under delegation.
- **No kernel finding** catches AI-actor commits that drift outside the convention. [G-0099](../../gaps/G-0099-worktree-isolation-parent-side-precondition.md) is partial-closed by today's session-layer PreToolUse hook (denies `isolation: "worktree"` Agent kwarg), but a subagent that escapes its assigned branch by hand (via `cd`, via `git -C <other-path>`, via `git checkout main` from inside the worktree, or via an off-tree merge) still goes undetected.

This epic ships the six surfaces in dependency order. Each milestone's tests land as **paired positive/negative cells** in the layer-4 extension of `internal/workflows/spec/`, with the existing drift policy ([ADR-0011](../../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md) §"Drift policy") guaranteeing the spec stays closed against the impl as future PRs touch the surface.

[E-0019](../E-0019-parallel-tdd-subagents-with-finding-gated-ac-closure/epic.md) (parallel TDD subagents — currently deferred) builds on top of this epic's deliverables: its cycle-level `aiwf-cycle-worktree-branch` per-commit redundancy ([ADR-0009](../../../docs/adr/ADR-0009-orchestration-substrate-vs-driver-split.md) Decision 3) is a sharpening of M-0106's scope-level rule, not a replacement (see §"Sequencing relative to ADR-0009 / E-0019" below).

## Scope

### In scope

- **Verb-level `aiwf authorize --branch <name>` flag** with commit-trailer recording (`aiwf-branch:` per §"Design decisions"). Wiring through Cobra completion per CLAUDE.md's auto-completion-friendly rule.
- **`internal/branchparse/` package extraction** (folded into M-0102): lift `parseEntityFromBranch` and the ritual-shape regexes from `internal/cli/status/worktrees.go:485` into a shared package consumed by both the existing `aiwf status --worktrees` correlation and M-0103's preflight detection. One regex set, two consumers — prevents drift between the two pattern lists.
- **AI-side preflight** in `aiwf authorize`: refuse opening a scope on `ai/<agent>` without a ritual branch context — accepted either via `--branch <name>` naming an existing branch *or* via current checkout matching a recognized ritual shape from `internal/branchparse/`. Refusal produces an actionable error naming the ritual to use. Sovereign override: `--force --reason "..."` per the existing kernel pattern.
- **`aiwfx-start-epic` reorder** (closes G-0116): the sovereign promote and the (optional) authorize land on the parent branch *before* the worktree/branch is cut. Edits land at `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md` (the canonical authoring location per [ADR-0014](../../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) + [ADR-0016](../../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md)). The stale "G-0059 frames the open question of which branch-model convention aiwf should bless" paragraph in step 6 retires — ADR-0010 is the answer.
- **`aiwfx-start-milestone` alignment**: the symmetric rule for milestones — `aiwf promote M-NNNN draft → in_progress` lands on the parent epic branch, *then* the milestone branch is cut via `aiwf authorize --branch milestone/M-NNNN-<slug>` (when the work is delegated). The "must be on epic branch already" precondition is tightened in step 3 — silent fallthrough to `git checkout -b epic/E-NNNN-<slug> if missing` is removed. Edits land at the embedded snapshot.
- **Kernel finding `isolation-escape`**: at `aiwf check` (pre-push), detect AI-actor commits whose `aiwf-entity` trailer points at an entity under an active scope whose `aiwf-branch:` doesn't match the commit's branch reachability. Fires only on AI-actor commits per the sovereignty principle. Cherry-pick handling — committer-vs-actor mismatch plus a cherry-pick marker in the commit body is treated as sovereign re-author and suppressed. Closes G-0099 fully.
- **Layer-4 branch-choreography spec cells**: each milestone's positive and negative cell tests register entries in `internal/workflows/spec/branch/` (exact package layout settled in the final M-0158); the drift policy under `internal/policies/` extends to cover branch-layer coverage closure. The meta-test ("every cell has at least one matching test") is the same shape M-0124 established for layers 1–3.
- **Test discipline** per CLAUDE.md: every AC under each milestone gets a Go test under `internal/policies/` or an equivalent fixture-validation; branch-coverage audit per milestone; the substring-vs-structural-assertion rule observed for the ritual-skill milestones.

### Out of scope

- **`aiwf status` enhancement for in-flight ritual branches** (the visibility-mitigation work named in ADR-0010's Validation section). Deferred to a later epic — *"after the model has lived for a few epics"* per the ADR. (Note: `aiwf status --worktrees` already correlates worktrees to entities via the same branch parser; the enhancement is the *summary view*, not the correlation logic.)
- **Cycle-level per-commit redundancy** (`aiwf-cycle-worktree-branch`) and `cycle-trailer-incomplete` checks — ADR-0009 Decision 2/3 substrate work, lands under E-0019.
- **Substrate-vs-driver split implementation** (ADR-0009 Decisions 1 & 2). Separate concern; tracked separately.
- **Parallel TDD subagent execution** (E-0019). Builds on this epic; its own scope.
- **`CLAUDE.md § "Working in this repo"` rewrite**. Doc-only; ships as a stand-alone `wf-patch` outside this epic, ahead of or alongside the implementation.
- **Retroactive enforcement.** The kernel finding is non-retroactive — it polices commits made under active scopes after the finding lands, not historical commits.

## Constraints

- **Author sovereignty preserved.** The finding fires on AI-actor commits only; human-actor commits are never policed by this surface. `--force --reason` remains the human-only sovereign override path; an `ai/...` invocation of `--force` is refused by the existing actor-shape rule.
- **Sovereign override is gated, audited, and rare by design.** Each chokepoint exposes an override; each override requires a typed signature (`--force --reason "..."` for the preflight; `aiwf-on-behalf-of: human/...` plus cherry-pick marker, or `aiwf-force: "..."` on the violating commit, for the finding). The override path produces a kernel-readable paper trail per [`docs/pocv3/design/provenance-model.md`](../../../docs/pocv3/design/provenance-model.md). It is **last resort, not routine** — its appearance in `aiwf history` is a flag, not a normalcy.
- **CLI auto-completion-friendly.** Per CLAUDE.md, the new `--branch` flag (and any new closed-set values it accepts) wires through `RegisterFlagCompletionFunc`; the completion-drift test in `cmd/aiwf/completion_drift_test.go` catches missing wiring.
- **Discoverability.** Every new flag, finding code, and trailer key is reachable via `aiwf <verb> --help`, the embedded skills under `.claude/skills/aiwf-*`, this epic's milestone specs, and CLAUDE.md cross-references.
- **One commit per mutating verb** (kernel invariant). The `aiwf authorize --branch` extension still produces exactly one authorize commit.
- **AC promotion requires mechanical evidence** (CLAUDE.md). Every AC under this epic's milestones has a Go test, finding-rule, or fixture-validation script that fails if the AC's claim breaks.
- **Layer-4 cells land in the spec, not as free-standing tests** ([ADR-0011](../../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md) §"Cell-coverage commitment"). Every new positive/negative test exercises a cell in `internal/workflows/spec/branch/`; the final M-0158 consolidates the cell table and extends the drift policy. Free-standing tests outside the spec table are a smell — they're either evidence of a missing cell or a redundant test, both of which the drift policy surfaces.
- **Ritual content edits land on the embedded snapshot.** Per [ADR-0016](../../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md), `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/<skill>/SKILL.md` is the canonical authoring location for ritual content; AC tests under `internal/policies/` assert against the embedded bytes via path constants (per G-0182's consolidation). One source of truth — no testdata-fixture / embedded-snapshot dual-edit dance, no cross-repo coordination.

## Design decisions

These four decisions were the prior "Open questions" entries; they're pre-decided here because they cross-constrain milestones and the spec table — deferring them to per-milestone drafting would force re-litigation at each AC pass.

| Decision | Choice | Rationale |
|---|---|---|
| **Trailer key name** | `aiwf-branch:` | Shorter and consistent with the existing `aiwf-actor:`, `aiwf-to:`, `aiwf-scope:` naming. Adds one line to `internal/gitops/trailers.go` constants + one regex case in `ValidateTrailer`. Forward-compatible with ADR-0009's eventual `aiwf-cycle-worktree-branch:` (the two compose: scope-level coupling is the foundation, cycle-level is per-commit redundancy that sharpens it). |
| **Branch-context detection in preflight** | Accept either signal: explicit `--branch <name>` referring to an existing branch, *or* current checkout matches a ritual shape from `internal/branchparse/`. Fail with actionable error if neither. | Two signals are no more complex to implement than one and substantially more usable: rituals produce recognizable shapes, ad-hoc invocations name the branch explicitly. The shared `branchparse` package guarantees the preflight's shape regex matches what `aiwfx-start-epic` / `aiwfx-start-milestone` produce — by construction, not by review. |
| **Finding scope** | Per-commit. One finding per violating commit. | Clear signal; the pre-push range naturally bounds noise (the operator sees only their own about-to-push commits). Per-scope-lifetime aggregation hides which commit is the offender. Defense-in-depth value beats noise concern. |
| **Finding severity** | Warning at first land. Tighten to error after one full epic of usage, gated on `false_positive_rate < threshold`. | Matches the M-0125 pattern: surface the rule with breathing room, then ratchet. Severity transition is recorded as a D-NNN at the time of tightening, not pre-committed here. |

Three further decisions land in milestone bodies (each is local to one milestone, not cross-cutting): the exact ritual-shape regex set (M-0102), the error message text for the preflight refusal (M-0103), and the spec-table package layout under `internal/workflows/spec/branch/` (M-0158).

## Corner cases — surfaced as spec cells

Per the user's directive *"these corner cases become part of these verifications"*, the corner cases below land as **named cells** in the layer-4 spec table (M-0158), each with paired positive/negative tests. The list is intended to be exhaustive of the cases I'm aware of today; future cases land via the drift-policy mechanism.

1. **AI-actor authorize on main, no `--branch`** → preflight rejects (illegal cell, rejection layer = verb-time, error code `branch-context-required`).
2. **AI-actor authorize with `--branch epic/E-NN-X` on a non-existent branch** → preflight rejects (illegal cell, rejection layer = verb-time, error code `branch-not-found`).
3. **AI-actor authorize on `epic/E-NN-X` without `--branch`, ritual shape matches** → preflight accepts; trailer records current branch (legal cell).
4. **AI-actor commit on main while scope's `aiwf-branch:` is `epic/E-NN-X`** → finding fires (illegal cell, rejection layer = check-time, finding code `isolation-escape`).
5. **AI-actor commit on `epic/E-NN-X` while scope's `aiwf-branch:` is `epic/E-NN-X`** → finding silent (legal cell).
6. **AI-actor commit on `epic/E-NN-X` while scope is paused** → finding silent (legal cell — commits made under a paused scope ride the same branch they were opened against; paused does not change the binding).
7. **AI-actor commit on `epic/E-NN-Y` while active scope's `aiwf-branch:` is `epic/E-NN-X`** → finding fires (illegal cell — branch mismatch is the same shape as main, just a different wrong target).
8. **Human cherry-pick of `ai/X` commit from `epic/E-NN-X` onto `main`** → finding silent (legal cell — committer ≠ actor + cherry-pick marker in body = sovereign re-author; the cherry-picked commit on main is the human's act).
9. **AI-actor commit on `epic/E-NN-X`, then human merges epic/E-NN-X into main** → finding silent on the merge (legal cell — first-parent reachability puts the `ai/X` commit behind the merge commit on main, not on main's first-parent spine; the merge commit's actor is human).
10. **Sovereign override on a violating commit via `--force --reason` on a subsequent `git commit --amend`** → finding silent (legal cell — the amend's `aiwf-force:` trailer is the gated override; per the trailer-shape rule, `--force` requires a `human/` actor, so the override is structurally human-sovereign).
11. **AI-actor commit before any scope has been opened on its entity** → finding silent (legal cell — no scope, no binding; defense lives elsewhere via the existing untrailered-entity audit).
12. **AI-actor commit on a worktree path that does not match its branch identity** (subagent did `git checkout main` from inside the worktree) → finding fires (illegal cell — branch identity, not path, is the load-bearing axis; case 4 covers this mechanically because the commit's branch is main).

The catalog above is the input to the M-0158's spec-table work, not a normative replacement. The milestone walks the catalog cell-by-cell, registers each as a `Rule` or `AntiRule` entry, pairs it with the test landed by the relevant prior milestone, and extends the drift policy to fail CI when a new branch-layer cell lands without a paired test.

## Sovereign override surface — gated, audited, last-resort

Per the user's directive *"absolutely watertight ... with the user overriding as last resort but gated"*, each layer's override is described explicitly so the *only* paths out are the ones below. Anything else is a bug.

| Layer | Override mechanism | Gate | Audit trail |
|---|---|---|---|
| **Pre-dispatch (PreToolUse hook)** | Operator chooses not to use `isolation: "worktree"` kwarg and instead uses the parent-side precondition pattern. | The hook only denies the unreliable kwarg; the precondition pattern is the supported path. No override needed because the hook does not block legitimate work. | Session-layer; not in git. The pattern itself produces a `git worktree add` commit visible in `git reflog` on the parent worktree. |
| **At-dispatch (M-0103 preflight)** | `aiwf authorize --force --reason "<one-sentence>"` | Existing trailer-shape rule: `--force` requires `human/` actor and non-empty `--reason`. AI invocation of `--force` is refused by the kernel. | `aiwf-force:` trailer on the authorize commit; readable via `aiwf history <id>` and surfaced by the standing `provenance-untrailered-entity-commit` audit. |
| **Post-hoc (M-0106 finding)** | (a) Re-author via cherry-pick (committer ≠ actor + marker = sovereign re-author, finding silent). (b) Amend the violating commit with `aiwf-force: "<reason>"` trailer + `human/` actor. | Cherry-pick marker is structural (git writes it); committer is structural. Trailer-shape rule gates the amend path. Neither path is invokable by an AI actor. | (a) The cherry-pick marker in the commit body is the trail. (b) `aiwf-force:` trailer on the amended commit is the trail; the original commit's hash changes, recorded via `aiwf-prior-entity:` if the amend rides through a verb. |
| **At-check (the F-NNNN waiver pattern)** | `aiwf promote F-NNNN waived --force --reason "..."` per ADR-0003. | Existing F-NNNN AC-closure rule and trailer-shape rule. | F-NNNN waiver commit, readable via `aiwf history F-NNNN`. |

**The override is intentionally noisy.** Each override path stamps a permanent, kernel-readable trailer. The override's *use* is a flag in `aiwf history`, not a normalcy. If the override starts appearing routinely, that's information the model needs revisiting — exactly the validation hook ADR-0010 §"Validation" already commits to.

## Sequencing relative to ADR-0009 / E-0019

ADR-0009 Decision 3 (currently `proposed`) names a finding also called `isolation-escape` but with a different mechanism: per-commit `aiwf-cycle-worktree-branch:` trailer redundancy, scoped to *cycle* commits (subagent work under E-0019's orchestration substrate). E-0030's M-0106 ships the **scope-level foundation**: `aiwf-branch:` recorded once on the authorize commit, read by walking back from each `ai-actor` commit to the most recent active scope on its entity. The two compose:

- **Foundation (E-0030 / M-0106)** — `aiwf-branch:` on authorize commit; per-commit walk-back to find active scope; reachability check. Polices every `ai-actor` commit under an authorized scope, whether or not E-0019's cycle substrate is in play.
- **Extension (E-0019)** — `aiwf-cycle-worktree-branch:` on *every* cycle commit (Decision 2). Lets the check rule skip the walk-back for cycle commits — same finding, cheaper read, per-commit-decidable. Adds zero new failure modes the foundation doesn't already catch.

This means **E-0030 does not depend on ADR-0009 ratifying.** It also means the finding *name* is reserved by both — they share it because they share the rule, not because they conflict. If ADR-0009 ratifies first, its design adjusts to "extends E-0030's `isolation-escape` with per-cycle-commit redundancy." If E-0030 ships first (likely), ADR-0009's Decision 3 description is amended to acknowledge the foundation. Either order works.

## Success criteria

- Every milestone listed in *Milestones* below is `done`.
- An AI-actor `aiwf authorize <id> --to ai/<agent>` invocation that does not pass `--branch` (and is not already on a recognized ritual branch) fails with an actionable error pointing at the ritual surface to use. The error names the override path explicitly (`--force --reason "..."`).
- `aiwf check` reports `isolation-escape` (and only `isolation-escape`, with no false positives on the corner cases enumerated above) when an AI-actor's commits violate ADR-0010's branch convention; reports nothing when the convention is followed.
- Running `aiwfx-start-epic` against this repo (or a fixture project) lands the promote-to-active and authorize commits on `main` (or the parent branch) *before* the epic branch is cut. The stale G-0059 paragraph at step 6 of the embedded skill is retired.
- The same holds for `aiwfx-start-milestone` against its parent epic branch; the "must be on epic branch already" precondition is enforced, not a silent fallthrough.
- The layer-4 branch-choreography spec table under `internal/workflows/spec/branch/` is populated, every cell has at least one matching test under `internal/policies/`, and the drift policy fails CI when a new branch-layer cell lands without a paired test (parallel to M-0124 / M-0125 for layers 1–3).
- The override surface is exercised by at least one positive test per layer (preflight, finding) — proving the override works as documented when needed, and exists nowhere else.
- [G-0099](../../gaps/G-0099-worktree-isolation-parent-side-precondition.md) and [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md) are addressed with `--by-commit` resolvers pointing at the relevant milestones in this epic.
- Author iteration on main remains unencumbered: no new pre-commit or pre-push blocker fires on human-actor commits.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Ritual skill updates lag the kernel surface, leaving operators unable to use the new flag through standard ritual flow | Low | Each ritual-update milestone (M-0104, M-0105) edits the embedded snapshot directly. Post-[ADR-0016](../../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md) the embedded snapshot is the canonical authoring location — no cross-repo coordination — so the kernel change and the ritual edit can land in the same commit (or back-to-back commits in the same wf-patch). |
| `isolation-escape` finding produces false positives on legitimate cross-branch work | Medium | Author override per ADR-0010 — the finding fires on AI-actor commits *only*; the human author's manipulations are sovereign. The cherry-pick case is mechanically handled via committer-vs-actor + marker recognition (corner case 8 above). Finding ships at *warning* severity for one epic of usage to surface unanticipated false-positive classes before tightening to *error*. |
| The cherry-pick recognition heuristic (committer ≠ actor + marker in body) misses a legitimate sovereign re-author path | Low | Documented fallback: `aiwf-force: "<reason>"` trailer on the cherry-picked or amended commit suppresses the finding via the existing force-trailer rule. The hint text on the finding names this path. If the heuristic-miss rate is non-negligible in practice, a `D-NNN` decision can elevate the explicit-trailer path to the canonical one. |
| Landing the chokepoint mid-stream breaks an in-flight epic | High | E-0030 lands on main; in-flight epics on their own branches inherit the change at next merge. The other session can adapt at its own pace, and `--force --reason` remains the escape hatch. The finding's *warning*-severity at first land further softens the landing for in-flight work. |
| The trailer-key extension breaks existing `aiwf history` rendering | Low | The new `aiwf-branch:` key is added alongside existing ones; renderer falls back gracefully when the key is absent (older commits). Drift-test under `internal/policies/trailer_keys.go` catches inconsistent emission. |
| Branch reachability via first-parent gives a wrong answer for an unusual merge shape (e.g., `git merge -s ours`, octopus merge with the wrong parent ordering) | Low | Per-commit fire + warning severity at first land surfaces the case without blocking the push. The hint text names the override path; the case becomes evidence for a refinement D-NNN if it recurs. |
| Layer-4 spec table's package layout under `internal/workflows/spec/branch/` differs from layers 1–3 enough to fragment the methodology | Medium | The M-0158's deliverable includes a one-page design note (recorded inline in the milestone body or as a D-NNN if the decision warrants its own entity) explaining the layout choice. The drift policy is shared with layers 1–3 by construction. |

## Milestones

<!--
Sequenced. M-0104 and M-0105 are siblings and can be parallelized after M-0103.
M-0106 also depends only on M-0102 + M-0103, so it can run in parallel with the rituals work.
The M-0158 (allocated via `aiwf add milestone --epic E-0030` once this body is committed)
consolidates the cells from all prior milestones; it lands last.
-->

- [M-0102](M-0102-aiwf-authorize-branch-flag-scope-branch-trailer-coupling.md) — `aiwf authorize --branch` flag + `aiwf-branch:` trailer + `internal/branchparse/` extraction · depends on: —
- [M-0103](M-0103-ai-side-preflight-aiwf-authorize-refuses-without-ritual-branch-context.md) — AI-side preflight: refuse AI-actor scope opening without ritual branch context · depends on: M-0102
- [M-0104](M-0104-aiwfx-start-epic-sequencing-fix-closes-g-0116.md) — `aiwfx-start-epic` sequencing fix (closes G-0116); retire stale G-0059 language · depends on: M-0102, M-0103
- [M-0105](M-0105-aiwfx-start-milestone-sequencing-alignment.md) — `aiwfx-start-milestone` sequencing alignment; tighten "must be on epic branch" precondition · depends on: M-0102, M-0103
- [M-0106](M-0106-kernel-finding-isolation-escape-closes-g-0099.md) — Kernel finding `isolation-escape` (closes G-0099 fully); cherry-pick recognition; sovereign override surface · depends on: M-0102, M-0103
- [M-0158](M-0158-layer-4-branch-choreography-spec-cells-drift-policy-extension.md) — Layer-4 branch-choreography spec cells + drift-policy extension · depends on: M-0102, M-0103, M-0104, M-0105, M-0106

## ADRs produced

This epic produces no new ADRs — it implements [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md) and extends [ADR-0011](../../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md) to layer 4. Any decision that surfaces during implementation (e.g., the exact ritual-shape regex set, the finding-severity transition timing) is recorded inline in the relevant milestone spec, or as a D-NNN if the decision warrants its own entity.

## References

- [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md) — Branch model: ritualized work on branches, author iteration on main (source of truth for this epic)
- [ADR-0011](../../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md) — Legal-workflow spec methodology (E-0033, done); §"Scope" carves out layer 4 for this epic
- [ADR-0009](../../../docs/adr/ADR-0009-orchestration-substrate-vs-driver-split.md) — Orchestration substrate (Decision 3: isolation as parent-side precondition — extends this epic's foundation per §"Sequencing relative to ADR-0009 / E-0019")
- [ADR-0014](../../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) — Embed-and-materialize rituals (E-0038, done); the embedded-snapshot edit shape M-0104/M-0105 follows
- [ADR-0016](../../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md) — Retire upstream rituals authoring channel (G-0193, done); collapses ritual authoring onto the embedded snapshot
- [G-0099](../../gaps/G-0099-worktree-isolation-parent-side-precondition.md) — worktree isolation; tier-3 partial-closed by `.claude/hooks/validate-agent-isolation.sh`; full closure under M-0106
- [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md) — sequencing fix; addressed by M-0104
- [E-0019](../E-0019-parallel-tdd-subagents-with-finding-gated-ac-closure/epic.md) — Parallel TDD subagents; sharpens this epic's `isolation-escape` with per-cycle-commit redundancy
- `internal/workflows/spec/` — the spec-cell package this epic extends with a `branch/` sub-package (M-0158)
- `internal/cellcoverage/` — the cell-test fixture helpers (`CellFixture.AuthorizeScope`, etc.) this epic's tests consume
- `internal/cli/status/worktrees.go:485` — `parseEntityFromBranch`, the helper M-0102 lifts into `internal/branchparse/`
- `docs/pocv3/design/provenance-model.md` — principal × agent × scope; sovereign override
- `CLAUDE.md § "Subagent worktree isolation"` — the precondition pattern this epic codifies in kernel form
- `CLAUDE.md § "Ritual content authoring"` — the embedded-snapshot edit shape M-0104/M-0105 use
