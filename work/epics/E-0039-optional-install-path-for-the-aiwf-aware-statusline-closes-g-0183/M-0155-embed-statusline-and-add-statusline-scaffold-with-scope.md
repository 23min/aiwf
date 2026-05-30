---
id: M-0155
title: Embed statusline and add --statusline scaffold with --scope
status: draft
parent: E-0039
depends_on:
    - M-0153
tdd: required
---
# M-0155 — Embed statusline and add --statusline scaffold with --scope

## Goal

Ship the statusline in the binary and let a consumer scaffold it via
`aiwf init/update --statusline [--scope project|user]` — writing the script,
the gitignore entry, and a printed activation snippet — with no settings write.

## Context

Builds on the embedded-artifact mechanism (ADR-0014 / E-0038) but applies one
deliberate difference: the statusline is embedded yet **excluded from the
unconditional refresh set**, so a consumer's edits survive `aiwf update`.
Requires the portable script from M-0153.

## Acceptance criteria

<!-- Formal ACs added at start-milestone via `aiwf add ac M-0155`. Intended shape: -->

`--statusline` on both `init` and `update` materializes the embedded script to
the scope-appropriate path **only if absent** (never clobbers); project scope
adds the `.claude/statusline.sh` gitignore entry and prints a relative-path
activation snippet; a subsequent bare `aiwf update` leaves an existing copy
untouched; `--scope user` targets `~/.claude/statusline.sh` with an absolute-
path snippet; the flag is wired through shell completion (completion-drift test).

## Constraints

- Embedded but excluded from the unconditional refresh set (scaffold-once).
- Project scope writes only inside the repo; `--scope user` is the explicit
  operator choice, never auto-selected from environment.
- No settings-file write in this milestone.

## Design notes

- `go:embed` for the script; `--statusline` shares one implementation across
  `init` and `update`.
- Scope table: project → `<repo>/.claude/statusline.sh`, relative command path;
  user → `~/.claude/statusline.sh`, absolute command path.

## Surfaces touched

- `internal/skills/` (embed + materialization carve-out)
- `cmd/aiwf/` init / update flag wiring + completion
- gitignore pattern emission

## Out of scope

- Settings wiring (M-0156) and doctor reporting (M-0157).

## Dependencies

- M-0153 (portable script).

## References

- [E-0039](epic.md) · ADR-0014 / E-0038 (embed precedent) · `.claude/statusline.sh`

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
