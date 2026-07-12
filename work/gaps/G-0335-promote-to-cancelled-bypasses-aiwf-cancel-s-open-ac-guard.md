---
id: G-0335
title: promote to cancelled bypasses aiwf cancel's open-AC guard
status: addressed
addressed_by_commit:
    - 6ec1c579
---
## What's wrong

The two surfaces that drive a milestone to `cancelled` disagree on their AC guard:

- `aiwf cancel <M>` **blocks** cancelling a milestone that has an open AC
  (`milestone-cancel-non-terminal-acs`: "dispose each … before cancelling"), per
  the deliberate cancel-cascade guard (D-0003 / D-0004).
- `aiwf promote <M> cancelled` reaches the same terminal state with **no** such
  guard.

Two paths to one transition disagree, so the guard `aiwf cancel` enforces is
trivially bypassable via `aiwf promote`.

## Evidence (traced + reproduced on v0.20.0)

- The promote CLI routes `args[1]` — including `cancelled` — through `verb.Promote`,
  never `verb.Cancel` (`internal/cli/promote/promote.go`).
- `verb.Cancel` carries the milestone cascade guard (`internal/verb/promote.go`,
  via `entity.MilestoneCanGoDone`); `verb.Promote` does not, and no check rule
  fires on `cancelled` with open ACs.
- Reproduced: with one open AC, `aiwf cancel <M>` was refused
  (`milestone-cancel-non-terminal-acs`) while `aiwf promote <M> cancelled` on the
  same milestone succeeded (`in_progress → cancelled`).

## Direction (decision needed)

Two clean resolutions:

1. Make `aiwf promote <id> cancelled` enforce the same cascade guard as
   `aiwf cancel` (consistency: both block open-AC cancellation).
2. Reject `cancelled` as an `aiwf promote` target and make `aiwf cancel` the single
   cancel surface — lean, because one verb then owns cancel semantics and its
   guards, matching the "what verb undoes this" design discipline (`cancel` is the
   dedicated inverse). The FSM keeps the `→ cancelled` edge for `cancel`'s use.

## Provenance

Surfaced by formal verification of aiwf v0.20.0; confirmed here by code trace and
measured reproduction.
