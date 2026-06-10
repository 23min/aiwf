---
name: wf-patch
description: One-off branch-and-merge ritual for fixes, chores, or tweaks too small to warrant a milestone — typos, config nudges, single-line bug fixes, dependency bumps. Creates a fix/patch/chore branch, lands one focused change, gates commit and merge behind explicit human approval. Use when the user describes a single focused change tied to one issue (or none).
---

# wf-patch

A lightweight ritual for changes too small to be a milestone but too significant to lose in a careless commit. The branch + explicit-merge shape is the audit trail; the commit and merge gates are the safety net.

## Gate discipline

Per CLAUDE.md §"Working with the user," every mutating action this skill walks you through — commit, push, merge, deleting a remote branch — is its own gate. The numbered steps below list when each gate fires; the standing invariant is **one approval per action, no bundling**.

If you find yourself authoring a single approval prompt that combines "commit + push" or "merge + delete branch," the prompt is wrong; split it. The temptation to bundle is strongest at the end of the procedure when the path forward looks obvious — that is exactly when each gate matters most.

This applies regardless of any cadence pattern inherited from a prior session's summary across `/compact`.

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

7. **After commit approval:** commit.

8. 🛑 **Push gate.** Show the local commit. Stop and wait for explicit "push" approval — even immediately after a clean commit.

9. **After push approval:** push to the remote.

10. 🛑 **Merge gate.** Confirm with the user before merging the patch back to mainline. The mechanism — open a PR, fast-forward main to the patch branch, cherry-pick onto main, rebase-and-merge, etc. — follows the consuming project's `CLAUDE.md` §"Working in this repo" policy (or equivalent). Reference the issue if one exists. The skill does not prescribe the mechanism; the project does.

11. **After merge:** delete the local branch. Confirm with the user before deleting the remote branch — local deletes are recoverable from `origin`, remote deletes are not.

12. **Reflection (optional).** If the patch surfaced a pattern, pitfall, or implicit decision worth keeping, record it where the project records such things. If the project has no such habit, skip — don't invent file conventions on the fly.

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

- 🛑 Never commit, push, merge, or delete a remote branch without explicit human approval. Each is its own gate; see steps 6, 8, 10, and 11.
- Tests must be green before the commit gate.
- Branch prefixes are `fix/`, `patch/`, `chore/`. No other prefixes for this skill.
