---
id: G-0708
title: Unattended aiwf update blocks on a prompt while holding the repo lock
status: addressed
discovered_in: E-0095
addressed_by_commit:
    - b2d7388da
---
## What's missing

An unattended `aiwf update` on a pseudo-terminal blocks on a prompt nobody will answer, holding the repo lock, and neither `update` nor `aiwf upgrade` offers a documented way to say no human is present.

Where it shows: `update` decides whether to prompt from `render.IsTTY(os.Stdin)` once a prompt is due. Its guidance selector is `cliutil.GuidanceSelector(false)` (`internal/cli/update/update.go:127`, run inside `initrepo.RefreshArtifacts`) and its hook-consent gate is `cliutil.GateHookDecisions(newHooks, enableHooks, false, false)` (`internal/cli/update/hooks.go:41`, reached from `update.go:177`). Both run between the lock acquired at `update.go:114` and its deferred release at `update.go:118`. `aiwf init` exposes `--no-prompt` for its guidance-selection and hook-consent prompts; `update` has no such flag. `aiwf upgrade` re-executes the new binary as a fixed `update --root <dir>` (`internal/cli/upgrade/upgrade.go:456`, via `syscall.Exec`, so stdin carries over) and has no flag to pass either.

The prompts recur on every run while their subject stays undecided: a registry hook absent from `aiwf.yaml`'s `hooks:` map — the state `init --no-prompt` and every non-terminal `update` leave behind — and, for guidance, any applicable pack neither selected nor ignored, since "not now" is not recorded.

Measured with aiwf v0.38.0 in the devcontainer:

```
$ aiwf init --help 2>&1 | grep -- '--no-prompt'
      --no-prompt                   never prompt for hook consent or guidance selection; leave undecided choices unchanged
$ aiwf update --help 2>&1 | grep -c -- '--no-prompt'
0
$ aiwf update --no-prompt --root <empty scratch dir>; echo "exit=$?"
aiwf: unknown flag: --no-prompt
exit=2
```

The hang, measured in a scratch repository prepared with `aiwf init --no-prompt </dev/null` (leaving one registry hook undecided), then `aiwf update --root <repo>` on a pseudo-terminal with no input, `claude` on PATH and the guidance catalogue reachable:

- `update` printed `code-health — … select (s) / not now (n) / ignore (i) [not now]:` and was still blocked 50 s later.
- Answered `n`, it printed `aiwf update: done.` and then `Enable hook "worktree-rituals-check.sh" — …? [y/N]`, still blocked 10 s later.
- Meanwhile a second `aiwf update` and an `aiwf add epic` each exited 2 after 2 s with `another aiwf process is running on this repo; retry in a moment`; `aiwf list`, `aiwf check` and `aiwf status` returned normally.
- The same run with stdin redirected from `/dev/null` exited 0 in 1.6 s. That redirect is the only way to suppress the prompts, and neither verb's `--help` mentions it.

## Why it matters

A caller running `update` or `upgrade` on a pseudo-terminal with nobody behind it — a devcontainer lifecycle command, as G-0446 observed for `init`, or a script run under `script(1)` — hangs indefinitely, and every mutating `aiwf` verb in that checkout refuses until the process is killed; linked worktrees hold their own lock and are unaffected. The hang follows `aiwf update: done.`, so a caller watching for completion is told the run finished. The only escape is an undocumented stdin redirect, so a caller that knows no human is present has no supported way to say so.
