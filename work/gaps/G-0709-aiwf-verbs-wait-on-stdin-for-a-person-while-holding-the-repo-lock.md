---
id: G-0709
title: aiwf verbs wait on stdin for a person while holding the repo lock
status: open
discovered_in: E-0095
---
## What's missing

`aiwf init`, `aiwf update` and `aiwf add --body-file -` wait on stdin for a person while holding the repo lock, so the lock's hold time is set by how long someone takes to answer or type.

Where it shows: `init` acquires the lock at `internal/cli/initcmd/initcmd.go:108` and defers its release (`:112`) to the return of `Run`; `update` does the same at `internal/cli/update/update.go:114` and `:118`. Every prompt either verb can raise runs after that point:

- guidance selection — `cliutil.GuidanceSelector` (`initcmd.go:116`, `update.go:127`), reached through `initrepo.Init` for `init` and `initrepo.RefreshArtifacts` for `update`;
- hook consent — `gateAndPersistHookDecisions` (`initcmd.go:175`) and `gateAndSyncHookDecisions` (`update.go:177`), through `cliutil.GateHookDecisions`;
- statusline confirmations — `cliutil.RunStatuslineScaffold` (`initcmd.go:187`, `update.go:196`), whose `promptYN` calls sit at `internal/cli/cliutil/statusline.go:67` and `:105`.

`aiwf add <kind> --body-file -` reads stdin to end-of-file at `add.go:277`, after acquiring the lock at `add.go:175`. `aiwf add ac` and `aiwf edit-body` read their stdin bodies before locking.

The lock is per checkout: `<root>/.git/aiwf.lock`, or `<worktree>/.aiwf.lock` in a linked worktree (`internal/repolock/repolock_unix.go:105-116`).

Measured with aiwf v0.38.0 in the devcontainer, in a scratch repository with a linked worktree. `aiwf update --statusline --scope project --root <repo>` ran on a pseudo-terminal, with a simulated person answering `n` about 10 s after each prompt appeared, and a competing `aiwf update --root … </dev/null` started 3 s into each prompt:

```
t+ 2.4s prompt #1: select (s) / not now (n) / ignore (i) [not now]:
t+ 7.4s  same checkout   -> exit=2 in 2.0s: another aiwf process is running on this repo; retry in a moment
t+10.4s  linked worktree -> exit=0 in 2.9s
t+17.4s  human answers 'n'
t+17.6s prompt #2: Enable hook "worktree-rituals-check.sh" …? [y/N]
t+22.7s  same checkout   -> exit=2 in 2.1s (same refusal)
t+29.7s  human answers 'n'
t+29.7s prompt #3: Wire statusLine into …/.claude/settings.local.json? [y/N]
t+34.8s  same checkout   -> exit=2 in 2.0s (same refusal)
t+41.8s  human answers 'n'
t+41.9s interactive update exited 0 after 3 prompts
t+45.2s  same checkout   -> exit=0 in 3.3s
```

`aiwf add epic --body-file -` on a pseudo-terminal with nothing typed was still waiting after 5 s, and a competing `aiwf update` in the same checkout exited 2 with the same refusal.

Relates to G-0446 and G-0708. G-0708 is the unattended case, where nobody answers; this record is the attended case, where the hold ends only when a person responds.

## Why it matters

While a prompt is on screen or a piped body is still being typed, every other mutating `aiwf` verb in the same checkout refuses. An operator who leaves an `init` or `update` prompt unanswered in one terminal blocks planning verbs from every other session in that checkout until they return. The refusal tells the blocked caller to "retry in a moment", but when the holder is waiting on a person there is no bound on the moment.
