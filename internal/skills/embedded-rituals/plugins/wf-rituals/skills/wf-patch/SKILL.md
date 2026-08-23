---
name: wf-patch
description: One-off branch-and-merge ritual for fixes, chores, or tweaks too small to warrant a milestone — typos, config nudges, single-line bug fixes, dependency bumps. Creates a patch/ branch (patch/G-NNNN-<slug> when it closes a gap), lands one focused change, gates commit, wrap, and push behind explicit human approval. Use when the user describes a single focused change tied to one issue (or none).
---

# wf-patch

A lightweight ritual for changes too small to be a milestone but too significant to lose in a careless commit. The branch + explicit-merge shape is the audit trail; the independent review and the three gates — commit, wrap, push — are the safety net.

## Gate discipline

Mutating actions are gated behind explicit human approval. This skill fires three gates:

1. **Commit gate** — after the independent review, before the commit lands.
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

### 1. Read the linked issue or task

If one exists, state the user-observable goal in your own words before touching code; if you can't, the change isn't ready.

### 2. Create a descriptive branch

Cut it from the project's mainline in its own worktree rather than switching the main checkout in place. `aiwf worktree add` creates the linked worktree and materializes rituals (skills, agents, templates, guidance) into it atomically, in one step (substitute your mainline branch):

```bash
aiwf worktree add patch/G-NNNN-<short-slug> --base main --print-path
```

This ritual runs as the calling session's own direct work (not a dispatched subagent), so call the harness `EnterWorktree(path: <printed path>)` tool right after `aiwf worktree add` succeeds: creating the worktree only puts it on disk, it does not relocate the session, and only `EnterWorktree` does that.

Name the branch for the gap it closes so the statusline can surface it:

- `patch/G-NNNN-<short-slug>` — closes a tracked gap (the common case). The
  statusline's session-entity HUD shows `G-NNNN` with its status glyph and
  color while you work — the same treatment epics get on a ritual branch.
- `patch/<short-slug>` — no tracked gap (a typo, a config nudge). No HUD
  entry, which is fine.

Keep the slug short and conventional. Branch lifetime is the duration of the patch.

### 3. Implement the change

Touch only what's needed. Resist refactoring along the way — that's what a milestone is for, not a patch.

If the patch fixes a **defect**, it lands with the check that pins it — a regression test that fails without the fix, or a lint or gate rule (`wf-codebase-health` D5). This sits on the same footing as the CHANGELOG entry in step 4: required, and not deferred to a follow-up. Two escapes, each *stated* at the commit gate so the human can veto it rather than discovering it later. First, a change with no logic to pin — a typo, whitespace, a dependency bump, a pure-config nudge — where you say "no logic to pin". That is the same *change class* step 6's review carve-out names, but a separate statement: one licenses skipping the review, this one licenses skipping the pin, and the gate expects both when both apply. Second, a defect you don't pin, because you can't or because you judge it not worth the test, where you say which and record why in the project's tracker. Neither escape is automatic: an unspoken skip is the silent correction this rule exists to stop. A patch that ships a fix, a changelog line, and no check leaves the project's checks exactly as thin as they were, which is how the same defect class comes back.

### 4. Add a CHANGELOG entry

Add a new sub-section under `## [Unreleased]` in `CHANGELOG.md`, using a Keep-a-Changelog category as the heading: `### Added — G-NNNN: <one-line summary>`, `### Changed — G-NNNN: <one-line summary>`, or `### Fixed — G-NNNN: <one-line summary>` (when the patch closes no tracked gap, drop the `G-NNNN:` prefix and carry just the summary). The body is a short paragraph distilling the **user-visible delta**.

This step always runs — there is no skip. Unlike a milestone, whose change can land inside its parent epic's one entry at `aiwfx-wrap-epic` time, a patch has no parent to roll up into: its own wrap is the only chance the change is ever recorded. For a genuinely internal-only patch (a test-only fix, an internal refactor with no observable behavior change), the entry still lands, but shrinks to one line stating plainly that nothing user-facing changed — never a full skip.

Stage `CHANGELOG.md` alongside the rest of the change; it rides the same commit gate as everything else (step 8), not a separate approval.

### 5. Verify locally

While iterating, a fast inner-loop gate (e.g. a `make check-fast` target: vet + lint + tests, without the race detector / coverage / end-to-end self-check) gives quicker feedback. Before staging the commit, run the project's full local CI gate — the same checks CI runs on push (e.g. a `make ci` target), not a subset — and confirm green; that gate, not the fast one, is what protects the merge to mainline in the wrap step. A linter that only runs in CI is debt waiting at the push boundary.

