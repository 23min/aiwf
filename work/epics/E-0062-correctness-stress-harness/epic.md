---
id: E-0062
title: Correctness stress harness
status: active
---

# E-0062 — Correctness stress harness

## Goal

An on-demand, real-git/real-process stress harness that exercises aiwf's
worktree, concurrency, and verb-sequencing correctness beyond what today's
example-based unit and integration tests cover, converting any violation it
finds into a reproducible gap.

## Context

This is epic 2 of 2 named in
[`docs/initiatives/robustness-correctness-stress-testing.md`](../../../docs/initiatives/robustness-correctness-stress-testing.md).
Epic 1, [E-0061](../E-0061-diagnostic-logging-and-correlation/epic.md)
(diagnostic logging and correlation), ships first: this epic's harness needs
a finished, merged `correlation_id` and `internal/logger` to instrument
against for RCA, not the in-progress version on an unmerged branch. aiwf's
schema tracks `depends_on` mechanically only at the milestone level, not the
epic level (`docs/pocv3/design/design-decisions.md`'s reference-field
table), so this epic's dependency on E-0061 is declared where it's actually
enforced: this epic's first milestone will carry `--depends-on M-0239`
(E-0061's capstone milestone, which itself depends on E-0061's other two).

**G-0212** (data-loss audit for verb composition) and **G-0269** (mutating
verbs lack a HEAD-drift guard against shared-worktree session races) are the
scenario catalogs this epic executes against; neither has a harness to run
in today.

**G-0211** (combinatorial verb-composition scenarios untested at
branch-choreography E2E) looked like it might belong here too, but turned
out to be stale: M-0159 (archived under E-0030) already built a real-git,
real-binary, combinatorial scenario-table framework
(`internal/cli/integration/branch_scenarios_*_test.go`,
`isolation_escape_*_scenarios_test.go`) covering essentially everything
G-0211 asked for, down to a deliberately documented list of exclusions
(worktree-branch mismatch is structurally untestable via real-git E2E for
that specific rule — `aiwf check`'s provenance walk is `git log HEAD`, not
`--all`, so a sibling worktree's commit is unreachable from another
worktree's check; the cherry-pick-silent scenario was blocked on G-0202,
which has since shipped and landed its scenario). G-0211 should be closed
separately, referencing that work — it is *not* in this epic's scope. The
one thread worth carrying forward is the `git log HEAD`-not-`--all`
reachability property itself: it's a real, documented architectural fact
about cross-worktree visibility, and this epic's multi-worktree scenario
tier is the right place to confirm its consequences are understood beyond
just the one rule that surfaced it.

## Scope

### In scope

- A harness driving the real, compiled `aiwf` binary as a subprocess
  throughout — never in-process function calls — against real, disposable
  git repositories.
- On-demand invocation only: a script/binary a human runs when they want
  it, never scheduled. Lives in its own tree, not part of the shipped
  `aiwf` binary.
- A streaming JSONL raw-report writer reusing E-0061's `O_APPEND`/
  one-`Write()`-per-record discipline, and a separate, abort-tolerant
  report-compose step that treats a truncated trailing line as "drop it,"
  never as a whole-report failure.
- A tiered scenario catalog:
  1. Single-process verb-sequence property tests.
  2. Multi-process / multi-worktree contention — `repolock` serialization,
     cross-worktree id-allocation races, and empirically confirming the
     consequences of `repolock`'s per-worktree lockfile scoping and the
     `git log HEAD`-not-`--all` cross-worktree reachability property
     (see *Context*).
  3. Fault injection (`kill -9`, disk-full, permission-denied) via
     externally observable proxies (probing `repolock`'s flock state,
     watching for `pathutil.AtomicWriteFile`'s temp file) — not a
     production-code failpoint hook, by default (see *Out of scope*).
  4. The named scenarios from G-0212 (reallocate races, edit-body races,
     archive-during-active-scope, force-push/cherry-pick vs.
     `acknowledge illegal`) and G-0269's HEAD-drift race.
  5. A concurrent-writer test proving E-0061's `O_APPEND` safety under real
     multi-process load, not just the package-level test built in E-0061
     itself.
- Cleanup discipline: a passing scenario cleans up its own temp repo(s); a
  failing one preserves them for post-hoc inspection.
- A `--repeat`/seed mechanism, since a single pass of a concurrency-shaped
  scenario proves nothing about a rare race.
- Manual triage: a violation the harness finds becomes a gap with a minimal
  regression test promoted into the normal, every-push suite.

### Out of scope

- Everything E-0061 builds (the logger, correlation id) — this epic consumes
  those, it does not rebuild them.
- Performance/throughput — the sibling initiative,
  [`check-performance-incremental-revwalk-cache.md`](../../../docs/initiatives/check-performance-incremental-revwalk-cache.md).
- Making the harness a blocking CI gate. It may later earn a
  `workflow_dispatch`-only entry point (still "on demand"), but that's a
  distribution question, not a decision this epic makes.
- A failpoint-style pause-hook mechanism in production code. External
  observation (see scenario tier 3) is the default for every race in this
  epic's catalog; a failpoint is deferred unless one of them (G-0269's
  read-HEAD-to-commit gap is the one candidate) proves unreproducible any
  other way after actually being attempted.
