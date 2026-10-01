---
id: G-0728
title: CLAUDE.md points the completion drift test at files that do not hold it
status: open
discovered_in: E-0098
---
## What's missing

`CLAUDE.md` names the wrong place for the completion drift test twice. Under "Go conventions",
"CLI conventions" (line 292), it gives the chokepoint as `cmd/aiwf/completion_drift_test.go`,
which does not exist; under "Engineering principles" (line 15) it says the drift test is in
`internal/policies/`. Both tests it means, `TestPolicy_FlagsHaveCompletion` and
`TestPolicy_PositionalsHaveCompletion`, live in `internal/cli/integration/completion_drift_test.go`.

Measured in the devcontainer at the head of the E-0098 epic branch:

```sh
ls cmd/aiwf/completion_drift_test.go
git ls-files | grep completion_drift
grep -n '^func Test' internal/cli/integration/completion_drift_test.go
```

Observed: `ls` reports no such file; `git ls-files` lists only
`internal/cli/integration/completion_drift_test.go`; that file defines
`TestPolicy_FlagsHaveCompletion` (line 42) and `TestPolicy_PositionalsHaveCompletion` (line 140).

## Why it matters

`CLAUDE.md` is where a contributor looks for the check that fails when a verb or flag ships
without completion wiring. Following either pointer finds no test, so the contributor cannot
read the rule they are bound by or run it alone before pushing.
