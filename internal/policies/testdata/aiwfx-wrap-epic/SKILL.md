---
name: aiwfx-wrap-epic
description: Closes an aiwf epic — verifies all milestones done, scaffolds a wrap artefact, harvests ADR candidates, runs scoped doc-lint, merges the epic branch into mainline with a trailered merge commit, promotes the epic to done. Use when the user says "wrap E-NN" or "close the auth epic" and every milestone in the epic is wrapped. Commit and push require explicit human approval.
---

# aiwfx-wrap-epic

Closes an epic. The epic itself is a coordination unit — closing it means: every milestone is `done`, the integration branch merges to mainline via a trailered merge commit, the wrap artefact captures what shipped and what didn't, and the epic's status flips to `done`.

## Principles

- **Wrap is closure, not release.** Tagging, packaging, publishing — those are `aiwfx-release`. This skill ends the planning unit.
- **Branch cleanup is opt-in.** Local branches are preserved (so `tig` / `gitk` keep labelling history); origin branches for completed milestones are deleted to reduce remote refname clutter.
- **Nothing is deleted at wrap.** Specs, tracking docs, the wrap artefact — all stay readable forever. Closure is a status change, not a deletion.
- **The merge commit is trailered.** The integration-target merge commit carries `aiwf-verb: wrap-epic`, `aiwf-entity: E-NNNN`, `aiwf-actor: human/<id>` trailers — exactly the keys the kernel's `provenance-untrailered-entity-commit` finding expects. Without the trailers, the rule fires once per entity file touched by the merge.

## Precondition

1. Every milestone in this epic has `status: done`. Run `aiwf check` and verify; if any are still `in_progress` or `draft`, stop and surface them.
2. The epic branch (if used) is up to date — every milestone's final merge commit is on it.
3. Working tree clean.
4. Integration target identified (usually `main`).

If any precondition fails, stop and report. Do not improvise around an unfinished epic.

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

Create `work/epics/E-NN-<slug>/wrap.md` (staged, not yet committed):

```markdown
# Epic wrap — E-NN

**Date:** <today>
**Closed by:** <actor>
**Integration target:** main
**Epic branch:** epic/E-NN-<slug>
**Merge commit:** <SHA — filled at step 5>

## Milestones delivered

- M-NNN — <title> (merged <short-sha>)
- M-NNN — <title> (merged <short-sha>)

## Summary

Two to four sentences on what shipped and why. Reference the goal from the epic spec; honest about what scope shifted mid-flight.

## ADRs ratified

- ADR-NNNN — <slug>          (or "none")

## Decisions captured

- D-NNN — <slug>             (or "none")

## Follow-ups carried forward

- G-NNN — <slug>             (gap that survives the epic)

## Handoff

What is ready for the next epic; what is deliberately left open.
```

Use **reference-phrasing for any list-derived count** ("every ADR listed in *ADRs ratified*" rather than "all 4 ADRs"). Avoids drift.

### 2. ADR check — harvest decisions worth keeping

Walk the epic's commits. For each candidate decision, ask: *"Would a future reader regret missing the reasoning?"* Signals an ADR is warranted:

- A default changed or a new default introduced.
- A strategy considered and rejected.
- A scope cut or framing shift affecting downstream work.
- A supersession of a prior ADR.

For each candidate, invoke `aiwfx-record-decision` and choose ADR (architectural, durable) or D-NNN (project-scoped, more local). Record the resulting ids in the wrap artefact's `## ADRs ratified` or `## Decisions captured` section.

### 3. Doc-lint sweep (scoped)

Invoke `wf-doc-lint` against the epic's change-set (every file touched on `epic/E-NN-<slug>` since it diverged from the integration target).

Append the report to `wrap.md` under a `## Doc findings` section. If findings include broken references or removed-feature docs, fix or open as gaps before proceeding. `wf-doc-lint` reports only — prose fixes are deliberate edits here.

### 4. 🛑 Merge gate — merge epic branch into integration target with a trailered merge commit

Confirm with the user. Then:

```bash
git checkout main
git pull --ff-only origin main
```

