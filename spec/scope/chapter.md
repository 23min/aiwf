# scope: one agent's delegation on one milestone

aiwf tracks planning state as markdown files in a git repository. A human lets an AI agent work
on a milestone on its own by opening a scope with `aiwf authorize`; the agent's verbs then run
under that scope, and the human can pause, resume and end it. This chapter says how the agent's
scopes on one milestone, the first and each one the human opens after an end, move between states
as the human, the agent and git act on them, as ADR-0056 has it for a scope opened from its
release on: a scope is named by a random id that its opening commit carries, which a rebase or a
copy keeps.

It is taken from aiwf's records: the provenance model (`docs/design/provenance-model.md`), the
`aiwf-authorize` skill, ADR-0036, ADR-0047, ADR-0056, D-0111, E-0100, CLAUDE.md, the gaps it
cites, and the maintainer's answers
in `spec/scope/audit/decisions.md` (cited as Q1 and on), which settle where those records and
aiwf's history disagree. The claims behind the answers are in `audit/docs.md` and
`audit/history.md`. The answers to the questions the chapter's two readings raised are in
`spec/scope/answers.md`, cited as "answers.md, Q1" and on.

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
- **branch**: the milestone's branch, named `milestone/<milestone>-<slug>` from the milestone as
  it was when the branch was cut, and checked out when each step starts and ends. The scope is
  opened on it and the agent works on it.

States:

- **none**: no scope has been opened, and the milestone is not `done`.
- **active**: the scope is active.
- **paused**: the scope is paused.
- **ended**: the human ended the scope, and the milestone is not `done`.
- **closed**: the milestone is `done`, and any scope on it is ended (Q10). aiwf records the scope
  as ended in both ended and closed; what tells them apart is that a `done` milestone takes no new
  scope (Q11), where after an end the human may open one.

Every run starts in none. None, active and paused are policy: what the agent may do in each is a
choice a later decision can change. Ended and closed are facts: each records an act no ordinary
aiwf verb reverses, an end, which only a new scope follows (ADR-0047, §"Nothing undoes an end"),
and a `done` milestone, which no aiwf verb but the human override `aiwf promote --force` moves
without `aiwf check` reporting it (answers.md, Q3).

Events. Each is one thing the human, the agent or git does:

- **authorize**: the human runs `aiwf authorize <milestone> --to ai/claude --reason "<text>"`.
- **act**: the agent runs `aiwf retitle <milestone> "<title>" --actor ai/claude --principal
  human/t`, which gives the milestone a new title. The agent's other verbs are not acts (§6;
  Q19).
- **pause**: the human runs `aiwf authorize <milestone> --pause "<reason>"`.
- **resume**: the human runs `aiwf authorize <milestone> --resume "<reason>"`.
- **end**: the human runs `aiwf authorize <milestone> --end --reason "<text>"`, without
  `--scope`, which ends the milestone's one scope that is not ended (Q9).
- **close**: the human meets the milestone's open ACs and promotes the milestone to `done`.
- **rebase**: `main` gains one commit that touches nothing of the milestone, and the branch is
  rebased onto it, keeping any merge an earlier copy made (`git rebase --rebase-merges`). The
  branch's own commits get new hashes, a scope's opening commit among them where one was opened,
  except those an earlier copy merged back in, which keep theirs (Q13).
- **copy**: the branch as it stands is kept, `main` gains a commit and the branch is rebased onto
  it as in rebase, and the kept branch is then merged back in, so each commit the rebase gave a
  new hash sits beside the commit it was rebased from, a scope's opening commit among them where
  one was opened (Q13).
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
17. In none, and in closed when no scope was opened, copy's git steps run as they do anywhere
    else, with no opening commit among the commits they copy, and the state stays as it is
    (answers.md, Q2).

### 2.7 check

18. check passes when `aiwf check` reports no errors; warnings are allowed. check never changes
    the state.
19. check passes in every state. An act commits only under an active scope (rules 6 and 7) and
    leaves nothing for `aiwf check` to report, and rebase, copy and reallocate change nothing
    about the scope (rules 15 and 16), so no step leaves a commit that names a scope it should
    not.

## 3. What must hold

- **S1. Only an active scope lets the agent act.** act is allowed in active and refused in every
  other state.
- **S2. Only the human moves the scope.** authorize, pause, resume, end and close are the only
  steps that change the state; rebase, copy, reallocate and check never do.
