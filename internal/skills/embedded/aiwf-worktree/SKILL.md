---
name: aiwf-worktree
description: Use whenever a ritual, subagent dispatch, or ad-hoc fix needs a new git worktree in this repo. Runs `aiwf worktree add` so the worktree is created AND has aiwf's rituals (skills, agents, templates, guidance) materialized into it in one atomic step — a bare `git worktree add` leaves those absent with no warning.
---

# aiwf-worktree

`aiwf worktree add` replaces the two-command sequence `git worktree add` + `aiwf init`/`aiwf update` with one atomic step. A worktree created any other way starts with none of aiwf's gitignored, materialize-on-demand artifacts (skills, agents, templates, per-turn guidance) — every ritual and slash-command becomes invisible inside it until someone remembers to run `aiwf update` by hand.

## When to use

Any time you are about to create a git worktree for branch work in this repo — starting an epic, starting a milestone, a one-off patch branch, or a subagent that needs an isolated checkout. Use this instead of a bare `git worktree add`.

## What to run

```bash
# Create a NEW branch off a base ref, at the default in-repo placement
aiwf worktree add <branch> --base <base-ref>

# Create a worktree at an explicit path (sibling directory, any custom location)
aiwf worktree add <branch> <path> --base <base-ref>

# Reuse an EXISTING local branch (omit --base; it only applies to new branches)
aiwf worktree add <branch>

# Compose with cd — only the absolute path is printed on success
cd "$(aiwf worktree add <branch> --print-path)"
```

- `<branch>` is required. When it does not already exist as a local branch, aiwf creates it fresh starting from `--base` (default: HEAD). When it already exists, aiwf reuses it and `--base` is rejected as a usage error — you cannot re-point an existing branch's start.
- `<path>` is optional. Omit it to resolve to the configured worktree-placement directory plus the branch name; pass it explicitly for a sibling directory or any other location. An explicit path is honored verbatim — it is never redirected back in-repo, even if it points outside the repo.
- `--print-path` suppresses every other output and prints only the resulting absolute path to stdout on success, nothing on failure. This is the only mode meant for shell composition (`cd "$(...)"`); don't parse the normal ledger output for the path.

## What aiwf does

1. Runs `git worktree add`, surfacing any git failure directly (branch already checked out elsewhere, path already exists, etc.) — never reports success on a failed creation.
2. Materializes rituals into the new worktree in the same step: skills, role agents, entity templates, and the per-turn guidance import — the identical pipeline `aiwf update` runs, just targeted at the fresh worktree instead of the current checkout.
3. Prints the resulting absolute path (or, under `--print-path`, only the path).

Immediately after, `aiwf doctor --root <path>` on the new worktree reports rituals as materialized — no separate `aiwf update` step needed.

## Don't

- Don't run a bare `git worktree add` for repo branch work — the resulting worktree silently has no skills, no agents, no templates, and no guidance import until someone notices and runs `aiwf update` by hand.
- Don't expect this verb to change your shell's current directory — no child process can `chdir` its parent. Compose with `cd "$(aiwf worktree add ... --print-path)"` instead.
- Don't pass `--base` when reusing an already-existing branch — aiwf rejects the combination rather than silently ignoring the flag.
- Don't parse the normal (non-`--print-path`) output for the path in a script — that output includes a materialization ledger; `--print-path` is the stable, script-safe surface.
