# milestone: one milestone and its one acceptance criterion

aiwf tracks planning state as markdown files in a git repository. A milestone is one unit of
work inside an epic. Its acceptance criteria (ACs) are sub-elements written in the milestone's
own file. This chapter says how one milestone and its single AC move between statuses through
aiwf's verbs, and which commits those verbs make. It is taken from aiwf's design documents, not
from its code. The sources are `docs/design/design-decisions.md` (DD), Pass B of the
legal-workflow catalogue `docs/design/legal-workflows-first-principles.md` (PB, rules R-FP-NNNN),
`README.md`, ADR-0036, and decisions D-0004 and D-0039. Where they don't settle a case, or
contradict each other, the chapter says so, marked "Open question:".

## 1. Glossary

The rest of this chapter uses only these names for the states, events and output.

Things:

- **milestone**: M-0001, under an epic whose status is `active` for the whole run. Its file
  records the milestone's `status`, which is one of `draft`, `in_progress`, `done` or `cancelled`
  (DD "Six entity kinds"). Its TDD policy is `tdd: none`, so no TDD phase applies (DD
  "Acceptance criteria and TDD").
- **AC**: M-0001/AC-1, the milestone's only acceptance criterion. It exists, with a non-empty
  body, from the start. Its `status` is one of `open`, `met`, `deferred` or `cancelled` (DD
  "Acceptance criteria and TDD").
- **operator**: a human who runs every verb directly. No agent and no `--force` take part.

States. A state is named for the milestone's status and the AC's status together, as
`<milestone>_<AC>`:

- **draft_open**, **draft_met**, **draft_deferred**, **draft_cancelled**: the milestone is
  `draft`.
- **in_progress_open**, **in_progress_met**, **in_progress_deferred**,
  **in_progress_cancelled**: the milestone is `in_progress`.
- **done_met**, **done_deferred**, **done_cancelled**: the milestone is `done`.
- **cancelled_met**, **cancelled_deferred**, **cancelled_cancelled**: the milestone is
  `cancelled`.

Every run starts in draft_open.

Events. Each is one aiwf command the operator runs:

- **start**: promote the milestone to `in_progress`.
- **finish**: promote the milestone to `done`.
- **back**: promote the milestone to `draft`.
- **cancel**: cancel the milestone, which sets it to its kind's cancel status, `cancelled`
  (README, `aiwf cancel <id>`).
- **meet**: promote the AC to `met`.
- **defer**: promote the AC to `deferred`.
- **drop**: cancel the AC, setting it to `cancelled`.
- **reopen**: promote the AC to `open`.
- **check**: run `aiwf check` over the repository.

Output:

- **commit**: a git commit the step makes. Only that a commit was made is named; its message,
  trailers and files belong to other checks.

## 2. The process

Each event is one step. A step either is allowed, and may move to another state, or is refused.
A refused step changes nothing.

### 2.1 The milestone's moves

1. A milestone moves `draft → in_progress → done`, and from `draft` or `in_progress` to
   `cancelled` (PB R-FP-0009 to R-FP-0012).