- The branch-choreography / isolation-escape combinatorial scenario surface
  G-0211 named — already covered by M-0159/E-0030's shipped work (see
  *Context*); re-scoping it here would duplicate existing coverage.
- Windows — `repolock` already refuses to run there.

## Constraints

- Harness code lives in its own tree (proposed: `internal/stresstest/` plus
  a thin `cmd/stresstest/` entry point), never scattered into production
  packages, and is never installed alongside `cmd/aiwf`.
- The raw-report writer reuses E-0061's `O_APPEND`/one-`Write()`-per-record
  discipline rather than inventing a second streaming primitive.
- Every scenario's oracle is a deterministic invariant check — a violation
  is something code asserts, not something a human eyeballs from output.
- A scenario that fails preserves its on-disk repo state; only a passing
  scenario cleans up after itself.

## Success criteria

- [ ] Every scenario in the catalog described in *Scope* has a deterministic
      pass/fail oracle.
- [ ] A run aborted mid-way (`SIGINT`/`SIGTERM`/`kill -9`) produces, when
      composed, a report that accurately reflects everything completed
      before the abort — no silent gap, no false "all clear."
- [ ] A violation the harness finds leaves enough behind (preserved repo
      state, a raw-report event, and a `correlation_id` into E-0061's
      diagnostic log) to be reproduced without re-running the whole
      campaign.
- [ ] Every violation found during this epic is triaged into a gap with a
      minimal regression test promoted into the normal, every-push suite.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| `go test -tags=stress -json` (reusing Go's streaming test-event format and `t.Run` reporting) vs. a fully bespoke `cmd/stresstest` driver | yes, for the skeleton milestone | Decided at the start of the harness-skeleton milestone |
| Local-only script vs. also a `workflow_dispatch` CI entry point | no | Revisit once the harness exists and someone asks for shared/CI-triggered runs |
| Report render format (markdown default; optional `--format=json`) | no | Decided during the skeleton milestone, following this repo's CLI convention |
| Whether `repolock`'s per-worktree lockfile scoping and the `git log HEAD`-not-`--all` cross-worktree reachability property are both fully-understood-and-safe, or hide a consequence nobody's confirmed | no, but the point of tier 2 | Empirically confirmed by the multi-worktree contention scenarios (tier 2), not assumed |
| Scenario catalog completeness — G-0274 (batch reallocate) and G-0157 (batched git subprocess fan-out) may suggest further scenarios | no | Revisited once the harness skeleton exists to receive them |

## Risks (optional)

| Risk | Impact | Mitigation |
|---|---|---|
| Fault-injection and contention scenarios are inherently timing-sensitive and could be flaky across different machines/CI environments | med | Oracles are invariant-based, not exact-timing-based; a scenario that doesn't hit its race window this run is a no-op, not a failure; `--repeat` buys statistical coverage rather than relying on one lucky run |
| The scenario catalog grows without bound as new ideas surface, turning the harness into an unmaintainable pile | low | Findings triage *out* of the harness into the normal suite once understood; the harness's job is discovery, not permanent regression-gating |
| G-0269's own HEAD-drift guard may not have landed by the time tier 4 runs its scenario | low | The scenario is expected-red until G-0269 lands a fix; that's a useful known-red case tied to the gap, not a harness defect |

## Milestones

- `M-0240` — Harness skeleton: driver loop (build-the-binary-under-test
  once, serial-by-default scenario iteration), the scenario interface, the
  streaming JSONL raw-report writer + separate compose step, cleanup
  discipline. · depends on: `M-0239` (E-0061's capstone milestone)
- `M-0241` — Property sequences and multi-worktree contention scenarios:
  single-process verb-sequence properties, plus multi-process/multi-worktree
  contention including the `repolock`-scoping and `git log HEAD`-reachability
  confirmations. · depends on: `M-0240`
- `M-0242` — Fault injection via external observation. · depends on:
  `M-0240`
- `M-0243` — Named scenarios from G-0212 and G-0269. · depends on: `M-0240`
- `M-0244` — Concurrent-writer test at scale; triage process. · depends
  on: `M-0241`, `M-0242`, `M-0243`
- `M-0249` — Scenario registry: wire `cmd/stresstest run` to the real
  catalog (G-0397: the CLI could only ever run the M-0240 placeholder
  scenario, never any of the 12 real ones the epic's own milestones
  built). Epic close moves here. · depends on: `M-0244`

## References

- [`docs/initiatives/robustness-correctness-stress-testing.md`](../../../docs/initiatives/robustness-correctness-stress-testing.md)
- [E-0061 — Diagnostic logging and correlation](../E-0061-diagnostic-logging-and-correlation/epic.md)
- G-0212 — data-loss audit for verb composition across kernel surface
- G-0269 — mutating verbs lack a HEAD-drift guard against shared-worktree session races
- G-0211 — combinatorial verb-composition scenarios untested at branch-choreography E2E (checked, stale, not in scope — see *Context*)
