---
id: M-0170
title: Firing tests for linter-config rules and the dormant forbidigo fix
status: in_progress
parent: E-0042
tdd: none
acs:
    - id: AC-1
      title: Fix the dormant forbidigo panic/os.Exit patterns
      status: met
    - id: AC-2
      title: Execution firing harness for linter-config rules, wired into the lint job
      status: met
---
## Deliverable

Close the linter-config vacuity surface G-0264 exposed: golangci-lint config
rules (`forbidigo`, `gocritic`, …) have **no firing evidence**, so a dormant
rule — one matching zero sites — is invisible. Two parts:

1. **Fix the dormant `forbidigo` patterns.** The `^panic\(` / `^os\.Exit\(`
   call-form patterns match zero sites under forbidigo v2 (it matches the bare
   qualified name, no trailing paren). Correct them to `^panic$` / `^os\.Exit$`.
2. **Add an execution firing harness for the linter-config rules.** A test that
   runs `golangci-lint` against fixtures and asserts each guarded rule actually
   fires — the firing-evidence mechanism for the golangci-config surface,
   parallel to the firing fixtures M-0166 adds for `internal/policies/` Go
   policies. This generalizes M-0167/AC-2's structural guard (config-presence)
   to real execution, and extends it across `gocritic` and `forbidigo`.

## Why

G-0264: the `panic`/`os.Exit` forbidigo rules went dormant on a toolchain bump
and nothing noticed, because the firing-fixture meta-gate only scans
`internal/policies/*.go` — golangci-lint config rules are an uncovered surface.
This is the G-0259 pathology one surface over: a chokepoint that reads as a
guarantee while detecting nothing.

## Mechanical evidence

- AC-1: the AC-2 harness's `forbidigo`-`panic` and `forbidigo`-`os.Exit` rows
  pass with the corrected patterns and fail if either reverts to the `\(` form
  (verified: old patterns 0 hits, new patterns 1 hit each on a library probe).
  Enabling the corrected patterns surfaced 58 legitimate `os.Exit(m.Run())`
  hits in `setup_test.go` files and zero in production, so AC-1 also scopes
  `forbidigo` off `_test.go` — after which a production-wide run is clean.
- AC-2: the harness runs in CI's lint job (where `golangci-lint` is on PATH),
  fail-closed when `golangci-lint` is required-but-absent; each row fails if its
  rule stops firing (config drift, a toolchain bump that changes matching, or a
  checker re-tag).

## Acceptance criteria

### AC-1 — Fix the dormant forbidigo panic/os.Exit patterns

**Deliverable** — In `.golangci.yml`, change the `forbidigo` patterns
`^panic\(` → `^panic$` and `^os\.Exit\(` → `^os\.Exit$`. forbidigo v2 matches
the bare qualified function name (no trailing call paren), so the `\(` form
matched nothing — empirically, a library `panic`/`os.Exit` probe produced 0
issues under the old patterns and 1 each under the new. Because the patterns
were dormant, enabling them surfaces 58 legitimate hits — all `os.Exit(m.Run())`
in `TestMain` across the `setup_test.go` files (the repo's own mandated
test-discipline pattern); production code has zero. So also add a `_test.go`
path exclusion for `forbidigo` — the rule targets production library code, and
`errcheck`/`gosec` already exclude `_test.go`. The existing `cmd/aiwf/main.go`
(the sanctioned `os.Exit`) and `internal/verb/apply.go` (the sanctioned
re-panic) exclusions remain. After the change, a production-wide `forbidigo` run
is clean (verified).

**Mechanical evidence** — The AC-2 harness carries rows for `forbidigo`-`panic`
and `forbidigo`-`os.Exit` that run `golangci-lint` against a fixture holding a
library `panic`/`os.Exit` and assert the rule fires; with the corrected patterns
they pass, and reverting either pattern to the `\(` form fails its row. Closes
the dormant-pattern half of G-0264.

### AC-2 — Execution firing harness for linter-config rules, wired into the lint job

**Deliverable** — Add a table-driven Go test (`TestGolangciConfigRulesFire`) with
one row per guarded config rule — `forbidigo`-`panic`, `forbidigo`-`os.Exit`,
`gocritic`-`filepathJoin`. Each row builds a self-contained temp module whose
code violates exactly that rule, runs `golangci-lint run` against the repo's
`.golangci.yml` from inside the module, and asserts the rule's identifier appears
in the output. Wire it into CI as a step in the existing **lint** job, where
`golangci-lint` is already on PATH via golangci-lint-action. The test is
**fail-closed**: with `AIWF_REQUIRE_GOLANGCI=1` (set on that step) it errors if
`golangci-lint` is absent; otherwise (the test job, local `go test`) it skips
gracefully — so it is a real chokepoint in the lint job and never a silent skip
there.

**Mechanical evidence** — The test itself: each row fails if its rule stops
firing (config drift, a toolchain bump that changes matching, or a checker
re-tag). The lint-job step makes it run in CI; the fail-closed guard turns
"golangci-lint went missing" into a CI error rather than a green skip.
`M-0167/AC-2`'s structural guard stays as the cheap always-on test in the test
job (defense in depth). Closes the no-firing-test half of G-0264.
