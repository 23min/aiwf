---
id: G-0616
title: Promote's branch guard sends the operator to a branch without the entity
status: open
priority: high
---
## What's missing

The promote branch guard tells the operator to check out a branch where the
entity may not exist, and following that instruction replaces a clear refusal
with a worse one.

`internal/verb/promote_branch_guard.go:50` refuses an activating promote landing
on the wrong branch and closes with `` `git checkout <expected>` and retry ``.
The guard compares branch names only. It does not ask whether the entity it is
refusing to promote is reachable from the branch it is sending the operator to.

Measured 2026-08-22 in a disposable repo built by following
`docs/workflows.md` §1, against `main@v0.32.0-928-gad5bc521d`:

- `aiwf add epic --title "Front-end auth widgets"` on branch
  `epic/E-0001-auth-rewrite` allocated E-0002 there.
- `aiwf promote E-0002 active` on that branch → exit 2, *"refusing to land on
  "epic/E-0001-auth-rewrite" — this activation is expected on "main" …
  `git checkout main` and retry"*.
- `git checkout main` then `aiwf promote E-0002 active` → exit 2,
  *"entity "E-0002" not found"*.
- `git ls-tree -r --name-only main | grep -c E-0002` → 0, while the same command
  against `epic/E-0001-auth-rewrite` → 2.

The remedy that works is to merge the branch carrying the entity into the
expected branch first, or to pass `--force --reason "..."`. The guard names
neither.

## Why it matters

Every other refusal measured in that walk named a remedy that worked: `--tdd`
is required, add an AC first, write prose under the AC heading, promote the AC
before the milestone. Those are the good failure mode — a clear stop with an
actionable next step. This one is the opposite: the operator does exactly what
the message says and lands on a message that no longer names the real problem,
because the entity has gone out of view.

The sequence is reachable from the documented flow rather than exotic.
`docs/workflows.md` §1 never mentions branches, so a reader following it
linearly stays on whatever branch the previous step left them on, allocates the
next entity there, and meets the guard. Planning a second epic while working on
a first produces the same shape.

A refusal that misdirects costs more than a refusal that merely stops, because
the operator spends their next action making the state harder to reason about.
