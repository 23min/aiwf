---
id: M-0168
title: Corpus-wide mutate-hunt sweep over the kernel packages
status: in_progress
parent: E-0042
tdd: none
acs:
    - id: AC-1
      title: Kernel-core swept and every survivor dispositioned
      status: open
---
## Deliverable

A `mutate-hunt` (gremlins) sweep over the load-bearing kernel packages, with
every surviving mutant dispositioned: either a new or strengthened assertion
that now kills it, or a documented, justified exclusion (equivalent mutant,
unreachable/defensive branch). This is probe 1 of the G-0262 corpus work — the
mechanical half. Probe 2 (the assertion-shape judgment `wf-vacuity` does and
gremlins cannot) is M-0169.

## Scope

Per-package mutation runs, prioritized by blast radius (G-0262):

- `internal/entity` (the FSM and id allocator), `internal/gitops`,
  `internal/verb`, `internal/check` first — the kernel.
- Renderers and CLI surfaces second, as budget allows.

Use the repo's tuning: `--workers 1`, `--timeout-coefficient 15`. Read
survivors carefully — equivalent-mutant and unreachable-branch noise are common
false positives and are not chased; real survivors are concrete file:line
entries that warrant a test or a refactor.

## Approach

gremlins is the mechanical version of `wf-vacuity`'s probe 1 ("can the tests
fail at all?") and the stronger signal where it is wired up. Each package is
swept to a JSON report; every `LIVED` mutant is read and assigned a verdict.
`NOT COVERED` mutants are coverage gaps (a distinct axis from assertion
strength) and are noted but not the milestone's focus. The output is a single
committed survivor-disposition record — the objective floor for the strength of
the kernel's test assertions.

## Mechanical evidence

For each survivor dispositioned `kill`, a new or strengthened test lands and a
targeted gremlins re-run on the affected file shows the mutant `KILLED` (the
before/after efficacy delta is the proof). Survivors dispositioned `equivalent`
or `unreachable` carry a written justification. The new tests ride the existing
diff-scoped coverage gate; `make ci` stays green.

## Acceptance criteria

### AC-1 — Kernel-core swept and every survivor dispositioned

**Deliverable** — Run gremlins (`--workers 1 --timeout-coefficient 15`) over the
four kernel-core packages — `internal/entity`, `internal/gitops`,
`internal/verb`, `internal/check` — and, as budget allows, the second-tier
surfaces (renderers, CLI). Produce a committed survivor-disposition record that
accounts for **100% of the LIVED mutants** in every swept package's report,
each carrying a verdict: `kill` (a strengthening test added), `equivalent`
(semantically identical mutant, justified), or `unreachable` (defensive or
genuinely unreachable branch, justified). Packages left unswept are listed as
explicitly deferred with a one-line reason, so the boundary of what was probed
is unambiguous.

**Mechanical evidence** — Every `kill`-verdict mutant has a new or strengthened
test such that a targeted gremlins re-run on the affected file reports it
KILLED; the before/after efficacy delta is recorded in the disposition record.
The new tests ride the existing diff-scoped coverage gate, and `make ci` stays
green.