Stage the merge **without committing** so the next step can attach the required trailers explicitly:

```bash
git merge --no-ff --no-commit epic/E-NN-<slug>
```

`--no-ff` preserves the epic as a single merge commit; `--no-commit` leaves the merge staged so the commit-emitting step is the one carrying trailers. Without `--no-commit`, git produces an untrailered merge commit and the kernel's `provenance-untrailered-entity-commit` rule fires once per entity file touched by the merge.

Resolve the operator identity from `git config user.email` (per CLAUDE.md *Provenance model* §"Identity is runtime-derived"); do not hardcode `<id>`. Then commit with the three required trailers and a Conventional Commits subject:

```bash
git commit -m "chore(epic): wrap E-NNNN — <epic title>" \
  --trailer "aiwf-verb: wrap-epic" \
  --trailer "aiwf-entity: E-NNNN" \
  --trailer "aiwf-actor: human/<id>"
```

The trailer keys are quoted from CLAUDE.md §"Commit conventions" verbatim — `aiwf-verb`, `aiwf-entity`, `aiwf-actor`. Variant casings (e.g. `Aiwf-Verb`) fail the kernel's trailer-keys policy. Record the resulting merge SHA in `wrap.md`.

**Do not push yet.**

### 5. Update `CHANGELOG.md` `[Unreleased]` and stage all wrap artefacts

The `[Unreleased]` section of `CHANGELOG.md` is a per-epic accumulator: every wrapped epic adds an entry here, and `aiwfx-release` later rolls the accumulated entries into a versioned `## [X.Y.Z]` heading. *Without this step, releases ship with empty changelog entries* — that's the `[Unreleased]` drift this step prevents.

Edit `CHANGELOG.md` to add a new sub-section under `## [Unreleased]`. Use a Keep-a-Changelog category as the heading: `### Added — E-NN: <one-line summary>`, `### Changed — E-NN: <one-line summary>`, or `### Fixed — E-NN: <one-line summary>` as appropriate. The body is a short paragraph (or bulleted milestone list, like prior epic entries in the file) summarising the **user-visible delta**: gaps closed, verbs added, behaviour changes, doctrine landed in `CLAUDE.md`. Internal refactors with no observable delta can be omitted; if everything is internal, a single line saying so still goes in (releases require *some* entry per the changelog-check workflow).

The `wrap.md` file already captures the structured detail (milestones, ADRs, gaps); the CHANGELOG entry distils it for a release reader who has not been following along. Reference-phrasing is fine ("every milestone listed in `wrap.md` …") to avoid drift between the two documents.

Then stage everything for the wrap commit:

```bash
git add CHANGELOG.md
git add work/epics/E-NN-<slug>/wrap.md
git status
```

### 6. 🛑 Commit gate

Show the user:
- The wrap artefact summary.
- `git diff --staged`.
- The proposed wrap-artefact commit message: `chore(E-NN): wrap epic — <one-line summary>` plus the same three trailers.

**Stop and wait for "commit" approval.**

### 7. After commit approval

The wrap-artefact commit (the CHANGELOG + `wrap.md` addition) is a normal mutating commit on top of the trailered merge — it carries the same three trailer keys so `aiwf history E-NNNN` surfaces it alongside the merge:

```bash
git commit -m "<approved-message>" \
  --trailer "aiwf-verb: wrap-epic" \
  --trailer "aiwf-entity: E-NNNN" \
  --trailer "aiwf-actor: human/<id>"
```

### 8. Promote the epic to `done` — last commit in the wrap bundle

```bash
aiwf promote E-NN done
```

