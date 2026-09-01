---
id: G-0658
title: Nothing pins that --root reaches the commit-msg guard that reads the index
status: open
discovered_in: M-0327
---
## What's missing

`aiwf check --commit-msg` takes a repo root and reads that repo's index to decide
whether a staged edit touches the ritual authoring tree
(`internal/cli/check/commit_msg.go`, `checkShippedSurfaceOwner`). The root
reaches it from the `--root` flag through `internal/cli/check/check.go`.

Nothing pins that threading. The seam test
(`internal/cli/integration/check_commit_msg_seam_test.go`) passes an empty
temporary directory as `--root`, which is not a git repository, so the guard's
`git diff --cached` fails and it returns early without consulting an index.
Replacing the flag's value with the empty string at the call site leaves that
test green, because the empty string falls back to the process working
directory, where nothing relevant is staged either. The test pins that the flag
is wired; it does not pin that its value arrives where the guard reads it.

## Why it matters

The guard is the only part of the commit-msg check that depends on repository
state rather than on the message alone, so `--root` is the one input whose
mis-threading changes what it decides. A regression there would make the guard
inspect the wrong repository — silently, since the failure mode is a guard that
finds nothing rather than one that errors.

Closing it needs a fixture that stages a path under the ritual authoring tree in
a real temporary repository and drives the verb through the dispatcher, so that
passing the wrong root produces a different verdict.
