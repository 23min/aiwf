---
id: G-0779
title: isolation-escape skips agent commits on ACs and on an epic scope's milestones
status: open
discovered_in: M-0364
---
## What's missing

`isolation-escape` judges only an agent commit whose `aiwf-entity` is the scope's own entity.
`RunIsolationEscape` (`internal/check/isolation_escape.go:254`) finds a commit's scope by its
exact `aiwf-entity` (`openers := openersByEntity[entity]`, `isolation_escape.go:337`), so a
commit on an acceptance criterion, `M-NNNN/AC-N`, matches no opener, and neither does a commit on
a milestone made under its epic's scope. M-0106's goal states the finding for every commit "made
under an active AI scope" that lands "on a branch that doesn't match the scope's `aiwf-branch:`
trailer"; its AC-1 specifies the lookup the code has, "walk back to the most recent
`aiwf-scope: opened` commit on the same entity".

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository.

```
$ aiwf authorize M-0001 --to ai/claude --reason go    # aiwf-branch: milestone/M-0001-first-milestone
$ git checkout main && git merge milestone/M-0001-first-milestone
$ aiwf promote M-0001/AC-1 met --actor ai/claude --principal human/t     # exit 0, on main
$ aiwf check --since <main before these acts>                            # no isolation-escape
$ aiwf retitle M-0001 "Renamed by agent" --actor ai/claude --principal human/t   # exit 0, on main
$ aiwf check --since <main before these acts>
isolation-escape (warning) × 1 — commit 3363d61: aiwf-actor: "ai/claude" on "main" escapes the active scope's bound branch "milestone/M-0001-first-milestone"
```

With a scope opened on `E-0001` from `epic/E-0001-engine` instead, merged to `main`:

```
$ aiwf retitle M-0001 "Milestone renamed by agent" --actor ai/claude --principal human/t   # exit 0, on main
$ aiwf check --since <main before these acts>                                             # no isolation-escape
$ aiwf retitle E-0001 "Epic renamed by agent" --actor ai/claude --principal human/t       # exit 0, on main
$ aiwf check --since <main before these acts>
isolation-escape (warning) × 1 — commit b78ad9d: aiwf-actor: "ai/claude" on "main" escapes the active scope's bound branch "epic/E-0001-engine"
```

## Why it matters

An agent's commits on a milestone's ACs (adding them, promoting them, stepping their TDD
phases), and its commits on the milestones of an epic it holds a scope on, escape the one finding
that reports branch drift when made off the bound branch, `main` included. The branch convention
the finding guards holds only for commits on the scope's own entity.
