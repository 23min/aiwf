---
id: G-0440
title: AC entities not created until start-milestone; milestones land bare on main
status: addressed
priority: high
addressed_by_commit:
    - 2c946eb0
---
## What's missing

`aiwfx-plan-milestones` fills each milestone spec's informal prose "Acceptance
Criteria" section and merges to main by default (its step 10), but never
calls `aiwf add ac` — the verb that creates the FSM-tracked `acs[]`
frontmatter entries and their `### AC-N` body headings. That call happens
inside `aiwfx-start-milestone`'s preflight instead ("Confirm the spec has its
ACs landed via `aiwf add ac`... If the spec was hand-written and `acs[]` is
empty, ask the user whether to add them now"), deep inside the worktree, well
after the milestone has already been sitting merged on main since planning.

The result: a `draft` milestone visible on main can carry zero AC entities,
or AC entities with empty `### AC-N` bodies, for however long it sits before
someone starts it — invisible to any reader (human or LLM) who hasn't
checked out the epic's worktree. G-0216/D-0039 already built the mechanical
"contract before code" guard for exactly this discipline, but it only fires
at the `draft → in_progress` transition — one FSM stage later than where the
visibility gap actually lives.

A structural fix: move `aiwf add ac` plus AC-body content-filling into
`aiwfx-plan-milestones`, before its merge-to-main step. Add a
warning-severity check-time finding (extending `internal/check/acs.go`
alongside `milestoneDoneIncompleteACs`) for a `draft` milestone with zero ACs
or empty AC bodies — warn, not block, consistent with D-0039's
block-at-transition/warn-at-rest split, since draft is a legitimate
mid-planning state.

## Why it matters

A reader — human or LLM — without the epic's worktree checked out should be
able to learn what a milestone is about from main alone. Today that
assumption fails for any milestone between allocation and start: the
informal prose section says what's planned, but the mechanically-tracked
contract (the actual ACs) doesn't exist yet where anyone without the
worktree can see it. This is the same "contract-first" discipline G-0216
names, just unenforced one stage earlier.
