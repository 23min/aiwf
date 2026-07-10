---
id: G-0388
title: pre-commit.local's policy_cmd lacks the -parallel 8 flake cap
status: addressed
discovered_in: M-0240
addressed_by_commit:
    - 7810aa45
---
## What's missing

`scripts/git-hooks/pre-commit`'s `policy_cmd` default runs
`go test -count=1 ./internal/policies/...` with no `-parallel` flag,
so it runs at Go's default (GOMAXPROCS-based) parallelism instead of
the `-parallel 8` cap the rest of this repo's own tooling enforces
uniformly (`Makefile`, CI workflows, per CLAUDE.md's own Test
discipline section: "race + git-subprocess fan-out flakes at default
parallelism").

## Why it matters

Encountered directly during M-0240's wrap: the pre-commit hook's
policy suite hit a genuine, reproducible `ETXTBSY` ("text file busy")
failure in `TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently`
— a test that passes reliably 3/3 in isolation but flakes under the
hook's uncapped parallelism, exactly the class of subprocess-exec race
the `-parallel 8` cap exists to bound. The fix is a one-line default
change: `go test -count=1 -parallel 8 ./internal/policies/...`.
