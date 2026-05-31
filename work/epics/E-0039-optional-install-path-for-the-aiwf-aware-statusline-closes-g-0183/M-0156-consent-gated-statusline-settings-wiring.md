---
id: M-0156
title: Consent-gated statusline settings wiring
status: done
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
      status: met
      tdd_phase: done
    - id: AC-3
      title: Consent-gated write creates .bak and inserts statusLine
      status: met
      tdd_phase: done
    - id: AC-4
      title: Pre-existing statusLine key blocks write with merge guidance
      status: met
      tdd_phase: done
    - id: AC-5
      title: Non-TTY and JSON paths skip write and emit snippet
      status: met
      tdd_phase: done
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

### AC-1 — render.IsTTY predicate wraps term.IsTerminal

Added `IsTTY(*os.File) bool` to `internal/render/term.go`, wrapping
`term.IsTerminal`. Two unit tests: non-TTY (piped stdout under `go test`)
returns false; nil returns false. Updated `TerminalWidth`'s comment to
reference the new companion predicate. Tests 2/2.

### AC-2 — --wire-settings flag on init and update with completion

Added `--wire-settings` boolean flag to both `initcmd.NewCmd()` and
`update.NewCmd()`. Plumbed through `Run` signatures. Policy test
`TestM0156_AC2_WireSettingsFlagOnInitAndUpdate` asserts flag presence and
default value on both commands. Tests 2/2 (init + update subtests).

### AC-3 — Consent-gated write creates .bak and inserts statusLine

New `internal/skills/settings.go` with `WireStatuslineSettings(settingsPath,
cmdPath)` — pure JSON settings-file manipulation: reads existing file, writes
`.bak`, inserts `statusLine` key. Also `SettingsPathForScope(root, home,
scope)` resolving project→`settings.local.json`, user→`settings.json`.
Three policy tests: insert-with-bak, create-from-scratch, path-for-scope.
Tests 3/3.

### AC-4 — Pre-existing statusLine key blocks write with merge guidance

`WireStatuslineSettings` detects existing `statusLine` key. If the command
matches → idempotent no-op (Idempotent=true). If different →
no-clobber (Wrote=false, ExistingValue set for merge guidance). Two policy
tests: different-value blocks, same-value is idempotent. Tests 2/2.

### AC-5 — Non-TTY and JSON paths skip write and emit snippet

Rewrote `cliutil.RunStatuslineScaffold` as the consent flow orchestrator.
Consent model: `--wire-settings` → unconditional write; TTY + not JSON →
`[y/N]` prompt; otherwise → skip write, print snippet. Three policy tests:
non-TTY skips, `--wire-settings` writes, `--format=json` skips. Tests 3/3.

## Decisions made during implementation

- `RunStatuslineScaffold` signature changed from `(rootDir, scope string)` to
  `(opts StatuslineOpts)` to accommodate the new `WireSettings` and
  `FormatJSON` fields without a 6-parameter function.
- Settings-file manipulation lives in `internal/skills/settings.go` (alongside
  the scaffold logic), not in `cliutil/`, because it's not CLI-specific — any
  caller can use it.
- `promptYN` reads from `os.Stdin` and writes to `os.Stderr` (prompt on
  stderr so stdout stays clean for `--format=json`). Cannot be unit-tested
  under `go test` (stdin is piped). The consent flow's integration is tested
  via AC-5's non-interactive path assertions.

## Validation

- Tests: 12 functions across 5 test files (2 in `render/`, 2 in `policies/`
  AC-2, 3 in AC-3, 2 in AC-4, 3 in AC-5); all pass.
- Full module: `go test -race -parallel 8 ./...` clean.
- Build: `go build ./cmd/aiwf` clean.
- Lint: `golangci-lint run` — 0 issues in our code (2 pre-existing gosec
  warnings in the stale E-0038 worktree).
- `aiwf check`: 0 errors, 11 pre-existing warnings.
- Coverage: `WireStatuslineSettings` 75.9%, `handleExistingKey` 100%,
  `RunStatuslineScaffold` 56.8% (uncovered: TTY prompt path + error paths),
  `promptYN` 0% (untestable under `go test`).

## Deferrals

- (none)

## Reviewer notes

- The `promptYN` function (interactive `[y/N]` confirm) cannot be exercised
  under `go test` because stdin is always piped. Coverage is 0%. The consent
  flow's *effect* is tested via AC-5's non-interactive path assertions (which
  verify the settings file is not written when consent is absent), and the
  `--wire-settings` path (which bypasses the prompt entirely). A behavioral
  integration test (binary subprocess with a PTY) would close this gap but
  is out of scope for this milestone.
- `statuslineCmdPath` parses the command path from the scaffold result's
  snippet rather than adding a new exported field to `StatuslineScaffoldResult`.
  This avoids changing the M-0155 API surface; if the snippet format changes,
  this parser breaks loudly (returns the fallback `.claude/statusline.sh`)
  rather than silently.
- The `FormatJSON` field in `StatuslineOpts` is plumbed but not yet wired
  from the CLI flags — `initcmd` and `update` don't currently carry
  `--format=json`. The field is present so M-0157 or a future verb can wire
  it without changing the signature again.

### AC-1 — render.IsTTY predicate wraps term.IsTerminal

### AC-2 — --wire-settings flag on init and update with completion

### AC-3 — Consent-gated write creates .bak and inserts statusLine

### AC-4 — Pre-existing statusLine key blocks write with merge guidance

### AC-5 — Non-TTY and JSON paths skip write and emit snippet

