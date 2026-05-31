---
name: wf-patch
description: One-off branch-and-merge ritual for fixes, chores, or tweaks too small to warrant a milestone — typos, config nudges, single-line bug fixes, dependency bumps. Creates a fix/patch/chore branch, lands one focused change, gates commit and merge behind explicit human approval. Use when the user describes a single focused change tied to one issue (or none).
---

# wf-patch

A lightweight ritual for changes too small to be a milestone but too significant to lose in a careless commit. The branch + explicit-merge shape is the audit trail; the commit and merge gates are the safety net.

## When to use

The user describes a single focused change that:

- Maps to one issue, or to a single-line summary if there's no issue.
- Doesn't need a spec or acceptance criteria.
- Will land in one merge to mainline.

If any of those break — unrelated changes bundled, an AC list emerging, or a planning conversation starting — stop and switch to a heavier ritual.

## Workflow

1. **Read the linked issue or task** if one exists. State the user-observable goal in your own words before touching code; if you can't, the change isn't ready.

2. **Create a descriptive branch** from the project's mainline:
   - `fix/<short-slug>` — bug fix.
   - `patch/<short-slug>` — small behavior tweak that isn't a bug.
   - `chore/<short-slug>` — config, dependencies, doc nudges, no behavior change.

   Keep the slug short and conventional. Branch lifetime is the duration of the patch.

3. **Implement the change.** Touch only what's needed. Resist refactoring along the way — that's what a milestone is for, not a patch.

4. **Verify locally.** Run the project's test suite and any linter the project has wired up. Confirm green before staging.

5. **Stage the change** and draft a commit message: one-line subject, optional body explaining *why*.

6. 🛑 **Commit gate.** Show the user the staged diff and the proposed commit message. **Stop and wait for explicit "commit" approval.** Never commit unprompted, even on what looks like a trivial change.

7. **After commit approval:** commit, push.

8. 🛑 **Merge gate.** Confirm with the user before merging the patch back to mainline. The mechanism — open a PR, fast-forward main to the patch branch, cherry-pick onto main, rebase-and-merge, etc. — follows the consuming project's `CLAUDE.md` §"Working in this repo" policy (or equivalent). Reference the issue if one exists. The skill does not prescribe the mechanism; the project does.

9. **After merge:** delete the branch locally and on the remote, confirm the change is on mainline.

10. **Reflection (optional).** If the patch surfaced a pattern, pitfall, or implicit decision worth keeping, record it where the project records such things. If the project has no such habit, skip — don't invent file conventions on the fly.

## What this skill explicitly does not do

- Does not write a spec or acceptance criteria. If you're tempted to, the work is too big for `wf-patch`.
- Does not run a TDD cycle. If the change requires test-first development, escalate to `wf-tdd-cycle` on the same patch branch.
- Does not touch planning state, milestones, or roadmaps. Patches are off-roadmap by design.
- Does not merge for you. The merge is your handoff to the project's normal merge flow.

## Anti-patterns

- *"While I was in there I also fixed X"* — split into two patches.
- *"It's just one line, no need for a separate branch"* — every patch goes through a branch and an explicit merge. The branch is the audit trail; the merge mechanism is project-specific.
- *"I'll update the roadmap from this patch"* — never.

## Constraints

- 🛑 Never commit, push, or merge without explicit human approval (steps 6, 7, 8).
- Tests must be green before the commit gate.
- Branch prefixes are `fix/`, `patch/`, `chore/`. No other prefixes for this skill.
