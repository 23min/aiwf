---
id: G-0377
title: Apply's staged-conflict guard is coarser than a directory-move's actual writes
status: open
discovered_in: M-0186
---
## What's missing

`checkStagedConflict` (internal/verb/apply.go) scopes its pre-flight guard to `planPaths`, which for an `OpMove` of a directory only names the directory's own Path/NewPath — not the nested files inside it. But the commit's actual writes are computed by `gatherCommitOps`, which walks a moved directory and produces one write per nested file at its new location.

So a user's staged edit to a file nested inside a directory a verb is about to move is neither flagged by the conflict guard (the guard never sees that nested path) nor left alone (Phase 1's `os.Rename` carries the file to its new location, and `gatherCommitOps` sweeps it into the verb's commit `ReconcilePaths` then stages). The two intents — the user's staged content, the verb's computed content for that same final path — can silently disagree without the guard ever refusing.

## Why it matters

Found during the M-0186 milestone-wrap independent code review (a fresh-context reviewer tracing `checkStagedConflict`'s scoping against `gatherCommitOps`'s actual write set). Not corruption — reallocate/rewidth (the verbs that move directories) normally run against a clean tree — but it is the one place the guard's granularity ("paths the Plan names") is coarser than the commit's real write set ("paths gatherCommitOps discovers by walking"), which is exactly the class of mismatch the guard exists to catch for flat writes.

## Possible directions (not decided)

- Extend `planPaths` to walk directory-move destinations the same way `gatherCommitOps` does, so the guard's path set matches the commit's write set exactly.
- Scope the fix to only the verbs that actually move directories with nested files (reallocate, rewidth) rather than a blanket walk for every `OpMove`.
- Confirm empirically whether this is reachable in practice today (do reallocate/rewidth ever run against a tree with unrelated staged nested-file edits?) before deciding whether to fix now or leave as a documented, accepted risk.
