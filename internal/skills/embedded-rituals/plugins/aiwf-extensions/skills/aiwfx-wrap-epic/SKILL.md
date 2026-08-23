---
name: aiwfx-wrap-epic
description: Closes an aiwf epic — verifies all milestones done, scaffolds a wrap artefact, harvests ADR candidates, runs scoped doc-lint, promotes the epic to done, then merges the epic branch into mainline with a trailered merge commit. Use when the user says "wrap E-NNNN" or "close the auth epic" and every milestone in the epic is wrapped. Commit and push require explicit human approval.
---

# aiwfx-wrap-epic

Closes an epic. The epic itself is a coordination unit — closing it means: every milestone is `done`, the integration branch merges to mainline via a trailered merge commit, the wrap artefact captures what shipped and what didn't, and the epic's status flips to `done`.

## Principles

- **Wrap is closure, not release.** Tagging, packaging, publishing — those are `aiwfx-release`. This skill ends the planning unit.
- **Branch cleanup is opt-in.** Local branches are preserved (so `tig` / `gitk` keep labelling history); origin branches for completed milestones are deleted to reduce remote refname clutter.
- **Nothing is deleted at wrap.** Specs (with their work-log sections), the wrap artefact — all stay readable forever. Closure is a status change, not a deletion.
- **The merge commit is trailered.** The integration-target merge commit carries `aiwf-verb: wrap-epic`, `aiwf-entity: E-NNNN`, `aiwf-actor: human/<id>` trailers — exactly the keys the kernel's `provenance-untrailered-entity-commit` finding expects. Without the trailers, the rule fires once per entity file touched by the merge.

## Precondition

