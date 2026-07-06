---
id: M-0235
title: 'Generalized hook registry: aiwf.yaml-declared, persisted consent'
status: draft
parent: E-0059
tdd: required
acs:
    - id: AC-1
      title: 'aiwf.yaml hooks: schema + aiwf.example.yaml regen'
      status: open
      tdd_phase: red
    - id: AC-2
      title: aiwf init gates undecided hooks via TTY prompt / --enable-hook flag
      status: open
      tdd_phase: red
    - id: AC-3
      title: aiwf update gates only newly-introduced hooks; syncs decided hooks silently
      status: open
      tdd_phase: red
    - id: AC-4
      title: 'Hooks settings writer: no-clobber, .bak backup, multi-event-array composition'
      status: open
      tdd_phase: red
---

## Goal

Build the infrastructure so any Claude Code hook aiwf ships (`SessionStart`,
`SubagentStart`, …) can be materialized into a consumer's `.claude/hooks/`
and activated in the consumer's `.claude/settings.json`, gated by a per-hook
decision recorded in `aiwf.yaml`'s new `hooks:` table — persisted once,
shared across a team's clones, rather than re-asked per invocation per
developer.

## Context

ADR-0015 gates the statusline's settings.json edit on a per-invocation TTY
prompt / `--wire-settings` flag, with no persisted record. ADR-0018
generalizes the underlying risk-calibrated-consent principle to a second
artifact (`CLAUDE.md`), using a different, fully-automatic mechanism suited
to that edit's near-zero risk. Neither fits a hook, which changes runtime
behavior and needs its own per-hook consent that persists rather than being
re-litigated every run. ADR-0032 captures this as the third instance of the
family; this milestone implements it. No concrete hook ships here — that is
the next milestone's job, registered against what this one builds.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0235 --title "..."`.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — `aiwf.yaml`'s schema gains a `hooks:` map
  (`hooks.<name>.enabled: true|false`); an absent key means undecided.
  `aiwf.example.yaml` regenerates to document it (ADR-0027).
- **AC-2 candidate** — On a fresh repo, `aiwf init` gates every registry hook
  with no recorded decision: a TTY `[y/N]` prompt (default declines) naming
  the hook and its one-line effect, or, absent a TTY, silent refusal unless
  `--enable-hook <name>` (repeatable) is passed. The decision — enabled or
  not — is baked into the freshly-written `aiwf.yaml`.
- **AC-3 candidate** — On an existing `aiwf.yaml`, `aiwf update` gates only
  hooks absent from the `hooks:` map (introduced by a newer aiwf version);
  every already-decided hook syncs silently every run — materialize +
  wire when `true`, remove + unwire when `false` — with no re-prompt.
- **AC-4 candidate** — The hooks settings writer targets the shared
  `.claude/settings.json`, preserves every unrelated key, refuses to clobber
  an existing non-aiwf entry for the same event (no-clobber, `.bak` before
  edit — mirroring `WireStatuslineSettings`), and composes correctly across
  multiple hook-event arrays (`SessionStart`, `SubagentStart`, `PreToolUse`,
  …) without duplicating entries on repeat runs.
- **AC-5 candidate** — A new "hooks" materialization category (parallel to
  the existing skills/agents/templates categories) embeds hook scripts;
  `aiwf doctor` reports drift (missing / stale / unwired / still-undecided)
  the same way it already does for rituals.

### AC-1 — aiwf.yaml hooks: schema + aiwf.example.yaml regen

### AC-2 — aiwf init gates undecided hooks via TTY prompt / --enable-hook flag

### AC-3 — aiwf update gates only newly-introduced hooks; syncs decided hooks silently

### AC-4 — Hooks settings writer: no-clobber, .bak backup, multi-event-array composition

## Constraints

- Never write `enabled: true` for a hook that hasn't been explicitly
  consented — the TTY-prompt / explicit-flag gate runs before the first
  write for every undecided hook, no exceptions.
- Settings target is the shared `.claude/settings.json`, never
  `.settings.local.json` — hooks are unconditionally materialized once
  enabled, unlike the personal opt-in statusline (ADR-0015).
- Neither ADR-0015's nor ADR-0018's own code paths change; this ships as an
  independent, parallel mechanism scoped to hooks only.

## Design notes

- ADR-0032 locks the mechanism this milestone implements: aiwf.yaml-declared
  `hooks:` map, per-hook consent gate on first decision only, shared
  `.claude/settings.json` target, no-clobber `.bak`-guarded writer.

## Out of scope

- The concrete `worktree-materialization-check` hook's own detection logic,
  script, and policy test — the next milestone, riding on this one's
  registry.
- Migrating the existing `.claude/hooks/validate-agent-isolation.sh`
  (G-0099) into this registry — tracked as a follow-up gap, not implemented
  here.

## Dependencies

- None within this epic — independent of M-0233/M-0234.

## References

- ADR-0032 — the consent mechanism this milestone implements.
- ADR-0015 / ADR-0018 — the sibling instances of the risk-calibrated-consent
  family this decision extends.
- ADR-0027 — the generated-`aiwf.example.yaml` convention this milestone's
  schema change follows.
- G-0374 — the gap this epic closes.
