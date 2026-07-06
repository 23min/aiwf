---
id: ADR-0032
title: 'Materialized hook consent: persisted per-hook aiwf.yaml registry'
status: proposed
---
# ADR-0032 — Materialized hook consent: persisted per-hook aiwf.yaml registry

> **Date:** 2026-07-06 · **Decided by:** human/peter

## Status vocabulary (aiwf)

aiwf's ADR statuses are: `proposed | accepted | superseded | rejected`.

## Context

ADR-0015 narrowed aiwf's original "never edits settings.json" stance to a
consent-gated one, scoped to the statusline (E-0039): a TTY `[y/N]` confirm, or
an explicit `--wire-settings` flag in non-TTY contexts, evaluated fresh on
every invocation. ADR-0018 generalized the underlying principle — consent
calibrated to a file's risk profile — to a second user-owned file
(`CLAUDE.md`), using a different mechanism (automatic, opt-out only) because
that edit's risk profile is negligible: marker-scoped, reversible, prose-only.

E-0059 introduces a third case: Claude Code hooks (`SessionStart`,
`SubagentStart`, and in future potentially others) that aiwf ships as
materialized artifacts and activates via a `.claude/settings.json` entry.
Neither existing mechanism fits:

- ADR-0015's per-invocation, unpersisted prompt does not scale past one
  feature — a second hook would mean a second bespoke flag, a third a third,
  which is the exact generalization ADR-0015's own consequences section
  warns against. It also carries no durable record: on a shared repo, every
  developer's own clone re-hits the same prompt independently, since the
  only evidence a decision was ever made lives in whether `settings.json`
  already has the key.
- ADR-0018's automatic, opt-out-only mechanism does not fit either: unlike a
  marker-scoped guidance import, a hook changes runtime behavior — it can
  deny an action or print a warning on every session. Each hook carries its
  own risk/value tradeoff and deserves its own explicit consent, not a
  blanket default-on.

## Decision

aiwf does not activate a materialized hook without consent, calibrated to
hooks' own profile: **per-hook granularity, decided once, persisted and
shared.** This is the third instance of the risk-calibrated-consent family
ADR-0015 and ADR-0018 established.

- `aiwf.yaml` gains a `hooks:` map: `hooks.<name>.enabled: true|false`.
  Absence of a key means undecided.
- **`aiwf init`** (no prior `aiwf.yaml`): every hook aiwf ships is undecided,
  so every one is gated before the fresh `aiwf.yaml` is written — a TTY
  `[y/N]` prompt naming the hook and its one-line effect (default declines),
  or, absent a TTY, silent refusal unless the operator passes
  `--enable-hook <name>` (repeatable) — the flag's presence on the command
  line is the consent record, mirroring `--wire-settings`'s reasoning.
- **`aiwf update`** (existing `aiwf.yaml`): only hooks absent from the
  `hooks:` map — i.e. introduced by a newer aiwf version than the consumer
  last ran — trigger the same prompt/flag gate. Every hook already decided,
  `true` or `false`, syncs silently on every run: materialize the script and
  wire the settings entry when `true`; remove both when `false`. There is no
  re-prompt once a decision is on record.
- The decision lives in `aiwf.yaml`, which is committed. The first developer
  to enable a hook commits that decision; every other clone's `aiwf init` /
  `aiwf update` reads it as already-settled and wires the hook
  non-interactively — the consent is made once per team, not once per
  developer per machine.
- Target file is the **shared** `.claude/settings.json`, not
  `.claude/settings.local.json`. ADR-0015 sends the statusline to the local
  file because the script itself is a personal, separately-opted-in
  preference — wiring the shared file risks pointing a teammate's config at
  a script that was never materialized on their machine. A hook has no such
  risk: once `enabled: true` is committed, `aiwf init` / `update`
  unconditionally materializes the script for every clone, the same way
  skills/agents/templates already are — so the shared file is the correct,
  and only correct, target.
- The settings writer preserves every unrelated key, refuses to clobber an
  existing hook entry that did not originate from aiwf (no-clobber, `.bak`
  before edit — the same discipline `WireStatuslineSettings` already
  applies), and composes correctly across hook event arrays (`SessionStart`,
  `SubagentStart`, `PreToolUse`, …), since more than one hook may register
  under the same event.

Neither ADR-0015 nor ADR-0018 is superseded. Each remains the canonical
mechanism for its own artifact (`settings.json`/statusline; `CLAUDE.md`
respectively). This decision adds materialized hooks as a third instance of
the same family principle, with its own mechanism suited to hooks' own risk
profile.

## Consequences

- A new "hooks" materialization category, parallel to the existing
  skills/agents/templates categories (`Target.HooksDir`, a
  `ListRitualHooks` alongside `ListRitualAgents`/`ListRitualTemplates`, and a
  matching drift/presence report from `MaterializedRituals`).
- `aiwf doctor` gains hook-specific reporting: materialized-but-not-wired,
  wired-but-stale, and any hook still undecided in the consumer's
  `aiwf.yaml`.
- `aiwf.example.yaml` regenerates to document the `hooks:` schema (per
  ADR-0027's generated-example convention).
- This licenses migrating the existing `.claude/hooks/validate-agent-isolation.sh`
  (G-0099), currently hand-committed and repo-local only, into the shipped
  registry as a natural second entry — tracked as a follow-up gap, not
  implemented by the milestones this ADR unblocks.
- Implementing milestones live under E-0059: one lands the registry,
  `aiwf.yaml` schema, consent flow, and settings writer; a second registers
  the first concrete hook (the worktree-materialization-check backstop
  this epic exists to ship) against it.

## References

- ADR-0015 — the `settings.json`/statusline instance of risk-calibrated
  consent; not superseded.
- ADR-0018 — the `CLAUDE.md` instance and the "risk-calibrated to the
  file's profile" framing this decision extends to a third case.
- ADR-0027 — generated `aiwf.example.yaml` convention, applied to the new
  `hooks:` schema.
- G-0374 — the gap E-0059 closes.
- G-0099 — the existing isolation-guard hook; a natural second registry
  entry once this decision ships, tracked as a follow-up gap rather than
  folded into this ADR's implementing milestones.
- E-0059 — the implementing epic.
