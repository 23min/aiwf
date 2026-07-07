---
title: 'Robustness: an on-demand correctness stress harness, and a retrace-ready diagnostic foundation'
status: captured
date: 2026-07-07
---

# Robustness: an on-demand correctness stress harness, and a retrace-ready diagnostic foundation

## Classifier note

This is an initiative document, following the precedent of
[`check-performance-incremental-revwalk-cache.md`](check-performance-incremental-revwalk-cache.md)
and [`id-lifecycle.md`](id-lifecycle.md): `initiative` is not yet an official
aiwf entity kind, so this lives under `docs/initiatives/` as a capture point
for a design that's concrete enough to scope but not yet built. This document
is closer to `id-lifecycle.md`'s shape than the revwalk-cache one — an
architecture proposal with real open questions, not a measured-and-prototyped
mechanism — because the harness itself doesn't exist yet.

## Why this exists

aiwf's whole value proposition is "guarantees about a markdown-and-frontmatter
project tree" (CLAUDE.md). A guarantee is only as strong as the adversarial
conditions it's been run against. The kernel has substantial unit and
property-test coverage built alongside the recent git/worktree and
performance work, but that coverage is example-based and mostly single-process
— it doesn't systematically exercise real concurrent `aiwf` invocations, real
git worktrees, real merges, or real process kills against real repositories.
Two open gaps already name this territory without a harness to execute
against it:

- **G-0212** ("data-loss audit for verb composition") catalogs known-risky
  scenarios — reallocate races, edit-body races, archive-during-active-scope,
  cross-process concurrent invocation, force-push/cherry-pick interaction
  with `acknowledge illegal` — and explicitly defers to "a future epic."
- **G-0269** is a live incident report, not a hypothetical: a parallel
  session's `git checkout` in a shared worktree caused an `aiwf promote` to
  land on the wrong branch, undetected until a later `git worktree list`.

Both describe the *what*; neither describes *how the kernel earns confidence
that this class of bug won't recur*. This initiative is that harness, plus
the diagnostic-log foundation that makes a failure it finds RCA-able rather
than a one-line "it failed once, couldn't reproduce."

## Scope

**Correctness only, not performance.** This initiative deliberately does not
touch throughput or latency — that's
[`check-performance-incremental-revwalk-cache.md`](check-performance-incremental-revwalk-cache.md)'s
territory, a sibling initiative already in flight. A stress run here may well
be slow; that's acceptable, because it never runs in the hot path (see
"On-demand, not scheduled" below).

### In scope

1. A multi-layer correctness stress harness driving real `aiwf` subprocesses
   against real, disposable git repositories — worktrees, merges, concurrent
   invocations, and fault injection.
2. The diagnostic-log foundation that makes anything the harness finds
   retrace-able: ADR-0017 (amended, see below) landing via G-0223, plus
   G-0232's correlation-id wiring.
3. A streaming, abort-safe raw-report mechanism and a separate report-compose
   step.
4. An on-demand invocation model — a script/binary a human runs when they
   want it, not a schedule.

### Out of scope (here)

- **Performance/throughput** of aiwf itself — sibling initiative, not this one.
- **Scheduled/cron execution** of the stress harness — explicitly rejected;
  see below.
- **Making the harness a blocking CI gate.** It may eventually earn a
  `workflow_dispatch`-only CI entry point (same shape as `mutate-hunt`), which
  is still "on demand," just triggerable by more than one person's local
  machine — but that's a distribution question, addressed as an open question
  below, not a decision this document makes.
- **Windows.** `repolock` already refuses to run there
  (`assertSupportedOS`); the harness follows the same platform floor as the
  rest of the kernel.

## Foundation: making findings retrace-able

A stress harness that finds a rare race and can't explain what happened is
barely better than not having one — "it failed once, four hours in, couldn't
reproduce" is not an actionable finding. Three pieces make a finding
retrace-able, and all three are logging/correlation work, not harness work:

### 1. ADR-0017, amended for concurrent-append safety

