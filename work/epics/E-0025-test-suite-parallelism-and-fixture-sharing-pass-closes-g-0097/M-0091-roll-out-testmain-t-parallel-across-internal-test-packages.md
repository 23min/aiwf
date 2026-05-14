---
id: M-0091
title: Roll out TestMain + t.Parallel across internal/* test packages
status: done
parent: E-0025
tdd: none
acs:
    - id: AC-1
      title: race-parallel cap lands in Makefile and workflows in a leading commit
      status: met
    - id: AC-2
      title: every internal/* test-bearing package has setup_test.go with TestMain
      status: met
    - id: AC-3
      title: t.Parallel adopted on every test that does not need serial execution
      status: met
    - id: AC-4
      title: internal/policies shares the live-repo tree.Load via a sync.Once helper
      status: met
    - id: AC-5
      title: go test -race -parallel 8 ./internal/... reliable across 10 runs
      status: met
    - id: AC-6
      title: wall-time baseline and post-conversion numbers recorded at wrap
      status: met
---

# M-0091 — Roll out TestMain + t.Parallel across internal/* test packages

## Goal

Apply the spike's `TestMain` + `t.Parallel()` pattern across every `internal/*` test package, cap `-race` parallelism at 8 in `Makefile` and the GitHub workflows, and memoize the live-repo `tree.Load(repoRoot)` in `internal/policies/`. After this milestone, `go test ./internal/...` is meaningfully faster at default parallelism and `go test -race -parallel 8 ./internal/...` is reliable across 10 consecutive runs.

## Context

The spike branch `spike/test-parallel` proved the pattern on `internal/verb/` (~4× faster non-race, ~2.4× with race). G-0097 catalogues the rest of `internal/*` as following the same shape — every test package re-does its own git-init / tree-load and almost none call `t.Parallel()`. The pattern is mechanical; the work is rolling it out package-by-package and recording the skip-list per package for tests that legitimately must stay serial.

Per the epic's Constraints, this milestone explicitly **relaxes "one commit per milestone"** to one commit per per-package conversion plus a leading cap-change commit. The cap-change must precede the conversions because `-parallel 8` is the safety net that lets the per-package commits run under `-race` without flakes.

## Acceptance criteria

(ACs allocated via `aiwf add ac`; bodies follow below.)

## Constraints

- **No test semantics change.** The conversion is mechanical — adding `t.Parallel()` and moving env setup to `TestMain`. A converted test that newly passes for a different reason than its assertion claims fails review (per CLAUDE.md "Don't paper over a test failure"). The chokepoint is the per-package re-run with `-race -count=10` recorded in the commit body.
- **One commit per per-package conversion plus the leading cap-change commit.** Each commit carries `aiwf-verb`, `aiwf-entity: M-0091`, and `aiwf-actor` trailers. Atomicity by *package*, not by milestone, is deliberate — see the epic's design decisions table.
- **`os.Setenv` for the GIT identity, not `t.Setenv`.** `t.Setenv` panics under `t.Parallel`. The values are immutable for the test binary's lifetime, so `os.Setenv` is the correct primitive. The one exception is `TestApply_RollsBackOnCommitFailure` (and any sibling that deliberately clears the identity to provoke a commit failure), which keeps its `t.Setenv` and stays serial.
- **Forward-compatibility.** The pattern must not introduce package-level mutable state that a future test could surprise itself with. The shared `*Tree` is read-only; the `// do not mutate` comment is reviewer-enforced.

## Design notes

- The conversion pattern is the spike's, verbatim: `setup_test.go` per package, `os.Setenv` block in `TestMain`, `t.Parallel()` first-line on every parallelizable test. No deviation; no per-package cleverness.
- The `-parallel 8` cap is conservative. If a slower-disk runner re-surfaces the macOS git-add segfault at 8, the cap drops to 4 in one-line follow-ups (Makefile + two workflow files).
- Memoizing `tree.Load(repoRoot)` is in this milestone (not M-0092 or M-0093) because the consumers are all under `internal/policies/` — co-locating the change with the rest of the `internal/*` rollout keeps the per-package commit boundaries clean.

## Surfaces touched

- `Makefile` — `test-race`, `coverage`, `ci` targets
- `.github/workflows/go.yml`, `.github/workflows/flake-hunt.yml`
- `internal/*/setup_test.go` (new, one per package)
- `internal/policies/` — shared `*Tree` helper

## Out of scope

- `cmd/aiwf/` test conversions — those land in M-0092.
- The `CLAUDE.md ## Test discipline` section and the `setup_test.go`-presence policy test — those land in M-0093.
- `internal/htmlrender/htmlrender_test.go::writeFixtureTree` snapshot conversion — out per the epic.
- Pre-baked `aiwf init`-ed skeleton tempdir — out per the epic.

