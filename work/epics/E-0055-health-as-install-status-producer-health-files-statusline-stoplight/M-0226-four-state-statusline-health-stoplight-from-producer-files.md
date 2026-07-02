---
id: M-0226
title: Four-state statusline health stoplight from producer files
status: draft
parent: E-0055
depends_on:
    - M-0224
tdd: required
---
## Deliverable

Replace the statusline's `aiwf check --fast` render block with a four-state
health stoplight fed by the producer health files. The statusline globs
`.claude/health.*.json`, unions the findings across producers, and renders one
glyph at the maximum severity — never running a check on the render path. Removes
the now-unnecessary TTL and HEAD-fold cache that existed only to make the
render-time check affordable. Supersedes the M-0193 health-glyph behaviour and
delivers the consumer half of G-0305.

State mapping:

- gray — no health file present, or none parse (unknown);
- green — at least one file present, findings empty or info-only (healthy);
- yellow — maximum severity is warn;
- red — maximum severity is error.

## Acceptance criteria (formalized at milestone start)

- **Four-state union render.** The statusline globs `.claude/health.*.json`,
  unions findings, and renders gray / green / yellow / red at maximum severity.
  Evidence: the existing statusline behavioural harness with a fixture per state,
  including a multi-producer union (dotfiles warn + aiwf error → red; dotfiles
  green + aiwf absent → green; no files → gray).
- **Always visible, no render-time check.** The stoplight renders on every prompt
  in an aiwf repo, and the statusline no longer invokes `aiwf check` on the render
  path. Evidence: a stub `aiwf` that fails if called with `check`; the
  embedded-copy drift test stays green.
- **Robust degrade.** A malformed or partial producer file is skipped and the
  union proceeds; if no file parses, the glyph is gray. Evidence: a fixture
  pairing a corrupt `health.dotfiles.json` with a valid `health.aiwf.json` renders
  aiwf's severity; an all-corrupt fixture renders gray.
