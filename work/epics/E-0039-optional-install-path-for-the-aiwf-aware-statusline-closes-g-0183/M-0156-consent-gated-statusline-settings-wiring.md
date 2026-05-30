---
id: M-0156
title: Consent-gated statusline settings wiring
status: draft
parent: E-0039
depends_on:
    - M-0154
    - M-0155
tdd: required
---
# M-0156 — Consent-gated statusline settings wiring

## Goal

Wire `statusLine` into the operator's settings with explicit per-invocation
consent — an interactive `[y/N]` confirm when a TTY is present, an explicit
`--wire-settings` flag otherwise — into `settings.local.json` (project scope) or
`~/.claude/settings.json` (user scope), never clobbering an existing key.

## Context

Requires the scaffold (M-0155) and the ratified stance ADR (M-0154). This
milestone introduces aiwf's **first interactive prompt**; the kernel has had no
interactive-confirm pattern before (only `term.IsTerminal` for width).

## Acceptance criteria

<!-- Formal ACs added at start-milestone via `aiwf add ac M-0156`. Intended shape: -->

With consent (TTY `y`, or `--wire-settings`), the verb writes the `statusLine`
key to the scope-appropriate settings file; without a TTY and without
`--wire-settings`, it never writes and prints the snippet; a pre-existing
`statusLine` key is never overwritten (the verb prints merge guidance); a `.bak`
is written before edit; re-running when the key already points at our script is
an idempotent no-op.

## Constraints

- The interactive prompt is gated strictly to this opt-in flag — no other verb
  gains a prompt.
- Non-TTY path uses the explicit `--wire-settings` flag, never a hidden prompt.
- `settings.local.json` is the project-scope target (personal, gitignored) —
  not the shared `settings.json`.
- No-clobber of an existing `statusLine`.

## Design notes

- TTY detection via `term.IsTerminal` (already present for width).
- JSON: parse → add-key-if-absent → marshal; `.bak` first; refuse if a
  `statusLine` already exists.

## Surfaces touched

- `cmd/aiwf/` (consent flow + `--wire-settings`)
- a settings-file editor helper
- CLAUDE.md + `doctor.go` comment (per the M-0154 ADR)

## Out of scope

- doctor reporting of wiring state (M-0157).

## Dependencies

- M-0154 (ratified stance ADR), M-0155 (scaffold).

## References

- [E-0039](epic.md) · M-0154 (stance ADR) · `internal/render/term.go`

---

## Work log

- (pending)

## Decisions made during implementation

- (none)

## Validation

- (pending)

## Deferrals

- (none)

## Reviewer notes

- (none)
