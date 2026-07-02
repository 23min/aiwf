---
id: ADR-0026
title: aiwf health is installation status, surfaced via producer health files
status: proposed
---
## Context

aiwf shipped a statusline health glyph (G-0290 / M-0193) that runs
`aiwf check --fast` on the render path and shows a warning glyph only when the
planning *tree* carries error-severity findings. Two problems surfaced in use:

- **Wrong axis.** "Health", in the operator's mental model and in ai-dotfiles'
  cross-producer health model (G-0305), means *is this installation configured
  correctly?* — the `aiwf doctor` domain (binary, config, actor, skills,
  filesystem, hooks, render). The shipped glyph instead reports planning-tree
  drift, a different concern `aiwf check` already owns.
- **Invisible when healthy.** The glyph renders nothing in the clean case (and
  there is no healthy indicator at all), so in normal operation the operator sees
  no signal and reasonably concludes the feature is not working.

Separately, ai-dotfiles settled a per-producer health-file convention: each
producer writes one `.claude/health.<source>.json`
(`{ generated_at, findings: [ { source, severity, message } ] }`, `severity` one
of `info` / `warn` / `error`, empty findings = healthy), and a statusline globs
the files, unions their findings, and renders one glyph — never running a check
itself.

## Decision

**aiwf's statusline "health" signal means installation and configuration health
— the `aiwf doctor` axis — surfaced through the per-producer health-file model,
not by running a check on the render path.**

1. **Producer.** `aiwf doctor` is the source of aiwf's health. It maps its
   findings onto the fixed ai-dotfiles schema and writes
   `.claude/health.aiwf.json` (`source: "aiwf"`), resolved to the main checkout
   even from a linked worktree, refreshed on the installation-state-changing
   lifecycle verbs. The ai-dotfiles schema is treated as fixed and external;
   aiwf validates against it on write.

2. **Consumer.** The statusline is a pure reader. It globs
   `.claude/health.*.json`, unions the findings across producers (aiwf,
   ai-dotfiles, and any future producer), and renders one **four-state stoplight**
   at the maximum severity:

   - **gray** — no health file present, or none parse (unknown);
   - **green** — at least one file present, findings empty or info-only (healthy);
   - **yellow** — maximum severity is `warn`;
   - **red** — maximum severity is `error`.

   The statusline never runs `aiwf check` (or any check) on the render path.

3. **Absence is unknown, globally.** "No health file at all" renders gray, not
   green. This is a consumer-side interpretation and does not change the fixed
   schema (whose own rule reads absence as healthy). Gray is a single global
   state rather than a per-expected-producer determination, which would require a
   producer registry the glob-based model deliberately avoids.

This supersedes the M-0193 render-time-check glyph behaviour. `aiwf check --fast`
itself remains — it is still the tree-health surface for `aiwf status`,
`aiwf doctor`, and scripts — but the statusline no longer drives off it.

## Consequences

- The healthy state becomes visible (green), which the shipped glyph never
  rendered — resolving the "it looks like nothing works" perception.
- aiwf and ai-dotfiles findings share one glyph, at no aiwf-side cost beyond
  writing aiwf's own health file.
- Health-file freshness follows the installation-state cadence (lifecycle verbs),
  not the render cadence; install and config health change rarely, so a written
  file stays fresh between those events without a render-time check.
- `aiwf doctor` must expose structured, severity-tagged findings — previously it
  is text-only (G-0070) — before it can produce the health file; this reframe
  pulls that in as a prerequisite.
- The statusline drops its render-time check and the TTL / HEAD-fold cache that
  existed only to make that check affordable.

## Alternatives considered

- **Keep the tree-check glyph (M-0193) and add producer plumbing on top.**
  Rejected: it conflates two axes — tree drift versus installation health — on
  one glyph and leaves the healthy state invisible.
- **`aiwf check` as the health producer.** Rejected: check reports tree health,
  not installation health; the operator-facing "is this set up correctly?"
  question is doctor's, and doctor's findings are what a health file should carry.
- **A four-state with per-producer "unknown".** Rejected as premature: it needs a
  registry of expected producers; gray-when-no-files-at-all is the simpler global
  signal, and the lifecycle write makes it transient.
