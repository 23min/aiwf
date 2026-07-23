---
id: G-0444
title: id-allocation.md cites renamed history-verb functions (runHistory->Run)
status: open
priority: low
discovered_in: M-0274
---
## What's missing

`docs/design/id-allocation.md:164` describes the history verb's `PriorIDs`-chain
handling using function names the M-0116 CLI restructure renamed:

- `runHistory` no longer exists — the history verb's entry point is now `Run`
  (`internal/cli/history/history.go:60`).
- `readHistoryChain` survives only in test comments under
  `internal/cli/integration/` (e.g. `canonicalize_history_test.go`,
  `show_cmd_test.go`), not in production code; the chain logic now lives inline
  in `history.go`.

The file-path half of this same line was already corrected (G-0443:
`cmd/aiwf/admin_cmd.go` → `internal/cli/history/history.go`), but the function
names on the line — plus the stale `readHistoryChain` mentions in the test
comments — were left. This is function-rename drift, a distinct class from the
file-path drift G-0443 closed; it was surfaced by G-0443's review and left out
of that patch's declared file-path scope on purpose.

## Why it matters

id-allocation.md is a Normative-tier design doc (current-truth). A reader who
follows it to `internal/cli/history/history.go` looking for `runHistory` or
`readHistoryChain` finds neither — the entry point is `Run` and the chain logic
is inline. Minor in blast radius, but it is real drift in a lockstep doc, and
it is the second drift class the same M-0116 restructure left behind (the
first, file paths, is closed as G-0443).

## Suggested approach

Update id-allocation.md:164's function names to the current symbols (`Run`;
describe the inline `PriorIDs`-chain handling rather than the retired
`readHistoryChain`), and refresh the `internal/cli/integration/` test comments
that still name `readHistoryChain`. Small, mechanical, no code change. No
mechanical guard is warranted: function-name references in prose are heuristic
to verify and would false-positive on historical/narrative mentions — the same
reason G-0443's guard was scoped narrowly to resolvable file paths, not symbol
names.
