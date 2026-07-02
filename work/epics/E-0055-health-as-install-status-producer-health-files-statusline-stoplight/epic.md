---
id: E-0055
title: 'Health as install status: producer health files + statusline stoplight'
status: active
---
## Goal

Give operators visibility of `aiwf` installation and configuration warnings and errors
in the Claude Code statusline: an always-visible stoplight (gray / green / yellow / red)
fed by per-producer `.claude/health.*.json` files. `aiwf` writes its own health file
from `aiwf doctor`'s warnings and errors; the statusline reads and unions the health
files and shows the maximum severity — never running a check on the render path.

## Context

The shipped statusline health glyph (G-0290 / M-0193) runs `aiwf check --fast` at render
and shows a warning only on planning-*tree* errors, rendering nothing when healthy — the
wrong axis (tree drift, not installation health) and invisible in the common case.
ai-dotfiles settled a per-producer health-file model (G-0305): each producer writes one
`.claude/health.<source>.json` and a statusline globs and unions them. This epic adopts
that model and corrects the axis to installation and configuration — the `aiwf doctor`
domain.

## Scope

### In

- `aiwf doctor` surfaces its warnings and errors as structured problems and writes
  `.claude/health.aiwf.json` (the fixed ai-dotfiles schema).
- A four-state statusline stoplight that reads `.claude/health.*.json`, unions the
  findings, and renders gray / green / yellow / red at the maximum severity.

### Out

- Changing the ai-dotfiles schema (fixed, external).
- A producer registry — "no health file present" is a single global unknown (gray).
- An `aiwf doctor --format=json` envelope, a report-wide findings rewrite, or tree-drift
  signalling on the statusline. `aiwf check --fast` stays for `aiwf status` / `aiwf
  doctor` / scripts; the statusline stops calling it.

## Constraints

- The ai-dotfiles schema is fixed: `{generated_at, findings:[{source, severity,
  message}]}`, `severity` one of info / warn / error, empty findings = healthy.
- The statusline never runs a check on the render path — it only reads and unions.
- Health-file writes are atomic and land in the main checkout's `.claude/` even when
  invoked from a linked worktree.
- `generated_at` is stamped at the CLI edge (no wall-clock in core).

## Success criteria

- On any aiwf repo the statusline always shows a stoplight, including a green healthy
  state the shipped glyph never rendered.
- A real installation or configuration warning or error turns the glyph yellow or red.
- ai-dotfiles' own findings union into the same glyph with no aiwf-side change.

## Milestones

1. Producer — `aiwf doctor` surfaces warnings and errors and writes
   `.claude/health.aiwf.json`.
2. Consumer — the four-state statusline stoplight.
