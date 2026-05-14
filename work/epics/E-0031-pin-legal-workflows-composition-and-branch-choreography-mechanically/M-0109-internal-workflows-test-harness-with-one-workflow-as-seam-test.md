---
id: M-0109
title: internal/workflows/ test harness with one workflow as seam test
status: draft
parent: E-0031
depends_on:
    - M-0108
tdd: required
---
## Goal

Build the `internal/workflows/` test harness — temp git repo helper, aiwf binary build helper, structured-markdown parser to read workflow definitions from the spec, runner that executes a workflow against the harness. Land one workflow end-to-end as the seam test that proves the pattern. Multi-branch fixture support is in-scope from day one.

## Context

M-0108 ships the spec; this milestone produces the mechanical consumer that turns the spec from prose into a chokepoint. Without this layer the spec is documentation; with it, drift in either direction (spec or skills) shows up as a test failure.

## Approach

Test package at `internal/workflows/`. Builds the aiwf binary into a tempfile (per CLAUDE.md "Test the seam, not just the layer" — binary-level integration). Each test sets up a fresh temp git repo with `aiwf init`, exercises a workflow by invoking the binary as a subprocess, and asserts the resulting tree state (frontmatter, status, commit trailers). Multi-branch fixtures use `git worktree add` or `git checkout -b` inside the temp repo to simulate the allocate-on-main → branch → merge contract. One workflow (likely `add-gap` for simplicity) lands as the seam test in this milestone; the rest follow in M-0110.

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac M-0109 --title "..."`. -->

## Surfaces touched

- `internal/workflows/` (new package)
- `internal/workflows/testdata/` (fixture helpers)

## Out of scope

- Tests for every workflow in the spec (M-0110)
- Drift-prevention test (M-0111)
- Fuzz harness (M-0112)

## Dependencies

- M-0108 (spec must exist to drive tests)
