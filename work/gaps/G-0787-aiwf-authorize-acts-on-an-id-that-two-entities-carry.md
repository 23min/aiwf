---
id: G-0787
title: aiwf authorize acts on an id that two entities carry
status: open
---
## What's missing

While two entities carry one id, after the branches that allocated it merge and before
`aiwf reallocate` renumbers one, `aiwf authorize <id>` opens, pauses and resumes scopes on that id
and exits 0. At `internal/verb/authorize.go:283`, `Authorize` takes the entity from `Tree.ByID`
(`internal/tree/tree.go:529`), which returns the first entity loaded with the id, and the commit
it writes names only the id. The verbs that write the entity's file refuse the same id, each
reporting `ids-unique`.

Measured 2026-11-24 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a
tree whose code is `main` at f068fa5a1 (`aiwf version`:
`v0.42.1-0.20261124212924-f068fa5a1f83+dirty`), in a disposable repository `a`: `aiwf init`
with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs. `b` is a
clone of `a`; each adds a milestone on `epic/E-0001-engine` and authorizes it before seeing the
other's, so both milestones are `M-0002`, "Side work" with a scope for `ai/side` and "Kept work"
with one for `ai/kept`. `b` is merged into `a`. Ids in the block name entities in that repository.

```
a$ aiwf check
work/epics/E-0001-engine/M-0002-side-work.md:2: error ids-unique: id "M-0002" is also used by work/epics/E-0001-engine/M-0002-kept-work.md …
a$ aiwf authorize M-0002 --pause hold
aiwf authorize M-0002 --pause                                                   # exit 0
a$ git log -1 --format=%B | grep aiwf-entity
aiwf-entity: M-0002
a$ aiwf show M-0002
M-0002 · Kept work · status: draft · tdd: none
  …
  Scopes (2):
    2b5499e  human/t → ai/side  state: active   opened 2026-11-24  events: 1
    67c8c13  human/t → ai/kept  state: paused   opened 2026-11-24  events: 2
```

In a second run of the same setup, the pause above followed by
`aiwf authorize M-0002 --resume go`, then
`aiwf authorize M-0002 --to ai/third --reason third --branch milestone/M-0002-kept-work`, each
exited 0, while `aiwf retitle M-0002 …`, `aiwf cancel M-0002 …`, `aiwf edit-body M-0002 …` and
`aiwf add ac M-0002 …` each exited 1 reporting `ids-unique`.

## Why it matters

A scope opened, paused or resumed while the id is shared records no more than the id, which named
two entities when the commit was written, so no reader can recover which entity the human meant.
The pause above landed on `ai/kept`'s scope, whose opener comes later in `git log` order: the
replay moves the most recently opened active scope among every commit carrying the id
(`scope.Target`, `internal/scope/scope.go:156-158`). If the human meant "Side work", `ai/side` is still
active, and nothing in the commit or in `aiwf show` says which was intended.

Every later reader inherits that guess, the agent's gate included. In a third run of the same
setup, the pause, then `aiwf reallocate work/epics/E-0001-engine/M-0002-side-work.md`
(`M-0002 -> M-0003`), then each agent's `aiwf retitle M-0003 …`: `ai/kept`'s exited 1 with
`provenance-no-active-scope`, and `ai/side`'s exited 0.
