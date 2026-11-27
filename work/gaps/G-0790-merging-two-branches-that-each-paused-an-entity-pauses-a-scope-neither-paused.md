---
id: G-0790
title: Merging two branches that each paused an entity pauses a scope neither paused
status: open
---
## What's missing

Merging two branches that each paused the same entity can pause a scope neither branch
paused. A pause commit names no scope: `scope.Replay` (`internal/scope/scope.go:132`) moves
the scope `scope.Target` (`scope.go:162`) picks, the most recently opened active one, and
applies an entity's pauses in `git log` order. With scopes A and B open, each branch's pause
moved B, the newer. Once both branches are merged, whichever order they merge in, the first
pause in log order moves B and the second finds B paused and moves A, which no one paused.
Every reader of the replay then holds A paused, including `aiwf show`, the verbs' scope gate
and `aiwf check`.

Measured 2026-11-27 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built
from `main` at 170e9f45f, in a disposable
repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in
progress, and `milestone/M-0001-first-milestone` checked out. Commit dates are set one
minute apart, so `git log` order is the order shown. The scope lines list each entry of the
JSON's `result.scopes` as SHA, agent and state. Ids in the block name entities in that
repository.

```
$ aiwf authorize M-0001 --to ai/claude --reason "claude's delegation"   # scope A
$ aiwf authorize M-0001 --to ai/codex --reason "codex's delegation"     # scope B
$ git checkout -b X
$ aiwf authorize M-0001 --pause "hold codex"
$ git checkout milestone/M-0001-first-milestone
$ git checkout -b Y
$ aiwf authorize M-0001 --pause "hold codex"
$ aiwf show M-0001 --format json
   c2841ea ai/claude active
   9807e4f ai/codex paused
$ aiwf retitle M-0001 "Retitled by claude" --actor ai/claude --principal human/t
  aiwf retitle M-0001 -> "Retitled by claude"
  # exit 0
$ aiwf check                                                           # exit 0
$ git checkout milestone/M-0001-first-milestone
$ git merge --no-ff X
$ git merge --no-ff Y
$ aiwf show M-0001 --format json
   c2841ea ai/claude paused
   9807e4f ai/codex paused
$ aiwf check
  error provenance-authorization-paused: commit 53531e3: aiwf-authorized-by: c2841ea references a scope that was paused
  # exit 1
$ aiwf retitle M-0001 "Retitled again by claude" --actor ai/claude --principal human/t
  aiwf retitle: provenance refused: actor "ai/claude" has no active scope authorizing this act (provenance-no-active-scope)
  # exit 1
$ aiwf authorize M-0001 --resume "pause lifted"                         # exit 0
$ aiwf show M-0001 --format json
   c2841ea ai/claude paused
   9807e4f ai/codex active
```

The same happens when both scopes are one agent's, the state G-0460 records. With one active
scope, the second merged pause finds none to move and the replay skips it.

## Why it matters

An agent loses a delegation no one withdrew. After the merge every verb `ai/claude` runs is
refused with `provenance-no-active-scope`, and `aiwf show` reports its scope paused, with
nothing recording that both pauses were meant for `ai/codex`.

The obvious repair makes it worse. `aiwf authorize --resume` moves the most recently opened
paused scope, so the human's resume revives `ai/codex`, the agent both branches paused, and
`ai/claude` stays paused.

`aiwf check` blocks the push of work the verb accepted. `ai/claude`'s retitle on branch Y ran
through the verb while its scope was active there; once both branches are merged it draws
`error provenance-authorization-paused`, and the pre-push hook refuses it until a human runs
`aiwf acknowledge illegal`.
