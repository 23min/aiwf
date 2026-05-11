---
id: M-0104
title: aiwfx-start-epic sequencing fix (closes G-0116)
status: draft
parent: E-0030
depends_on:
    - M-0102
    - M-0103
tdd: required
---

## Goal

Reorder `aiwfx-start-epic` so the sovereign promote (`aiwf promote E-NN active`) and authorize (`aiwf authorize E-NN --to ai/<id> --branch epic/E-NN-<slug>`) commits fire on `main` *before* the worktree/branch is cut. Closes [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md).

## Context

G-0116 documented the sequencing inversion in today's `aiwfx-start-epic`: step 5 (worktree placement) precedes step 8 (sovereign promote) and step 9 (optional authorize). With M-0103's preflight active, the existing ordering would *fail* — the worktree-first cut hits the preflight before any ritual branch context exists.

This milestone fixes the ordering so the ritual works with the chokepoint. It also implements [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s sequencing rule for opening an epic: state-announcement commits on main, *then* branch cut, *then* implementation work on the branch.

Cross-repo via the fixture pattern (CLAUDE.md § "Cross-repo plugin testing"). The canonical authoring location is `internal/policies/testdata/aiwfx-start-epic/SKILL.md` in this repo; the rituals-repo commit lands separately at wrap.

## Out of scope

- `aiwfx-start-milestone` (M-0105 — symmetric fix one level down).
- Kernel finding for post-hoc detection (M-0106).
- Other ritual surfaces beyond `aiwfx-start-epic`.
- G-0059's open ladder steps beyond what this milestone needs.

## Dependencies

- **M-0102** — the `--branch` flag the new ordering invokes.
- **M-0103** — the preflight that makes the ordering necessary.

## Open questions for AC drafting

- **Step placement in the new sequence:** Where exactly do promote + authorize land — preflight (steps 1–4), or interleaved with the conversation about branch policy (steps 5–6)? Tentatively: after the preflight conversation confirms trunk-based policy, run promote + authorize, then cut the worktree.
- **G-0116's "lighter alternative":** Should we also surface trunk-vs-PR choice as an explicit Q&A step earlier in the flow, or unconditionally apply the new order? G-0059's resolution doesn't pick this; this milestone can decide for `aiwfx-start-epic` specifically.
- **Rituals-repo commit gate:** Land the rituals-side commit before or after this milestone wraps? Default per the fixture pattern: at milestone wrap, rituals-repo commit's SHA goes into *Validation*.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0104` time. -->