1. Every milestone in this epic has `status: done`. Run `aiwf check` and verify; if any are still `in_progress` or `draft`, stop and surface them.
2. The epic branch (if used) is up to date — every milestone's final merge commit is on it.
3. Working tree clean.
4. Integration target identified (usually `main`).
5. The project's full local CI gate is green on the epic branch **after integrating current mainline** — the same checks CI runs on push (e.g. a `make ci` target), not a subset. A gate run that predates mainline's latest commits is green on a tree that omits them; it doesn't cover the branch that's about to merge. See "Reconcile the epic branch with mainline" below for the integrate-then-gate mechanics. Long-lived epic branches accumulate lint debt invisibly across milestone wraps; the merge to mainline is the last local moment to catch it. (If the last green run of that gate predates only frontmatter commits — e.g. milestone `promote`s, which touch no Go/build inputs — it is still valid; re-run it only when Go/build inputs changed since. Don't re-run a still-green gate.)
6. Neither the epic's own spec nor any milestone's left a gap open that it explicitly claims to fix. `aiwfx-wrap-milestone`'s own wrap step should already have closed the milestone-level ones; this is the backstop for a milestone wrapped under an older ritual version, or one closed outside the ritual. Nothing checks the epic's own spec earlier.

If precondition 1–5 fails, stop and report. Do not improvise around an unfinished epic. Precondition 6 is not a stop condition — the epic itself is otherwise ready — but a disposition to settle first: if a claimed-fixed gap surfaces still open, either close it (`aiwf promote G-NNNN addressed --by-commit <sha>`, citing the implementing commit), or, when the work advanced it without finishing it, correct the claim to say what landed and what remains. Don't close a partly-addressed gap to satisfy the check. Silence is the forbidden outcome: don't let it become a bare `## Follow-ups carried forward` entry instead.

## One-time setup (per consumer repo)

`wrap.md` is an extension artefact, not a kernel-recognized entity file. The aiwf kernel's `aiwf check` enforces a closed tree shape under `work/` and will flag `wrap.md` as `unexpected-tree-file` unless it's whitelisted. Add this once to the consumer repo's `aiwf.yaml`:

```yaml
# wrap.md is the artefact emitted by aiwf-extensions:aiwfx-wrap-epic.
# It's not a kernel-recognized entity, so whitelist the path so
# `aiwf check` doesn't flag it as an unexpected-tree-file.
tree:
  allow_paths:
    - "work/epics/E-*/wrap.md"
```

If you skip this, the first `aiwf check` after step 6 will warn (or, under `tree.strict: true`, error). Add the entry before staging the wrap artefact.

## Workflow

### 1. Scaffold the wrap artefact

Create `work/epics/E-NNNN-<slug>/wrap.md` (staged, not yet committed):

```markdown
# Epic wrap — E-NNNN

**Date:** <today>
**Closed by:** <actor>
**Integration target:** main
**Epic branch:** epic/E-NNNN-<slug>

## Milestones delivered

- M-NNNN — <title> (merged <short-sha>)
- M-NNNN — <title> (merged <short-sha>)

## Changelog entry

### <Added|Changed|Fixed> — E-NNNN: <one-line summary>

<The user-visible delta for a release-notes reader who has never seen the epic spec: verbs added, behaviour changed, gaps closed. Pick the category the epic's dominant delta falls under; add a second `###` entry only when one category genuinely misrepresents what shipped. One bullet per milestone that shipped a distinct user-visible change; a single paragraph when one covers it. Leave a purely internal milestone out — but when the whole epic is internal, say so in one line rather than omitting the entry.>

## Summary

Two to four sentences on what shipped and why. Reference the goal from the epic spec; honest about what scope shifted mid-flight.

## ADRs ratified

- ADR-NNNN — <slug>          (or "none")

## Decisions captured

- D-NNNN — <slug>             (or "none")

## Follow-ups carried forward

- G-NNNN — <slug>             (gap that survives the epic)

## Handoff

What is ready for the next epic; what is deliberately left open.
```

Use **reference-phrasing for any list-derived count** ("every ADR listed in *ADRs ratified*" rather than "all 4 ADRs"). Avoids drift.

**`## Changelog entry` is authored once — here.** Everything beneath that heading, the `###` category line included, is copied verbatim into `CHANGELOG.md` at step 6; nothing is re-authored there. It sits directly beneath `## Milestones delivered` so that list is in front of you while you write, and beside `## Summary`, which covers the same epic for a reader who has seen the epic spec.

Because that one section travels, keep its references inside it. The reference-phrasing rule above still applies — but a phrase reaching *out* of the section, like "every milestone listed in *Milestones delivered*", resolves here and dangles in `CHANGELOG.md`, where no such section exists. Name the milestones, or name `wrap.md` itself.

### 2. ADR check — harvest decisions worth keeping

Walk the epic's commits. For each candidate decision, ask: *"Would a future reader regret missing the reasoning?"* Signals an ADR is warranted:

- A default changed or a new default introduced.
- A strategy considered and rejected.
- A scope cut or framing shift affecting downstream work.
- A supersession of a prior ADR.

For each candidate, invoke `aiwfx-record-decision` and choose ADR (architectural, durable) or D-NNNN (project-scoped, more local). Record the resulting ids in the wrap artefact's `## ADRs ratified` or `## Decisions captured` section.

### 3. Doc-lint sweep (scoped)

Invoke `wf-doc-lint` against the epic's change-set (every file touched on `epic/E-NNNN-<slug>` since it diverged from the integration target).

Append the report to `wrap.md` under a `## Doc findings` section. If findings include broken references or removed-feature docs, fix or open as gaps before proceeding. `wf-doc-lint` reports only — prose fixes are deliberate edits here.

### 4. 🛑 Declared-sequence gate — close the epic (terminal local sequence)

This is the epic's terminal sequence of *local, reversible* mutations. Present it as a single **declared-sequence gate** that enumerates every action verbatim; the user may approve a subset ("all except the promote"), and any deviation (a merge conflict, a check finding, unexpected dirty state) aborts the sequence and re-gates from the point of deviation. **Excluded from this gate:** the push (step 10) and the origin-branch deletes (step 11) — those are outward and stand as their own gates, never batched here.

The enumerated local sequence is **wrap-artefact commit → promote-done → roadmap regen → merge**:

1. **Wrap-artefact commit** — the CHANGELOG `[Unreleased]` entry + `wrap.md`, trailered (step 6).
2. **Promote** the epic to `done` — status-flip commit (step 7).
3. **Roadmap regen** — regenerate `ROADMAP.md` now that the epic shows `done` (step 8).
4. **Merge** the epic branch into the integration target with a trailered merge commit (step 9).

**Every commit but the merge lands on the epic branch**, so the integration target
receives exactly one commit and never has to be checked out. That matters because
a branch can only be checked out once: under the in-repo worktree convention the
target is already held by another worktree, and `git checkout main` fails outright.
Keeping the sequence on one branch also removes an ordering trap — merging before
the promote leaves the target carrying the merge without the status flip.

Once the sequence is approved, execute it:

### 5. Reconcile the epic branch with mainline

Run this immediately before the merge — not as an earlier precondition a concurrent push can invalidate. The target is your *local* mainline (the branch the epic merges into), not the remote-tracking ref.

1. Fetch, then fast-forward local mainline to its upstream — folds in commits another clone pushed; concurrent local commits are already on it (substitute your mainline branch and remote; a project with no remote skips this step):

   **Stay on the epic branch here.** Steps 6 through 8 commit on it, so unlike the
   other wrap rituals this one must not move into the target's worktree yet — it
   drives that worktree by path, and does so in a single command. A shell variable
   does not survive to the next command, where each command runs in its own shell
   as it does for an assistant driving one per tool call, so the resolution and the
   command that uses it belong together. Substitute the project's mainline ref for
   `main`; `substr($0,10)` rather than `$2` so a path containing spaces survives:

   ```bash
   TARGET=main
   TARGET_WT=$(git worktree list --porcelain \
     | awk -v b="refs/heads/$TARGET" \
           '/^worktree /{wt=substr($0,10)} $0=="branch "b{print wt; exit}')
   git fetch
   if [ -n "$TARGET_WT" ]; then
     git merge --ff-only "origin/$TARGET"
   else
     git fetch origin "$TARGET:$TARGET"   # no worktree holds it — move the ref directly
   fi
   ```

   The `else` arm is what a fast-forward looks like when no worktree holds the
   target. Do **not** check the target out here to obtain one: this worktree holds
   the epic branch, and steps 6 through 8 commit on it. `git fetch <remote>
   <ref>:<ref>` updates a branch without any worktree, and refuses at exit 128
   naming the holding path when one exists — so the two arms are mutually exclusive
   by git's own behaviour rather than by the operator choosing correctly.

   Read the first arm by exit code rather than message: it aborts at exit 128 with
   *"Not possible to fast-forward"* where the target's upstream has diverged from
   the epic, and does nothing at exit 0 where that upstream is already an ancestor.

2. Check whether mainline has advanced past the epic branch's fork point (substitute the project's mainline ref):

   ```bash
   git merge-base --is-ancestor main epic/E-NNNN-<slug>
   ```

