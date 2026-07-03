---
id: G-0293
title: Promote tdd_phase live, not in a burst at milestone wrap
status: addressed
prior_ids:
    - G-0292
discovered_in: M-0189
addressed_by_commit:
    - 76829a69
---
## Problem

This gap has two facets, both rooted in `aiwfx-start-milestone` step 8's
"defer all commits to wrap" guidance.

### Facet 1 — phase ladder bursted at wrap destroys TDD evidence

The `tdd_phase` ladder (`red -> green -> refactor -> done`) on a `tdd: required`
milestone's ACs carries signal **only when promoted contemporaneously**. The
value is temporal: `aiwf history M-NNN/AC-N` is supposed to show `red` (failing
test written) with a timestamp *before* `green` (code makes it pass). That gap in
time is the evidence the test existed and failed first.

When all transitions are stamped in a burst at milestone wrap — all carrying the
same timestamp — there is zero evidence the test came first. The trail is
indistinguishable from "wrote the code, wrote the test after, back-stamped the
ladder." It records the shape of TDD while proving none of the substance. The
ladder becomes ceremony.

### Facet 2 — implementation code is never staged or committed

The wrap path never commits the implementation source. `wf-tdd-cycle`'s phase
promotes commit only the milestone spec's `acs[]` frontmatter, not the `.go`
files. `aiwfx-start-milestone` step 8 says *"do not commit the implementation yet
— wrap bundles the implementation,"* but `aiwfx-wrap-milestone` step 7 stages
**only the spec** (`git add work/epics/.../M-NNN-<slug>.md`). Followed literally,
the implementation code is never staged or committed: the milestone -> epic merge
carries the phase-promote commits and the spec, but not the code. Meanwhile step
6's work-log wants a `commit <SHA>` per AC that does not yet exist. Three
statements, no coherent commit model among them.

## Root cause

`aiwfx-start-milestone`'s "defer all commits to wrap" guidance (step 8) is
correct for `tdd: none` (no phases to lose, code committed once is fine) but,
carried into a `tdd: required` milestone, both collapses the phase promotes into
a wrap-time burst and leaves the implementation code uncommitted because the wrap
stages only the spec.

## Deeper issue (honor-system vs. mechanical)

Even *live* phase promotes are not a mechanical guarantee. The kernel's
`acs-tdd-audit` only enforces "`met` requires `tdd_phase: done`" — it never checks
that `red` preceded `green` by a real interval, nor that the test actually failed
at `red`. So the ladder's meaningfulness rests on operator honesty, which bumps
against "framework correctness must not depend on LLM behavior." The actual
mechanical TDD floor in this repo is the diff-scoped coverage gate (G-0067):
every changed line must be tested or the merge fails. The phase ladder is the
soft narrative layer on top.

## Decision (Model 1)

Commit code incrementally per AC, with phase promotes firing live:

- `aiwfx-start-milestone` step 6 commits each AC's implementation code on the
  milestone branch as the AC completes; the work-log `commit <SHA>` becomes real
  and bisectable.
- Phase promotes fire **live during the `wf-tdd-cycle`**, at the moment each
  transition actually happens — never bursted at wrap. The declared-sequence gate
  (the wrap terminal-sequence batcher) explicitly excludes phase promotes for
  exactly this reason: their signal is their timing.
- `aiwfx-start-milestone` step 8 and `aiwfx-wrap-milestone` step 7 are rewritten
  to match: the implementation is already committed per-AC; the wrap adds the
  wrap-side spec edits + the closure promote + the merge.

Optional complementary follow-up (not committed here): a `check` finding that
warns when an AC's `red`/`green`/`done` transitions all land in one commit (or
within a tiny timestamp window), making the honor-system partly mechanical.

## Discovered in

M-0189 (`worktree.dir` config knob). The ACs were seeded `red`, the
implementation done with genuine test-first discipline (each test written first
and observed red, then green), but the phase promotes were deferred — so the
ladder would have been stamped at wrap. M-0189 was closed with the ladder stamped
plus a work-log note that the phases are retroactive (the test suite + 100% diff
coverage are the real evidence). This gap captures the systemic fix so the next
`tdd: required` milestone does it right. Facet 2 (implementation never staged) and
the Model-1 decision were added after an audit of the milestone rituals.
