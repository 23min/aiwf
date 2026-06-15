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
`@.claude/aiwf-guidance.md` import line; the guidance content itself lives in a
file aiwf already owns under `.claude/` (G-0243, building on the gate-discipline
work in G-0242).

ADR-0015's settings.json consent model defaults to opt-in: refuse in non-TTY,
require an explicit flag. Applied literally to `CLAUDE.md`, that default would
leave the guidance unwired in the most common agentic case — a headless
`aiwf init` run via Bash — which is exactly the forgot-to-configure failure the
feature exists to prevent.

The two edits also have materially different risk profiles. `settings.json` is
structured and tooling-readable: a bad edit can clobber config or, through a
shared file, force a broken artifact onto a teammate's clone. The `CLAUDE.md`
edit is one human-readable, append-only, marker-scoped line that clobbers
nothing, breaks nothing if present-but-unwanted, and is trivially reversible.

## Decision

aiwf does not edit a user-owned consumer file without consent, **calibrated to
that file's risk profile.** Two instances:

1. **`settings.json` (per ADR-0015) — opt-in.** A TTY `[y/N]` confirm (default
   declines) or an explicit `--wire-settings` flag in non-TTY contexts. Never
   clobbers an existing key; writes a `.bak` before editing.

2. **`CLAUDE.md` (this ADR) — default-on.** `aiwf init` and `aiwf update` wire
   the marker-wrapped `@.claude/aiwf-guidance.md` import line by default,
   **including in non-TTY contexts**, with `--no-wire-claudemd` to decline.
   Running `aiwf init` is itself the consent to adopt aiwf; the edit is never
   silent (a printed notice announces it), is marker-scoped (content outside the
   markers is left verbatim), is reversible, and creates `CLAUDE.md` if absent.

The departure from ADR-0015's default — default-on rather than opt-in, wiring in
non-TTY rather than refusing — is deliberate and licensed by the risk-profile
difference above.

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

## Consequences

- A `--no-wire-claudemd` flag lands on `aiwf init` and `aiwf update`; the wiring
  reuses E-0039's consent machinery where it composes
  (`internal/cli/cliutil/statusline.go`, `internal/skills/settings.go`) rather
  than forking a parallel flow.
- CLAUDE.md and the `internal/cli/doctor/doctor.go` stance comment are updated to
  describe the generalized stance: `settings.json` opt-in, `CLAUDE.md`
  default-on.
- An advisory `aiwf doctor` finding (`claudemd-guidance-unwired`) surfaces an
  unwired tree with the exact remediation command — the recurring safety net that
  makes a declined or hand-removed wiring self-healing without re-fighting the
  operator's choice.
- A fresh `aiwf init` always leaves the guidance loading every turn (including in
  a headless session and after `/compact`) with no extra step.
- This grants aiwf no other consumer-root write target: `CLAUDE.md` is the only
  file this ADR authorizes beyond `settings.json`, and the refresh behavior of
  every other materialized artifact (skills, hooks, agents, templates) is
  unchanged.
- ADR-0015 is **not** superseded — it remains the `settings.json` instance of
  this principle. This ADR generalizes the principle and adds the `CLAUDE.md`
  instance. ADR-0015's own status is a separate matter, out of scope here.

## References

- ADR-0015 — the `settings.json` instance of this principle (per-invocation
  consent), whose consent machinery this decision reuses.
- G-0243 — the gap this decision unblocks; G-0242 — the gate-discipline rule that
  motivated reaching consumer `CLAUDE.md`.
- E-0040 — the implementing epic.
- ADR-0014 / E-0038 — the embed-and-materialize mechanism the guidance file ships
  through.
