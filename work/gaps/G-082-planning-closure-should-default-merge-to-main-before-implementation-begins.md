---
id: G-082
title: Planning closure should default-merge to main before implementation begins
status: open
discovered_in: E-21
---

# G-082 — Planning closure should default-merge to main before implementation begins

## What's missing

`aiwfx-plan-epic` and `aiwfx-plan-milestones` close their planning conversations and point the operator at the next step (typically `aiwfx-start-milestone`), but neither skill recommends merging the planning commits to main first. The current convention is *"planning lives on the ritual branch alongside implementation; merge happens once at epic wrap."* That works for solo flows but holds settled data hostage on a long-lived branch and doesn't compose well across parallel epics or multiple worktrees.

The recommendation should fire at *whichever* planning skill is the last to close:

- `aiwfx-plan-epic` exits at *"plan milestones now, or stop here?"* — if **stop here**, the planning commits for the epic alone want to merge to main now.
- `aiwfx-plan-milestones` always exits at *"sequence confirmed"* — both the epic spec and milestone-allocation commits want to merge together.

The skill-level surface and a kernel-level check rule are both candidate landing places for the guarantee. Layer the resolution: skill prompt first; check rule when usage shows the prompt isn't sticky.

## Why it matters

Settled planning is durable kernel data — entities, ACs, depends_on edges, ADR allocations, ROADMAP renders. Holding it on a ritual branch:

1. **Hides the entity tree from other worktrees, machines, or operators.** Other Claude Code sessions, or a sibling worktree, see only main's view; the freshly-allocated `M-NNN` ids are invisible until merge.
2. **Lets implementation milestone branches stack on a long-lived parent.** A `milestone/M-078-...` branched from `epic/E-NN-...` rather than from main carries the parent's planning + the milestone's implementation. As the epic accumulates milestones, the diff at epic wrap balloons.
3. **Increases the surface for cross-branch id-allocation surprises.** Parallel epics on independent ritual branches each walk their own filesystem-only view of `next-free-id`. Branch A allocates `M-078` on its branch; branch B, started from main before that merge, also picks `M-078`. The conflict only surfaces at merge time.
4. **Forces the merge ceremony to happen all at once at epic wrap when the diff is largest.** Earlier, smaller merges are easier than one big one.

The friction is bounded for a strict solo flow with one in-flight epic at a time. It scales badly the moment a second worktree, second epic, or second operator enters the picture — which is the natural growth path for any project that succeeds.

## Reproducer

Today's `aiwfx-plan-milestones` skill body, last visible step:

> 9. **Confirm the sequence with the user.** Walk through the milestone list together. Identify any scope adjustments before drafting begins.
>
> ## Next step
>
> → `aiwfx-start-milestone <M-NNN>` for the first milestone in the sequence.

No mention of merging planning to main. The same shape applies to `aiwfx-plan-epic`, which currently closes by pointing at `aiwfx-plan-milestones` without a stop-here exit explicitly recommending the merge.

## Resolution shape — two layers

### Layer 1 — skill-level (v1, ships first)

Edit both planning skills' bodies to add a closing prompt at planning closure. Concrete shape (refine at implementation):

> **Planning is closed. Default behavior is to merge to main now. Decline only with a specific reason — entity shape uncertain, near-term re-planning expected, team convention overrides. Merge now? (Y/n)**

When the operator confirms: drive the merge (FF if main is an ancestor of the ritual branch, otherwise three-way; abort cleanly on conflicts). When the operator declines: capture the one-line reason in the conversation transcript.

For `aiwfx-plan-epic`: bifurcate the closing step.

- *"Plan milestones now, or stop here?"*
  - **Plan milestones now** → call `aiwfx-plan-milestones`; merge prompt fires at *its* end (one merge covering both skills' commits).
  - **Stop here** → merge prompt fires now at this skill's end.

For `aiwfx-plan-milestones`: the merge prompt is unconditional at sequence-confirmation step.

The framing is *strong recommendation with explicit decline*, not optional guidance. The default is to merge; declining is an active operator decision with a stated reason.

### Layer 2 — kernel-level check rule (v2, optional later)

Add a check rule:

> **warning: `not-merged-before-implementation`** — milestone `M-NNN` at status `in_progress` whose `aiwf add milestone` commit is not reachable from `main`. Hint: planning didn't merge before implementation began. Resolution: FF-merge the planning commits to `main`, or `--force` the start with a reason if the per-branch divergence is intentional.

This is a real kernel-side gate. The pre-push hook catches it; the rule is mechanical (does `git merge-base --is-ancestor <add-commit> main` succeed?). Operator can `--force` past it with a documented reason for the rare case the per-branch divergence is intentional.

Layer 2 satisfies the kernel principle *"framework correctness must not depend on the LLM's behavior"*: the skill prompt is advisory; the check is authoritative.

**Sequencing:** ship Layer 1 first. Let usage show whether the skill prompt is sticky enough on its own. File Layer 2 as a follow-up gap (or absorb into this gap's wrap) if real flows demonstrate the prompt is being skipped.

## Out of scope

- **Auto-merging from the skill.** The merge is operator-initiated; the skill prompts and walks through, doesn't take the action sovereignly. Sovereign acts trace to a named human (kernel principle on provenance).
- **Defining how parallel epics' planning interleaves on main.** Separate concern: the merge-now-default reduces the surface but doesn't eliminate it. If two epics both merge planning to main and both later allocate milestones independently, id-collisions can still happen. That's a broader gap (probably G-059's territory).
- **The branch model itself.** G-059 covers *"how do epic→milestone hierarchies mirror git branches?"* This gap fills a specific recommendation gap *inside* that broader model — it does not propose to redefine the model.
- **Retrofitting Layer 2 onto already-in-flight milestones.** When the rule lands, it warns going forward; existing in-flight work isn't blocked retroactively.

## Resolution sketch

The skill body edits are small — two skills, one prompt apiece, plus the bifurcation in `aiwfx-plan-epic`. Each skill body is in `aiwf-extensions/skills/aiwfx-plan-*/SKILL.md`. The kernel rule (Layer 2) is a new check under `internal/check/` with a small fixture-driven test. The whole gap closes in one small milestone for Layer 1; Layer 2 is its own milestone if and when filed.

## Discovered in

- E-21 milestone planning, 2026-05-08. Same session that surfaced G-081. Discussion turned from "did the rename collision suggest we should merge planning to main earlier?" (answer: not directly; G-081 is the right rename fix) to "but yes, generally — and the planning skills don't currently encode that." This gap captures the latter.

## References

- G-081 — sibling gap on the rename verb's pre-flight. Same family ("verbs / skills should refuse-with-guidance, not allow-then-warn"); different specific case.
- G-059 — broader gap on the canonical branch model. This gap fills a specific recommendation slot inside G-059's territory, not a redefinition of it.
- G-063 — *"no start-epic ritual"*. Adjacent ritual gap.
- `aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md`, `aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md` — the skill bodies that need the prompt edits.
- CLAUDE.md *Engineering principles* §"Framework's correctness must not depend on the LLM's behavior" — informs Layer 2's existence rationale.
- CLAUDE.md *Engineering principles* §"Errors are findings, not parse failures" — informs Layer 2's shape (warning, not refusal; the operator can `--force` past with a reason).