### 6. Independent review of the diff — not self-review

Dispatch a *fresh-context* reviewer (a subagent with no authorship attachment) over the staged diff, briefed adversarially per `wf-review-code` §"Independence" (enumerate the load-bearing claims, instruct *verify by measuring not reasoning*, name the risk areas). Run the **code-quality lens** (`wf-review-code`: correctness, untested branches, conventions, documentation); if the patch introduced a non-trivial design — a new module/package boundary, core abstraction, or data model (the `wf-rethink` trigger; see its §"The non-trivial-design trigger" for the full criterion and the skip-list) — also run the **design-quality lens** (`wf-rethink`) on that unit. **If the patch added or changed tested logic, also run the test-sufficiency lens** (`wf-vacuity`) on the unit's assertions — a required invocation whenever there are assertions to attack; it is automatically satisfied when the change ran through `wf-tdd-cycle` (whose own vacuity step covers it), so this is the backstop for logic touched outside a full cycle. `wf-vacuity`'s output is advisory: strengthen a weak assertion or surviving mutant where you can, otherwise surface it at the commit gate for the human to weigh — it does not mechanically block. Fix every blocking finding inline; re-run step 5 if code changed; re-capture the fingerprint below if you've already taken it, since an inline fix changes the staged diff it must match; then **confirm the fixes** — mechanically for mechanical findings (re-run the gate/scan), by re-dispatching a fresh reviewer for judgment-level ones. The author re-reading their own diff is *not* a substitute — it is the failure mode this step exists to close. The commit gate presents the independently-reviewed diff and the review outcome, not a raw one.

*The one carve-out:* a change with no logic to review — a typo, whitespace, a dependency-version bump, a pure-config nudge — may rely on the mechanical gates plus a self-skim, **but only if you state that explicitly at the commit gate** ("no independent review: <reason>"), so the skip is the human's to veto rather than a silent default.

**Reviewer-dispatch contract.** A dispatched reviewer verifying revert or vacuity behavior (the `wf-vacuity` mutation probe run under this step) must not mutate the shared working tree or index at all — no `git stash`, no in-place revert-then-restore against the live tree the orchestrating session is about to commit from. A plain `stash`/`pop` restores content unstaged regardless of its staged state beforehand, which can silently desync the index from the diff staged before dispatch. Instead, the reviewer reads pre-edit content via `git show HEAD:<path>` (or an equivalent read-only plumbing command) or performs the mutate/revert cycle in its own isolated worktree — never against the checkout the commit gate is about to act on.

**Before dispatching**, capture the current `git diff --cached` (redirect to a scratch file, or hash it) — this is the fingerprint the commit gate (step 8) re-verifies against before the commit runs, so a reviewer that touched shared git state despite the contract above can't silently ship a corrupted commit.

### 7. Stage the change and draft a commit message

One-line subject, optional body explaining *why*.

### 8. 🛑 Commit gate

Immediately before showing the diff below, re-run `git diff --cached` and confirm it is byte-identical to the fingerprint captured before dispatching the reviewer in step 6 — never trust staging state carried across a subagent dispatch. A bare non-empty check is not sufficient: a reviewer that mutated shared git state can leave the index non-empty while missing the actual fix (the commit that lands would be a broken intermediate, passing only its own pinning test). If the diff has changed, stop — do not proceed to the commit message below — and re-stage and re-review before continuing. (Under the one carve-out in step 6 — no reviewer dispatched, no fingerprint captured — this re-check has nothing to compare against and is a no-op; the exemption travels with the carve-out.)

Show the user the staged diff, the independent-review outcome (or the named carve-out from step 6), the pinning statement from step 3 (the check that landed, or the named escape and where it is recorded), the green-gate evidence, and the proposed commit message. **Stop and wait for explicit "commit" approval.** Never commit unprompted, even on what looks like a trivial change.

### 9. After commit approval

Re-run `git diff --cached` against the same fingerprint one more time, immediately before running `git commit` — the gap between gate approval and execution is exactly where an unnoticed mutation would land uncaught. Only then commit. The patch branch is normally never pushed — the merge in step 12 is local, so the branch lives and dies on this machine.

### 10. 🛑 Wrap gate (declared sequence)

Present the enumerated terminal sequence and wait for approval:

- **Merge** the patch branch to mainline (step 12).
- **Tracker closure**, if the patch closes a tracked item (step 13).
- **Cleanup** — delete the local branch; remove the worktree if one was used (step 14).

