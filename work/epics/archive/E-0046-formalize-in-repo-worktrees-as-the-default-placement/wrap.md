# Epic wrap — E-0046

**Date:** 2026-06-28
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0046-formalize-in-repo-worktrees-as-the-default-placement
**Merge commit:** 0ba3fb29

## Milestones delivered

- M-0188 — Pin that the loader ignores in-repo worktrees under .claude/worktrees (merged 790bc55c)
- M-0189 — Add worktree.dir config knob defaulting to .claude/worktrees (merged 70c8c9b5)
- M-0190 — Default the start rituals to in-repo worktree placement (merged 7f35f6e5)

## Summary

Formalizes in-repo worktrees under `.claude/worktrees/<branch>/` as aiwf's default placement
for ritual worktrees, overriding the near-universal sibling-worktree git convention. M-0188
pins (with a nested-checkout regression fixture) that the loader and `aiwf check` never descend
into `.claude/worktrees/`, so a nested in-repo checkout cannot surface phantom duplicate
entities — de-risking the default flip. M-0189 adds the `worktree.dir` knob to `aiwf.yaml`
(default `.claude/worktrees`) with a nil-tolerant getter and a greppable `aiwf doctor`
`worktree-dir:` line the rituals read. M-0190 flips `aiwfx-start-epic` / `aiwfx-start-milestone`
to recommend in-repo placement as the default while retaining the per-invocation override, and
hardens `config.WorktreeDir()` to reject a repo-escaping `worktree.dir` so the value the rituals
consume can never place a worktree outside the repo. The motivating rationale — a sandboxed
devcontainer session can only root its cwd in a worktree under the mounted workspace, and
`$HOME`-placed worktrees are wiped on container rebuild — was proven empirically this session and
is recorded in ADR-0023 and inline in both rituals.

## ADRs ratified

- ADR-0023 — default-to-in-repo-worktree-placement-under-claude-worktrees (accepted during the epic)

## Decisions captured

- none as standalone D-NNN entries — the milestone-local decisions (e.g. AC-4's escape check lives
  in the `WorktreeDir()` getter, not a markdown use-site) are recorded in each milestone spec's
  "Decisions made during implementation" section.

## Doc findings

Clean. `wf-doc-lint`-equivalent scan of the epic change-set (every file touched on the epic branch
since it diverged from `main`): no broken code references, no removed-feature docs, no TODO/FIXME
in shipped prose. The ADR-0023 links in both rituals resolve to the accepted ADR.

## Follow-ups carried forward

- G-0293 — Promote tdd_phase live, not in a burst at milestone wrap (discovered in M-0189; a
  methodology gap on the red→green→done ladder being stamped at wrap rather than contemporaneously;
  open).

## Handoff

In-repo worktree placement is now the documented, mechanically-guarded default: the knob, its
`aiwf doctor` surface, the loader guard, the escape rejection, and the ritual default-flip all
landed and are pinned by tests. Ready for the next epic. Deliberately left open: G-0293 (live
tdd_phase promotion) is a separate methodology concern, not in this epic's scope.
