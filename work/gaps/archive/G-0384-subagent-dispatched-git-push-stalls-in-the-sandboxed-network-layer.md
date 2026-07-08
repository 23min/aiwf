---
id: G-0384
title: Subagent-dispatched git push stalls in the sandboxed network layer
status: addressed
addressed_by_commit:
    - ee10940f
---
## What's missing

`aiwfx-release`'s push gates (steps 6 and 7 — pushing the release-prep commit, pushing the tag) have the dispatched `deployer` subagent execute `git push` itself, from within its own sandboxed Bash tool context. Twice during a single release run (`v0.26.1`), that push stalled in the sandbox's network layer: reads (fetch, `gh api`, `git log`) worked fine, but the push write hung. Both times the workaround was disabling the Bash tool's sandbox (`dangerouslyDisableSandbox`) for that one command, after killing the hung attempt.

The orchestrating session's own Bash tool did not hit this when it ran the equivalent pushes directly earlier in the same session (the `v0.26.0` release). The stall correlates with *which process runs the push* — a dispatched subagent's sandboxed context — not with git/GitHub connectivity in general.

## Why it matters

Disabling the sandbox is a bypass of a safety mechanism, not a fix, and it isn't something a shipped ritual should rely on turn after turn — a future session either repeats the same kill-and-retry dance or, worse, defaults to disabling the sandbox pre-emptively for every subagent push "just in case," which is a bigger bypass than the problem warrants. Left as-is, every `aiwfx-release` run that dispatches to `deployer` risks the same stall, with no guidance in the skill or the agent card about what to do when it happens.
