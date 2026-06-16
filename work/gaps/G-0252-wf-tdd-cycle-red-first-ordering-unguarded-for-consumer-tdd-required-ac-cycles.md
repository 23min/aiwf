---
id: G-0252
title: wf-tdd-cycle red-first ordering unguarded for consumer tdd:required AC cycles
status: open
discovered_in: M-0066
---
## What's missing

A mechanical RED-first ordering guard for consumer-milestone `tdd: required`
acceptance-criterion cycles. The `wf-tdd-cycle` skill asks the assistant to
write a failing test before the implementation, but nothing enforces the
ordering: a cycle can reach AC `met` with `tdd_phase: done` while the test was
in fact written after (or alongside) the implementation. The standing checks
(`aiwf check`, `acs-tdd-audit`) confirm a test *exists* for a `done` AC; they
cannot tell that the test *preceded* the code.

This was split out of G-0067 when its sub-goal (a) — the diff-scoped coverage
gate for aiwf's own Go code — landed. Sub-goal (a) made branch coverage
mechanical for the kernel itself; this gap is the remaining, orthogonal
concern: ordering enforcement for consumer AC cycles.

Candidate mechanisms (carried over from G-0067's original list):

- `aiwf promote --phase green` runs the test suite and refuses any new test
  that does not fail-then-pass: the kernel checks that some commit between the
  AC's add and the green-promote contains a test that, run against the parent
  of the impl commit, would fail. Real chokepoint, but expensive (runs tests
  at promote time) and language-specific.
- An `aiwf-red-commit` trailer on the AC: `aiwf promote --phase red --commit
  <SHA>` records the failing-test commit; promote-to-green refuses unless that
  SHA is reachable from the parent of the green commit. Pins ordering as a
  deliberate act without inspecting test content.
- AC-scope cap as planning discipline: when an AC's expected diff exceeds ~50
  lines of impl, split it — large ACs drift off red-first regardless of how
  strict the skill is.

## Why it matters

CLAUDE.md's load-bearing principle is "the framework's correctness must not
depend on the LLM's behavior." Until red-first ordering is mechanical, a `met`
AC under `tdd: required` means "the LLM said it followed the discipline," not
"the discipline held." The diff-scoped coverage gate from G-0067 closes the
"untested branch ships" half of this concern; the ordering half remains open.
