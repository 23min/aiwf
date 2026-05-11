---
name: aiwfx-plan-milestones
description: Decomposes an approved aiwf epic into a sequenced set of independently-shippable milestones. Allocates each M-NNNN via `aiwf add milestone --epic E-NNNN`, scaffolds each milestone spec from the plugin's template, sequences them by dependency. Use after `aiwfx-plan-epic` when the user says "break this into milestones" or "plan the work for E-NNNN".
---

# aiwfx-plan-milestones

Decomposes an existing epic into milestones. The skill drives the conversation about *what to ship in what order*; aiwf owns id allocation and per-milestone commits.

## When to use

An epic spec exists. The user says: *"break E-NNNN into milestones"*, *"plan the work for the auth epic"*, *"sequence the milestones for X"*.

If the epic doesn't exist yet, use `aiwfx-plan-epic` first.

## Workflow

1. **Read the epic spec.** Open `work/epics/E-NNNN-<slug>/epic.md`. Understand:
   - The goal — what the epic is delivering.
   - The scope (in / out).
   - The constraints — what each milestone must respect.
   - The success criteria — what "done" looks like at epic close.

2. **Decompose into milestones.** Each milestone:
   - Is **independently shippable**. After M-0001 lands, the system is in a coherent state even if M-0002 never runs.
   - Has clear, **testable acceptance criteria**.
   - Targets **1–3 days of focused work**. If a candidate is bigger, split it. If smaller, fold it into a sibling.
   - Has explicit dependencies (or none). Forward-flowing — M-0002 may depend on M-0001; never the reverse.

3. **Sequence them.** Foundational first. Group related work; don't scatter concerns. Identify any milestones that can be parallelized (no dependency between them).

4. **Allocate each milestone via aiwf.** For each one in sequence:

   ```bash
   aiwf add milestone --epic E-NNNN --title "<imperative title>"
   ```

   `aiwf` allocates the next free `M-NNNN` (global, not epic-scoped), creates `work/epics/E-NNNN-<slug>/M-NNNN-<slug>.md` with the minimal body skeleton, sets `parent: E-NNNN` in frontmatter, and produces one commit per milestone with `aiwf-verb: add` trailers.

5. **Replace each milestone's body with the rich template** at this plugin's `templates/milestone-spec.md`. Fill in:
   - **Goal** — 1–2 sentences.
   - **Context** — what exists before; what must be in place; why now.
   - **Acceptance criteria** — testable, numbered (AC1, AC2, …).
   - **Constraints** — non-negotiable invariants for *this* milestone.
   - **Design notes** — locked decisions; reference ADRs by id.
   - **Out of scope** — what this milestone explicitly does NOT do.
   - **Dependencies** — prior milestones, external deps, decision records.

   Frontmatter (`id`, `parent`, `status: draft`) was set by `aiwf add` — don't touch.

6. **Declare milestone dependencies via verb, not by hand-editing frontmatter.** Two writer surfaces, both producing one atomic commit with `aiwf-verb` trailers (M-0076):

   ```bash
   # At allocation time: pass --depends-on on aiwf add milestone
   aiwf add milestone --epic E-NNNN --tdd <policy> \
     --title "..." --depends-on M-PPPP[,M-QQQQ]

   # Post-allocation (the edge surfaced after the milestone was allocated):
   aiwf milestone depends-on M-NNNN --on M-PPPP[,M-QQQQ]

   # Empty an existing list:
   aiwf milestone depends-on M-NNNN --clear
   ```

   Replace-not-append semantics: a second `--on` invocation replaces the list, it does not extend. To add a single dependency to an existing list, pass the full updated list. `--on` and `--clear` are mutually exclusive. Each id passed to `--depends-on` or `--on` must already resolve to an existing milestone — typos and forward-references are refused with an error naming the unresolvable id. Cycle detection happens at the next `aiwf check` (and pre-push hook); the writers don't pre-check global DAG validity.

   Do not hand-edit `depends_on:` in frontmatter. `aiwf edit-body` refuses frontmatter changes, and a plain `git commit` against the milestone file trips the kernel's `provenance-untrailered-entity-commit` warning. Both writer verbs above leave a trailered commit that `aiwf history M-NNNN` can render.

7. **Update the epic's Milestones list.** Edit the epic spec to list all milestones in execution order. Use the format from the epic template — link, one-line description, dependencies.

8. **Update `ROADMAP.md`** by running:

   ```bash
   aiwf render roadmap --write
   ```

9. **Confirm the sequence with the user.** Walk through the milestone list together. Identify any scope adjustments before drafting begins.

10. **Merge planning to main.** Planning is closed; the entity tree on this ritual branch now diverges from main. Default behavior is to merge to main now so the freshly-allocated `M-NNNN` ids, the epic's updated Milestones list, and any `depends_on` edges are visible to other worktrees, machines, or operators. Held on a long-lived branch, planning data is hostage: other Claude Code sessions see only main's view, parallel epics walk separate filesystem-only `next-free-id` views (id collisions surface only at eventual merge), and milestone branches stack on a long-lived parent — making the epic-wrap diff balloon.

    Prompt the user as a strong recommendation with explicit decline:

    > Planning is closed. Default behavior is to merge to main now. Decline only with a specific reason — entity shape uncertain, near-term re-planning expected, team convention overrides. Merge now? (Y/n)

    When the operator confirms, drive the in-place merge:

    ```bash
    git checkout main
    git merge --ff-only <ritual-branch>     # falls back to a three-way merge if FF isn't possible
    ```

    When the operator declines, capture the one-line reason in the conversation transcript so future readers know.

    **Workflow assumption — single checkout, not a worktree.** The skill assumes the operator runs planning in a single checkout (the same one calling the skill). Planning is sequential conversation; it doesn't benefit from worktree-level parallelism, and the cwd-and-session switching that worktrees impose adds friction without payoff. Worktrees are an implementation-phase tool, not a planning-phase tool.

## What this skill does NOT do

- Does not draft individual milestone specs in deep detail — that happens just-in-time when each milestone is started (via `aiwfx-start-milestone`). This skill produces the milestone *list and shape*, not the full spec body for milestones not yet started.
- Does not promote any milestone past `draft`. Promotion happens at `aiwfx-start-milestone`.

## Anti-patterns

- *Front-loading detail.* Don't write 10 fully-specced milestones up front. Spec details rot fast; AC definitions written 6 weeks before the work starts are usually wrong.
- *Inventing global ordering when only local matters.* If M-0003 and M-0004 don't depend on each other, leave their order soft.
- *Scope creep mid-decomposition.* If decomposition surfaces work that wasn't in the epic, decide: amend the epic spec (and re-confirm with the user) or capture as a gap (`aiwf add gap`) for later.

## Next step

→ `aiwfx-start-milestone <M-NNNN>` for the first milestone in the sequence. Run after the planning commits have landed on main (per step 10) so subsequent implementation branches can fork cleanly from main rather than stacking on the ritual branch.
