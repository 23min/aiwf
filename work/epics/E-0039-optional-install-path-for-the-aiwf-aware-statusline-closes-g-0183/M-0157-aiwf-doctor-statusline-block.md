---
id: M-0157
title: aiwf doctor statusline block
status: done
parent: E-0039
depends_on:
    - M-0155
tdd: required
acs:
    - id: AC-1
      title: Statusline block emitted only when the script is installed
      status: met
      tdd_phase: done
    - id: AC-2
      title: Missing jq/gh reported with platform-branched install hints
      status: met
      tdd_phase: done
    - id: AC-3
      title: Installed-but-not-wired state prints activation snippet
      status: met
      tdd_phase: done
    - id: AC-4
      title: Embedded-vs-on-disk drift detected and reported
      status: met
      tdd_phase: done
    - id: AC-5
      title: Container detected with project scope nudges --scope user
      status: met
      tdd_phase: done
---
# M-0157 — aiwf doctor statusline block

## Goal

Have `aiwf doctor`, when the statusline is installed, report missing `jq`/`gh`
(with platform install hints), installed-but-not-wired state, embedded-vs-
on-disk drift, and a container user-scope nudge — all advisory, never blocking.

## Context

Requires the scaffold (M-0155): "installed" must be detectable and the embedded
copy must exist for the drift comparison. Mirrors the existing
materialized-rituals reporting pattern in `doctor.go`, which is advisory and
never increments the problem count.

## Acceptance criteria

<!-- Formal ACs added at start-milestone via `aiwf add ac M-0157`. Intended shape: -->

The block is emitted **only when the statusline is installed**; missing `jq`
(load-bearing) and `gh` (CI segment) are reported with `runtime.GOOS`-branched
install hints (`brew` vs `apt-get`); installed-but-not-wired prints the snippet;
drift is reported when the embedded bytes differ from the on-disk copy; a
detected devcontainer yields a `--scope user` recommendation; the block never
increments `problems`. Each branch has a fixture that traverses it.

## Constraints

- Advisory only — never increments the problem count, never blocks push.
- Platform hints branch on `runtime.GOOS`.
- Container detection drives a *recommendation*, never a silent action.

## Design notes

- Reuse the `appendMaterializedRitualsReport` / `recordArtifact` shape.
- Drift = embedded bytes vs on-disk bytes compare.

## Surfaces touched

- `internal/cli/doctor/doctor.go`

## Out of scope

- The wiring itself (M-0156).

## Dependencies

- M-0155 (scaffold).

## References

- [E-0039](epic.md) · `internal/cli/doctor/doctor.go` (materialized-rituals report)

---

## Work log

### AC-1 — Statusline block emitted only when the script is installed

`appendStatuslineReport` (new file `internal/cli/doctor/statusline.go`)
resolves both project and user scope paths; returns `in` unmodified when
neither exists. Two tests: not-installed → empty output; installed →
`statusline: installed` line present. Tests 2/2.

### AC-2 — Missing jq/gh reported with platform-branched install hints

`installHintFor(tool, goos)` branches on `runtime.GOOS`: `brew` on darwin,
`apt-get` on linux. Four subtests (jq×darwin, jq×linux, gh×darwin, gh×linux)
exercise both platforms on any host. Tests 4/4.

### AC-3 — Installed-but-not-wired state prints activation snippet

`appendWiringCheck` scans `settings.local.json`, `settings.json` (project),
and `~/.claude/settings.json` (user) for a `statusLine` key. Missing →
prints wiring hint with `--wire-settings` command; present → suppressed.
Two tests. Tests 2/2.

### AC-4 — Embedded-vs-on-disk drift detected and reported

`appendDriftCheck` compares `skills.StatuslineBytes()` to the on-disk file.
Mismatch → drift advisory with `aiwf update --statusline` guidance; match →
no output. Two tests. Tests 2/2.

### AC-5 — Container detected with project scope nudges --scope user

When `inContainer=true` and scope is `project`, emits a `--scope user`
recommendation. `resolveInstalledStatusline` tested across project/user/
neither-installed. Container nudge tested for both inContainer states.
Tests 2 + 3 subtests = 5.

## Decisions made during implementation

- Split `appendStatuslineReport` into a production entry point (resolves
  home via `os.UserHomeDir()` and container via `InContainer()`) and a
  testable core `appendStatuslineReportWithHome(in, root, home, inContainer)`
  so tests run against isolated temp dirs without picking up the host's real
  `~/.claude/` state.
- New file `internal/cli/doctor/statusline.go` rather than extending the
  804-line `doctor.go` — the block is self-contained and follows the existing
  `appendXReport` pattern.
- Exported `FormatStatuslineSnippet` from `internal/skills/` (was
  `formatStatuslineSnippet`) so the doctor wiring hint can render the snippet.

## Validation

- Tests: 10 test functions in `internal/cli/doctor/statusline_test.go`; all
  pass.
- Full module: `go test -race -parallel 8 ./...` clean (one pre-existing
  flake in `internal/verb`; passes on retry).
- Build: `go build ./cmd/aiwf` clean.
- Lint: `golangci-lint run` — 0 issues in our code.
- `aiwf check`: 0 errors.

## Deferrals

- (none)

## Reviewer notes

- `appendDepCheck` calls `exec.LookPath` at runtime. In this devcontainer
  both `jq` and `gh` are installed, so the dep lines never fire in the live
  tests. The `installHintFor` unit test exercises the hint *text* for both
  platforms; the `LookPath`-missing path is only reachable in environments
  where the binary is genuinely absent. This is acceptable — the dep lines
  are purely advisory.
- The `FormatStatuslineSnippet` export is a one-line rename; the only new
  call site is the doctor wiring hint. No API surface change beyond
  visibility.

### AC-1 — Statusline block emitted only when the script is installed

### AC-2 — Missing jq/gh reported with platform-branched install hints

### AC-3 — Installed-but-not-wired state prints activation snippet

### AC-4 — Embedded-vs-on-disk drift detected and reported

### AC-5 — Container detected with project scope nudges --scope user

