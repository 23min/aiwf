---
id: M-0156
title: Consent-gated statusline settings wiring
status: draft
parent: E-0039
depends_on:
    - M-0154
    - M-0155
tdd: required
acs:
    - id: AC-1
      title: render.IsTTY predicate wraps term.IsTerminal
      status: met
      tdd_phase: done
    - id: AC-2
      title: --wire-settings flag on init and update with completion
      status: open
      tdd_phase: green
    - id: AC-3
      title: Consent-gated write creates .bak and inserts statusLine
      status: open
      tdd_phase: red
    - id: AC-4
      title: Pre-existing statusLine key blocks write with merge guidance
      status: open
      tdd_phase: red
    - id: AC-5
      title: Non-TTY and JSON paths skip write and emit snippet
      status: open
      tdd_phase: red
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
interactive-confirm pattern before (the only TTY-awareness today is
non-interactive width detection inside `render.TerminalWidth`).

## Acceptance criteria

<!-- Formal ACs added at start-milestone via `aiwf add ac M-0156`. Intended shape: -->

With consent (TTY `y`, or `--wire-settings`), the verb writes the `statusLine`
key to the scope-appropriate settings file; without a TTY and without
`--wire-settings`, it never writes and prints the snippet; in `--format=json`
mode the verb never prompts (it mirrors the non-TTY path: without
`--wire-settings` it skips the write and returns the activation snippet in the
JSON `result`); a pre-existing `statusLine` key is never overwritten (the verb
prints merge guidance); a `.bak` is written before edit; re-running when the key
already points at our script is an idempotent no-op; the new `--wire-settings`
flag is wired through shell completion (completion-drift test).

## Constraints

- The interactive prompt is gated strictly to this opt-in flag — no other verb
  gains a prompt.
- Non-TTY path uses the explicit `--wire-settings` flag, never a hidden prompt.
- `--format=json` is a non-interactive path: it never prompts and treats a
  missing `--wire-settings` exactly like the non-TTY case (skip + snippet in
  `result`).
- `settings.local.json` is the project-scope target (personal, gitignored) —
  not the shared `settings.json`.
- No-clobber of an existing `statusLine`.

## Design notes

- TTY detection needs a small exported predicate — add `render.IsTTY(*os.File) bool`
  wrapping `term.IsTerminal`. `internal/render/term.go` today exposes only
  `TerminalWidth`; its own comment already anticipates this separate predicate.
  `golang.org/x/term` is already a module dependency, so this adds no new dep.
- JSON: parse → add-key-if-absent → marshal; `.bak` first; refuse if a
  `statusLine` already exists.

## Surfaces touched

- `cmd/aiwf/` (consent flow + `--wire-settings`, completion wiring)
- a settings-file editor helper
- `internal/render/term.go` (new `IsTTY` predicate)
- (the stance prose — CLAUDE.md + `doctor.go` — is owned by M-0154, not this
  milestone; this milestone adds only operational `--wire-settings` usage notes
  if any are needed)

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

### AC-1 — render.IsTTY predicate wraps term.IsTerminal

### AC-2 — --wire-settings flag on init and update with completion

### AC-3 — Consent-gated write creates .bak and inserts statusLine

### AC-4 — Pre-existing statusLine key blocks write with merge guidance

### AC-5 — Non-TTY and JSON paths skip write and emit snippet

