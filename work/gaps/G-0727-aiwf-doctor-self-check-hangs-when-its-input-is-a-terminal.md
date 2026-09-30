---
id: G-0727
title: aiwf doctor --self-check hangs when its input is a terminal
status: open
discovered_in: M-0360
---
## What's missing

`aiwf doctor --self-check` stops for good when its standard input is a terminal. Its first step
runs `init` with `--root` and `--actor` but not `--no-prompt`
(`internal/cli/doctor/selfcheck.go`, the step labelled `init`). `init` asks questions whenever
standard input is a terminal: the guidance-pack selection (`GuidanceSelector` in
`internal/cli/cliutil/guidance.go`) and one `[y/N]` per optional Claude hook (`interactive` in
`internal/cli/cliutil/hooks.go`). The self-check's output shows none of those questions, so it
waits on an answer nobody is asked for. `make ci` runs the self-check last (`selfcheck:` in the
`Makefile`), so `make ci` typed at a terminal hangs after every other step has passed.

Measured in the devcontainer (Linux, x86_64), with a binary built from `5f0aa0fc1`, giving the
self-check a terminal through `script`:

```sh
timeout 90 script -qec "aiwf doctor --self-check" /tmp/selfcheck-tty.log </dev/null
```

Expected it to finish. Observed exit 124 from `timeout` after 90 seconds, with the log ending at
`self-check repo: /tmp/aiwf-self-check-937370664` and nothing after it. The same command without
a terminal passes all 29 steps. Running `aiwf init --root <tmp> --actor human/selfcheck` under
`script` the same way shows the question it stops on:

```
Enable hook "worktree-rituals-check.sh" — Warns (without blocking) when a session or subagent starts inside a .claude/worktrees/ checkout whose rituals aren't materialized.? [y/N]
```

`make ci` typed into an interactive terminal in a newly built devcontainer sat at
`./bin/aiwf doctor --self-check` for over 40 minutes, the process asleep with no child; the same
`make ci` with `</dev/null` finished and passed.

## Why it matters

`make ci` is the local gate `CLAUDE.md` asks for before a merge to `main` or a push, and a
person runs it from a terminal. There it hangs silently at the last step, after the slow ones,
with nothing on screen to say it is waiting; the reader's only way out is to find the cause, as
here, or to abandon the gate.