- **S3. Ended stays ended.** Nothing brings an ended scope back; from ended, only authorize
  opens a scope, and it is a new one. Nothing leaves closed. Closed is not final: §2 decides
  every step there, and each is refused, is a no-op or leaves closed as it is (answers.md, Q1).
- **S4. Asking for what already holds changes nothing.** A repeat of an allowed authorize,
  pause, resume or close is a no-op.
- **S5. check passes.** In every state the steps reach, `aiwf check` reports no errors.

## 4. Scenarios

Each starts in none. A note names the answer that decides a step, or a gap that records what aiwf
does there before ADR-0056 is built; `conform.md` classifies each divergence its margin-2 run
found.

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
- The milestone starts `in_progress` with `tdd: none` and its ACs' bodies written, so close can
  meet the ACs without their TDD phases and promote the milestone to `done`. No step changes
  the TDD policy (Q19).
- The scope is opened from ADR-0056's release on, so its opening commit carries an id (Q16).
- Every aiwf that commits in the scope or reads it is from that release on: an older aiwf
  misreads an end this release writes, and a commit it makes points at the opening commit's SHA,
  a pointer a rebase can still break (ADR-0056, §Consequences).
- The agent acts only through aiwf verbs.
- The branch's own commits are the only ones that change the milestone; `main`'s commits in
  rebase and copy touch nothing of it, so copy's merge meets no conflict: each rebased twin
  makes the same change as the kept commit it was made from.
- One client plays one sequence of steps at a time (E-0100, §"Out of scope").

## 6. Not covered

Left out of the machine, each with the answer that leaves it out:

- **Old scopes.** A scope named by its opening commit's hash, which aiwf writes before ADR-0056's
  release, rebases and copies the way G-0751, G-0752 and G-0755 record, and ADR-0056 leaves it
  so. The epic that builds ADR-0056 adds a unit test: on one such scope, made by hand as an
  opening commit, an agent commit and an end, aiwf reads the scope as ended and the agent commit
  as authorized (Q16).
- **Ending a scope by naming it**, `--end --scope <its id>`, which does nothing when that scope
  is already ended (Q9).
- **cancel** of the milestone, which ends a scope as close does (Q14).
- **The agent's other verbs** on the milestone and its ACs (Q19), among them its change of the
  milestone's TDD policy to `required`, after which the human's close is refused in every state
  that follows, and its own close or cancel, which ends its scope with `aiwf-scope-ends` on the
  agent's commit, as the human's close does.
- **authorize `--force`** on a `done` milestone, a human override (Q11).
- **promote `--force`** out of `done` or `cancelled`, a human override. No ordinary transition
  leaves either, so there is no ordinary un-cancel (answers.md, Q3; Q15).
- **Other actors and other scopes**: another agent's scope, a `bot/` actor, a scope delegated to
  a human, and a scope on the parent epic, which also lets the agent act on the milestone
  (D-0111, §Reasoning) (Q14, Q16).
- **Commits made around the verbs**: an agent commit made by hand, which `aiwf check` should
  judge; Q17 records where it does not.
- **Other git steps on the branch**: a cherry-pick of the opening commit alone, which copies
  less than copy does, and a merge of `main` in place of a rebase, which keeps every hash (Q13).
- **Names with no event behind them**: `revoke` and `authorize-end`, names proposed for what
  `--end` does (Q15).
- **Scopes proposed and never built**: re-binding a scope to another branch, an agent
  authorizing another, and scopes limited by time, by verb or by pattern (Q15).
- **Branch rules**: `--branch`, the checkout, the branch rungs and `isolation-escape` (E-0100,
  §"Out of scope"; Q16).
- **Other git and aiwf steps** that change what `aiwf check` reports or what the branch rules
  say, not the scope: push, wrap commits after a close, `aiwf acknowledge illegal`, a branch
  rename, `git cherry-pick -x` of an agent commit, which copies an act and not the opening
  commit, a merge of a ritual branch into `main`, judged by first-parent ancestry, force-push and
  `git reset --hard`, a rebase in progress (Q15).
- **`aiwf promote --audit-only`** to `done`, which records a status the milestone's file already
  has and ends the scope as close does, and **`aiwf archive`**, which sweeps nothing while the
  epic is active (Q15).
- **Amend, squash and the host's merge methods**, which Q15 leaves to E-0100's verb commit
  output machine. A squash that keeps only a commit's title loses its trailers (ADR-0056,
  §Consequences), so one that takes in the opening commit loses the scope.
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
- More than one scope on the milestone that is not ended, which no step here produces (Q3,
  Q4).
