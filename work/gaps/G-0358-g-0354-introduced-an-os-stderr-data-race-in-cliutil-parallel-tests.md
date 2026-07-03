---
id: G-0358
title: G-0354 introduced an os.Stderr data race in cliutil parallel tests
status: open
---
## What's missing

The work that addressed G-0354 (`aiwf update --remove`, commit `adfb96f7`)
introduced a data race in the `internal/cli/cliutil` test suite that flaked
`make ci` / CI intermittently under `-race -parallel 8`.

`RunStatuslineRemove` (in `internal/cli/cliutil/statusline.go`) writes its
error and refusal messages with `fmt.Fprintf(os.Stderr, …)` — a **read** of
the process-global `os.Stderr`. Its tests (`TestRunStatuslineRemove_*`) are all
`t.Parallel()` and several deliberately drive the refusal branch (asserting
`rc == ExitFindings`), so they exercise those `Fprintf` reads concurrently.

Meanwhile a long-standing test, `TestParseTestsFlag`'s `malformed` subtest
(`verbhelpers_test.go`, from 2026-05-17), **mutates** the same global
`os.Stderr` (`os.Stderr, _ = os.Open(os.DevNull)` + restore) while it too runs
`t.Parallel()`. That writer had no parallel reader before G-0354, so it was a
dormant violation of the repo's "shares stdout/stderr capture → serial" test
rule. G-0354 supplied the missing half — the first parallel reader of the
global — turning it into a live write∥read race at `statusline.go:162` vs
`verbhelpers_test.go:80/81`.

## Why it matters

The race is intermittent (~60% of full-suite `make ci` runs), so it presents as
"flaky CI" rather than a clear failure — the most expensive kind of defect to
diagnose, and one that erodes trust in the gate for every clone until fixed. It
also slipped past G-0354's own CI, which happened to schedule a passing
interleaving on the push that landed it: proof that a single green `-race` run
is not evidence of race-freedom for a low-probability window.

## Resolution

Fixed by commit `4b7634e9` (`test(cliutil): serialize TestParseTestsFlag to fix
os.Stderr data race`): `TestParseTestsFlag` is made fully serial and recorded in
the `cliutil` serial skip-list, per the repo's stdout/stderr-capture rule. A
deterministic reproducer — `go test -race -count=300 -run
'TestParseTestsFlag|TestRunStatuslineRemove' ./internal/cli/cliutil/` — went from
43 detected races to 0 (also clean at count=500), and full `make ci` is green.

Filed retroactively against G-0354 to record the regression and its root cause;
the discovery happened during the G-0356 wrap, where a reconcile with trunk
pulled the already-merged G-0354 code into a branch whose `make ci` then flaked.
