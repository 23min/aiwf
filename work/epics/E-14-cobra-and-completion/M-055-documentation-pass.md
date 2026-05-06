---
id: M-055
title: Documentation pass
status: in_progress
parent: E-14
acs:
    - id: AC-1
      title: Each verb's --help has at least one example invocation
      status: met
    - id: AC-2
      title: No 'previously was' or migration notes in any user-facing docs
      status: met
    - id: AC-3
      title: README CLI section reflects the Cobra-shaped surface
      status: met
    - id: AC-4
      title: CLAUDE.md Go conventions references Cobra as standard CLI library
      status: open
---

## Goal

Final pass on user-facing docs. Each verb's `--help` reads cleanly with at least one example invocation; README explains the CLI surface and the completion install one-liner; `CLAUDE.md` § Go conventions names Cobra as the standard CLI library. No "previously was" / "renamed from" / migration notes anywhere in user-facing docs — the surface is described as it is, not as it changed.

## Approach

Systematic walk through every verb's help text, the README, and CLAUDE.md. Treat any reference to pre-Cobra behavior as a defect to delete. Help-text examples should be small but real — copy-pastable invocations the user can try, not pseudocode.

## Acceptance criteria

### AC-1 — Each verb's --help has at least one example invocation

### AC-2 — No 'previously was' or migration notes in any user-facing docs

### AC-3 — README CLI section reflects the Cobra-shaped surface

### AC-4 — CLAUDE.md Go conventions references Cobra as standard CLI library