2. `done` and `cancelled` are terminal for the milestone: no move leaves them (PB R-FP-0013,
   R-FP-0014). A milestone never moves back to `draft` (PB §1, "Terminal states stay terminal:
   there is no demotion"; R-FP-0008 for the epic, which the milestone mirrors).
3. `draft → done` is refused (PB R-FP-0015).
4. **finish needs the AC closed.** A milestone may not move to `done` while its AC is `open`;
   `met`, `deferred` and `cancelled` are acceptable (DD "Milestone-done implies AC progress").
5. **cancel needs the AC closed.** Cancelling a milestone is refused while its AC is `open`;
   the operator first takes the AC to `met`, `deferred` or `cancelled` (D-0004).
6. **start needs an AC.** A milestone with no ACs is refused at `draft → in_progress` (D-0039,
   point 1). An AC counts only when it is `open` or `met`, so start is refused while the AC is
   `deferred` or `cancelled` (answers.md, Q4): such a milestone has nothing to deliver, the
   emptiness D-0039 refuses.
   DD says the opposite of D-0039: "What's not a kernel rule. No 'milestone must have ≥1 AC' —
   ACs remain optional", and PB R-FP-0166 repeats it. The chapter follows D-0039, the later and
   accepted decision; `docs/audits/entity-truth-audit.md` records DD's sentence as stale.

### 2.2 The AC's moves

7. An AC moves from `open` to `met`, `deferred` or `cancelled` (PB R-FP-0046 to R-FP-0048), and
   from `met` to `deferred` or `cancelled` (DD: "`met` may move to `deferred`/`cancelled` if
   scope changes after the fact"; PB R-FP-0049, R-FP-0050).
8. `deferred` and `cancelled` are terminal for the AC (DD: "`deferred` and `cancelled` are
   terminal"). No move leaves them, so reopen from `deferred` is refused. PB R-FP-0051 left
   `deferred → open` to its Pass C (PB Q1), which settled it as DD says ("Q1 (deferred is
   terminal)", `internal/workflows/spec/rules.go`).
9. Nothing moves an AC from `met` back to `open`: no source lists that move.
10. **The milestone's status and the AC's moves.** While the milestone is `draft` or
    `in_progress`, the AC moves as rules 7 to 9 say (answers.md, Q2). Once the milestone is
    `done` or `cancelled`, its AC is frozen: every step that would change the AC is refused
    (answers.md, Q3), as PB R-FP-0064's "ACs cannot outlive the milestone" reads. DD's
    "`met` may move … if scope changes after the fact" means after `met`, while the milestone
    is live. The one exception is an `open` AC on a finished milestone, which may still be
    closed; no step here reaches that state, which only `--force`, a merge or a hand edit
    produces. A step that asks for the status the AC already has still changes nothing, and
    rule 11 applies to it.

### 2.3 Asking for the status already there

11. A promote whose target is the status already recorded, or a cancel of something already at
    a terminal status, changes nothing. It succeeds, exits 0 and makes no commit (ADR-0036,
    "Decision"). For cancel, "already at a terminal status" means any terminal status, so
    cancelling a `done` milestone is such a step too (ADR-0036, its Option A).
    This covers the AC as well (answers.md, Q1): meet on a `met` AC, defer on a `deferred` AC
    and reopen on an `open` AC succeed with no commit, and so does drop on a `deferred` or
    `cancelled` AC.

### 2.4 check

12. `aiwf check` passes when it reports no errors; warnings are allowed (README: exit code `0`
    "no errors (warnings allowed)"). check never changes the state.
13. Every verb computes the tree it would write, checks it, and writes only when the check
    raises no new finding; otherwise the working tree is never touched (DD "One git commit per
    mutating verb"). Starting from a tree that passes, every state the verbs reach therefore
    passes check.

### 2.5 What a step sends

14. A verb step that changes the milestone's file makes exactly one commit (DD "One git commit
    per mutating verb"; CLAUDE.md commitment 7). A step that changes nothing (rule 11) makes no
    commit. A refused step makes no commit. check makes no commit.

## 3. What must hold

- **M1. Terminal is terminal.** Once the milestone is `done` or `cancelled`, no step changes
  its status.
- **M2. No done with open work.** No step leaves the milestone `done` or `cancelled` while its
  AC is `open`.
- **M3. One commit or none.** Every step makes at most one commit, and a step that changes
  nothing makes none.
- **M4. The verbs keep the tree clean.** check passes in every state the verbs reach.

## 4. Scenarios

Each starts in draft_open.

| ID | Steps | The state after each step |
|---|---|---|
| S01 | start, meet, finish | in_progress_open, in_progress_met, done_met |
| S02 | start, finish | in_progress_open, then finish is refused |
| S03 | start, defer, finish | in_progress_open, in_progress_deferred, done_deferred |
| S04 | cancel | refused: the AC is open |
| S05 | start, drop, cancel | in_progress_open, in_progress_cancelled, cancelled_cancelled |
| S06 | finish | refused: `draft → done` |
| S07 | start, back | in_progress_open, then back is refused |
| S08 | start, start | in_progress_open, in_progress_open, the second making no commit |
| S09 | start, meet, finish, cancel | in_progress_open, in_progress_met, done_met, done_met, the cancel making no commit |
| S10 | start, defer, reopen | in_progress_open, in_progress_deferred, then reopen is refused |
| S11 | start, meet, reopen | in_progress_open, in_progress_met, then reopen is refused |

## 5. Assumptions

- **A1.** The repository starts with the epic `active`, M-0001 `draft` with `tdd: none`, and
  AC-1 `open` with a non-empty body, and `aiwf check` passes on it. Every step runs on the
  epic's branch, where the branch model expects a milestone's work (the layer-4 branch cells,
  `internal/workflows/spec/branch`).
- **A2.** The operator is human, so no authorization scope is involved (PB R-FP-0131).
- **A3.** The state is read from the milestone's file: its `status` and its AC's `status`.

## 6. Not covered

- More than one AC, and adding ACs.
- TDD phases, and milestones with `tdd: required` or `tdd: advisory`.
- `--force`.
- The epic's own status and moves, and archiving.
- Agents, authorization scopes and provenance.
- Renaming, retitling and editing bodies.
- git operations other than the commits the verbs make: rebases, merges, pushes, and other
  clones.

## 7. What the lifecycle does not show

- What a commit contains: its message, trailers and files.
- Which warnings check reports.
- The text of refusals.
- The AC's body, and the milestone's body sections.
