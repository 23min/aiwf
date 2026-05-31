---
id: M-0155
title: Embed statusline and add --statusline scaffold with --scope
status: done
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
      status: met
      tdd_phase: done
    - id: AC-5
      title: 'User scope: writes ~/.claude/statusline.sh with absolute-path snippet'
      status: met
      tdd_phase: done
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

### AC-1 — Statusline script embedded in the aiwf binary via go:embed

Added `//go:embed embedded-statusline/statusline.sh` to
`internal/skills/skills.go` and a `StatuslineBytes()` accessor.
The embed source at `internal/skills/embedded-statusline/statusline.sh`
is kept byte-equal to the canonical `.claude/statusline.sh` by
`TestM0155_AC1_StatuslineEmbedded`'s drift assertion — any future edit
to either file that isn't mirrored to the other fails CI with a clear
remediation hint. Tests 1/1. Closed in afb48651.

### AC-2 — init and update grow a --statusline flag with --scope project|user

Both `internal/cli/initcmd/initcmd.go` and `internal/cli/update/update.go`
now register `--statusline` (bool) and `--scope` (string, project|user
closed set). `--scope` is wired through Cobra's
`RegisterFlagCompletionFunc` with `FixedCompletions` — the M-054
drift-prevention chokepoint catches a missing one anyway, but the
per-flag assertion in `TestM0155_AC2_StatuslineFlagsOnInitAndUpdate`
fails earlier with a focused error message. Tests 2/2 (init + update
subtests). Closed in fadb8b42.

### AC-3 — Scaffold-if-absent: bare aiwf update leaves an existing script untouched

New helper `skills.ScaffoldStatuslineWithHome(root, home, scope)` in
`internal/skills/statusline.go` is the dedicated write path —
**separate from `Materialize`** whose contract is wipe-and-rewrite.
The helper writes the embed bytes to the destination only when no
file already exists there; a pre-existing copy is preserved verbatim
(including operator edits). `TestM0155_AC3_ScaffoldStatuslineWritesIfAbsent`
drives both legs via `t.TempDir`: fresh destination → write +
bytes-equal-embed; pre-existing destination → no write + sentinel
content preserved. Tests 2/2 (sub-tests). Closed in 5e6bd548.

### AC-4 — Project scope: relative-path snippet + scoped .claude/statusline.sh ignore

Project scope writes to `<root>/.claude/statusline.sh` and idempotently
appends `.claude/statusline.sh` to `<root>/.gitignore` via the new
`ensureStatuslineGitignoreEntry` helper. The ignore line is emitted
**only on the `--statusline` install path** — never in the unconditional
`GitignorePatterns()` set (asserted by
`TestM0155_AC4_GitignorePatternsNotGlobal`) — so a consumer who never
opts in keeps a clean `.gitignore`. Activation snippet uses a
repo-relative command path so it works from any cwd inside the repo.
Tests 2/2 (one full project-scope + one global-patterns guard).
Closed in 70b407b8.

### AC-5 — User scope: writes ~/.claude/statusline.sh with absolute-path snippet

User scope routes through `os.UserHomeDir()` (production path) and
through `ScaffoldStatuslineWithHome(root, home, ...)` (test path) to
write `<home>/.claude/statusline.sh`. No gitignore touch (user scope
lives outside any tracked tree). Activation snippet uses an absolute
command path so the same script renders correctly from any worktree of
any repo in the same (dev)container.
`TestM0155_AC5_UserScopeWritesHomeWithAbsoluteSnippet` pins the
destination, the gitignore-not-created guard, and the absolute-path
shape of the snippet. Tests 1/1. Closed in 1d895b47.

## Decisions made during implementation

- (none)

## Validation

- Tests: 6 functions in three new files under `internal/policies/`
  (`m0155_statusline_embed_test.go`, `m0155_statusline_flag_test.go`,
  `m0155_statusline_scaffold_test.go`); all pass with race detector.
- Full module: `go test ./...` clean apart from the pre-existing
  `TestFSMHistoryConsistent_PerfBudget` flake (timing-sensitive perf
  test under parallel load — already documented as environmental,
  passes in isolation).
- Build: `go build ./cmd/aiwf` clean.
- Lint: `golangci-lint run` 0 issues (one gofumpt grouping fix applied
  mid-cycle to the scaffold-test file's constants).
- `aiwf check`: 0 errors on M-0155; 16 advisory warnings (all
  pre-existing or `acs-tdd-audit`-shaped, which is benign now that
  phases are tracked end-to-end under `tdd: required`).
- Smoke render via the embedded binary not executed in this milestone
  (M-0157's doctor block will exercise that path end-to-end); the
  byte-equality drift test in AC-1 is the structural-evidence proxy.

## Deferrals

- (none)

## Reviewer notes

- The five ACs were implemented in one coherent pass (RED tests
  written upfront, then one implementation diff brought them all
  GREEN). This departs from the strict per-AC red→green cycle that
  `wf-tdd-cycle` suggests under `tdd: required`, but the ACs are
  tightly coupled — AC-2's flags only do something useful once
  AC-3's scaffold helper exists; AC-4 and AC-5 are just per-scope
  branches inside the same helper. The kernel-level phase tracking
  (`--phase red → green → done` per AC) was still applied so the audit
  trail in `aiwf history M-0155/AC-N` records the discipline; what
  was condensed is the implementation order, not the test-before-code
  ordering.
- The `internal/skills/embedded-statusline/statusline.sh` file is a
  byte-equal copy of the canonical `.claude/statusline.sh`. There is
  no `make sync-statusline` target yet — the drift test catches a
  forgotten sync at CI time. If editing the script becomes friction-
  prone (more than a couple of misses), adding the make target is the
  obvious next step; for now, manual copy keeps tooling minimal.
- `runStatuslineScaffold` lives in `internal/cli/cliutil/statusline.go`
  rather than in `initcmd` or `update` because both commands call it.
  Putting it in `cliutil` keeps the Exit-code semantics local to that
  package (which is what cliutil is for) and avoids a cross-import
  between `initcmd` and `update`.
- Test injection via `ScaffoldStatuslineWithHome` (two-arity pattern):
  the parameter-injected variant is exported for tests; the production
  variant `ScaffoldStatusline` resolves `os.UserHomeDir()` internally.
  Tests use the inner helper with a `t.TempDir`-anchored fake home so
  they stay parallel-safe (no `t.Setenv("HOME", ...)` needed, which
  would conflict with `t.Parallel`).
- ADR-0015 cross-reference appears in the activation-snippet preamble
  printed by the scaffold helper. M-0156's wiring milestone replaces
  the manual paste step the snippet describes with the actual settings
  write (consent-gated per the ADR).
- A `--dry-run` `aiwf init --statusline` reports "dry-run — statusline
  scaffold skipped." rather than scaffolding. `aiwf update` has no
  `--dry-run` mode currently, so its `--statusline` flag always
  executes the scaffold when set.

### AC-1 — Statusline script embedded in the aiwf binary via go:embed

### AC-2 — init and update grow a --statusline flag with --scope project|user

### AC-3 — Scaffold-if-absent: bare aiwf update leaves an existing script untouched

### AC-4 — Project scope: relative-path snippet + scoped .claude/statusline.sh ignore

### AC-5 — User scope: writes ~/.claude/statusline.sh with absolute-path snippet

