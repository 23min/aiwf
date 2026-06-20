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

A `mutate-hunt` (gremlins) sweep over the load-bearing kernel packages, with the
surviving mutants triaged and the high-value real survivors killed. This is
probe 1 of the G-0262 corpus work — the mechanical half. Probe 2 (the
assertion-shape judgment `wf-vacuity` does and gremlins cannot) is M-0169.

## Scope

Per-package mutation runs, prioritized by blast radius (G-0262):

- `internal/entity` (the FSM and id allocator), `internal/gitops`,
  `internal/verb`, `internal/check` — the kernel.

Use the repo's tuning: `--workers 1`, `--timeout-coefficient 15`.

The sweep surfaced far more survivors than the milestone anticipated (entity 18,
gitops 15, verb 89 LIVED, plus check). The milestone scope's own guidance — "read
survivors carefully; equivalent-mutant and unreachable-branch noise are common
false positives and are not chased" — is the operative principle at this volume:
a large fraction is boundary noise (`<`→`<=` after a `!=` guard, capacity hints,
no-op `>` vs `>=` max-updates). Mutation testing is inherently slow (one suite
run per mutant) and `--workers 1` is forced (higher counts time out on this
repo), so neither more workers nor CI matrix-parallelism reduces the irreducible
cost, which is the human triage of survivors, not the sweep wall-clock.

## Approach — value-tiered

Most test-strength gain for least kill-test churn:

1. Record the per-package gremlins **efficacy baseline** — the objective floor,
   the milestone's stated Outcome.
2. **Kill** the high-value, low-cost real survivors: concrete logic gaps in pure
   functions and core paths, where a small test or table extension flips the
   mutant. Each kill is confirmed by injecting its *exact* mutation, running the
   focused test (red), and reverting — faster and more targeted than a full
   gremlins re-run.
3. **Document** the equivalent-mutant and boundary-noise survivor classes by
   pattern, naming the per-package survivor counts so the un-itemized remainder
   is visible rather than silently dropped.

This is a deliberately tiered pass, not a claim that every survivor was
itemized line-by-line.

## Mechanical evidence

Each killed survivor has a test that goes **red** when its exact mutation is
injected into the implementation (recorded in the disposition record) and green
on real code. `make ci` stays green with the new tests in place.

## Acceptance criteria

### AC-1 — Kernel-core baselined; high-value survivors killed; noise documented

**Deliverable** — A committed survivor-disposition record covering the
kernel-core sweep that: (a) records the per-package gremlins **efficacy
baseline** (entity/gitops/verb/check — the milestone's Outcome floor); (b)
**kills** the high-value, low-cost real survivors (concrete logic gaps in pure
functions and core paths); (c) **documents** the equivalent-mutant and
boundary-noise survivor classes by pattern, naming the per-package survivor
counts so the un-itemized remainder is visible. Deliberately value-tiered — most
strength for least churn — *not* a 100%-individual-disposition claim.

**Mechanical evidence** — Each killed survivor has a test that goes red when its
exact mutation is injected (recorded in the record) and green on real code;
`make ci` stays green.
