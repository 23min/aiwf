---
id: E-0028
title: 'Start-epic ritual: sovereign activation with preflight, branch/worktree choice, and optional delegation (closes G-0063 start-side)'
status: active
---

# E-0028 — Start-epic ritual

## Goal

Ship the `aiwfx-start-epic` ritual plus its supporting kernel chokepoints so epic activation becomes a deliberate sovereign act with preflight checks, an explicit branch/worktree choice, and an optional principal-to-agent delegation hand-off. Closes the start-side scope of G-0063; the wrap-side concerns spawn a follow-up gap at this epic's wrap.

## Context

G-0063 frames epic activation as a sovereign moment that the kernel today treats as a one-line FSM flip: `aiwf promote E-NN active` requires no preflight, no human-only enforcement, and no pairing with `aiwf authorize`. The other lifecycle skills already exist (`aiwfx-plan-epic`, `aiwfx-plan-milestones`, `aiwfx-start-milestone`, `aiwfx-wrap-milestone`, `aiwfx-wrap-epic`) — only the start-epic ritual is missing, which collapses the sovereign delegation moment into a side effect of the first milestone start.

Prior work in E-0017 already shipped the kind-generalized `entity-body-empty` finding (M-0066), so the body-completeness chokepoint G-0063 wanted is already in place — this epic relies on it rather than re-introducing it. The work here is the *epic-active-specific* preflight rule, the *sovereign-act* enforcement rule, and the skill that orchestrates the conversation.

## Scope

### In scope

