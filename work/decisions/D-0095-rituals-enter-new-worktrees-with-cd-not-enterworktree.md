---
id: D-0095
title: Rituals enter new worktrees with cd, not EnterWorktree
status: accepted
relates_to:
    - G-0673
    - G-0413
---
> **Date:** 2026-09-17 · **Decided by:** human/peter

## Question

After `aiwf worktree add`, a ritual whose own session keeps working in the new
worktree has to move the session there. Claude Code offers two ways: its
`EnterWorktree` tool, or a plain `cd`. The same session later merges from a
different worktree — mainline's for `wf-patch` and `aiwfx-wrap-epic`, the epic
branch's for `aiwfx-wrap-milestone`. Which way should the start rituals use?

## Decision

The start rituals move the session into a new in-repo worktree with
`cd "<printed path>"` and never call `EnterWorktree`, and tell the session to
stay inside the repository while it works there, running anything that needs
another directory in a subshell. The wrap rituals' merge steps find the worktree
holding their merge target with `git worktree list | grep -F '[<branch>]'`, `cd`
there, and keep the branch check in the same command as the merge; `wf-patch`
checks the branch in the same command as its commit.

## Reasoning

`EnterWorktree` confines the session, and the rituals' merge runs in another
checkout by design. By Claude Code 2.1.233, a session that called it is refused
`cd` and `git -C` into any other checkout, and any compound command the
confinement cannot verify stays inside; subagents the session dispatches are
confined the same way. Counted on 2026-09-16 from this repo's Claude Code
session transcripts, which hold versions 2.1.220, 2.1.221 and then 2.1.233
onward: under 2.1.220–2.1.221, 4,559 main-session and 6,567 subagent commands
ran inside tool-entered worktrees and none was refused; from 2.1.233, 65 of
1,856 main-session commands and 220 of 2,997 subagent commands were refused —
20 for reaching another checkout, the rest ordinary loops, pipelines and
variable expansions. Read from the 2.1.273 source and not measured: the
confinement is active whenever the session holds a tool-entered worktree, and no
setting turns it off (`worktree.bgIsolation` governs background sessions only).

G-0413 gave three reasons for entering with the tool, each resting on the claim
that nothing else moves the session. Measured 2026-09-16 under Claude Code
2.1.273 in the devcontainer, a plain `cd` moves it into an in-repo worktree.
After `cd /workspaces/aiwf/.claude/worktrees/patch/aiwf-add-stale-body-claims`,
a separate command's `pwd` printed that path, and Claude Code replaced the
working-directory section of the session's instructions: it named the worktree
as the primary working directory and added worktree-specific guidance. From
there, `git -C /workspaces/aiwf rev-parse --abbrev-ref HEAD` printed `main`, and
`cd /workspaces/aiwf` succeeded.

- **Statusline.** Read from the same source and not measured: the statusline
  and hook inputs take their working directory from the value `cd` updates.
- **Session-end keep/remove prompt.** Read from the `EnterWorktree` tool
  description: `ExitWorktree` will not remove a worktree entered by path, which
  is how the rituals entered. Whether the session-end prompt offers removal for
  such a worktree was not checked.
- **Working-directory-dependent state.** The tool description names system
  prompt sections, memory files and the plans directory. The working-directory
  section follows a plain `cd`, as measured above. Memory files and the plans
  directory were not checked; automatic memory is off in the environment the
  measurement ran in.

Alternatives considered:

- **Keep entering, and exit before the merge.** It fixes the merge step only. A
  trip to main mid-work (filing a gap there), git commands in reviewer
  subagents' own worktrees, and ordinary compound commands stay refused. It also
  depends on the session calling `ExitWorktree`, whose tool description tells it
  not to call it unprompted.
- **Keep entering, and route cross-checkout work through `aiwf` subprocesses
  that take `--root`.** That depends on the confinement not recognising those
  commands, which runs against its stated purpose and can change with any Claude
  Code release.
- **Keep the multi-line `git worktree list --porcelain | awk` lookup.** It
  resolved the path and changed directory in one command, matched the branch
  exactly against `refs/heads/`, and read the path mechanically. The `grep -F`
  form is exact on the bracketed branch name and leaves the path to be read
  from its output; the branch check in the merge command is what stops a merge
  in the wrong directory under either form, so the shorter one gives up no
  guarantee the merge relies on. `-F` matters: without it the whole pattern is
  one bracket expression. Measured 2026-09-16 in the devcontainer, where `grep` is
  ugrep: `grep '[main]'` matched a line for another branch, and
  `grep '[patch/G-0673-x]'` stopped with an invalid character range error
  (exit 2); with `-F`, each matched only its own line.

## Consequences

- A `cd` to a directory outside the one the session started in does not hold:
  Claude Code returns the session to its starting directory after the command.
  Measured 2026-09-16 under 2.1.273 in the devcontainer (a bare host was not
  measured): from the patch worktree, `cd <scratchpad> && pwd` printed the
  scratchpad, and the next command's `pwd`
  printed `/workspaces/aiwf` on `main` — the main checkout, not the worktree.
  `( cd <scratchpad> && pwd )` in a subshell left the session in the worktree.
  Two things follow. A session that strays loses its worktree silently, which
  the start rituals' stay-inside instruction and the branch checks before the
  merge and the patch commit are there to catch; other commits a ritual makes
  carry no such check. And a sibling worktree cannot be entered by `cd` from a
  session started in the repository, so the start rituals send work there to a
  new session or a dispatched subagent. A session holding a tool-entered
  worktree was returned to that worktree instead — observed in a 2.1.220
  session transcript on 2026-08-19 — which a plain `cd` gives up.
- Read from the 2.1.273 source and not measured: the confinement also refuses an
  `Edit` or `Write` aimed at the main checkout while the session stands in a
  worktree. Nothing replaces that; an edit landing in the wrong checkout is
  caught only by noticing it.
- None of the ritual instructions this decision sets is pinned by a test: they
  are shipped prose, and D-0070 rules out asserting that shipped prose contains
  a phrase.
