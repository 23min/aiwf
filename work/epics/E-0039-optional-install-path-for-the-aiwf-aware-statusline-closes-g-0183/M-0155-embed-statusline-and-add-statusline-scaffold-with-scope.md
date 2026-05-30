---
id: M-0155
title: Embed statusline and add --statusline scaffold with --scope
status: in_progress
parent: E-0039
depends_on:
    - M-0153
tdd: required
acs:
    - id: AC-1
      title: Statusline script embedded in the aiwf binary via go:embed
      status: met
      tdd_phase: done
    - id: AC-2
      title: init and update grow a --statusline flag with --scope project|user
      status: met
      tdd_phase: done
    - id: AC-3
      title: 'Scaffold-if-absent: bare aiwf update leaves an existing script untouched'
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'Project scope: relative-path snippet + scoped .claude/statusline.sh ignore'
      status: open
      tdd_phase: done
    - id: AC-5
      title: 'User scope: writes ~/.claude/statusline.sh with absolute-path snippet'
      status: open
      tdd_phase: red
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
the scope-appropriate path **only if absent** (never clobbers) via a dedicated
scaffold-if-absent write path — **not** routed through `Materialize`, whose
contract is wipe-and-rewrite; project scope adds the `.claude/statusline.sh`
gitignore entry **only on the `--statusline` install path** (never in the
unconditional `GitignorePatterns()` set, which `ensureGitignore` reconciles on
every bare `update`) and prints a relative-path activation snippet; a subsequent
bare `aiwf update` touches neither the script nor the `.gitignore`; `--scope user`
targets `~/.claude/statusline.sh` with an absolute-path snippet; both
`--statusline` and `--scope`'s closed value set (`project|user`, via
`cobra.FixedCompletions`) are wired through shell completion (completion-drift test).

## Constraints

- Embedded but excluded from the unconditional refresh set (scaffold-once).
- Project scope writes only inside the repo; `--scope user` is the explicit
  operator choice, never auto-selected from environment.
- The `.gitignore` entry is emitted only by the `--statusline` install path,
  never appended to the unconditional `GitignorePatterns()` — so a consumer who
  never opts in keeps a clean `.gitignore`, and this repo's own deliberate
  `!.claude/statusline.sh` un-ignore (it tracks the canonical copy) is never
  contradicted.
- No settings-file write in this milestone.

## Design notes

- `go:embed` for the script; `--statusline` shares one implementation across
  `init` and `update`.
- Scaffold-if-absent is a new write path (e.g. a `WriteStatuslineIfAbsent`
  helper), the deliberate exception to the materializer's "files are a cache,
  not state" contract — embedded so fixes ship with the binary and `doctor` can
  detect on-disk drift (M-0157), written once so user tweaks survive `update`.
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

### AC-1 — Statusline script embedded in the aiwf binary via go:embed

### AC-2 — init and update grow a --statusline flag with --scope project|user

### AC-3 — Scaffold-if-absent: bare aiwf update leaves an existing script untouched

### AC-4 — Project scope: relative-path snippet + scoped .claude/statusline.sh ignore

### AC-5 — User scope: writes ~/.claude/statusline.sh with absolute-path snippet

