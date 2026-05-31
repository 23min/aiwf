---
id: M-0157
title: aiwf doctor statusline block
status: in_progress
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
      status: open
      tdd_phase: green
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

- (pending)

## Decisions made during implementation

- (none)

## Validation

- (pending)

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — Statusline block emitted only when the script is installed

### AC-2 — Missing jq/gh reported with platform-branched install hints

### AC-3 — Installed-but-not-wired state prints activation snippet

### AC-4 — Embedded-vs-on-disk drift detected and reported

### AC-5 — Container detected with project scope nudges --scope user

