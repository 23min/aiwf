---
id: ADR-0018
title: Risk-calibrated consent for user-owned file edits
status: accepted
---
## Context

aiwf has documented — in CLAUDE.md and in `internal/cli/doctor/doctor.go` — that
it does not silently edit user-owned files. ADR-0015 narrowed the original "never
edits settings.json" stance to a consent-gated stance for the statusline wiring
(E-0039): a TTY `[y/N]` confirm, or an explicit `--wire-settings` flag in non-TTY
contexts.

E-0040 introduces a second user-owned-file edit. To make aiwf's advisory rules —
the ones with no mechanical chokepoint — bind in consumer trees, aiwf needs to
reach the consumer's `CLAUDE.md`: the only surface re-injected on every turn and
re-read fresh after `/compact`. The mechanism is a single marker-wrapped
`@.claude/aiwf-guidance.md` import block; the guidance content itself lives in a
file aiwf already owns under `.claude/` (G-0243, building on the gate-discipline
work in G-0242).

ADR-0015's settings.json consent model defaults to opt-in: refuse in non-TTY,
require an explicit flag. Applied literally to `CLAUDE.md`, that default would
leave the guidance unwired in the most common agentic case — a headless
`aiwf init` / `aiwf update` run via Bash — which is exactly the
forgot-to-configure failure the feature exists to prevent.

The two edits also have materially different risk profiles. `settings.json` is
structured and tooling-readable: a bad edit can clobber config or, through a
shared file, force a broken artifact onto a teammate's clone. The `CLAUDE.md`
edit is one marker-scoped block that touches only aiwf's own region, preserves
surrounding content verbatim, and is reversible.

## Decision

aiwf does not edit a user-owned consumer file without consent, **calibrated to
that file's risk profile.** Two instances:

1. **`settings.json` (per ADR-0015) — opt-in.** A TTY `[y/N]` confirm (default
   declines) or an explicit `--wire-settings` flag in non-TTY contexts. Never
   clobbers an existing key; writes a `.bak` before editing.

2. **`CLAUDE.md` (this ADR) — automatic and self-maintaining.** `aiwf init` and
   `aiwf update` maintain a marker-wrapped `@.claude/aiwf-guidance.md` import
   block in the consumer's root `CLAUDE.md`: adding it when absent, refreshing it
   when present, and re-adding it if a prior block was removed — exactly how
   skills and hooks are materialized. There is deliberately **no CLI flag**.
   Adopting aiwf (running `init`/`update`) is the consent; the edit is announced
   (a printed ledger step), marker-scoped (only aiwf's own region is touched and
   surrounding content is preserved verbatim, with line-anchored detection so a
   marker mentioned in user prose is inert), reversible, and creates `CLAUDE.md`
   if absent. A consumer opts out via a single set-once `aiwf.yaml` knob,
   `guidance.wire_claudemd: false` (default `true`).

The departure from ADR-0015's default — automatic with a config opt-out rather
than per-invocation prompt consent — is deliberate and licensed by the
risk-profile difference above and by parity with skill/hook materialization
(also automatic, no opt-out flag, location/marker-scoped).

The imported file lives in-repo under `.claude/`, never `~/.claude/` or an
absolute path. That is the only zone free of Claude Code's import-approval dialog
(verified empirically on Claude Code 2.1.177; an out-of-repo import silently
fails to load in a headless session).

### Inclusion principle for the guidance fragment

The fragment is the only aiwf content re-anchored on every turn; to stay
high-signal it must not accrete. A rule may enter the fragment only if it clears
all three criteria:

1. **No mechanical chokepoint** — if `aiwf check`, a hook, or a policy test
   already catches it, the check is the guarantee and per-turn anchoring is
   redundant.
2. **Invisible-until-named** — its violation is silent drift, not a loud failure.
3. **Governs the agent** operating aiwf or its interaction with the human.

Hard boundary: the fragment says nothing about the consumer's own code (language,
test, or build conventions). That is the consumer's own `CLAUDE.md`'s job.

**Sanctioned exception.** A single one-line pointer to an *existing* chokepoint is
admissible even though it fails criterion 1: it spares the agent a predictable
failed push without re-litigating the guarantee. The id-shape rule (backed by the
`body-prose-id` check) is the lone current instance; any future exception is added
here explicitly so the carve-out cannot quietly widen.

## Consequences

- The wiring is a focused `ensureGuidanceImport` step in the init/update pipeline
  (`internal/initrepo`), gated by the `aiwf.yaml` knob. There is **no CLI flag**
  and **no reuse of E-0039's statusline consent machinery**: that machinery
  implements the opt-in TTY-prompt / `--wire-settings` flow, which does not apply
  to an automatic default-on edit. (An earlier flag-based draft of this decision
  was superseded in review before the epic shipped.)
- CLAUDE.md's "what aiwf materializes" section is updated to describe the guidance
  fragment, the `CLAUDE.md` import write-channel, and the `guidance.wire_claudemd`
  opt-out. The narrative design docs are updated at the epic wrap.
- An advisory `aiwf doctor` finding (`claudemd-guidance-unwired`) surfaces an
  unwired tree with the exact remediation command (`aiwf update`, which self-heals
  the import). It is advisory-only and respects the opt-out.
- A fresh `aiwf init` — and every subsequent `update` — leaves the guidance
  loading every turn (including in a headless session and after `/compact`) with
  no extra step, and self-heals if the block is removed.
- This grants aiwf no other consumer-root write target: `CLAUDE.md` is the only
  file this ADR authorizes beyond `settings.json`; the refresh behavior of every
  other materialized artifact (skills, hooks, agents, templates) is unchanged.
- ADR-0015 is **not** superseded — it remains the `settings.json` instance of
  this principle. This ADR generalizes the principle and adds the `CLAUDE.md`
  instance. ADR-0015's own status is a separate matter, out of scope here.

## References

- ADR-0015 — the `settings.json` instance of this principle (per-invocation
  consent); its opt-in prompt machinery does not apply to the automatic
  `CLAUDE.md` case.
- G-0243 — the gap this decision unblocks; G-0242 — the gate-discipline rule that
  motivated reaching consumer `CLAUDE.md`.
- E-0040 — the implementing epic.
- ADR-0014 / E-0038 — the embed-and-materialize mechanism the guidance fragment
  ships through.
