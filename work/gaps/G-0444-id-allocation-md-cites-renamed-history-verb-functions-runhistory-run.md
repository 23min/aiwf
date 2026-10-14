---
id: G-0444
title: id-allocation.md cites renamed history-verb functions (runHistory->Run)
status: open
priority: low
discovered_in: M-0274
---
## What's missing

Comments and test failure messages name the history verb's functions by names that no longer exist. The verb's entry point is `Run` (`internal/cli/history/history.go:57`), and the chain read is `ReadHistoryChain` in `internal/entityview/historyevent.go:117`; no `runHistory`, `readHistory` or `readHistoryChain` is declared anywhere.

- Comments: `internal/cli/history/history.go:80` and `internal/cli/show/show.go:527` (`readHistory`), `internal/cli/integration/canonicalize_history_test.go:19,24` and `show_cmd_test.go:260` (`readHistoryChain`), `integration_g37_test.go:928` (`runHistory`), `history_cmd_test.go:153` (`readHistory`).
- Test failure messages beginning `readHistory`, in `cancel_same_state_noop_test.go`, `history_cmd_test.go`, `move_same_state_noop_test.go`, `promote_same_state_noop_test.go`, `rename_retitle_same_state_noop_test.go` and `status_cmd_test.go` under `internal/cli/integration/`.

```
$ git grep -nE 'func (\([^)]*\) )?(runHistory|readHistory|readHistoryChain)\(' -- '*.go' | wc -l
0
$ git grep -nwE 'runHistory|readHistoryChain|readHistory' -- 'internal/*.go' 'cmd/*.go' | wc -l
17
```

Expected: no name in a comment or message that no function carries. Observed 2026-10-13 in the aiwf devcontainer, as above.

## Why it matters

A reader following a comment, or a failing test's message, to the function it names finds nothing, and has to search out where the history verb expands an id and reads its chain.
