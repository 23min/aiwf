---
id: G-0741
title: aiwf doctor names no blocker when legacy ai-dotfiles delivery blocks guidance
status: wontfix
discovered_in: M-0353
---
## What's missing

When legacy ai-dotfiles delivery blocks project guidance, `aiwf update` names the blocker and its fix on a `blocked:` line, and `aiwf doctor` on the same repository names neither. Only installation calls `projectguidance.CheckCompatibility` (`internal/initrepo/project_guidance.go`); doctor's guidance rows come from `appendProjectGuidanceReport` in `internal/cli/doctor/project_guidance.go`, which inspects the installed tree and never consults delivery compatibility. The refused run creates no `.guidance/` and records the refusal nowhere: the blocker reaches only that run's report, through `internal/cli/cliutil/guidance_report.go`.

Measured on 2026-10-24 in the aiwf devcontainer (Linux, x86_64) with the v0.41.0 release binary, in a fresh clone of `github.com/23min/loom` at `7547d6602`, whose `aiwf.yaml` has no `guidance` block. `HOME` is a scratch directory holding `github.com/23min/ai-dotfiles` at `2552653`, installed by its own `install.sh`, which leaves `~/.local/bin/dotfiles-sync` and a `~/.claude/CLAUDE.md` importing `~/.agents/guidance/` and no `dotfiles-guidance-check`; `CLAUDE_CONFIG_DIR` and `CODEX_HOME` are unset.

```
$ aiwf update --root . </dev/null          # exit 0
Guidance:
  selected:  not adopted (guidance.packs unset)
  adopted:   none
  installed: none
  ignored:   none
  blocked:   incompatible ai-dotfiles guidance delivery: detected <scratch>/home/.claude/CLAUDE.md but dotfiles-guidance-check is unavailable; update/reconcile ai-dotfiles and make its checker available on PATH, then retry
$ aiwf doctor --root . </dev/null | grep 'project policy'
project policy: selected: not adopted (guidance.packs unset)
project policy: installed revision: unknown (no intact owned index); upstream freshness not checked
$ aiwf doctor --root . </dev/null | grep -c -i -E 'dotfiles|incompatib|blocked'
0
```

In a copy of that clone with `guidance: packs: [code-health]` appended to `aiwf.yaml`, `aiwf update` prints the same `blocked:` line, and doctor's guidance rows advise the rerun that line refuses:

```
project policy: selected: code-health
project policy: installed revision: unknown (no intact owned index); upstream freshness not checked
project policy: .guidance/index.md: selected guidance is not installed; run aiwf update
project policy: CLAUDE.md: selected host has no recorded engineering guidance route; run aiwf update
project policy: AGENTS.md: selected host has no recorded engineering guidance route; run aiwf update
```

Expected: doctor names the blocked delivery and the action that clears it, as E-0095's success criterion "When guidance cannot be installed, the update report states why and what clears it, and `aiwf doctor` agrees" states.

## Why it matters

A maintainer who runs `aiwf doctor` to learn why a repository has no guidance is not told that their personal ai-dotfiles setup is the cause. With packs selected they are told to run `aiwf update`, which refuses again for the reason doctor omits. The cause and its fix appear only in the output of an `init` or `update` run, and an unattended `aiwf upgrade` prints that output to whatever ran it; once it scrolls away, no read-only verb reproduces it.

The installation-health file the statusline reads is written by the same refused run from a doctor pass (`doctor.WriteHealth`, called at the end of `internal/cli/update/update.go`). In the measurement above, `.claude/health.aiwf.json` holds an actor error and a hook-drift warning and no guidance finding, so the statusline shows no guidance problem while guidance is blocked.
