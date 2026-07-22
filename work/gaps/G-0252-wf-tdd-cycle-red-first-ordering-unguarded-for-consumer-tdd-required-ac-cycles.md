---
id: G-0252
title: wf-tdd-cycle red-first ordering unguarded for consumer tdd:required AC cycles
status: open
priority: high
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

**Mechanism, per [D-0047](../decisions/D-0047-contract-first-ac-timing-and-red-first-ordering-enforcement.md):**
`aiwf promote M-NNN/AC-N --phase red` refuses if the working tree's diff
against HEAD touches any non-test path (test-path classification via a glob,
the same "paths:" oracle pattern `aiwf-area` uses for area classification).
`--phase green` refuses unless the diff has grown to include a non-test path
since red. No new commit and no new trailer — the check inspects existing
working-tree state at each phase-promote call, so an honest cycle satisfies
it for free.

Earlier candidates, considered and rejected (see D-0047's Reasoning):

- Running the test suite at `--phase green` to confirm fail-then-pass: the
  strongest signal, but expensive and language-specific — the same objection
  D-0038 raised against `--evidence`'s `go test -list` toolchain coupling.
- An `aiwf-red-commit` SHA trailer: proves a commit exists and is reachable
  before green, not that *that commit's diff* contains the test — the same
  "existence not relevance" gap D-0038 named, one layer down.
- AC-scope cap as planning discipline: not mechanical by itself; still a
  reasonable complementary practice, but doesn't guard anything on its own.

What the diff-shape check does not close: whether the red-phase test
actually fails for the right reason. That judgment stays with
`wf-tdd-cycle`'s own instruction, `wf-vacuity`, and `wf-review-code` — the
same boundary D-0038 drew between mechanizable structural claims and
semantic judgment.

## Why it matters

CLAUDE.md's load-bearing principle is "the framework's correctness must not
depend on the LLM's behavior." Until red-first ordering is mechanical, a `met`
AC under `tdd: required` means "the LLM said it followed the discipline," not
"the discipline held." The diff-scoped coverage gate from G-0067 closes the
"untested branch ships" half of this concern; the ordering half remains open.