ADR-0017 (opt-in `slog`, default off, one daily file under
`$XDG_STATE_HOME`) is the right shape and stays as-is on the big decisions.
It had one unresolved interaction with this repo's own atomic-write
discipline: `internal/policies/atomic_write_chokepoint.go` currently bans any
raw `O_APPEND`/`O_WRONLY` file write in production code, mandating
`pathutil.AtomicWriteFile` (temp + fsync + rename) instead. That's the right
rule for entity files — it is the wrong rule for a shared, append-only,
multi-writer log, where temp+rename would mean read-whole-file →
append-in-memory → atomic-replace, unsafe under concurrent writers and
exactly backwards from what a log needs.

The ADR now specifies the actual safety property instead: `internal/logger`
opens its file handle with `O_APPEND`, and every record is written as one
`Write()` call. POSIX guarantees a single `write(2)` under `O_APPEND` is
atomic with respect to the file offset — concurrent processes appending to
the same daily file are serialized by the kernel, never interleaved or
torn. This is a hard implementation constraint (no buffered writer that could
split a record across two syscalls), not a suggestion, and
`atomic_write_chokepoint.go` needs an explicit, reasoned allowlist entry for
this one call site — pointing back to the ADR, not a bare `//nolint`-shaped
exemption.

This directly answers the original worry that drove this whole discussion
(would a stress harness's own logging distort the concurrency it's trying to
observe?): no separate collector process is needed, because the file append
itself is already lock-free and safe under real concurrent writers. The
harness's stress runs are in fact the natural place to *prove* this — see
"Diagnostic-log concurrent-writer test" in the scenario catalog below.

### 2. G-0232: correlation_id, resurrected

`render.Envelope.Metadata.correlation_id` is declared but dead — no caller
populates it. Wiring it (Cobra root mints a per-invocation id; every verb
threads it into the envelope; the same id becomes `logger.WithVerb`'s
`run_id`) is what turns "an envelope line" and "a diagnostic log line" into
one cross-referenceable fact about one invocation. Without this, RCA means
manually eyeballing timestamps across two unrelated streams; with it, it's
one grep.

This gap was already correctly scoped (envelope wiring, mutating-verb
metadata, the deferred `--trace` flag) — nothing about its content needs
revision, only its priority: it moves from "nice to have once someone asks
for cross-envelope correlation" to "a load-bearing prerequisite for RCA on
anything the stress harness finds."

### 3. A third correlation layer: the harness's own raw-report stream

The harness needs an event stream distinct from both of the above — ADR-0017
logs answer *what did this one `aiwf` invocation do internally*; the raw
report answers *what did the harness do, and what did it observe*, across
potentially many invocations and processes making up one scenario. Each raw
report event should carry the harness's own scenario/step identifiers **and**
the `correlation_id` of every `aiwf` subprocess spawned in that step. That
shared key is what lets a human (or the report-compose step) walk from "step
14 of scenario `worktree-race-3` failed" to "here are the three `aiwf`
invocations involved, and here is each one's diagnostic-log trace."

The raw-report writer should reuse the exact same discipline as
`internal/logger`'s file handle (§1 above) — `O_APPEND`, one `Write()` per
event — rather than inventing a second streaming primitive. One safe,
tested, append-only writer serving both the production diagnostic log and
the harness's report stream is a straightforward application of this repo's
own single-source-of-truth principle, not a new abstraction for its own sake.

## The harness

### On-demand, not scheduled

The harness is a script/binary a human runs when they want it — not a cron
job, not a periodic workflow. This repo already has precedent for
heavy, non-gating, manually-triggered tooling: `mutate-hunt` is
`workflow_dispatch`-only, and `make diag-aiwf` / `bin/aiwf-diag` is a
worktree-scoped dev tool that's built and invoked by hand, never part of the
shipped binary. The stress harness follows the same shape: it lives in this
repo but is not installed via `go install` alongside `cmd/aiwf`, and nothing
about it runs automatically.

Proposed home: either a dev-only `cmd/stresstest` Go binary (same tier as
`bin/aiwf-diag`) or a `go test -tags=stress` suite under
`internal/stresstest/` — the fork is discussed in "Orchestration mechanics"
below and left open — invoked either way via a `make stress` target. It
drives the real, currently-built `aiwf` binary as a subprocess throughout —
never in-process function calls — because the entire point is catching
OS-level behavior (real `flock` contention, real process kills, worktrees as
directory-scoped `HEAD`) that calling Go functions directly cannot exercise.

### Orchestration mechanics

