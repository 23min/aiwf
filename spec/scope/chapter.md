# scope: one agent's delegation on one milestone

aiwf tracks planning state as markdown files in a git repository. A human lets an AI agent work
on a milestone on its own by opening a scope with `aiwf authorize`; the agent's verbs then run
under that scope, and the human can pause, resume and end it. This chapter says how one such
scope moves between states as the human, the agent and git act on it, as ADR-0056 has it for a
scope opened from its release on: a scope is named by a random id that its opening commit
carries, which a rebase or a copy keeps.

It is taken from aiwf's records, not its code: the provenance model
(`docs/design/provenance-model.md`), ADR-0047, ADR-0056, CLAUDE.md, and the maintainer's answers
in `spec/scope/audit/decisions.md` (cited as Q1 to Q18), which settle where those records and
aiwf's history disagree. The claims behind the answers are in `audit/docs.md` and
`audit/history.md`.

## 1. Glossary

The rest of this chapter uses only these names for the states and events.

Things:

- **milestone**: one milestone under an epic whose status is `active` throughout. It starts
  `in_progress`, with ACs whose bodies are written. Its status becomes `done` when it is closed.
  A reallocate gives it a new id; it is the same milestone, and every later step names it by
  that id.
- **human**: `human/t`, the principal. Runs every step but act.
- **agent**: `ai/claude`, the one agent the scope is for. Runs act.
- **scope**: the human's permission for the agent to work on the milestone. Its opening commit
  carries its id (ADR-0056).
- **branch**: the milestone's branch, `milestone/<milestone>-<slug>`, checked out throughout. The
  scope is opened on it and the agent works on it.

States:

- **none**: no scope has been opened, and the milestone is not `done`.
- **active**: the scope is active.
- **paused**: the scope is paused.
- **ended**: the human ended the scope, and the milestone is not `done`.
- **closed**: the milestone is `done`, and any scope on it is ended (Q10).

Every run starts in none.

Events. Each is one thing the human, the agent or git does:

- **authorize**: the human runs `aiwf authorize <milestone> --to ai/claude --reason "<text>"`.
- **act**: the agent runs one aiwf verb that changes the milestone, with
  `--actor ai/claude --principal human/t`.
- **pause**: the human runs `aiwf authorize <milestone> --pause "<reason>"`.
- **resume**: the human runs `aiwf authorize <milestone> --resume "<reason>"`.
- **end**: the human runs `aiwf authorize <milestone> --end --reason "<text>"`, without
  `--scope`, which ends the milestone's one scope that is not ended (Q9).
- **close**: the human meets the milestone's open ACs and promotes the milestone to `done`.
- **rebase**: `main` gains one commit that touches nothing of the scope, and the branch, which
  holds the scope's opening commit, is rebased onto it, so that commit gets a new hash (Q13).
- **copy**: a second copy of the opening commit lands in the branch's history: the branch as it
  stands is kept, `main` gains a commit and the branch is rebased onto it, and the kept branch is
  then merged back in, so the original opening commit sits beside its rebased twin (Q13).
- **reallocate**: the human runs `aiwf reallocate <milestone>`, which gives the milestone a new
  id (Q15).
- **check**: the human runs `aiwf check`.

Variables: none. Outputs: none.

## 2. The process

Each event is one step. A step is allowed or refused. An allowed step either moves to another
state, or leaves the state as it is; one that leaves it because what it asks for already holds
is a no-op, which succeeds, exits 0 and makes no commit (CLAUDE.md, commitment 7). A refused step
changes nothing and makes no commit.

### 2.1 Opening a scope