aiwf validates `active → done`, rewrites frontmatter, commits with `aiwf-verb: promote`. (If the epic is still `proposed`, that means no milestone ever started — wrap doesn't apply. Investigate.)

**Why promote is last (closes G-0119).** The `aiwf promote E-NN done` commit ends the authorize scope that opened with `aiwfx-start-epic`. Any commit produced *after* this — wrap artefact, CHANGELOG entry, reallocates, or other wrap-bundle commits — would carry `aiwf-authorized-by:` referencing the just-ended scope and trigger the kernel's `provenance-authorization-ended` finding on push, blocking the wrap with no clean remediation short of `--no-verify` or history rewrite. Keeping `aiwf promote E-NN done` as the last commit in the wrap bundle guarantees every other wrap commit lives under the live scope, and the scope-ending promote is itself the natural last act before the push gate.

The completion date is recorded in `wrap.md` (step 1) and is recoverable from the `aiwf-verb: promote` commit via `aiwf history E-NN`. Do not add a `completed:` field to the epic frontmatter — aiwf's epic schema does not include it, and the parse failure cascades into unresolved-reference findings on every entity that links to this epic.

### 9. 🛑 Push gate

Confirm. Then:

```bash
git push origin main
```

### 10. Branch cleanup (origin only)

Plan the deletions first. List the milestone and epic branches to delete. For each, verify it's merged:

```bash
git branch -r --merged main | grep "milestone/M-NNN"
git branch -r --merged main | grep "epic/E-NN"
```

If a branch isn't shown as merged, stop and report — don't force.

After explicit approval per branch (or batch approval for the full list), delete on origin only:

```bash
git push origin --delete milestone/M-NNN-<slug>
git push origin --delete epic/E-NN-<slug>
```

Local branches are not touched. Operators prune local branches on their own schedule.

### 11. Update the roadmap

```bash
aiwf render roadmap --write
```

### 12. Record learnings

Append to `work/agent-history/<agent>.md` (whoever closed the epic): patterns that worked across the epic, pitfalls, conventions established. 2–5 line entries. Roll up older history if the file exceeds ~200 lines.

## Constraints

- 🛑 **Never merge or push without explicit approval** (steps 4, 9, 10).
- 🛑 **The merge commit and the wrap-artefact commit both carry the three required trailers.** Skipping either is the regression the kernel's `provenance-untrailered-entity-commit` finding catches.
- 🛑 **`aiwf promote E-NN done` is the last commit in the wrap bundle** (step 8). It ends the active authorize scope; any commit produced after it carries an ended-scope `aiwf-authorized-by:` and fails the kernel's `provenance-authorization-ended` check on push. Closes G-0119.
- Every milestone must be `done` before wrap — `aiwf check` and `aiwf history E-NN` confirm.
- Branch-cleanup is origin-only. Do not delete local branches.
- The wrap artefact is mandatory. Don't close an epic without one.

## Anti-patterns

- *Wrapping while a milestone is still `in_progress`.* Run `aiwf check` first.
- *Force-deleting an unmerged branch.* Reconcile the work or the name; don't force.
- *Slipping a code change into the wrap commit.* If the change is real, it's a milestone or a `wf-patch`.
- *Skipping the ADR harvest.* The window to record "why we did it this way" closes when the team forgets.
- *Pushing before approval.*
- *Merging without `--no-commit`.* Produces an untrailered merge commit; the kernel rule fires once per entity file touched.
- *Hardcoding `<id>` in the actor trailer.* Resolve from `git config user.email` at run time per the provenance model.
- *Promoting the epic to `done` before the wrap-artefact and other wrap-bundle commits.* Ends the authorize scope mid-bundle; subsequent commits carry an ended-scope `aiwf-authorized-by:` and fail `provenance-authorization-ended` on push. Promote is step 8, after the wrap-artefact commit — see G-0119.

## Out of scope

Version-tag cuts, the `[Unreleased]` → `[X.Y.Z]` rename, package publishing, and deployment. Those belong to `aiwfx-release`.

**Note:** *Adding* the per-epic entry under `## [Unreleased]` in `CHANGELOG.md` is **in scope** for this skill (step 5). The `[Unreleased]` heading is the per-epic accumulator; `aiwfx-release` only rolls the accumulated entries forward when cutting a version. Skipping the CHANGELOG-update step at wrap is the failure mode that produces empty release notes — this skill owns prevention.

## Next step

If a release follows: → `aiwfx-release`.
If not: → `aiwfx-plan-epic` for whatever's next, or stop here.
