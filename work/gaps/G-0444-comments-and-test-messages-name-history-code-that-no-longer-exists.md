---
id: G-0444
title: Comments and test messages name history code that no longer exists
status: open
priority: low
discovered_in: M-0274
---
## What's missing

Comments and test failure messages name the history verb's code by names that no longer exist:

- Twelve name `readHistory` for the single-id read, which is `ReadHistory` in `internal/entityview/historyevent.go:106`: the `t.Fatalf("readHistory…")` messages in `cancel_same_state_noop_test.go`, `history_cmd_test.go`, `move_same_state_noop_test.go`, `promote_same_state_noop_test.go`, `rename_retitle_same_state_noop_test.go` and `status_cmd_test.go` under `internal/cli/integration/`, each just after a call to `entityview.ReadHistory`; the comment at `internal/cli/integration/history_cmd_test.go:153`; and the comment at `internal/cli/show/show.go:527`.
- `internal/cli/history/history.go:80` names `readHistory` for the read across all of an entity's ids, which is `ReadHistoryChain` (`internal/entityview/historyevent.go:117`), called at `history.go:101`.
- `internal/cli/integration/canonicalize_history_test.go:19,24` and `internal/cli/integration/show_cmd_test.go:260` name `readHistoryChain`, which is `ReadHistoryChain`.
- `internal/cli/integration/integration_g37_test.go:928` names `runHistory`; the verb's entry point is `Run` (`internal/cli/history/history.go:57`).
- `internal/cli/integration/canonicalize_history_test.go:25` and `internal/entity/canonicalize_test.go:79` cite `admin_cmd.go`, a file the repository no longer holds.

```
$ git grep -nE 'func (\([^)]*\) )?(runHistory|readHistory|readHistoryChain)\(' -- '*.go' | wc -l
0
$ git grep -nwE 'runHistory|readHistoryChain|readHistory' -- 'internal/*.go' 'cmd/*.go' | wc -l
17
$ git ls-files | grep -c admin_cmd
0
```

Expected: no name in a comment or message that no function or file carries. Observed 2026-10-14 in the aiwf devcontainer, as above.

## Why it matters

A reader following one of these names finds no such function or file. For a failure message the function meant is called on the line above under a different capitalization; for a comment, it takes a search to learn which of `Run`, `ReadHistory` and `ReadHistoryChain` is meant, or where the code `admin_cmd.go` held now lives.