3. If that check fails: integrate mainline into the epic branch, resolve any conflicts there, and re-run the project's full local CI gate on the reconciled epic branch. Mainline can move again during that gate, so re-run this check immediately before merging.

4. Only once the check passes does the sequence run. The check is repeated immediately before the merge (step 9), since mainline can move while the wrap commits are being made.

### 6. Wrap-artefact commit — CHANGELOG `[Unreleased]` + `wrap.md`

Steps 6 to 8 all commit on the epic branch. Assert that before the first one —
run from the main checkout they would land on the target instead, and step 9's
merge would then be a no-op:

```bash
[ "$(git rev-parse --abbrev-ref HEAD)" = "epic/E-NNNN-<slug>" ] || echo "not on the epic branch"
```

The `[Unreleased]` section of `CHANGELOG.md` is a per-epic accumulator: every wrapped epic adds an entry here, and `aiwfx-release` later rolls the accumulated entries into a versioned `## [X.Y.Z]` heading. *Without this step, releases ship with empty changelog entries* — that's the `[Unreleased]` drift this step prevents.

Copy `wrap.md`'s `## Changelog entry` section into `CHANGELOG.md` — from its `###` category line down to the next `##` heading, landing at the top of `## [Unreleased]`, newest entry first. Verbatim means the finished text, not the template — step 1 is where the placeholders get filled. Don't re-author the prose here — writing it a second time from the same memory is what the copy rule exists to stop. If reading it in place shows it wrong, fix it in `wrap.md` and copy again, so the two never disagree.