1. From none, authorize opens the scope: none moves to active (provenance model, §"Scope
   states").
2. From active, authorize is a no-op: the agent already has the permission (Q4).
3. From paused, authorize is refused; lifting a pause is resume's (Q4).
4. From ended, authorize opens a new scope, and ended moves to active. The ended scope stays
   ended (ADR-0047, §"Nothing undoes an end": "The inverse of an end is a fresh grant").
5. From closed, authorize is refused: a `done` milestone takes no new scope (Q11).

### 2.2 The agent

6. In active, act is allowed and the state stays active.
7. In none, paused, ended and closed, act is refused (Q7, Q8; for paused, the `aiwf-authorize`
   skill, §Lifecycle: "While paused, agent verbs targeting the scope refuse with
   `provenance-no-active-scope` until the scope is resumed").

### 2.3 Pause and resume

8. From active, pause moves to paused; from paused, resume moves to active (provenance model,
   §"Scope states").
9. From paused, pause is a no-op, and from active, resume is a no-op (Q9).
10. From none, ended and closed, pause and resume are refused: there is no scope to pause or
    resume (Q9).

### 2.4 Ending

11. From active and from paused, end moves to ended; the milestone's status does not change
    (Q6).
12. From none, ended and closed, end is refused: no scope is left that is not ended (Q9;
    ADR-0047, §"An operator ends a scope by naming it, and `--end` defaults to the only
    candidate": "With no `--scope` and no candidate, the verb refuses rather than converging").

### 2.5 Closing the milestone

13. From none, active, paused and ended, close moves to closed. A scope that is active or paused
    is ended by the close (provenance model, §"Ending covers paused scopes"; Q5); one already
    ended is left as it is (Q10).
14. From closed, close is a no-op: the milestone is already `done` (Q9; ADR-0036).

### 2.6 What git does to the branch

15. rebase leaves every state as it is, and so does copy (Q1, Q2, Q3). A scope is the human's
    permission, and a rebase or a copy of its opening commit neither grants nor withdraws it; the
    scope's id, which both keep, is what its pointers hold (ADR-0056).
16. reallocate leaves every state as it is: the milestone's number changes, the permission does
    not (Q15). The scope stays the milestone's under its new id, so every later step on it
    behaves as these rules say.
17. In none, and in closed when no scope was opened, there is no opening commit, so copy has
    nothing to copy.

### 2.7 check

18. check passes when `aiwf check` reports no errors; warnings are allowed. check never changes
    the state.
19. check passes in every state. An agent's verb commits only under an active scope (rules 6
    and 7), and rebase, copy and reallocate change nothing about the scope (rules 15 and 16), so
    no step leaves a commit that names a scope it should not.

## 3. What must hold

- **S1. Only an active scope lets the agent act.** act is allowed in active and refused in every
  other state.
- **S2. Only the human moves the scope.** authorize, pause, resume, end and close are the only
  steps that change the state; rebase, copy, reallocate and check never do.
- **S3. Ended stays ended.** Nothing brings an ended scope back; from ended, only authorize
  opens a scope, and it is a new one. Nothing leaves closed.
- **S4. Asking for what already holds changes nothing.** A repeated authorize, pause, resume
  or close is a no-op.
- **S5. check passes.** In every state the steps reach, `aiwf check` reports no errors.

## 4. Scenarios

Each starts in none.

| ID | Steps | The state after each step |
|---|---|---|
| S01 | authorize, act, pause, act, resume, act | active, active, paused, then act is refused, active, active |
| S02 | authorize, end, act | active, ended, then act is refused |
| S03 | authorize, act, close, act | active, active, closed, then act is refused |
| S04 | authorize, act, rebase, check | active, active, active, active (check passes; G-0751) |
| S05 | authorize, end, rebase, act | active, ended, ended, then act is refused (G-0752) |
| S06 | authorize, close, rebase, act | active, closed, closed, then act is refused (G-0752) |
| S07 | authorize, copy, pause, act | active, active, paused, then act is refused (G-0755) |
| S08 | authorize, end, copy, act | active, ended, ended, then act is refused (Q3) |
| S09 | authorize, reallocate, pause, resume | active, active, paused, active (Q15) |
| S10 | authorize, reallocate, close, act | active, active, closed, then act is refused (Q15) |
| S11 | authorize, authorize, pause, act | active, active (no-op), paused, then act is refused (Q4) |
| S12 | authorize, pause, authorize | active, paused, then authorize is refused (Q4) |
| S13 | authorize, pause, pause, resume, resume | active, paused, paused (no-op), active, active (no-op) (Q9) |
| S14 | authorize, end, end, authorize, act | active, ended, then end is refused, active, active (Q9) |
| S15 | act | act is refused (Q8) |
| S16 | close, authorize | closed, then authorize is refused (Q10, Q11) |
| S17 | authorize, pause, close, resume | active, paused, closed, then resume is refused (Q5) |

## 5. Assumptions

- One human, one agent and one milestone. The epic stays `active` and no scope is opened on it.
- The milestone starts `in_progress` with its ACs' bodies written, so close can meet them and
  promote the milestone to `done`.
- The scope is opened from ADR-0056's release on, so its opening commit carries an id (Q16).
- The agent acts only through aiwf verbs.
- The branch's own commits are the only ones that change the milestone; `main`'s commits in
  rebase and copy touch nothing of it. Where copy's merge meets a file both sides changed, the
  merge keeps the branch's version.
- One client plays one sequence of steps at a time (E-0100, §"Out of scope").

## 6. Not covered

Left out of the machine, each with the answer that leaves it out:

- **Old scopes.** A scope named by its opening commit's hash, which aiwf writes before ADR-0056's
  release, rebases and copies the way G-0751, G-0752 and G-0755 record, and ADR-0056 leaves it
  so. The epic that builds ADR-0056 adds a unit test that its readers still read such a scope as
  ended and its agent commits as authorized; a scope open across the switch to that release is
  outside the machine (Q16).
- **Ending a scope by naming it**, `--end --scope <its id>`, which does nothing when that scope
  is already ended (Q9).
- **cancel** of the milestone, which ends a scope as close does (Q14).
- **authorize `--force`** on a `done` milestone, a human override (Q11).
- **Other actors and other scopes**: another agent's scope, a `bot/` actor, a scope delegated to
  a human, and a scope on the parent epic, which also lets the agent act on the milestone
  (D-0007) (Q14, Q16).
- **Commits made around the verbs**: an agent commit made by hand, which `aiwf check` should
  judge; Q17 records where it does not.
- **Branch rules**: `--branch`, the checkout, the branch rungs and `isolation-escape` (E-0100,
  §"Out of scope").
- **Other git and aiwf steps** that change what `aiwf check` reports or what the branch rules
  say, not the scope: push, wrap commits after a close, `aiwf acknowledge illegal`, a branch
  rename, force-push and `git reset --hard`, a rebase in progress (Q15).
- **`aiwf archive` and `aiwf promote --audit-only`**, which change the milestone; whether the
  scope follows them is not measured (Q15).
- **Amend, squash and the host's merge methods**, which the verb commit output machine (M-0366)
  takes up. A squash that keeps only a commit's title loses its trailers and with them the scope
  (ADR-0056, §Consequences) (Q15).
- **A `main` that carries a change to the milestone**, such as a cancel made on another clone,
  which a rebase brings onto the branch (Q2).
- **`aiwf import`**, which is deprecated (G-0667) (Q17).

## 7. What the lifecycle does not show

- What a step writes: the scope's id, the trailers and which value a pointer holds. The machine
  has no outputs; a pointer that holds the wrong value shows only through what rebase and copy
  then do.
- Whether a step made a commit. A no-op and an allowed step that stays look the same.
- Refusal messages and codes, and the warnings `aiwf check` prints.
- What `aiwf show` prints beyond which state the scope is in.
- More than one scope on the milestone at once, which no step here produces (Q3, Q4).
