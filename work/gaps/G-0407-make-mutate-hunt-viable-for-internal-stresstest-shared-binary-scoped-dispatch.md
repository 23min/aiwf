---
id: G-0407
title: 'Make mutate-hunt viable for internal/stresstest: shared binary + scoped dispatch'
status: open
---
## What's missing

G-0405 documented that `mutate-hunt` never finishes for
`./internal/stresstest` within its 90-minute job ceiling. This gap
captures the accepted strategy for actually fixing that, decided after
discussing and empirically checking several candidate directions.

**Root cause, confirmed by reading the code:** `sharedbinary_test.go`'s
`sharedTestBinary(t)` builds the real `aiwf` binary via `os.MkdirTemp`
+ `BuildBinary`, gated only by an in-process `sync.Once`. Gremlins
launches one fresh `go test` process per mutant; `sync.Once` state
doesn't survive across process boundaries, so **every single mutant
re-pays the full `aiwf` build cost** on top of its own test run. Since
gremlins only mutates files inside `internal/stresstest` — never
`cmd/aiwf` or anything in its dependency graph — the built binary is
never affected by whatever mutation is currently under test, so
reusing one externally-built binary across every mutant is safe.

## Decided direction (two parts, do both)

**1. Pre-built binary reuse.** Teach `sharedTestBinary` (and its
`sharedLockHolderBinary` sibling) to check an env var first — e.g.
`AIWF_STRESSTEST_PREBUILT_BINARY` — and use that path directly,
skipping the build, when it's set and points to an existing
executable. Fall back to today's build-from-scratch behavior when
unset, so every normal `go test` invocation (local dev, CI's regular
test job) is unaffected. Then have `mutate-hunt.yml` build the `aiwf`
binary once, before invoking gremlins, and export its path via that
env var. This removes the actual multiplier, not just the symptom.

**2. Scoped multi-dispatch via gremlins' own `--exclude-files`.**
Confirmed this flag exists (`-E, --exclude-files stringArray`,
filepath-regexp). Rather than one dispatch covering the whole package,
split into several dispatches, each excluding everything but one file
group (e.g. `verb_sequence*.go`, `concurrent_*.go`, the fault-
injection scenarios). `gh workflow run` can only set inputs a workflow
already declares, so this needs one small addition: an `exclude_files`
input on `mutate-hunt.yml`'s `workflow_dispatch` block, wired through
to gremlins' flag, plus the file-group split documented in the
workflow's header comment. Which groups and how many dispatches stays
an operator choice at invocation time. Independent of part 1 landing.

**Deferred, decide after 1 and 2 land:**

- **Raising `--workers` above 1.** The current `--workers 1` was
  tuned because concurrent workers thrash the build cache for
  `internal/entity` specifically (per the workflow's own header
  comment). Once part 1 removes the repeated-build cost for
  `internal/stresstest`, the remaining per-mutant workload is mostly
  I/O-bound (subprocess launches, git operations) and might
  parallelize safely — but this needs its own empirical check once 1
  and 2 are in place, not before (testing it now would still hit the
  build-cache contention the doc warns about).

**Explicitly rejected, do not revisit without new information:**

- **`gremlins unleash --diff <ref>` scoping.** This looked like the
  ideal fix on paper — mutate only what changed, matching the
  workflow's own stated cadence ("after a substantive test-suite
  change"). Tried it three ways locally (a raw commit SHA, `origin/
  main`, `HEAD~50`) against a package with a *confirmed* 64-file/
  9797-line diff (verified via plain `git diff --stat`); every attempt
  reported `Runnable: 0, Mutator coverage: 0.00%`. Either it's invoked
  wrong or broken in gremlins v0.6.0 — not confirmed working, not
  worth pursuing further without new information surfacing.
- **Just raising `timeout-minutes`.** Brute force, doesn't address
  the actual per-mutant waste, ties up a runner longer for no
  efficiency gain. Rejected in favor of the two real fixes above.

## Scope

Revised on closer inspection: neither part needs epic-level planning.

**Part 1** (prebuilt-binary reuse) is patch-shaped: an env-var-gated
branch in `sharedbinary_test.go`'s `sharedTestBinary`/
`sharedLockHolderBinary` (unset env var ⇒ byte-identical current
behavior) plus one new step in `mutate-hunt.yml`. Single-file per
change, mechanical, low blast radius despite the helper backing every
real-subprocess scenario in the package.

**Part 2** (scoped multi-dispatch) is also patch-shaped: one new
`exclude_files` input on `mutate-hunt.yml`'s `workflow_dispatch` block
wired to gremlins' `--exclude-files`, plus the file-group split
documented in the workflow's header comment. A CI-workflow-only
change, no production or test-helper code touched.

**Deferred `--workers` tuning** stays a small empirical follow-up once
both patches land and get exercised against `internal/stresstest` — a
re-run and observe, not a planning task.

## References

- G-0405 — mutate-hunt never finishes for internal/stresstest within
  the 90-min ceiling (the problem this gap's strategy resolves)
- G-0403 — mutate-hunt silently reports empty results for a
  subpackage + /... pattern (a separate, already-filed pkg_pattern bug
  in the same workflow)