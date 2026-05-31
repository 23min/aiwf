# Epic wrap — E-0039

**Date:** 2026-05-31
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0039-statusline-install
**Merge commit:** 71fd995d

## Milestones delivered

- M-0153 — Statusline script portability and robustness fixes (merged 0d21ee15)
- M-0154 — ADR: amend settings.json stance to consent-gated (merged 650385a2)
- M-0155 — Embed statusline and add --statusline scaffold with --scope (merged 14359661)
- M-0156 — Consent-gated statusline settings wiring (merged 2198ef3f)
- M-0157 — aiwf doctor statusline block (merged aa370e10)

## Summary

E-0039 ships the aiwf-aware Claude Code statusline as an optional install path
via `aiwf init/update --statusline`. The script is embedded in the binary
(`go:embed`), scaffolded on first use (never clobbered), and activated via
consent-gated settings wiring (`--wire-settings` or TTY `[y/N]` confirm per
ADR-0015). `aiwf doctor` reports dep availability, wiring state, drift, and a
container scope nudge. The shipped script is portable (macOS + Linux) and
hardened against index-lock contention. Closes G-0183.

## ADRs ratified

- ADR-0015 — Settings.json edits require explicit per-invocation consent

## Decisions captured

- (none beyond ADR-0015)

## Follow-ups carried forward

- G-0187 — Statusline rendering has no end-to-end behavioral test (deferred
  from M-0153; content assertions pin structural form but cannot detect
  bash-precedence bugs caught at the smoke phase)

## Handoff

The statusline install path is fully operational. A downstream consumer runs
`aiwf update --statusline --wire-settings` and gets a working HUD in one
command. The deferred G-0187 is the only open gap — it improves test
confidence but does not block any consumer functionality.
