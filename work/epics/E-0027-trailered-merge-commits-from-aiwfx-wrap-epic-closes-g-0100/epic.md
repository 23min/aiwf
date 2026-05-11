---
id: E-0027
title: Trailered merge commits from aiwfx-wrap-epic (closes G-0100)
status: proposed
---

# E-0027 — Trailered merge commits from `aiwfx-wrap-epic`

## Goal

Change `aiwfx-wrap-epic`'s merge step so the merge commit it produces carries `aiwf-verb: wrap-epic`, `aiwf-entity: E-NNNN`, and `aiwf-actor: human/<id>` trailers. The merge commit becomes self-describing — `aiwf history E-NNNN` surfaces the merge event — and the kernel's `provenance-untrailered-entity-commit` rule stays strict, with the ritual now aligned to pass it cleanly.

## Context

E-0024 and E-0026 wrapped during the M-0089 session. Each ran `aiwfx-wrap-epic`, which executed `git merge --no-ff <branch>` and produced an untrailered merge commit. The kernel's `provenance-untrailered-entity-commit` rule fires once per entity file touched by each untrailered merge — currently 4 instances on this tree, and the count will grow by ~4 on every future epic wrap.

The original gap (G-0100) framed this as "rule should skip merge commits." On reflection, that route would hide a real signal: any non-ritual merge that hand-edits an entity file would also pass silently. The cause-side fix is to align the ritual with the kernel rule — bring the merge commit into trailer compliance — so the chokepoint keeps catching regressions while the ritual stops producing them.

`aiwfx-wrap-epic` lives in the `ai-workflow-rituals` plugin repo at `/Users/peterbru/Projects/ai-workflow-rituals/` (distributed via the Claude Code marketplace). Per CLAUDE.md *Cross-repo plugin testing*, the canonical authoring location during the milestone is a fixture in this repo under `internal/policies/testdata/<skill-name>/SKILL.md`; AC tests under `internal/policies/` assert content claims against the fixture. At wrap, the fixture content is copied to the rituals repo as a separate commit there.

## Scope

### In scope

- Update `aiwfx-wrap-epic`'s SKILL.md (in the fixture authoring location) so the merge instructions read as: `git merge --no-ff --no-commit <branch>` followed by `git commit -m "<merge subject>" --trailer "aiwf-verb: wrap-epic" --trailer "aiwf-entity: E-NNNN" --trailer "aiwf-actor: human/<id>"` (substituting concrete identity at run time).
- A drift-check test in `internal/policies/` that asserts the fixture body names the trailered-commit sequence structurally (per CLAUDE.md *Substring assertions are not structural assertions*; walk the markdown section that documents the merge step).
- A drift-check test that compares the fixture against the local marketplace cache (`~/.claude/plugins/cache/ai-workflow-rituals/.../SKILL.md`) per the M-0079 precedent — skips cleanly when the cache is absent.
- At wrap: copy the fixture content into the rituals repo at `aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md`; record the rituals-repo commit SHA in the milestone's *Validation* section.

### Out of scope

- Backfilling historical untrailered merges (E-0024, E-0026). Per CLAUDE.md *no backwards-compat hacks*. The 4 existing warnings stay as advisory artifacts. The cause-side fix prevents new instances; the historical 4 are accepted.
- Other ritual surfaces (`aiwfx-wrap-milestone`, `aiwfx-plan-epic`, etc.). `aiwfx-wrap-milestone` doesn't produce a merge commit; it commits the milestone spec edits directly via verb commits that are already trailered. If audit later reveals other untrailered ritual outputs, file separate gaps.
- Loosening the kernel rule. The rule stays strict.
- `git config` changes (e.g., setting `merge.commit` templates). The ritual instruction is sufficient; system-wide config is out of scope.
- Test of a real epic wrap end-to-end. The kernel rule is the runtime chokepoint; if the ritual breaks again, the rule catches it on the next push. A dedicated end-to-end test of the wrap ritual is overkill for one merge invocation.

## Constraints

- **Kernel rule unchanged.** The `provenance-untrailered-entity-commit` rule stays strict; this epic does not modify `internal/policies/` rule logic beyond adding the drift-check test for the fixture.
- **No history rewrite.** The 4 historical warnings remain; no rebases, no `git filter-repo`.
- **Trailer keys verbatim from CLAUDE.md "Commit conventions".** `aiwf-verb`, `aiwf-entity`, `aiwf-actor`, formatted exactly as the trailer-keys policy expects.
- **Cross-repo workflow per CLAUDE.md.** Fixture-first authoring; tests in this repo; copy to rituals repo at wrap; record SHA in *Validation*.

## Success criteria

- [ ] `aiwfx-wrap-epic` SKILL.md (fixture) names the `git merge --no-ff --no-commit` + `git commit --trailer ...` sequence structurally in its merge-step section.
- [ ] Drift-check test in `internal/policies/` walks the fixture's merge-step section and asserts the trailer-emitting instructions are present in the right place (not flat substring grep).
- [ ] Drift-check test compares fixture vs. local marketplace cache and either passes or skips (cache absent); fails only on real divergence.
- [ ] Rituals-repo SHA is recorded in the milestone's *Validation* section.
- [ ] When the **next** epic is wrapped via `aiwfx-wrap-epic`, the merge commit carries the three required trailers and produces zero new `provenance-untrailered-entity-commit` findings against that merge. This is observed post-wrap, not pre-tested.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the rituals plugin's current fixture for `aiwfx-wrap-epic` already exist in `internal/policies/testdata/`? If not, this milestone introduces the first one — minor extra scaffolding. | no | Check during milestone start; scaffold if absent. |
| Should the merge commit's subject follow Conventional Commits (e.g., `chore(epic): wrap E-NNNN`)? CLAUDE.md "Commit conventions" calls for Conventional Commits on subject lines. | no | Yes, follow Conventional Commits. Resolve in the SKILL.md instruction during the milestone — `chore(epic): wrap E-NNNN — <title>` is the natural shape. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Operator (or LLM agent) running the ritual under non-default git config (`merge.ff = only`, custom merge driver, etc.) and the `--no-ff --no-commit` sequence behaves unexpectedly. | low | The kernel rule catches the regression at the next push; documented merge edge cases per ADR-0004 §"Merge edge cases" remain the operator's playbook. |
| Future rituals plugin upstream change that re-introduces untrailered merges. | low | The drift-check test in `internal/policies/` is the canary — compares fixture vs. cache. Fixture and rituals-repo content must stay in sync; the test fires if they drift. |

## Milestones

- [M-NNNN](work/epics/E-0027-trailered-merge-commits-from-aiwfx-wrap-epic-closes-g-0100/M-NNNN-...md) — `aiwfx-wrap-epic` SKILL.md fixture emits trailered-merge instructions; drift-check tests · depends on: —

## References

- G-0100 — gap this epic closes.
- M-0089 — surfaced the friction observably (per-code summary made the 4 warnings scannable).
- E-0024 wrap commit (`31c68fb`), E-0026 wrap commit (`fa2abf2`) — the historical untrailered merges this epic prevents from recurring.
- CLAUDE.md "Commit conventions" — `aiwf-verb` / `aiwf-entity` / `aiwf-actor` trailer specification.
- CLAUDE.md "Cross-repo plugin testing" — the fixture-first authoring + drift-check pattern this epic follows.
- M-0079 — precedent for embedded-skill drift-check tests in `internal/policies/`.
- `internal/policies/` — where the untrailered-entity audit and the new drift-check test live.
- `aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md` in the rituals repo — the upstream content this epic synchronizes via the fixture.
