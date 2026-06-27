---
id: E-0046
title: Formalize in-repo worktrees as the default placement
status: active
---

# E-0046 — Formalize in-repo worktrees as the default placement

## Goal

Make in-repo worktrees (`.claude/worktrees/<branch>/`) aiwf's default placement for
ritual worktrees, so a Claude session inside a sandboxed devcontainer can root in its
worktree — and record the non-obvious rationale so the default is not reverted to the
sibling-worktree git convention.

## Context

The rituals (`aiwfx-start-epic`, `aiwfx-start-milestone`) create a git worktree for
implementation work and currently offer three placements as a free choice with no
default: switch branches in the main checkout, an in-repo worktree under
`.claude/worktrees/`, or a sibling directory next to the repo.

This session established empirically that a Claude Code session running in a devcontainer
is **sandbox-confined to the workspace folder**: a `cd` into a sibling worktree (outside
the workspace) is reset back to the workspace root, while a `cd` into an in-repo worktree
(a subdirectory of the workspace) persists — and the statusline correctly follows the
session into it, with Claude Code even populating the `workspace.git_worktree` field. So
in a container:

- **Sibling worktrees are unreachable** as a session's working directory; a session can
  never root in one, so surfaces that derive context from the cwd (the statusline's
  active-entity, branch, CI) cannot reflect the work.
- **`$HOME`-placed worktrees are wiped on container rebuild** — `$HOME` is typically not a
  persistent mount — a real lost-work hazard.
- **In-repo worktrees are reachable, persistent, and gitignored** (`.claude/*`), so they
  neither clutter the parent directory nor live at the mercy of an ephemeral mount.

The CLAUDE.md "Subagent worktree isolation" section already names `.claude/worktrees/<name>`
for transient agent worktrees. This epic promotes in-repo to the documented default for
session/epic worktrees, makes it a configurable default, and records the reasoning.

## Scope

### In scope

- An ADR recording the decision (in-repo is the default) and the devcontainer-sandbox
  rationale.
- A `worktree.dir` key in `aiwf.yaml` (default `.claude/worktrees`) giving a project a
  persistent placement default.
- The rituals (`aiwfx-start-epic` / `aiwfx-start-milestone`) read the knob and default to
  in-repo placement, with rationale inline; the per-invocation placement override is
  retained.
- A loader-guard regression test pinning that `aiwf check` / the loader ignores
  `.claude/worktrees/`, so a nested in-repo checkout cannot surface phantom duplicate
  entities.

### Out of scope

- The aiwf-aware statusline HUD and health surfaces — branch-derived active-entity
  display, active-only epic filter, the `health.aiwf.json` writer, the glob-reader warning
  triangle. Tracked as a separate epic.
- The worktree-race / allocator cluster: G-0269 (HEAD-drift guard), G-0272
  (sibling-worktree id allocation), G-0277 (status staleness vs unmerged worktree), G-0157
  (status worktree batching).
- Subagent worktree isolation (G-0099) and the `start-epic` worktree-vs-promote sequencing
  question (G-0116) — adjacent; may be brushed when editing `start-epic`, but are not
  deliverables here.

## Constraints

- Placement is **config-driven, not hardcoded**: the kernel default is `.claude/worktrees`,
  overridable via `aiwf.yaml worktree.dir`. The correct placement is environment-dependent
  (in-repo for sandboxed devcontainers; siblings acceptable on a bare host), so a defaulted
  knob — not a hardcode.
- The rituals retain the per-invocation placement override; the knob sets the *default*,
  not a lock.
- The loader's exclusion of `.claude/worktrees/` must be a **pinned regression test**, not
  an assumption — the in-repo default is only safe if `aiwf check` provably never descends
  into a nested checkout.
- The ADR follows the repo's ADR discipline: it records the decision and rationale only,
  with no gate or schedule language in the body.

## Success criteria

- [ ] Running the start rituals defaults to creating the worktree under the configured
  `worktree.dir` (in-repo), with the sibling and main-checkout options still selectable
  per invocation.
- [ ] `aiwf.yaml worktree.dir` is honored: setting it relocates the ritual default; leaving
  it unset falls back to `.claude/worktrees`.
- [ ] `aiwf check` reports no phantom or duplicate-entity findings when an in-repo worktree
  holding a full second checkout exists under `.claude/worktrees/` — pinned by a regression
  test.
- [ ] The decision and its devcontainer-sandbox rationale are captured in an accepted ADR
  (see *ADRs produced*).

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the loader already ignore `.claude/worktrees/`, or is an explicit exclusion needed? | no | The loader-guard milestone verifies against a fixture with a nested in-repo checkout, then pins the result either way. |
| Should `worktree.dir` accept an absolute path or multiple roots, or only a repo-relative directory? | no | The config-knob milestone starts with a single repo-relative directory (YAGNI); revisit if a consumer needs more. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The loader actually descends into `.claude/worktrees/` today, so flipping the default would break `aiwf check`. | high | The loader-guard milestone is sequenced first and verifies/pins the behavior before the ritual default flips. |
| Concurrent sessions sharing the main checkout race the planning commits. | low | Trunk-based atomic commits; `aiwf reallocate` resolves any id collision at merge. |

## Milestones

- [M-0188](work/epics/E-0046-formalize-in-repo-worktrees-as-the-default-placement/M-0188-pin-that-the-loader-ignores-in-repo-worktrees-under-claude-worktrees.md) — pin that the loader ignores in-repo worktrees under `.claude/worktrees`. · depends on: —
- [M-0189](work/epics/E-0046-formalize-in-repo-worktrees-as-the-default-placement/M-0189-add-worktree-dir-config-knob-defaulting-to-claude-worktrees.md) — add the `worktree.dir` config knob, default `.claude/worktrees`. · depends on: —
- [M-0190](work/epics/E-0046-formalize-in-repo-worktrees-as-the-default-placement/M-0190-default-the-start-rituals-to-in-repo-worktree-placement.md) — default the start rituals to in-repo placement, reading the knob. · depends on: M-0189

## ADRs produced

- ADR-0023 — Default to in-repo worktree placement under `.claude/worktrees`.

## References

- CLAUDE.md — "Subagent worktree isolation" (`.claude/worktrees/<name>` precedent) and
  "Worktree binary discipline".
- `aiwfx-start-epic` SKILL.md — current three-way worktree placement Q&A.
- Adjacent gaps: G-0116, G-0099, G-0269, G-0272, G-0277, G-0157.
