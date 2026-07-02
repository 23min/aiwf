# Epic wrap — E-0055

**Date:** 2026-07-02
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0055-health-as-install-status-producer-health-files-statusline-stoplight
**Merge commit:** f6081400

## Milestones delivered

- M-0224 — aiwf health: doctor writes health.aiwf.json + statusline stoplight (merged b8f52c11)

## Summary

Gives operators visibility of aiwf's installation and configuration warnings and errors in
the Claude Code statusline. `aiwf doctor` now surfaces its warnings and errors as structured
problems and writes them to `.claude/health.aiwf.json` (the fixed ai-dotfiles per-producer
schema), refreshed on `aiwf update`; the statusline reads and unions `.claude/health.*.json`
and renders an always-visible four-state stoplight (green `●` / yellow `▲` / red `▲` / gray
`●`) at the maximum severity, never running a check on the render path. This corrects the
health axis from the shipped tree-check glyph (G-0290 / M-0193) to installation and
configuration health, and makes the healthy state visible.

Scope shifted mid-flight: an initial single-source-of-truth rewrite of the entire doctor
report was over-built and reverted in favour of an additive `problems int → []Problem`
thread with byte-identical human output; the `aiwf doctor --format=json` envelope was cut as
out of scope. The three planned milestones collapsed to one (M-0225 and M-0226 were folded
in and cancelled).

## ADRs ratified

- ADR-0026 — aiwf health is installation status, surfaced via producer health files

## Decisions captured

- none (the reframe is recorded in ADR-0026)

## Follow-ups carried forward

- G-0344 — statusline: version-stamp + upgrade-only auto-refresh on plain aiwf update

## Handoff

The health *data* path is worktree-safe (main-checkout resolution). The statusline script
itself is user-scope by default — worktree-safe, but host-shared under the common `~/.claude`
bind mount; G-0344 captures the follow-up to make that shared artefact versioned and
self-updating. `aiwf doctor --format=json` (G-0070) remains open and unrelated.
