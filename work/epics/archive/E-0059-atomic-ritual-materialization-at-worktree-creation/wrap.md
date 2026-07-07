# Epic wrap — E-0059

**Date:** 2026-07-06
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0059-atomic-ritual-materialization-at-worktree-creation
**Merge commit:** f5106cc19d4dade8a9fadb64330609cb44c346aa

## Milestones delivered

- M-0233 — aiwf worktree add verb: atomic creation with ritual materialization (merged 17c7cdc1)
- M-0234 — Rewire aiwf rituals and CLAUDE.md to use aiwf worktree add (merged 8c74c1c6)
- M-0235 — Generalized hook registry: aiwf.yaml-declared, persisted consent (merged 6902a8df)
- M-0236 — Ship the worktree-materialization-check SessionStart hook (merged bcf343b1)

## Summary

A freshly-cut git worktree now carries the same materialized `.claude/skills/`,
`.claude/agents/`, `.claude/templates/`, and `.claude/aiwf-guidance.md` as the main
checkout, atomically, via `aiwf worktree add` (M-0233), with every ritual call site in
this repo rewired to it (M-0234). A generalized, consent-gated hook registry was added
to `aiwf.yaml` (M-0235), and its first concrete consumer — a `SessionStart`/
`SubagentStart` hook that warns (without blocking) when a session starts inside a
`.claude/worktrees/` checkout whose rituals aren't materialized — closes the
session-level backstop the epic's goal called for (M-0236). Together these close the
gap where ritual discipline (TDD, vacuity, rethink, gate rules) could silently vanish
just because work happened to be isolated in a worktree.

## ADRs ratified

- ADR-0032 — Materialized hook consent: persisted per-hook aiwf.yaml registry

## Decisions captured

- none

## Follow-ups carried forward

- G-0099 — Worktree isolation must be a parent-side precondition (pre-existing;
  the `isolation: "worktree"` kwarg's own resolution remains pinned to ADR-0009 /
  E-0019, unaffected by this epic).
- Migrating `.claude/hooks/validate-agent-isolation.sh` into the new hook registry
  (noted as a follow-up gap in M-0236's spec `## Out of scope`; not yet filed as its
  own gap entity — the sibling hook keeps working exactly as before, unaffected by
  the new registry's existence).

## Doc findings

Scoped to the epic's full change-set (62 files since the fork point from `main`).

- **Broken code references:** none. Every backticked `aiwf <verb>` reference in the touched `SKILL.md` files (`aiwfx-start-epic`, `aiwfx-start-milestone`, `wf-patch`, `aiwf-worktree`) resolves to a real, current Cobra command — spot-checked `aiwf worktree add --help` directly.
- **Removed-feature docs:** none.
- **Orphan files:** none. The one file the diff shows as deleted (`M-0235-session-start-hook-flags-worktrees-missing-materialized-rituals.md`) is a legitimate `aiwf retitle` rename artifact (confirmed via `git log --follow`), not an orphaned or abandoned doc.
- **Documentation TODOs:** none found across the non-markdown changed files (grepped for `TODO`/`FIXME`/`XXX`).

Clean — 0 findings.

## Handoff

The hook registry (`skills.ShippedHooks`, `HookDef.Events`, `MaterializeHooks`/
`WireHookSettings`/`UnwireHookSettings`, `HookDrift`) is now real, tested
infrastructure with one concrete entry — a future hook is a registry addition plus
its own script, not new plumbing. `aiwf worktree add` is the one true path for
creating a worktree with live rituals; nothing in this epic depended on the
E-0019/ADR-0009 substrate rewrite, so that dependency remains exactly as deferred as
it was before this epic started.
