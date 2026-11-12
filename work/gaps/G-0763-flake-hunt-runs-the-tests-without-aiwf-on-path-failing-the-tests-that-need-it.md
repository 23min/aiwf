---
id: G-0763
title: flake-hunt runs the tests without aiwf on PATH, failing the tests that need it
status: open
priority: medium
---
## What's missing

The `flake-hunt` job in `.github/workflows/flake-hunt.yml` (lines 69–90) checks
out, sets up Go, sets a git identity and runs the race-detector sweep. No step
puts `aiwf` on PATH. The test suite needs it there: the `test` job in
`.github/workflows/go.yml` installs it first, at lines 140–141, "Install aiwf to
PATH (required for hooks in tests)".

Tests that resolve `aiwf` from PATH fail under flake-hunt. Run 43292158635,
dispatched on a branch at `778c2c0ab` with `-race -parallel 8 -count=10`:

- Six tests in `internal/cli/doctor` failed all 10 repeats:
  `TestRun_SelfCheckEndToEnd`,
  `TestAppendPreCommitHookReport_PostG0135StatusDrift`,
  `TestAppendPreCommitHookReport_PostG0135ChainProblem`,
  `TestAppendPostCommitHookReport_PostG0135ChainProblem`,
  `TestAppendCommitMsgHookReport_OurHook` and
  `TestAppendCommitMsgHookReport_ChainProblem`.
- Two tests in `internal/cli/integration` failed 1 of 10 repeats:
  `TestDoctor_CheckLatest_ProxyDisabled` and
  `TestDoctorReport_EnvLine_InformationalOnly`, each reporting 4 problems where
  it expects 0. The first prints its report, and its four problems are the
  "aiwf binary not found on PATH" rows for the four hooks.

The integration pair pass only once another test in the same process has called
`testutil.AiwfBinary` (`internal/cli/cliutil/testutil/proc.go:91`), which builds
`aiwf` and prepends its directory to the process PATH (line 124). The build runs
once per test process and the PATH change lasts for every later repeat. Both
tests call `t.Setenv`, so they run before any parallel test resumes, and they
fail only in the first repeat: one failure in ten is the expected count, not
chance. In the run's log both FAIL lines come before the first of the ten
once-per-repeat lines.

Reproduced at `8ef91d79c` (go1.25.12, git 2.51.1, Linux), where
`$(go env GOPATH)/bin` holds the only `aiwf` on PATH:

```
$ PATH=$(echo "$PATH" | tr ':' '\n' | grep -vx "$(go env GOPATH)/bin" | paste -sd: -) \
    go test -count=1 -run '^(TestRun_SelfCheckEndToEnd|TestAppend(Pre|Post)CommitHookReport_PostG0135(StatusDrift|ChainProblem)|TestAppendCommitMsgHookReport_(OurHook|ChainProblem)|TestDoctor_CheckLatest_ProxyDisabled|TestDoctorReport_EnvLine_InformationalOnly)$' \
    ./internal/cli/doctor/ ./internal/cli/integration/ 2>&1 | grep -E -- '^--- FAIL|^(ok|FAIL)\s+github'
--- FAIL: TestRun_SelfCheckEndToEnd (1.53s)
--- FAIL: TestAppendCommitMsgHookReport_ChainProblem (0.00s)
--- FAIL: TestAppendCommitMsgHookReport_OurHook (0.01s)
--- FAIL: TestAppendPostCommitHookReport_PostG0135ChainProblem (0.01s)
--- FAIL: TestAppendPreCommitHookReport_PostG0135ChainProblem (0.01s)
--- FAIL: TestAppendPreCommitHookReport_PostG0135StatusDrift (0.01s)
FAIL	github.com/23min/aiwf/internal/cli/doctor	1.546s
--- FAIL: TestDoctor_CheckLatest_ProxyDisabled (0.25s)
--- FAIL: TestDoctorReport_EnvLine_InformationalOnly (0.25s)
FAIL	github.com/23min/aiwf/internal/cli/integration	0.515s
```

With PATH left unchanged, the same command prints `ok` for both packages. With
`aiwf` removed from PATH again, a test that builds the binary changes the
outcome from the first repeat to the second:

```
$ PATH=<as above> go test -count=2 -v \
    -run '^(TestAddAC_RepeatedTitle_BinaryEndToEnd|TestDoctor_CheckLatest_ProxyDisabled)$' \
    ./internal/cli/integration/
=== RUN   TestAddAC_RepeatedTitle_BinaryEndToEnd
=== PAUSE TestAddAC_RepeatedTitle_BinaryEndToEnd
=== RUN   TestDoctor_CheckLatest_ProxyDisabled
--- FAIL: TestDoctor_CheckLatest_ProxyDisabled (0.20s)
=== CONT  TestAddAC_RepeatedTitle_BinaryEndToEnd
--- PASS: TestAddAC_RepeatedTitle_BinaryEndToEnd (1.72s)
=== RUN   TestAddAC_RepeatedTitle_BinaryEndToEnd
=== PAUSE TestAddAC_RepeatedTitle_BinaryEndToEnd
=== RUN   TestDoctor_CheckLatest_ProxyDisabled
--- PASS: TestDoctor_CheckLatest_ProxyDisabled (0.19s)
=== CONT  TestAddAC_RepeatedTitle_BinaryEndToEnd
--- PASS: TestAddAC_RepeatedTitle_BinaryEndToEnd (0.46s)
FAIL	github.com/23min/aiwf/internal/cli/integration	2.581s
```

## Why it matters

flake-hunt exists to find tests that fail intermittently. These tests fail because
the workflow skips a setup step the suite requires, so `internal/cli/doctor` and
`internal/cli/integration` are reported failed whatever their tests do, and a real
flaky test in either package is hidden among them.

The integration pair mislead further. Failing once in ten repeats is what a flaky
test looks like, so a reader investigates the tests rather than the workflow,
when the failure is the same repeat in every run.
