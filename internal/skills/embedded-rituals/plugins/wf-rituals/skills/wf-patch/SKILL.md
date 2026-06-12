---
name: wf-patch
description: One-off branch-and-merge ritual for fixes, chores, or tweaks too small to warrant a milestone — typos, config nudges, single-line bug fixes, dependency bumps. Creates a fix/patch/chore branch, lands one focused change, gates commit, wrap, and push behind explicit human approval. Use when the user describes a single focused change tied to one issue (or none).
---

# wf-patch

A lightweight ritual for changes too small to be a milestone but too significant to lose in a careless commit. The branch + explicit-merge shape is the audit trail; the self-review and the three gates — commit, wrap, push — are the safety net.

## Gate discipline

Per CLAUDE.md §"Working with the user," mutating actions are gated behind explicit human approval. This skill fires three gates:

1. **Commit gate** — after self-review, before the commit lands.
2. **Wrap gate (declared sequence)** — one approval covering the patch's *enumerated* terminal sequence: local merge to mainline, tracker closure (e.g. `aiwf promote G-NNNN addressed --by-commit <sha>`) when the patch closes a tracked item, and cleanup (local branch deletion, worktree removal). The gate question lists every action verbatim; approval binds to exactly that list, and the user may approve a subset. Any deviation — merge conflict, check finding, unexpected dirty state, anything not on the list — aborts the sequence and re-gates from the point of deviation.
3. **Push gate** — push to origin is never part of the wrap sequence. It is the only action that leaves the machine; it always stands alone.

The consolidation at the wrap gate is sound only because mechanical gates carry the safety load: the full local CI gate green at the verify step, plus whatever pre-push validation the project wires up. If the project has no mechanical gates, fall back to one approval per action — the declared-sequence gate is earned, not free.

Never bundle beyond the declared sequence. "Commit + merge" in one prompt is wrong; "merge + push" in one prompt is wrong. The temptation to bundle is strongest at the end of the procedure when the path forward looks obvious — that is when the push gate matters most.

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

4. **Verify locally.** Run the project's full local CI gate — the same checks CI runs on push (e.g. a `make ci` target), not a subset. A linter that only runs in CI is debt waiting at the push boundary. Confirm green before staging.

5. **Self-review the diff.** Walk the full diff with reviewer eyes (the `wf-review-code` checklist shape): correctness, untested branches, conventions, documentation impact. Fix findings inline and re-run step 4 if code changed. This is the user's review moment too — the commit gate presents the reviewed diff, not a raw one.

6. **Stage the change** and draft a commit message: one-line subject, optional body explaining *why*.

7. 🛑 **Commit gate.** Show the user the staged diff, the self-review outcome, the green-gate evidence, and the proposed commit message. **Stop and wait for explicit "commit" approval.** Never commit unprompted, even on what looks like a trivial change.

8. **After commit approval:** commit. The patch branch is normally never pushed — the merge in step 9 is local, so the branch lives and dies on this machine.

9. 🛑 **Wrap gate (declared sequence).** Present the enumerated terminal sequence and wait for approval:
   - Merge to mainline. The mechanism — fast-forward, rebase-and-merge, cherry-pick — follows the consuming project's `CLAUDE.md` §"Working in this repo" policy (or equivalent). The skill does not prescribe the mechanism; the project does.
   - Tracker closure, if the patch closes a tracked item (e.g. `aiwf promote G-NNNN addressed --by-commit <sha>`).
   - Cleanup: delete the local branch; remove the worktree if one was used.

   List each action verbatim in the gate question. Approval binds to exactly the list; honor a partial approval. If anything deviates mid-sequence — conflict, check finding, dirty state — stop and re-gate.

   *PR-flow projects:* if the project merges via PR rather than locally, the declared sequence does not apply — pushing the branch and opening the PR are outward actions. Gate the branch push, open the PR per project policy, and let the forge's review flow take over.

10. **After wrap approval:** execute exactly the approved sequence, in order. Report each action as it completes.

11. 🛑 **Push gate.** Mainline now carries the patch (and the closure commit, if any). Show what will be pushed and wait for explicit "push" approval. If a remote copy of the patch branch exists, confirm its deletion separately — remote deletes are not recoverable from local state.

12. **After push approval:** push.

13. **Reflection (optional).** If the patch surfaced a pattern, pitfall, or implicit decision worth keeping, record it where the project records such things. If the project has no such habit, skip — don't invent file conventions on the fly.

## What this skill explicitly does not do

- Does not write a spec or acceptance criteria. If you're tempted to, the work is too big for `wf-patch`.
- Does not run a TDD cycle. If the change requires test-first development, escalate to `wf-tdd-cycle` on the same patch branch.
- Does not touch planning state, milestones, or roadmaps. Patches are off-roadmap by design. (Tracker closure of the item the patch fixes — a gap promote, an issue close — is the one exception, and it rides the wrap gate.)
- Does not merge without approval. The wrap gate is the handoff; the merge mechanism is the project's.

## Anti-patterns

- *"While I was in there I also fixed X"* — split into two patches.
- *"It's just one line, no need for a separate branch"* — every patch goes through a branch and an explicit merge. The branch is the audit trail; the merge mechanism is project-specific.
- *"The wrap was approved, so I'll push too"* — the wrap gate never covers the push. Outward actions stand alone.
- *"I'll update the roadmap from this patch"* — never.

## Constraints

- 🛑 Never commit, merge, promote, push, or delete a branch without explicit human approval. Three gates: commit (step 7), wrap (step 9, declared sequence), push (step 11).
- The full local CI gate must be green before the commit gate.
- Branch prefixes are `fix/`, `patch/`, `chore/`. No other prefixes for this skill.