Every epic gets an entry — step 1's template says what to write when the whole epic is internal.

Then stage and commit both, **on the epic branch**. The message and trailers were approved as part of the declared-sequence gate (step 4) — there is no separate commit gate. The commit carries the same three trailer keys as the merge, so `aiwf history E-NNNN` surfaces it alongside:

```bash
git add CHANGELOG.md
git add work/epics/E-NNNN-<slug>/wrap.md
git commit -m "chore(E-NNNN): wrap epic — <one-line summary>" \
  --trailer "aiwf-verb: wrap-epic" \
  --trailer "aiwf-entity: E-NNNN" \
  --trailer "aiwf-actor: human/<id>"
```

### 7. Promote the epic to `done` — last verb-driven commit in the bundle

```bash
aiwf promote E-NNNN done
```

aiwf validates `active → done`, rewrites frontmatter, commits with `aiwf-verb: promote`. (If the epic is still `proposed`, that means no milestone ever started — wrap doesn't apply. Investigate.)

**Why promote is last among verb-driven commits.** The `aiwf promote E-NNNN done` commit ends the authorize scope that opened with `aiwfx-start-epic`. Any commit produced *after* this that goes through a kernel verb — wrap artefact, reallocates, or other verb-driven wrap-bundle commits — would carry `aiwf-authorized-by:` referencing the just-ended scope and trigger the kernel's `provenance-authorization-ended` finding on push, blocking the wrap with no clean remediation short of `--no-verify` or history rewrite. Keeping the promote as the last verb-driven commit guarantees every other verb commit lives under the live scope. The two commits that follow it — the roadmap regen (step 8) and the merge (step 9) — are hand-composed via plain `git commit`, never routed through the CLI's scope-lookup/trailer-decoration path, so neither can receive an auto-stamped `aiwf-authorized-by` whatever its position.

The completion date is recorded in `wrap.md` (step 1) and is recoverable from the `aiwf-verb: promote` commit via `aiwf history E-NNNN`. Do not add a `completed:` field to the epic frontmatter — aiwf's epic schema does not include it, and the parse failure cascades into unresolved-reference findings on every entity that links to this epic.

### 8. Regenerate the roadmap

Still on the epic branch, and after the promote — this is the first point at which the roadmap can reflect the epic's final `done` state:

```bash
aiwf render roadmap --write
```

`--write` only rewrites `ROADMAP.md` on disk — it never commits. Stage and commit any resulting change with the same trailer set as the rest of the bundle:

```bash
git add ROADMAP.md
git commit -m "docs(roadmap): regenerate after E-NNNN wrap" \
  --trailer "aiwf-verb: wrap-epic" \
  --trailer "aiwf-entity: E-NNNN" \
  --trailer "aiwf-actor: human/<id>"
```

If `aiwf render roadmap --write` reported the file already up to date, skip the `git add`/`git commit` — there is nothing to stage.

### 9. Merge the epic branch into the integration target with a trailered merge commit

Everything above is committed on the epic branch, so this merge is the only commit the integration target receives.

Everything the epic branch needed is now committed, so this is where the session
moves into the target's worktree. Resolve and `cd` in a single command, and confirm
where you landed — the target is still never checked *out*, only moved *into*:

```bash
TARGET=main
TARGET_WT=$(git worktree list --porcelain \
  | awk -v b="refs/heads/$TARGET" \
        '/^worktree /{wt=substr($0,10)} $0=="branch "b{print wt; exit}')
cd "${TARGET_WT:?the target is checked out nowhere — create a worktree for it with aiwf worktree add, then cd there}"
git rev-parse --abbrev-ref HEAD          # prints the target; stop if it does not
```

Mainline can move while the wrap commits are being made, so re-run step 5's
ancestor check now — this is the "immediately before the merge" step 5 promises:

```bash
git merge --ff-only "origin/$TARGET"
git merge-base --is-ancestor "$TARGET" epic/E-NNNN-<slug>
```

If that fails, mainline advanced: go back to step 5, integrate it into the epic
branch, and re-run the full local gate there before returning.

Stage the merge **without committing** so the commit-emitting step is the one carrying trailers:

```bash
[ "$(git rev-parse --abbrev-ref HEAD)" = "$TARGET" ] || { echo "not in the target's worktree — re-run the resolution above"; exit 1; }
git merge --no-ff --no-commit epic/E-NNNN-<slug>
```

Assert and merge in one command, aborting rather than warning: a working directory can be lost between commands — a re-gated sequence, a context compaction, a harness that resets it when a command ends outside the repository — and the loss does not announce itself where you are looking. Without the assertion a lost directory puts the merge on the epic branch, where it reports `Already up to date.` at exit 0 and the target receives nothing.

`--no-ff` preserves the epic as a single merge commit. `--no-commit` is what lets the trailers be attached deliberately, so `aiwf history E-NNNN` surfaces the merge alongside the rest of the bundle. (Note the kernel's `provenance-untrailered-entity-commit` rule is *not* the backstop here — it skips ordinary multi-parent merges by construction — so nothing catches an untrailered merge for you.)

Resolve the operator identity from `git config user.email` — identity is runtime-derived, not stored; do not hardcode `<id>`. Then commit with the three required trailers and a Conventional Commits subject:

```bash
git commit -m "chore(epic): wrap E-NNNN — <epic title>" \
  --trailer "aiwf-verb: wrap-epic" \
  --trailer "aiwf-entity: E-NNNN" \
  --trailer "aiwf-actor: human/<id>"
```

The trailer keys are exact — `aiwf-verb`, `aiwf-entity`, `aiwf-actor`. Variant casings (e.g. `Aiwf-Verb`) fail the kernel's trailer-keys policy.

**Do not push yet.**

### 10. 🛑 Push gate

Push is outward and irreversible — its own gate, never part of the declared-sequence gate above. Confirm. Then:

```bash
git push origin "$TARGET"
```

Push from the worktree holding the target, not from the epic worktree. The
pre-push hook runs `aiwf check` against the pushing worktree's branch range, so
pushing from here is what puts the merge commit inside the audited range.

### 11. 🛑 Origin branch cleanup — one gate per delete

Plan the deletions first. List the milestone and epic branches to delete. For each, verify it's merged:

```bash
git branch -r --merged main | grep "milestone/M-NNNN"
git branch -r --merged main | grep "epic/E-NNNN"
```

If a branch isn't shown as merged, stop and report — don't force.

Each `git push origin --delete` is an **outward, irreversible action — its own gate.** Confirm per branch and delete one at a time; **never batch-approve the list** (a batched delete removes per-action judgment on irreversible remote refs). Local branches are not touched (operators prune those on their own schedule):

```bash
git push origin --delete milestone/M-NNNN-<slug>   # its own gate
git push origin --delete epic/E-NNNN-<slug>          # its own gate
```

## Constraints

- 🛑 **The terminal local sequence — wrap-artefact commit, promote-done, roadmap regen, merge — runs under one declared-sequence gate (step 4)**, enumerated verbatim and subset-approvable. The push (step 10) and each origin-branch delete (step 11) are outward and keep their own gates; never batch them.
- 🛑 **The merge commit and the wrap-artefact commit both carry the three required trailers.** Skipping either is the regression the kernel's `provenance-untrailered-entity-commit` finding catches.
- 🛑 **`aiwf promote E-NNNN done` is the last *verb-driven* commit in the bundle** (step 7). It ends the active authorize scope; any commit produced after it that routes through a kernel verb carries an ended-scope `aiwf-authorized-by:` and fails the kernel's `provenance-authorization-ended` check on push. The roadmap regen (step 8) and the merge (step 9) both follow it and are safe — each is hand-composed via plain `git commit`, never routed through the CLI's scope-lookup path, so neither can receive that trailer regardless of position.
- 🛑 **Every commit but the merge lands on the epic branch, and the integration target is never checked out** (step 9). A branch can be checked out once; under the in-repo worktree convention the target is held by another worktree, so step 9 changes directory into that worktree rather than checking the branch out.
- 🛑 **Mainline is reconciled into the epic branch before the merge (step 5), not resolved on mainline mid-merge.** After fetching and fast-forwarding local `main`, if `git merge-base --is-ancestor main epic/E-NNNN-<slug>` is false, integrate mainline into the epic branch, resolve conflicts, and re-run the full local gate there first.
- Every milestone must be `done` before wrap — `aiwf check` and `aiwf history E-NNNN` confirm.
- Branch-cleanup is origin-only. Do not delete local branches.
- The wrap artefact is mandatory. Don't close an epic without one.

## Anti-patterns

- *Wrapping while a milestone is still `in_progress`.* Run `aiwf check` first.
- *Force-deleting an unmerged branch.* Reconcile the work or the name; don't force.
- *Slipping a code change into the wrap commit.* If the change is real, it's a milestone or a `wf-patch`.
- *Skipping the ADR harvest.* The window to record "why we did it this way" closes when the team forgets.
- *Writing the CHANGELOG entry fresh at step 6.* It already exists — `wrap.md`'s `## Changelog entry`, written at step 1. Copy it; if it reads wrong, fix the source and copy again.
- *Pushing before approval.*
- *Merging without `--no-commit`.* Produces an untrailered merge commit; the kernel rule fires once per entity file touched.
- *Hardcoding `<id>` in the actor trailer.* Resolve from `git config user.email` at run time per the provenance model.
- *Promoting the epic to `done` before the wrap-artefact commit.* Ends the authorize scope mid-bundle; subsequent verb-driven commits carry an ended-scope `aiwf-authorized-by:` and fail `provenance-authorization-ended` on push. Promote is step 7, after the wrap-artefact commit — the "Why promote is last among verb-driven commits" section above explains why.
- *Merging before the promote.* Leaves the integration target carrying the merge without the status flip, so the epic reads `active` on mainline and the roadmap regen has nothing to reflect. The merge is step 9, last.
- *Resolving a mainline conflict on mainline itself, mid-merge.* If mainline has advanced past the epic branch's fork point, reconcile on the epic branch (step 5) and re-run the gate there — mainline only ever receives an already-validated result.

## Out of scope

Version-tag cuts, the `[Unreleased]` → `[X.Y.Z]` rename, package publishing, and deployment. Those belong to `aiwfx-release`.

**Note:** *Adding* the per-epic entry under `## [Unreleased]` in `CHANGELOG.md` is **in scope** for this skill — as a copy of `wrap.md`'s `## Changelog entry` (step 6), never as fresh prose. The `[Unreleased]` heading is the per-epic accumulator; `aiwfx-release` folds it into a version section when cutting a release. Skipping the CHANGELOG step at wrap is the failure mode that produces empty release notes — this skill owns prevention.

## Next step

If a release follows: → `aiwfx-release`.
If not: → `aiwfx-plan-epic` for whatever's next.