- **Kernel rule** — new `aiwf check` finding `epic-active-no-drafted-milestones` (warning when an `active` epic has zero `draft`-status milestones; informs the skill's preflight).
- **Kernel rule** — human-only enforcement on `aiwf promote E-NN active`: the verb refuses non-human actors unless the standard `--force --reason <text>` sovereign override is used (mirrors how `--force` already requires `human/...` actors). This is the new sovereign-act rule G-0063 names.
- **Rituals plugin skill** — `aiwfx-start-epic` lives upstream in `aiwf-extensions/skills/`, authored during the milestone as a fixture under `internal/policies/testdata/aiwfx-start-epic/SKILL.md` per the CLAUDE.md cross-repo plugin-testing convention. The skill orchestrates:
  1. Preflight read of the epic spec (Goal/Scope/Out-of-scope concrete; relies on `entity-body-empty`).
  2. Drafted-milestone check (relies on the new `epic-active-no-drafted-milestones` finding).
  3. `aiwf check` clean of refusal-level findings.
  4. Project tests/build advisory pass.
  5. **Worktree-placement prompt** (Q&A): *none / `.claude/worktrees/<branch>/` / `../aiwf-<branch>/`*.
  6. **Branch prompt** (Q&A; deliberately stubbed pending G-0059): *stay current / create branch `<name>`*.
  7. Delegation prompt: in-loop, or delegate via `aiwf authorize E-NN --to ai/<id>`.
  8. Sovereign promotion: `aiwf promote E-NN active` (commit 1).
  9. Optional `aiwf authorize E-NN --to ai/<id>` (commit 2).
  10. Hand-off to `aiwfx-start-milestone` or subagent spawn.
- **Drift-check test** in this repo asserting the rituals-repo SHA recorded at wrap matches the local plugin cache when present (per the M-0090 precedent).
- **Filing a follow-up gap at wrap** capturing the wrap-side concerns deliberately deferred (see Out of scope).

### Out of scope

- **Wrap-side behavior change.** `aiwf promote E-NN done` continues to auto-end scopes for now; the sub-decision in G-0063 ("scope ends *before* `done`") is deferred to a follow-up epic with its own ADR. The skill's design assumes the current wrap-side behavior; the follow-up will adjust both verb and `aiwfx-wrap-epic` together.
- **Human-only enforcement on `aiwf promote E-NN done`.** Pairs with the deferred wrap-side change; out.
- **`aiwfx-wrap-epic` updates** for the new wrap-timing. Pairs with the deferred wrap-side change; out.
- **G-0059 branch-model resolution.** The skill's branch prompt is interactive precisely because G-0059 has not settled on a convention. When G-0059 lands, the prompt's default can tighten; this epic does not block on or scope that work.
- **Generalizing the sovereign-act rule to other kinds.** Whether `contract-active`, `ADR-accepted`, or other transitions are sovereign-act-shaped is a separate open question — out for this epic.
- **Subagent-spawn mechanics** (delegating mode's hand-off). The skill prescribes the *what* (open an `aiwf authorize` scope, hand off the work); the *how* of spawning the subagent is Claude Code surface and explicitly outside kernel/plugin scope.

## Constraints

- **Skill authored via the canonical fixture pattern.** `SKILL.md` lives at `internal/policies/testdata/aiwfx-start-epic/SKILL.md` during the milestone; the rituals-repo copy is a wrap-time commit. The drift-check ships in the same milestone and skips cleanly when the plugin cache is absent (per CLAUDE.md).
- **Sovereign promotion is a single commit.** `aiwf promote E-NN active` keeps its existing "one verb = one commit" shape; the optional authorize is a *separate* second commit. The skill orchestrates both; it does not collapse them.
- **Preflight uses existing kernel signals.** Body completeness uses the existing `entity-body-empty` finding; FSM legality uses the existing `aiwf promote` check. The new rules add the *epic-active-specific* gap (`no drafted milestones`) and the *sovereign-act* enforcement, not duplicate work.
- **`--force --reason` remains the sovereign override.** Non-human actors are refused by default for `aiwf promote E-NN active`, but the standard `--force --reason <text>` path stays open for sovereign acts (mirrors the existing pattern). Tests must cover both refusal and override paths.
- **What undoes this?** `aiwf promote E-NN proposed` reverses the activation (FSM allows the backward step); `aiwf authorize E-NN --end` ends any opened scope. The skill is non-mutating up to step 8; aborting before that costs nothing.

## Success criteria

- [ ] `aiwf check` emits `epic-active-no-drafted-milestones` (warning) on a tree where an `active` epic has zero drafted milestones.
- [ ] `aiwf promote E-NN active` refuses non-human actors with a clear error pointing at the sovereign-act rule and the `--force --reason` override; the `--force` path succeeds with proper trailers.
- [ ] `aiwfx-start-epic` lands in the rituals plugin at the expected path, registered in `plugin.json`, and is invocable via `/aiwfx-start-epic` after `/reload-plugins`.
- [ ] The skill's preflight walks every step listed under *Scope → In scope → Rituals plugin skill*; a fresh contributor running it through this repo's E-NNNN-of-the-day can activate an epic from `proposed` without invoking any kernel verb directly.
- [ ] Worktree-placement and branch prompts surface as deliberate Q&A choices; selecting *direct on main* is a valid path, not a special case requiring override.
- [ ] Drift-check test in `internal/policies/` skips cleanly without a plugin cache and fails when the rituals-repo content diverges from the fixture (matches the M-0090 precedent).
- [ ] A follow-up gap is filed at wrap capturing the deferred wrap-side concerns (scope-end-before-done behavior change, human-only enforcement on `done`, `aiwfx-wrap-epic` update).
- [ ] G-0063 promoted to `addressed` at wrap with a trailer referencing this epic.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the skill default to creating a worktree, or default to "stay on main"? | No (cosmetic) | Decide during the skill milestone; record in the milestone's *Design notes*. |
| Should the drafted-milestone preflight emit a refusal (not a warning) if the epic is being activated with zero drafted milestones? | No | Default to warning per G-0063's "preflight checks" table; tighten in a follow-up if real usage shows the warning being ignored. |
| Where does the `aiwf authorize` scope-end happen if the user aborts mid-skill after step 9? | No | The skill documents that scope ending is the human's responsibility on abort; ergonomic helpers can come later. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The sovereign-act rule on `aiwf promote E-NN active` breaks an existing automation that uses a non-human actor. | Medium | Audit the repo's `aiwf-*` automation paths (CI, hooks) before landing; the `--force --reason` override stays available for any legitimate non-human invocation; ship the rule with a clear error message. |
| The skill orchestrates two commits (promote + authorize) that drift apart if the user aborts between them. | Low | The skill makes the authorize prompt atomic with promote in the conversation — the user answers both before either commit lands. Tested with a "user aborts between prompts" path. |
| Cross-repo fixture-vs-rituals-repo drift bites again (per G-0088 territory). | Low | The drift-check test is part of the same milestone; lands red until the rituals-repo SHA is recorded, then green. |

## Milestones

<!-- Refined in aiwfx-plan-milestones. Sequence: kernel chokepoints first so the skill's
     preflight relies on real findings; skill ships last. -->

- [M-0094](M-0094-add-aiwf-check-finding-epic-active-no-drafted-milestones.md) — `epic-active-no-drafted-milestones` check rule (kernel). · depends on: —
- [M-0095](M-0095-enforce-human-only-actor-on-aiwf-promote-e-nn-active.md) — Sovereign-act enforcement on `aiwf promote E-NN active`. · depends on: —
- [M-0096](M-0096-ship-aiwfx-start-epic-skill-with-worktree-and-branch-preflight-prompts.md) — `aiwfx-start-epic` skill (fixture + drift-check + ACs). · depends on: M-0094, M-0095
- [M-0097](M-0097-close-m-0094-95-96-verification-seams-m-0095-automation-audit-chokepoint-and-ac-5-drift-comparator.md) — Close verification seams: M-0095 audit chokepoint + AC-5 drift comparator. · depends on: M-0094, M-0095, M-0096

## ADRs produced (optional)

(None expected. The skill is documented in its own SKILL.md; the kernel rules are documented in their `--help` text and `aiwf check` finding hints. If the sovereign-act rule's generalization to other kinds becomes a real question during implementation, that is its own decision, recorded then via `aiwfx-record-decision`.)

## References

- [G-0063](../../gaps/G-0063-no-defined-start-epic-ritual-epic-activation-is-a-deliberate-sovereign-act-with-preflight-optional-delegation-but-kernel-treats-it-as-a-one-line-fsm-flip.md) — gap framing.
- G-0059 — branch-model gap; the skill's branch prompt is the placeholder pending its resolution.
- E-0017 — entity-body chokepoint epic (done); supplies the `entity-body-empty` finding this epic relies on.
- M-0066 — `entity-body-empty` rule (done).
- CLAUDE.md *Cross-repo plugin testing* — convention for authoring plugin SKILL.md via fixture.
- CLAUDE.md *Provenance is principal × agent × scope* — the model this epic's sovereign-act rule operationalizes.
