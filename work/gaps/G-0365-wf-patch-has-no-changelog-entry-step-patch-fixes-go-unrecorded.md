---
id: G-0365
title: wf-patch has no CHANGELOG entry step; patch fixes go unrecorded
status: addressed
addressed_by_commit:
    - 3406e50a
---
## Problem

`wf-patch`'s `SKILL.md` has no step that adds a `CHANGELOG.md` entry under
`## [Unreleased]` — grepping the skill body for "changelog" returns zero
hits. Whether a patch's user-visible delta ever gets recorded depends
entirely on someone adding the entry by hand, separately from the ritual,
after the fact.

This is confirmed, not theoretical: of the 16 `patch/*` branches merged in
this repo's own history, several genuinely user-visible fixes have zero
`CHANGELOG.md` entry. Checked directly against the five most recent patches
(all merged 2026-07-05, after the `v0.24.1` tag): G-0363 (drops a stray
`completed:` key from the shipped epic-spec template), G-0361 (routes
release-cutting intent to the deployer agent), G-0352 (fixes the shipped
statusline script's token-count source), G-0332 (fixes `aiwf status`
worktree-altitude rendering), and G-0350 (decouples `roadmap --write` from
committing) — all five are genuine, shipped, user-visible behavior changes,
and none appear anywhere in `CHANGELOG.md`. `[Unreleased]` is currently
empty despite this activity.

There's direct evidence this has lapsed before: one of the 16 patch merges
in history is `patch/changelog-retrofill`, an explicit catch-up commit
backfilling `[Unreleased]` entries for several already-wrapped epics.

## Why it matters

`aiwfx-wrap-epic` already has this exact step, working, as precedent (step
7: "Wrap-artefact commit — CHANGELOG `[Unreleased]` + `wrap.md`"), and is
explicit about the failure mode: "Skipping the CHANGELOG-update step at
wrap is the failure mode that produces empty release notes — this skill
owns prevention." `aiwfx-release`'s own skill only *moves* whatever's
already sitting under `[Unreleased]` into the new version heading — it
never synthesizes entries from patch or gap history. So without a mandated
step somewhere, a release can ship real, user-visible changes with zero
changelog record, silently, and nothing catches it.

`wf-patch` is the sharpest case: unlike a milestone (whose changes can
plausibly roll up into its parent epic's one changelog entry at
`aiwfx-wrap-epic` time — worth confirming that's actually the intended
design rather than the same gap), a patch has no parent epic to roll up
into. Its wrap is the only wrap that will ever happen for that change. If
`wf-patch` doesn't record it, nothing ever will short of manual retrofill.

## Direction (not prescribed)

Add a changelog-entry step to `wf-patch`'s wrap sequence, modeled directly
on `aiwfx-wrap-epic` step 7: a `### <Added|Changed|Fixed> — G-NNNN:
<one-line summary>` heading under `## [Unreleased]`, with a short paragraph
distilling the user-visible delta (internal-only patches may need a
narrower "state so and note nothing user-facing changed" allowance, or may
legitimately skip it — worth deciding explicitly rather than defaulting to
always-required). Gate it the same way the rest of the wrap's local,
reversible changes are gated today — part of the same reviewable diff
before the wrap commit, not a separate approval.

## Provenance

Found 2026-07-05 investigating why `ROADMAP.md` shows no unreleased
patches that closed gaps (see the sibling gap on `ROADMAP.md`'s epic-only
render scope) — checking whether `CHANGELOG.md` at least covered the same
ground surfaced that it doesn't either, reliably.
