---
id: M-0134
title: 'CLAUDE.md DO/DON''T refresh: container primary, macOS host fallback'
status: in_progress
parent: E-0035
tdd: required
acs:
    - id: AC-1
      title: 'CLAUDE.md test-running guidance: container-primary, macOS-host fallback'
      status: met
      tdd_phase: done
---
## Goal

After M-0132 + M-0133 landed, the devcontainer is the actual default
dev surface — `make ci` runs green from VS Code's "Reopen in
Container" without remembering any macOS-specific discipline. But
`CLAUDE.md`'s `### Testing` area still positions the macOS host
wrapper (`scripts/sign-and-run.sh`, the `-parallel 8` cap, the
G-0127 / G-0128 / G-0133 diagnostic discipline) as the primary path,
and ends with "Structural fix (Linux devcontainer) is parked."
The structural fix isn't parked — it shipped. Refresh CLAUDE.md
so the container-primary path leads and the macOS-host wrapper is
clearly demoted to a fallback for the rare case where the operator
must run tests on the host.

## Approach

Two coordinated changes to `CLAUDE.md`'s `## Go conventions →
### Testing` area, landing as a single AC with a mechanical
structural assertion.

1. **Rewrite the existing `#### Running tests on macOS — use the
   wrapper` subsection.** Split it into two adjacent
   subsections: `#### Running tests in the devcontainer (primary)`
   leads (anything goes — `make test`, bare `go test`, focused
   `go test -run TestX`, all work because Linux has no signing
   requirement; the section points at `.devcontainer/README.md`
   for operational setup and does *not* duplicate operational
   instructions, single source of truth). `#### Running tests on
   macOS host (fallback)` follows, carrying the existing wrapper
   discipline content — `sign-and-run.sh`, the `GOFLAGS` export
   pattern, the Do / Don't list, and the G-0127 / G-0128 / G-0133
   diagnostic references — preserved as the "if you must run on
   macOS" path. The "Structural fix (Linux devcontainer) is parked."
   sentence is deleted. The "Defaults, not a chokepoint. Nothing
   catches a bare `go test`" caveat is scoped explicitly to the
   macOS-fallback subsection (it doesn't apply in the container,
   where bare `go test` just works).

2. **Mechanical structural assertion** in `internal/policies/`
   walks `CLAUDE.md`'s markdown heading hierarchy and pins the new
   structure: the devcontainer subsection appears before the
   macOS-host subsection under `### Testing`; the devcontainer
   subsection's body indicates explicitly that no wrapper is
   required on Linux; the macOS-host subsection's body still
   mentions `sign-and-run.sh`, `make test`, and at least one of
   the G-0127 / G-0128 / G-0133 diagnostic ids; the literal
   "Structural fix (Linux devcontainer) is parked." phrase is
   absent from the entire file. Assertions resolve to specific
   parsed sub-trees under each named heading, not flat substring
   matches against the file body — per CLAUDE.md *"Substring
   assertions are not structural assertions."*

Design choices locked at scoping time (kept for reviewer context):
- **Hard demotion**, not soft mention — macOS guidance moves to a
  sub-subsection clearly labeled "fallback," not a leading
  paragraph that warns and then continues.
- **The "Defaults, not a chokepoint" footgun is noted as
  known-limitation, not fixed here** — adding a pre-commit guard
  against bare `go test` is a separate gap candidate.
- **Cross-references point, don't duplicate** — the new
  container-primary subsection links to `.devcontainer/README.md`
  for operational details; no duplication.
- **Tight scope: test-running section only** — the existing
  Operator-setup → Devcontainer subsection and the Go-conventions
  intro are not touched. Broader cleanup is a follow-up gap if
  it surfaces.

## Acceptance criteria

ACs land via `aiwf add ac M-NNNN`.

### AC-1 — CLAUDE.md test-running guidance: container-primary, macOS-host fallback

**Pass criterion**: `CLAUDE.md`'s `## Go conventions → ### Testing`
area carries two adjacent subsections in this order:
`#### Running tests in the devcontainer (primary)` first,
`#### Running tests on macOS host (fallback)` second.

The devcontainer subsection's body indicates explicitly that no
test wrapper is required on Linux (any `go test` invocation works
because there is no signing requirement), references
`.devcontainer/README.md` for operational setup, and does not
duplicate the operational instructions. The macOS-host subsection's
body carries the demoted wrapper-discipline content —
`scripts/sign-and-run.sh`, the `make test` / `make test-race` /
`make coverage` Do list, the bare `go test` Don't, the `GOFLAGS`
export pattern for focused runs outside `make`, the "Defaults, not
a chokepoint" caveat (scoped here, not floating at file scope), and
at least one of the G-0127 / G-0128 / G-0133 diagnostic gap ids.
The literal string "Structural fix (Linux devcontainer) is parked."
is absent from the entire file.

A Go test under `internal/policies/` walks `CLAUDE.md`'s markdown
heading hierarchy and pins the above structurally — assertions
resolve to specific parsed sub-trees under each named heading, not
flat substring matches against the file body (per CLAUDE.md
*"Substring assertions are not structural assertions"*). The test
runs against the live `CLAUDE.md` (not a fixture) following the
`sharedRepoTree`-style pattern already in `internal/policies/`.

**Edge cases**: tolerate `\r\n` line endings and trailing whitespace
on heading lines; if the test cannot locate either subsection,
fail with a message naming the expected heading shape so the
operator knows what to add. Don't touch the existing `## Operator
setup → ### Devcontainer` subsection (separate concern; scope-creep
risk). Preserve the existing Do / Don't formatting (markdown bold
labels) in the fallback block. The wrapper-discipline content
stays substantively unchanged — only its position and framing
move.

**Code references**: `CLAUDE.md` (the test-running subsection in
the Go-conventions Testing area, currently `#### Running tests on
macOS — use the wrapper`); new policy + test under
`internal/policies/` (e.g., `m0134_claude_md_test_section.go` +
`_test.go`, following the `m0132_*` precedent for CLAUDE.md
section assertions).