List each action verbatim in the gate question. Approval binds to exactly the list; honor a partial approval. If anything deviates mid-sequence — conflict, check finding, dirty state — stop and re-gate.

*PR-flow projects:* if the project merges via PR rather than locally, the declared sequence does not apply — pushing the branch and opening the PR are outward actions. Gate the branch push, open the PR per project policy, and let the forge's review flow take over.

Once the sequence is approved, execute it in order:

### 11. Reconcile mainline with the patch branch

Run this immediately before the merge — not as an earlier precondition a concurrent push can invalidate. The target is your *local* mainline (the branch the patch forked from and merges into), not the remote-tracking ref.

**Move into mainline's worktree; do not check mainline out.** A branch can be checked out in one worktree at a time, so where the patch was cut into its own worktree (step 2) mainline is held elsewhere and `git checkout main` fails with *"fatal: … is already used by worktree at …"*. Changing directory into the worktree that already holds it moves no branch and cannot fail that way. Do it once, here — steps 12 through 15 then run as plain commands carrying no path:

```bash
MAIN_WT=$(git worktree list --porcelain \
  | awk -v b="refs/heads/main" \
        '/^worktree /{wt=substr($0,10)} $0=="branch "b{print wt; exit}')
cd "${MAIN_WT:?mainline is checked out nowhere — take the fallback below}"
git rev-parse --abbrev-ref HEAD          # prints main; stop if it does not
```

Resolve and `cd` in a **single** command. A shell variable does not survive to the next one — where each command runs in its own shell, as it does for an assistant driving one per tool call, a `MAIN_WT` read back at step 12 is empty, and `git -C ""` does not fail: it runs against the current worktree. `${MAIN_WT:?…}` aborts rather than moving nowhere, and the `rev-parse` states where you landed. Working directory is what survives, which is why nothing below carries a path — within the repository; a worktree placed outside it may be reset back to the repo root, which the `rev-parse` also catches.

`substr($0,10)` rather than `$2` so a worktree path containing spaces survives.

If the `cd` aborts, mainline is checked out nowhere. Check it out here and stay put — you are then already in the right directory, and the rest of the ritual reads the same. That is safe in this ritual specifically: nothing after this step needs the patch branch checked out, and step 14 deletes it, which requires that it is not.

```bash
git checkout main
```

1. Fetch, then fast-forward local mainline to its upstream — folds in commits another clone pushed; concurrent local commits are already on it (substitute your mainline branch and remote; a project with no remote skips the fetch and this fast-forward, but still needs the `cd` above):

   ```bash
   git fetch
   git merge --ff-only origin/main
   ```

2. Check whether mainline has advanced past the patch branch's fork point (substitute the project's mainline ref):

   ```bash
   git merge-base --is-ancestor main <branch>
   ```

3. If that check fails: integrate mainline into the patch branch, resolve any conflicts there, and re-run the full local CI gate (step 5) on the reconciled branch. Mainline can move again during that gate, so re-run this check immediately before merging.

4. Only once the check passes does the merge (step 12) run.

### 12. Merge the patch branch to mainline

Step 11 left this session standing in mainline's worktree, so the merge needs no path. Assert that in the same command, and abort rather than warn — a working directory can be lost between commands (a re-gated sequence, a context compaction, a harness that resets it when a command ends outside the repository), and the loss does not announce itself where you are looking:

```bash
[ "$(git rev-parse --abbrev-ref HEAD)" = main ] || { echo "not in mainline's worktree — re-run step 11"; exit 1; }
git merge --no-ff --no-commit <branch>
git commit -m "Merge patch/<branch>: <summary>"
```

Substitute your mainline branch in the test. Checking rather than assuming is what makes the plain commands safe: without it, a lost directory puts the merge on the patch branch, where it reports `Already up to date.` at exit 0 and mainline receives nothing.

`--no-ff` keeps the patch as one identifiable merge commit — the branch + explicit-merge audit trail this ritual exists to produce. `--no-commit` leaves the merge staged so the commit above carries the deliberate summary message, not git's default.

### 13. Tracker closure

If the patch closes a tracked item:

```bash
aiwf promote G-NNNN addressed --by-commit <sha>
```

No `--root`: the closure commit belongs on mainline, and step 11 put this session there. Run this from the patch worktree instead and the commit lands on the patch branch, after which step 14's branch delete fails with *"the branch … is not fully merged"*.

Standing in the right place also fixes what the guard proves. The closure is mechanically guarded — `aiwf` refuses a `--by-commit` SHA unreachable from the resolved root's `HEAD` — so from mainline's worktree a refusal means the merge did not land: reconcile and merge first (steps 11–12), and don't `--force` past it. From the patch worktree the same guard rejects the merge commit even when the merge landed perfectly, while accepting the patch commit whether it landed or not.