**Driver loop.** Build the binary under test once per run — worktree-scoped,
same discipline as `make diag-aiwf`; never trust whatever's on `PATH`.
Iterate the scenario catalog serially: the concurrency under test happens
*within* a scenario, between simulated actors sharing one disposable repo,
not *across* scenarios, so nothing is gained by running scenario A and
scenario B concurrently — that would only complicate the report stream for
no correctness benefit, given performance is explicitly out of scope. Each
scenario gets its own repo(s)/worktree(s) and, per the repeat knob below, its
own sequence of attempts, aggregated into one scenario-level raw-report
entry.

**Scenario shape.** Each scenario is Go code, not a declarative spec — the
catalog is fixed and authored by us, not something an operator composes at
runtime, so a DSL would be premature abstraction. A small interface
(`Setup(dir) actors`, `Run(harness)`, `Verify(dir) []Violation`) is enough.
One real fork, left open below: build scenarios as ordinary Go tests under a
`stress` build tag and drive them via `go test -tags=stress -json`, letting
that already-streaming, already-abort-tolerant JSON-event format *be* the raw
report (no bespoke runner; `t.Run` gives sub-scenario reporting and `-run`
filtering for free) — versus a fully bespoke `cmd/stresstest` driver. Nothing
about `go test` prevents a test function from spawning real subprocesses and
sending real signals, so the `go test` route costs little and buys existing
tooling for free.

**Creating real concurrency — two mechanisms, not one.**

- *Loose/statistical races*, for questions like "does `repolock` actually
  serialize" or "does cross-worktree id allocation ever produce a real
  duplicate": start every actor's subprocess as close together as the
  scheduler allows (a shared "go" channel; each actor's goroutine calls
  `cmd.Start()` right after it closes), then repeat. No synchronization
  beyond "start together" is needed — real scheduler jitter does the work,
  and repetition buys statistical coverage. Covers most of scenario tier 2.
- *Directed races*, for a narrow time-of-check-to-time-of-use window (G-0269's
  read-HEAD → … → commit gap): blind timing rarely lands in a fast,
  in-process window without heroic repeat counts. Prefer an **externally
  observable proxy that needs zero production-code changes** wherever the
  window has one: `repolock`'s flock state is probeable from outside (a
  non-blocking `flock` attempt from the harness tells it exactly when
  another process holds the lock); `pathutil.AtomicWriteFile`'s
  temp-file-then-rename leaves a filesystem trace a watcher (`fsnotify` or
  polling) can catch mid-flight. Both let the harness pause, kill, or launch
  a competing actor at the right instant against the real, unmodified
  binary. Reserve an actual failpoint hook (an env-var-gated pause point,
  no-op unless set — the same opt-in shape as `AIWF_LOG`) only for windows
  with no external proxy; G-0269's gap is the one candidate so far, since
  nothing touches disk during it. That's a real decision, not an
  implementation detail — a failpoint means a small test-only seam in
  production code, which cuts against this repo's general aversion to
  speculative hooks even though it's well-precedented (etcd/TiKV's
  `failpoint`, FoundationDB's simulation hooks) for exactly this failure
  class. See Open Questions.

**Fault injection.** Same external-observation preference as directed races:
poll for the lock being held or the temp file existing, then `SIGKILL` at
that instant, rather than blind random-delay-then-kill. Random delay is the
fallback where no clean external proxy exists.

### Two output streams, two different questions

- **Diagnostic log** (ADR-0017, emitted by the `aiwf` subprocesses under
  test) — "what did this invocation do internally."
- **Stress report** (emitted by the harness itself) — "what did the harness
  run, and what did it find." These are deliberately separate: the harness
  isn't instrumenting itself via `AIWF_LOG`, it's an independent observer
  recording its own scenario-level narrative, cross-referenced to the
  diagnostic log via `correlation_id` as described above.

### Streaming raw report, composed later — abort-safety