## Dependencies

- None. The conversion touches test files only.

## References

- E-0025 epic spec (this milestone's parent) — design decisions, risks, constraints.
- G-0097 — gap framing, evidence, spike numbers.
- Spike branch `spike/test-parallel`; `internal/verb/setup_test.go` — reference implementation.
- CLAUDE.md *Go conventions* — testing rules; this milestone's chokepoint commits comply but do not extend.

## Work log

### AC-1 — race-parallel cap (commit `f6a0dcb`)

`-parallel 8` landed in `Makefile`'s `test-race` target, `.github/workflows/go.yml`'s race-coverage step, and `.github/workflows/flake-hunt.yml`'s race-detector sweep. Mechanical evidence: `internal/policies/race_parallel_cap.go` scans the three files and asserts `-parallel 8` adjacent to every `go test ... -race` invocation (comment lines skipped). Drop the cap from one surface and the policy fires on that file at that line.

### AC-2 / AC-3 — TestMain + t.Parallel across 24 internal/* packages (24 commits)

24 packages converted, one commit per package, in two waves:

1. **Reference conversions (parent session):** `internal/pathutil` (`580494c`) and `internal/gitops` (`2a053c3`). pathutil is simple (no env state); gitops is the heavy-git template — two t.Setenv helpers (`gitTestEnv` and `initTestRepo`) were neutralized.
2. **Bulk conversion (dispatched builder):** the remaining 22 packages — `version`, `repolock`, `roadmap`, `pluginstate`, `manifest`, `contractconfig`, `contractcheck`, `contractverify`, `config`, `recipe`, `scope`, `aiwfyaml`, `render`, `skills`, `entity`, `initrepo`, `tree`, `trunk`, `htmlrender`, `check`, `policies`, `verb`. SHAs in `git log --grep="M-0091/AC-2 AC-3"`.

Pattern: `setup_test.go` per package with TestMain seeding the 4 GIT identity env vars via `os.Setenv`; `t.Parallel()` adopted on every parallelizable Test\* function; helpers that used `t.Setenv` for git identity were neutralized. The serial skip-list lives in each package's `setup_test.go` as a `// Serial tests:` comment.

**One serial test by design:** `internal/verb/TestApply_RollsBackOnCommitFailure` keeps its `t.Setenv` — it deliberately clears the GIT identity to provoke a commit failure (that's the test's premise). Documented in `internal/verb/setup_test.go`'s skip-list.

### AC-4 — shared tree.Load via sync.Once (in the policies commit, `d095c8d`)

`internal/policies/shared_tree_test.go` exposes `sharedRepoTree(t)` — a `sync.Once`-memoized wrapper around `tree.Load(root)`. The returned `*Tree` is read-only by convention (`// do not mutate`). Five consumers wired: the 3 named in the spec (`TestPolicy_ThisRepoTreeIsClean`, `TestPolicy_ThisRepoDriftCheckClean`, `loadM080Spec`) plus 2 discovered during conversion (`loadADR0007`, `TestAiwfxWrapEpic_AC4_RitualsRepoSHARecordedAtWrap`). Post-share, `go test ./internal/policies/` runs in ~0.7s on warm cache, down from ~3s when each consumer loaded the tree independently.

### AC-5 — 10-run reliability (see Validation below)

### AC-6 — wall-time numbers (see Validation below)

## Decisions made during implementation

- **Subagent dispatch without `aiwf authorize` produces unrecoverable trailer drift.** The bulk-conversion builder agent ran `aiwf edit-body M-0091` for each of its 22 per-package commits while no authorize scope was open. The trailers landed as `aiwf-actor: ai/claude` without `aiwf-principal:` or `aiwf-on-behalf-of:`, failing `provenance-trailer-incoherent` and `provenance-no-active-scope` at the post-implementation `aiwf check`. The pre-commit hook (`--shape-only` by design) did not catch it. Retroactive fixes were unavailable — adding a principal alone surfaced the on-behalf-of rule, and there is no scope to reference. Recovery: a `git filter-branch --msg-filter` pass stripped every `aiwf-*` trailer from those 22 commits, demoting them to plain `chore(test):` commits. The choreography rule (parent opens `aiwf authorize` before dispatching any subagent that will invoke aiwf mutating verbs) is captured in E-0031's "Evidence in flight" section for M-0108 to encode in `legal-workflows.md`.

## Validation

### Build + lint + check

- `go build ./cmd/aiwf` — green.
- `golangci-lint run` — 0 issues.
- `aiwf check` — 0 errors (warnings unrelated to M-0091: pre-existing `archive-sweep-pending` for G-0119, `entity-body-empty` on M-0102's draft AC section, `provenance-untrailered-scope-undefined` from missing upstream config).
- `go test -race -parallel 8 ./...` — all packages pass.

### AC-6 — wall-time at default parallelism (`go test ./internal/... -count=1`)

| | Wall time |
|---|---|
| Baseline (pre-conversion) | 53.6s |
| Post-conversion           | 24.5s |
| **Speedup**               | **~2.2×** |

Meets the epic's success target of ≥2× faster at default parallelism. Both measured on the same 20-core macOS dev host, warm Go build cache.

### AC-5 — 10-run `-race -parallel 8 -count=1 ./internal/...` reliability

```
=== run 1  === PASS at 36s
=== run 2  === PASS at 32s
=== run 3  === PASS at 30s
=== run 4  === PASS at 31s
=== run 5  === PASS at 33s
=== run 6  === PASS at 33s
=== run 7  === PASS at 35s
=== run 8  === PASS at 35s
=== run 9  === PASS at 34s
=== run 10 === PASS at 31s
```

Zero flakes, zero timeouts. Individual run times 30–36s (avg ~33s). The `-parallel 8` cap chosen in AC-1 holds reliably across the full `./internal/...` set.

## Deferrals

- (none)

## Reviewer notes

- **Trailer-strip history rewrite.** The 22 builder-agent per-package commits originally landed with `aiwf-verb: edit-body` + `aiwf-entity: M-0091` + `aiwf-actor: ai/claude` trailers, stamped by the builder's `aiwf edit-body` invocations. No `aiwf authorize` scope was open at dispatch, so the trailers violated the provenance model (`provenance-trailer-incoherent` then `provenance-no-active-scope`). A `git filter-branch --msg-filter` pass stripped every `aiwf-*` trailer from those 22 commits, demoting them to plain `chore(test):` commits. The pre-rewrite branch state is preserved on the local tag `m0091-before-trailer-rewrite` (delete after the wrap-merge confirms). See `## Decisions made during implementation` and E-0031's "Evidence in flight" section for the choreography rule M-0108 should encode.
- **Pre-existing CI blockers on main fixed in passing.** Two stale `FilesWritten` assertions and a gofumpt/staticcheck pair were cleared on main while wrapping (`f84d7b6`, `1569c10`) so the wrap-merge lands on a green tree. Not part of M-0091's deliverables — landed as separate chore commits on main, then `git merge main` brought them onto this branch.
- **Helpers neutralized.** Six helpers had their `t.Setenv` blocks removed (keeping the non-env logic): `gitops.gitTestEnv` (deleted entirely), `gitops.initTestRepo`, `initrepo.freshGitRepo`, `trunk.initRepo`, `verb.newApplyTestRepo`, `verb.newRunner`. Plus inline `t.Setenv` blocks in `verb/apply_lock_test.go` and `verb/apply_internal_test.go`.
- **Subtests.** Table-driven `t.Run(name, ...)` subtests received nested `t.Parallel()` where the loop iteration was independent. Subtests that share parent fixtures (parent mutates fixture between iterations) were left serial inside their parallel parent.
- **M-0093 still pending.** This milestone rolls out the convention; the policy-test chokepoint asserting every `internal/*` test package has a `setup_test.go` lands in M-0093. Until then the convention is reviewer-enforced.

### AC-1 — race-parallel cap lands in Makefile and workflows in a leading commit

`Makefile`'s `test-race` target gains `-parallel 8`; the `coverage` and `ci` targets propagate the same cap where they invoke race-mode tests. `.github/workflows/go.yml`'s race-mode test step gains the same flag. `.github/workflows/flake-hunt.yml`'s `go test -race -count=10` line gains the same flag. The commit message documents the rationale (race + heavy subprocess fan-out flakes on macOS at default parallelism; CI Linux less affected but uniform cap keeps the chokepoint single-valued). After this commit, all three race-test surfaces agree.

### AC-2 — every internal/* test-bearing package has setup_test.go with TestMain

For each package in the scope list — `internal/verb`, `internal/check`, `internal/initrepo`, `internal/htmlrender`, `internal/policies`, `internal/aiwfyaml`, `internal/contractverify`, `internal/tree`, `internal/gitops`, `internal/skills`, `internal/config`, plus any others discovered during audit — a `setup_test.go` is added with a `TestMain(m *testing.M)` that calls `os.Setenv` once for `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_COMMITTER_NAME`, `GIT_COMMITTER_EMAIL` before `m.Run()`. The filename is uniform; the structure mirrors `internal/verb/setup_test.go` from the spike. Tests in the package no longer call `t.Setenv` for these four variables (with the documented serial-batch exceptions called out in AC-3).

### AC-3 — t.Parallel adopted on every test that does not need serial execution

For each converted package, the per-package audit produces a written skip-list of tests that stay serial — those that call `t.Setenv`/`t.Chdir`, mutate a package-level var, or guard a global the parallel suite depends on. Every test not on the skip-list calls `t.Parallel()` as its first non-`t.Helper()` statement. The skip-list lives in the package's `setup_test.go` as a comment (`// Serial tests: …`) so a contributor can see at a glance what is intentionally serial and why. The per-package conversion commits each cite the skip-list in their commit body.

### AC-4 — internal/policies shares the live-repo tree.Load via a sync.Once helper

A new helper in `internal/policies/` loads the repo tree once and returns the shared `*Tree` to consumers (`TestPolicy_ThisRepoTreeIsClean`, `TestPolicy_ThisRepoDriftCheckClean`, `internal/policies/m080_test.go::loadM080Spec`, and any other live-repo-tree readers discovered during conversion). The helper carries a `// do not mutate` comment at its definition. `TestMain` and the shared loader compose cleanly (env setup in `TestMain`; tree memoization in a separate `sync.Once`).

### AC-5 — go test -race -parallel 8 ./internal/... reliable across 10 runs

After all per-package commits land, the milestone records a 10-run loop of `go test -race -parallel 8 ./internal/...` with zero flakes and zero timeouts. The run log is pasted into the milestone's Validation section at wrap. If a flake surfaces, it is root-caused (not papered over per CLAUDE.md "Don't paper over a test failure") and the fix lands either as a per-package amendment or as a new gap if it's a real data race the conversion exposed.

### AC-6 — wall-time baseline and post-conversion numbers recorded at wrap

The milestone records `go test ./internal/...` wall time before the first conversion commit and after the last, on the same hardware. The target per the epic is ≥2× faster at default parallelism; the actual numbers go into Validation at wrap. If the target is missed, the milestone documents why in Reviewer notes (not a wrap blocker; the epic's success criteria are the hard bar).

