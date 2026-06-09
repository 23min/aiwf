---
name: aiwfx-plan-epic
description: Scopes, refines, and documents a new aiwf epic. Allocates the next E-NN id via `aiwf add epic`, scaffolds an epic spec from the plugin's template, fills in goal/context/scope/constraints/success criteria, and stages the spec for the user's review. Use when the user says "plan feature X", "design the system for Y", or "I need to build Z". Calls aiwf — install only with the aiwf-extensions plugin.
---

# aiwfx-plan-epic

Scopes a new epic and produces its spec. The skill does the planning conversation and the spec authoring; aiwf owns id allocation and commit.

## When to use

The user wants to start planning a feature, capability, or initiative that doesn't fit inside a single milestone. Phrases that trigger this skill: *"plan feature X"*, *"design the system for Y"*, *"I want to build Z"*, *"start an epic for …"*.

If the work fits in one milestone, skip this skill and use `aiwfx-plan-milestones` directly under an existing epic.

## Workflow

1. **Understand the request.** Ask, don't guess.
   - What problem does this solve?
   - Who benefits?
   - What's in scope; what's explicitly out of scope?
   - What constraints (tech stack, timeline, dependencies)?
   - What does success look like at epic close?

2. **Check existing context.**
   - Read `ROADMAP.md` (if present) for current epics and priorities.
   - Walk `work/epics/` for related or overlapping epics.
   - Walk `work/gaps/` for previously deferred work that fits the new epic's scope.

3. **Confirm scope with the user.** Spell back what you understood. Get a yes before writing.

4. **Allocate the id and scaffold the spec.**

   ```bash
   aiwf add epic --title "<imperative title>"
   ```

   `aiwf` allocates the next free `E-NN`, creates `work/epics/E-NN-<slug>/epic.md` with the minimal body skeleton (`## Goal / ## Scope / ## Out of scope`), and produces one commit with `aiwf-verb: add` trailers.

5. **Replace the body with the rich template** at this plugin's `templates/epic-spec.md`. Fill in:
   - **Goal** — 1–2 sentences, value-shaped.
   - **Context** — what exists; why now; prior epics this builds on.
   - **Scope** — in / out, both populated.
   - **Constraints** — invariants, banned shortcuts, shim policies.
   - **Success criteria** — observable outcomes at epic close, *not* tests.
   - **Open questions** — what's blocking; how each gets resolved.
   - **Risks** — only if there are real risks.
   - **Milestones** — known candidates with one-line descriptions; refine via `aiwfx-plan-milestones`.

   Keep frontmatter (`id:`, `status:`) untouched — `aiwf add` set those correctly. The spec's body is where the planning conversation lives.

6. **Use reference-phrasing for list-derived counts.** When success criteria reference a list defined elsewhere in the spec, phrase as a reference, not a count. *"Every ADR listed in the *ADRs produced* table is merged"* not *"all 16 ADRs merged"*. Counts drift; references don't.

7. **Update `ROADMAP.md`** by running:

   ```bash
   aiwf render roadmap --write
   ```

   This regenerates the markdown table of epics + milestones from the current tree. Don't hand-edit the roadmap.

8. **Optional tracker linkage.** If the project mirrors planning into an external issue tracker, create or link the epic record according to the project's convention.

## What this skill does NOT do

- Does not promote the epic to `active`. The epic stays `proposed` until milestones are planned and work begins. Use `aiwf promote E-NN active` when ready.
- Does not break the epic into milestones. That's `aiwfx-plan-milestones`.
- Does not commit the body fill — the body edit happens in the working tree; the user commits when the spec is ready (or runs `aiwfx-plan-milestones` next which produces its own commit).

## Anti-patterns

- *Planning the epic and immediately starting work.* Don't skip review. The epic spec is the place where scope changes are cheap; once milestones are running, scope changes are expensive.
- *Hand-writing scalar counts.* "5 milestones" rots; "every milestone listed below" doesn't.
- *Treating "Open questions" as scratch.* If a question is blocking, state how it gets resolved.
- *Inventing id-shaped labels for not-yet-allocated milestones.* Per CLAUDE.md and G-0184: don't write `M-a`, `M-alpha`, `M-NNNN`, "Phase 1", "alpha/beta" anywhere — committed prose **or** conversation. The mechanical chokepoint `body-prose-id` catches malformed shapes that leak into committed bodies; the discipline above keeps the conversation clean. **In conversation**, when sequencing several not-yet-allocated milestones, short numeric labels (`M-1`, `M-2`, `M-3`) are acceptable as conversational shorthand — distinguishable from canonical ids (`M-0001`+) by their narrow width. Once `aiwf add milestone` runs, the verb assigns the canonical id and the deliverable name becomes the slug; replace the casual labels with the real ids in any prose that lands in entity bodies.

## Closing the planning session

Planning is closed. Two paths from here:

- **Continue to milestones now** → invoke `aiwfx-plan-milestones`. The merge-to-main prompt fires at *its* end, covering both skills' commits in one operation.
- **Stop here** → the epic spec is settled but no milestones are planned yet. The planning commits live on the ritual branch; merge them to main now so the freshly-allocated `E-NNNN` id and the spec are visible to other worktrees, machines, or operators.

For the stop-here path, prompt the user as a strong recommendation with explicit decline (not optional guidance):

> Planning is closed. Default behavior is to merge to main now. Decline only with a specific reason — entity shape uncertain, near-term re-planning expected, team convention overrides. Merge now? (Y/n)

When the operator confirms, drive the in-place merge:

```bash
git checkout main
git merge --ff-only <ritual-branch>     # falls back to a three-way merge if FF isn't possible
```

When the operator declines, capture the one-line reason in the conversation transcript so future readers know.

**Workflow assumption — single checkout, not a worktree.** The skill assumes the operator runs planning in a single checkout (the same one calling the skill). Planning is sequential conversation; it doesn't benefit from worktree-level parallelism, and the cwd-and-session switching that worktrees impose adds friction without payoff. Worktrees are an implementation-phase tool, not a planning-phase tool.

## Next step

→ `aiwfx-plan-milestones` to break the epic into sequenced milestones, or merge to main and pause if stopping here. Run `aiwfx-start-milestone <M-NNNN>` only after the planning commits have landed on main.