This is the concrete design constraint from the original ask ("the test can
be aborted without losing all the data it has been gathering"):

- The raw report is JSONL — one line, one `Write()` call, per completed unit
  of work (scenario start, a step's result, an assertion outcome). Never
  buffered across a whole scenario or the whole run; each line is flushed to
  disk before the harness starts the next unit of work.
- On `SIGINT`/`SIGTERM`, the harness finishes its current step's write and
  stops starting new scenarios, rather than tearing down mid-write. But it
  must also tolerate a hard `kill -9` leaving the *last* line truncated —
  that's an accepted case, not a bug, and the compose step treats a
  malformed trailing JSON line as "truncated, drop it," never as a
  whole-report failure. This is the same "errors are findings, not parse
  failures" posture `aiwf check` already holds toward the entity tree,
  applied to the harness's own output.
- **Report composition is a separate step**, reading whatever the raw JSONL
  file contains (complete or partially aborted) and rendering a
  human-readable summary — pass/fail per scenario, an invariant-violation
  list with enough detail (and `correlation_id`s) to reproduce and grep the
  diagnostic log, following this repo's own "human-readable default,
  `--format=json` secondary" CLI convention.
- Splitting capture from render is deliberate beyond abort-safety: a stress
  campaign might run for hours; the report format can be improved and
  old raw-report files re-rendered without re-running the campaign.

### Cleanup discipline

A scenario that passes cleans up its own temp repo(s)/worktree(s). A scenario
that fails **preserves** them — the actual on-disk git state at the moment of
failure is itself RCA material, separate from both log streams, and deleting
it on failure would throw away the one artifact a human might most want to
open and poke at by hand.

### Repetition and reproducibility, because real concurrency is non-deterministic

A single pass of a contention scenario proves nothing about a rare race. The
harness needs a `--repeat N` (or `--duration`) knob to hammer
concurrency-shaped scenarios repeatedly; "ran once, passed" and "ran once,
never saw the race" look identical from the outside. Each attempt logs the
random seed it used (actor-start jitter, chosen kill delay where randomized,
repeat index) into its raw-report event — a violation found on, say, repeat
#47 should be replayable by re-running with that seed, which narrows the
search even though true OS scheduling isn't fully deterministic.

## Scenario catalog (tiered)

1. **Single-process verb-sequence properties** — extends the existing
   `internal/entity/transition_property_test.go` pattern: generate random
   (legal and illegal) verb sequences against one real temp repo via the real
   binary, assert invariants after each step (`aiwf check` clean or
   expected-findings-only; exactly one commit per mutation; FSM legality).
2. **Multi-process / multi-worktree contention** — real concurrent `aiwf`
   subprocesses: same working copy (repolock must serialize, never produce a
   duplicate id), sibling worktrees of one repo (id collisions are expected
   and must surface as a `check` finding resolved by `aiwf reallocate` — this
   is accepted-eventual by design, not a bug to prevent), and the G-0269
   HEAD-drift race scripted directly (this one should currently fail — a
   useful known-red case tied to that gap until it lands a guard).
3. **Fault injection** — `kill -9` mid-verb, both during and outside the
   `repolock` hold; disk-full and permission-denied paths on the atomic-write
   route. Assert: the lock releases via kernel fd cleanup, no half-written
   entity file survives, no lockfile permanently blocks a future run.
4. **Named scenarios from G-0212** — reallocate races, edit-body races
   (concurrent `aiwf edit-body` on the same entity from different worktrees),
   archive-during-active-scope, force-push/cherry-pick interaction with
   `acknowledge illegal`.
5. **Diagnostic-log concurrent-writer test** — spawn N real `aiwf`
   subprocesses with `AIWF_LOG=debug` pointed at one shared daily log file;
   assert every line is well-formed, every `run_id` is present exactly once,
   and no line is torn or interleaved. This is the harness proving out
   ADR-0017's own §5 safety claim under real load — not hypothetical, since
   the harness exists anyway.

## Success criteria

"Bulletproof" cashes out to concrete, checkable properties, not a feeling:

- Every scenario in the catalog has a deterministic pass/fail oracle (an
  invariant check), not a human eyeballing output.
- A run that finds a violation leaves enough behind (preserved repo state +
  raw report event + `correlation_id` into the diagnostic log) that the
  violation is reproducible without re-running the whole campaign.
- An aborted run's raw report, composed as-is, accurately reflects everything
  completed before the abort — no silent gap, no false "all clear."
- Every finding the harness surfaces is triaged into a gap with a minimal
  regression test promoted into the normal (fast, every-push) suite — the
  stress harness's job is discovery, not standing in as the permanent
  regression gate for a bug once it's understood.

## Open questions

- **Distribution: local-only script, or also a `workflow_dispatch` CI
  entry point?** Both are "on demand" in the sense that mattered for this
  decision (neither is a schedule); the CI form makes it runnable by anyone
  without a local Go toolchain and captures artifacts centrally, at the cost
  of a second invocation path to keep in sync with the local one. Not
  resolved here.
- **Driver mechanism: `go test -tags=stress -json`, or a bespoke
  `cmd/stresstest` binary?** The `go test` route reuses an already-streaming,
  already-abort-tolerant JSON event format plus `t.Run`/`-run` for free; a
  bespoke driver gives full control over process orchestration and report
  shape at the cost of building and maintaining it. See "Orchestration
  mechanics."
- **Failpoint-style hooks for narrow directed races, or external-observation
  and statistics only?** External proxies (probing `repolock`'s flock state,
  watching for `pathutil.AtomicWriteFile`'s temp file) cover most races with
  zero production-code changes. G-0269's read-HEAD-to-commit gap has no such
  proxy; closing it deterministically means a small, opt-in, `AIWF_LOG`-shaped
  test-only seam in production code. Worth deciding deliberately rather than
  defaulting into it — see "Orchestration mechanics."
- **Report render format** — markdown (git-diffable, pasteable) is the
  obvious default per this repo's CLI conventions; whether a `--format=json`
  machine-readable summary is worth adding alongside it (for, say, a future
  dashboard) is open until something actually consumes it.
- **Repolock's cross-worktree scope** — confirmed-but-unverified-under-load:
  a linked worktree's `.git` is a *file*, not a directory, so
  `repolock.lockfilePath` falls through to a different lockfile per worktree
  rather than sharing one with the main checkout. That appears intentional
  given the accepted-eventual id-collision design (CLAUDE.md's "Id-collision
  resolution at merge time"), but scenario tier 2 above is what actually
  confirms the *consequence* of that scoping — a stray duplicate id — is
  always caught and always resolvable, not just assumed to be.
- **Auto-filing gaps from findings** — whether the compose step should open
  a draft gap automatically for a new invariant violation, or whether that
  stays a manual triage step. Leaning manual (auto-filing an aiwf entity is a
  mutation, and mutations from an unattended tool sit awkwardly against this
  repo's per-mutation approval-gate discipline) but not decided.
- **Scenario catalog completeness** — the tiered list above is a strong
  starting set, not a closed one; G-0274 (batch reallocate) and G-0157
  (batched git subprocess fan-out) may each suggest further scenarios once
  the harness skeleton exists to receive them.

## Relationship to other work

- **G-0212** — this initiative is the harness that gives that gap's catalog
  somewhere to run; the catalog above absorbs its named scenarios directly.
- **G-0269** — becomes both a scenario (tier 2) and, once its own guard
  lands, a regression test the harness's fault-injection tier exercises
  going forward.
- **G-0223 / ADR-0017** — the logging implementation gap; this initiative
  adds the concurrent-append constraint (§1 above) as part of ratifying that
  ADR.
- **G-0232** — the correlation-id wiring; reclassified here from "nice to
  have" to "prerequisite for RCA."
- **`check-performance-incremental-revwalk-cache.md`** — the sibling
  initiative covering performance; deliberately out of scope here, cited so
  the correctness/performance split reads as a decision, not an omission.
- **`mutate-hunt`** — a different axis entirely (does the *test suite* catch
  injected code mutations — a test-quality signal), not a duplicate of this
  harness (does the *system* behave correctly under real adversarial
  conditions — a system-correctness signal).

## Provenance

Emerged from a 2026-07-07 discussion prompted by wanting aiwf to be
"bulletproof" after a period of substantial git-layer and performance work.
The discussion first evaluated and rejected a stdout-piped-to-a-batching-
collector-process design for diagnostic logging (overengineered for a
short-lived CLI tool; ADR-0017's file-based approach already covers the real
need, once its one gap — concurrent-append safety versus this repo's
atomic-write chokepoint — is closed), then scoped the stress-testing side
against the existing G-0212/G-0269 gaps, and folded in the specific
streaming/abort-safety and on-demand-invocation requirements from that
conversation. No code has been written yet; this document precedes epic/
milestone planning.
