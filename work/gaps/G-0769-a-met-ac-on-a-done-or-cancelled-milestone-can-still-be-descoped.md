---
id: G-0769
title: A met AC on a done or cancelled milestone can still be descoped
status: open
---
## What's missing

D-0109 freezes a finished milestone's ACs: once the milestone is `done` or `cancelled`, a
`met` AC is not moved to `deferred` or `cancelled`. The kernel still allows both moves.
`promoteAC` and `cancelAC` in `internal/verb/ac.go` never read the parent milestone's status, so
the move succeeds on a finished milestone and makes a commit, and `aiwf check` then reports no
error. The same moves are stated without the parent's status in the spec table's `met` cells in
`internal/workflows/spec/rules.go`, the comment on `acTransitions` in
`internal/entity/transition.go`, and design-decisions.md ("if scope changes after the fact").

Measured on Linux 7.0.14 in the devcontainer, with the installed `aiwf` built from main at
417f992f7 (no Go change since, up to 2de605437), in a scratch repository: E-0001 `active`,
M-0001 `draft` with `tdd: none` and AC-1 `open`, on the epic branch. Every entity id in this
section is the scratch repository's own; in this tree those ids name unrelated archived
entities.

```
$ aiwf promote M-0001 in_progress
aiwf promote M-0001 draft -> in_progress
exit 0
$ aiwf promote M-0001/AC-1 met
aiwf promote M-0001/AC-1 open -> met
exit 0
$ aiwf promote M-0001 done
aiwf promote M-0001 in_progress -> done
exit 0
$ aiwf promote M-0001/AC-1 deferred
aiwf promote M-0001/AC-1 met -> deferred
exit 0
$ grep -E "^status:|^      status:" work/epics/*/M-0001-*.md
status: done
      status: deferred
```

From the same `done` state, `aiwf cancel M-0001/AC-1` printed `aiwf cancel M-0001/AC-1 ->
cancelled`, exit 0. After start, meet and `aiwf cancel M-0001` (`aiwf cancel M-0001 ->
cancelled`, exit 0), `aiwf promote M-0001/AC-1 deferred` printed `aiwf promote M-0001/AC-1 met
-> deferred`, exit 0, leaving the milestone `cancelled` and AC-1 `deferred`.

## Why it matters

A finished milestone's record changes after it closed. A criterion it delivered as `met` comes
to read `deferred` or `cancelled` while the milestone stays `done`, and `aiwf check` raises no
error, so the change of scope lands silently inside a closed record rather than in a new entity
that references it.
