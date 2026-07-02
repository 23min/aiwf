---
id: E-0055
title: 'Health as install status: producer health files + statusline stoplight'
status: active
---
## Goal

Replace the tree-check-driven, silent-when-healthy statusline glyph with an
always-visible four-state stoplight (gray / green / yellow / red) driven by
per-producer `.claude/health.*.json` files. aiwf becomes a *producer* that
writes its installation and configuration health (the `aiwf doctor` domain) to
`health.aiwf.json`; the statusline becomes a pure *consumer* that globs the
health files, unions their findings, and renders one glyph at the maximum
severity — never running a check on the render path.

## Context

The shipped statusline health glyph (G-0290) drives off `aiwf check --fast` —
the planning-*tree* health axis — and renders nothing when the tree is clean.
Two consequences: it surfaces the wrong axis (tree drift, not installation
health), and it is invisible in the common healthy case, so an operator
concludes it is not working at all. Meanwhile ai-dotfiles has settled a
per-producer health-file model (G-0305): each producer writes one
`.claude/health.<source>.json`, and a statusline globs and unions them without
running any checks. This epic adopts that model, corrects the health axis to
installation and configuration, and makes the healthy state visible.

## Scope

### In

- Structured, severity-tagged findings out of `aiwf doctor`, plus a
  `--format=json` envelope (closes G-0070).
- `aiwf doctor --write-health`: map doctor findings onto the fixed ai-dotfiles
  schema and atomic-write `.claude/health.aiwf.json`, resolved to the main
  checkout even from a linked worktree, refreshed by the lifecycle verbs.
- A four-state statusline stoplight consumer that globs `.claude/health.*.json`,
  unions findings, and renders gray / green / yellow / red at maximum severity —
  superseding the `aiwf check --fast` render block.

### Out

- Changing the ai-dotfiles health-file schema — treated as fixed and external;
  aiwf validates against it on write.
- A producer registry — "no health file present" is a single global unknown
  (gray) state, not a per-expected-producer determination.
- Tree-drift signalling on the statusline — `aiwf check --fast` remains for
  `aiwf status`, `aiwf doctor`, and scripts, but the statusline stops calling it.

## Constraints

- The ai-dotfiles schema is fixed: `{ "generated_at": <ISO8601 UTC>, "findings":
  [ { "source", "severity", "message" } ] }`, `severity` one of `info` / `warn`
  / `error`, empty findings = healthy.
- The statusline renders on every prompt and must never run a check on that path
  — it only reads and unions the health files.
- Health-file writes are atomic (temp + rename) and land in the main checkout's
  `.claude/` even when invoked from a linked worktree.
- `generated_at` is stamped at the CLI edge; core write logic takes the
  timestamp as a parameter (no wall-clock in core).

## Success criteria

- On any aiwf repo the statusline always shows a stoplight glyph, including a
  green "healthy" state that the shipped glyph never rendered.
- A real installation or configuration problem turns the glyph yellow or red
  within one lifecycle cycle.
- ai-dotfiles' own findings union into the same glyph with no aiwf-side change.

## Milestones

1. Structured doctor findings and a `--format=json` envelope (closes G-0070) —
   the prerequisite that gives the health file a typed, severity-mapped source.
2. aiwf as a health producer: `aiwf doctor --write-health`, schema-mapped atomic
   write, main-checkout resolution, contract-tested against the fixed schema, and
   wired into the lifecycle verbs.
3. The four-state statusline stoplight consumer, replacing the check-at-render
   block (supersedes the M-0193 health-glyph behaviour).

An ADR recording the reframe — health means installation and configuration
(doctor), surfaced via producer health files and a four-state stoplight,
superseding the tree-check glyph — is to be recorded early in the epic.