### 14. Cleanup

Delete the local branch; remove the worktree if one was used.

Order matters and the branch cannot go first: git refuses to delete a branch a worktree still holds — *"error: cannot delete branch … used by worktree at …"* — and you cannot remove the worktree you are standing in. Leave it, then remove it, then delete the branch:

```bash
PATCH_WT=$(git worktree list --porcelain \
  | awk -v b="refs/heads/<branch>" \
        '/^worktree /{wt=substr($0,10)} $0=="branch "b{print wt; exit}')

# Step 11 already moved this session out of the patch worktree, so it can be removed.
# Where the session entered it through the harness `EnterWorktree` tool rather than by
# `cd`, leave it with `ExitWorktree` first.
[ -n "$PATCH_WT" ] && git worktree remove "$PATCH_WT"

git branch -d <branch>
```

### 15. 🛑 Push gate

Mainline now carries the patch (and the closure commit, if any). Push is outward and irreversible — its own gate, never part of the declared-sequence gate above. Show what will be pushed and wait for explicit "push" approval. Then:

```bash
git push origin main
```

Push from the worktree holding mainline — step 11 put this session there and step 14 removed the other one. The pre-push hook runs `aiwf check` against the pushing worktree's branch range, so pushing from there is what puts the merge commit inside the audited range.

If a remote copy of the patch branch exists, confirm its deletion separately — remote deletes are not recoverable from local state.

### 16. Reflection (optional)

If the patch surfaced a pattern, pitfall, or implicit decision worth keeping, record it where the project records such things. If the project has no such habit, skip — don't invent file conventions on the fly.

## What this skill explicitly does not do

- Does not write a spec or acceptance criteria. If you're tempted to, the work is too big for `wf-patch`.
- Does not run a full TDD cycle. If the change requires test-first development, escalate to `wf-tdd-cycle` on the same patch branch. That is not an exemption from pinning: a patch that fixes a defect still lands the check that fails without the fix (step 3).
- Does not touch planning state, milestones, or roadmaps. Patches are off-roadmap by design. (Tracker closure of the item the patch fixes — a gap promote, an issue close — is the one exception, and it rides the wrap gate. The `CHANGELOG.md` entry, step 4, is likewise off-roadmap but always required.)
- Does not merge without approval. The wrap gate is the handoff; the merge always lands as a `--no-ff` commit, never a fast-forward.

## Anti-patterns

- *"While I was in there I also fixed X"* — split into two patches.
- *"It's just one line, no need for a separate branch"* — every patch goes through a branch and an explicit `--no-ff` merge. That pairing is the audit trail.
- *"It's internal, no need for a CHANGELOG entry"* — every patch adds one, even if it's a single line stating nothing user-facing changed. A patch has no parent epic to roll the change into later.
- *"I'll add the regression test in a follow-up"* — the follow-up is where pinning goes to die, and the fix is cheapest to pin while you still hold why it broke. Land the check with the fix, or state at the commit gate that you are not pinning it, why, and where that is recorded.
- *"The wrap was approved, so I'll push too"* — the wrap gate never covers the push. Outward actions stand alone.
- *"I reviewed it myself, it looks fine"* — self-review is not the gate. Step 6 dispatches a fresh-context reviewer; the author cannot see their own blind spots. The only exception is the explicitly-stated no-logic carve-out.
- *"I'll update the roadmap from this patch"* — never.

## Constraints

- 🛑 Never commit, merge, promote, push, or delete a branch without explicit human approval. Three gates: commit (step 8), wrap (step 10, declared sequence), push (step 15).
- The full local CI gate must be green before the commit gate.
- Every patch adds a `CHANGELOG.md` entry under `## [Unreleased]` (step 4) — always, with a minimal one-line form for internal-only patches. No skip.
- 🛑 A patch that fixes a defect lands the check that pins it (step 3) — a regression test that fails without the fix, a lint rule, or a gate entry. Two named escapes only: a change with no logic to pin, and a defect you don't pin — because you can't, or because you judge it not worth the test — stated at the commit gate and recorded in the project's tracker. Both are stated explicitly so the human can veto them; neither is a silent default.
- Branch is `patch/G-NNNN-<short-slug>` when the patch closes a gap, else `patch/<short-slug>`. The single `patch/` prefix is the convention for this skill; the gap id, when present, is what the statusline's session-entity HUD reads.
