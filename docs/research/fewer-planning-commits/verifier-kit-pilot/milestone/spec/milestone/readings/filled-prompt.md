You are one of two independent readers of a chapter. Turn it into a spine: one state machine
in the YAML format described under "Spine format" below. Another reader does the same without
seeing your work, and a tool compares the two readings case by case. Every place where you
differ becomes a question for the chapter's owner, so read the chapter as written. Do not
improve it, and do not fill gaps from what such systems usually do.

Everything you need is in this message. Do not read, search for or run anything else.

Rules:

1. Use only the state, event and output names in the chapter's glossary, spelled exactly as
   there. The glossary declares no variables, so declare none, and declare the glossary's
   outputs under `outputs:`. Give every rule that moves or stays `outputs:` with what
   its step sends, `[<output>]` or `[]` for nothing; where the chapter doesn't say what a step
   sends, the rule is undecided (rule 3). Use `machine: milestone`.
2. Decide every case. For every state that is not final and every event, write exactly one
   rule: `to: <state>`, `stay: true`, `ignore: true` or `reject: true`. Final states get no
   rules.
3. Where the chapter does not decide a case, do not guess. Leave that rule out and write, where
   it would go, a comment `# undecided: <what the chapter does not say>`.
4. Give every state a `kind` (fact, policy or design) and mark the final ones `terminal: true`.
   Set `initial`. Write no `examples`.
5. Answer with the YAML file only, nothing before or after it.

## Chapter

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
   point 1). This milestone always has AC-1.
   Open question: does a milestone whose only AC is `cancelled`, or `deferred`, count as one
   with no ACs for this rule? DD keeps a cancelled AC in the milestone's list of ACs ("Cancelled
   ACs stay in `acs[]` at their original position"), but D-0039 doesn't say whether such a
   milestone has ACs in its sense.
   Open question: DD says the opposite of D-0039: "What's not a kernel rule. No 'milestone must
   have ≥1 AC' — ACs remain optional", and PB R-FP-0166 repeats it. The chapter follows D-0039,
   the later and accepted decision. The question is whether DD and PB are now wrong.

### 2.2 The AC's moves

7. An AC moves from `open` to `met`, `deferred` or `cancelled` (PB R-FP-0046 to R-FP-0048), and
   from `met` to `deferred` or `cancelled` (DD: "`met` may move to `deferred`/`cancelled` if
   scope changes after the fact"; PB R-FP-0049, R-FP-0050).
8. `deferred` and `cancelled` are terminal for the AC (DD: "`deferred` and `cancelled` are
   terminal"). No move leaves them, so reopen from `deferred` is refused.
   Open question: PB R-FP-0051 calls `deferred → open` possibly legal "if work resumes" and
   leaves it to its Pass C (PB Q1). DD says `deferred` is terminal. The chapter follows DD.
9. Nothing moves an AC from `met` back to `open`: no source lists that move.
10. **The milestone's status and the AC's moves.**
    Open question: may the AC move (meet, defer, drop) while the milestone is `draft`? No
    source says.
    Open question: may the AC move while the milestone is `done` or `cancelled`? DD allows
    `met → deferred/cancelled` "if scope changes after the fact" without naming the
    milestone's status. PB R-FP-0064 says an AC's lifecycle "is bounded by its parent
    milestone's lifecycle" and that "ACs cannot outlive the milestone". Neither says whether a
    terminal milestone's AC may still change.

### 2.3 Asking for the status already there

11. A promote whose target is the status already recorded, or a cancel of something already at
    a terminal status, changes nothing. It succeeds, exits 0 and makes no commit (ADR-0036,
    "Decision"). For cancel, "already at a terminal status" means any terminal status, so
    cancelling a `done` milestone is such a step too (ADR-0036, its Option A).
    Open question: ADR-0036 speaks of "the entity" and of the kinds' statuses. Does it cover an
    AC as well? If it does, meet on a `met` AC, defer on a `deferred` AC and reopen on an `open`
    AC succeed with no commit, and drop on a `deferred` or `cancelled` AC does too. If it
    doesn't, those steps are refused as moves the AC's lifecycle doesn't list (PB R-FP-0053,
    "`open → open` … is not a verb-allowed transition (or, if allowed, is a no-op)", leaves it
    open).

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
  AC-1 `open` with a non-empty body, and `aiwf check` passes on it.
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


## Spine format

# Spine format

A spine is one state machine for one feature: which states exist, which events can happen,
and what each event does in each state for each combination of variable values, including, when
the spine declares outputs, what the step sends to another system. It covers lifecycle,
permissions and outputs only. Computations, content and timing belong in other views (see
`docs/design.md`). Changing this format needs an issue and the owner's approval first.

## File

`spec/<machine>/spine.yaml`. Top-level keys, and nothing else:

| Key | Required | Holds |
|---|---|---|
| `machine` | yes | the machine's name |
| `section` | no | the chapter section it was ratified from |
| `vars` | no | each variable mapped to a short, finite list of values, each listed once (a repeat is a `listed-twice`) |
| `states` | yes | each state mapped to `{kind: fact|policy|design, terminal: true}` (`terminal` optional) |
| `initial` | yes | the starting state |
| `events` | yes | the list of event names, each listed once (a repeat is a `listed-twice`) |
| `outputs` | no | the list of output names, each listed once (a repeat is a `listed-twice`); see "Outputs" |
| `rules` | yes | the list of rules |
| `examples` | no | worked examples that `check` replays |

```yaml
machine: book
section: lending-one-book
vars:
  who: [member, librarian, visitor]
states:
  on_shelf: {kind: policy}
  on_loan:  {kind: policy}
  held:     {kind: design}
  lost:     {kind: fact, terminal: true}
initial: on_shelf
events: [lend, renew, return, hold, write_off]
rules:
  - {from: on_shelf, event: lend, when: "who == 'member'", to: on_loan}
  - {from: on_shelf, event: lend, when: "who != 'member'", reject: true}
  - {from: on_loan, event: renew, when: "who != 'visitor'", stay: true}
  - {from: on_shelf, event: return, reject: true}
  - {from: on_loan, event: hold, when: "who == 'librarian'", ignore: true}
examples:
  - name: a member borrows and returns
    steps: ["lend who=member", "return who=member"]
    expect: on_shelf
```

## State kinds

- **fact:** records something irreversible in the world (a book written off as lost). Changes
  should be rare and need a decision.
- **policy:** a business choice, expected to change, with a decision behind each change.
- **design:** an implementation choice; it may change freely while facts and policy hold.

## Rules

Each rule has `from`, `event`, an optional `when` and exactly one outcome; in a spine that
declares outputs, a rule that moves or stays also says what it sends (see "Outputs"):

| Outcome | Meaning |
|---|---|
| `to: <state>` | the event is allowed and moves to that state |
| `stay: true` | allowed, same state |
| `ignore: true` | accepted as a no-op, not an error |
| `reject: true` | refused |

For every non-terminal state, every event and every combination of variable values, exactly
one rule must match. `check` reports `undecided-cell`, `missing-case` and `overlap` otherwise.

## Conditions

`when` is a condition over the declared variables, whose values are all text:

- `==` and `!=` compare a variable or a text with a variable or a text: `who == 'member'`,
  `borrower != owner`.
- `in` and `not in` have a variable or a text on the left. On the right is a list of one or more
  texts, `who in ['member', 'librarian']`, or a variable or a text, which tests for a substring:
  `'lib' in who`.
- `not`, `and` and `or` join comparisons, `not` binding tightest and `or` loosest; parentheses
  group them.
- A text is in single or double quotes and holds any character but its own quote and a
  backslash. A variable is named as declared, in letters, marks, digits and `_`, starting with a
  letter or `_`; `and`, `or`, `not` and `in` are not names.
- Spaces, tabs and line breaks may go before, between and after the parts.

Anything else, such as a number, a variable alone, a list beside `==` or an undeclared name, is
a `bad-condition`. Numeric rules ("over 1,000") are split into named ranges at the thresholds the
chapter gives, so the cases stay enumerable.

## Outputs

An output is something a step sends to another system: a commit, a push, a mail, a published
event. A spine that declares `outputs:`, a list of names, says on every rule that moves or stays
what its step sends: `outputs: [<output>]`, or `outputs: []` for nothing. Only the name is
written; what is sent, such as which address or which files, belongs to content checks. The
answer to the caller is the outcome, not an output.

```yaml
outputs: [send_notice]
rules:
  - {from: on_shelf, event: lend, when: "who == 'member'", to: on_loan, outputs: []}
  - {from: on_loan, event: renew, when: "who != 'visitor'", stay: true, outputs: [send_notice]}
  - {from: on_shelf, event: lend, when: "who != 'member'", reject: true}
```

- A rule sends at most one output; a list of more is a `spec-shape`. A step that sends two
  things usually hides a state between them.
- In a spine that declares `outputs:`, a rule that moves or stays without `outputs` is a
  `missing-key`, and an output the list doesn't name is an `unknown-output` from `check`.
- An `ignore` or `reject` sends nothing, so its rule has no `outputs`; nor has any rule of a
  spine that declares no outputs. Either is an `unknown-key`.
- What a step sends is part of its outcome: two rules that match the same case and move alike
  but send differently are an `overlap`, and `diff` compares what each case sends.

`spine conform` compares what each step of the code sends, as the adapter reports it, with what
the rule says (`tool-contract.md`, "The adapter protocol"). A spine without `outputs:` says
nothing about what its steps send.

## Text only

Every key, and every name and value the format defines, is text: the machine and section,
state names, events, outputs, `initial`, a rule's `from`, `event`, `to`, `when` and outputs,
variable names and values, and in worked examples and behavior files `name`, `expect` and each
step. Steps pass values as text, so a number could never match one. Quote anything YAML would
read as something else: `"yes"`, `"no"`, `"on"`, `"off"`, `"012"`, `"1.5"`, `"2026-10-14"`,
`"null"`. A key read as true or false is a `yaml-boolean-key` and such a value a
`yaml-boolean-value`; any other key that isn't text is an `unknown-key`, and any other value a
`spec-shape`.

That is the reading of plain words that YAML 1.1 readers share, so a spine means the same
whatever YAML a tool reads it with:

- `yes`, `no`, `on`, `off`, `true` and `false`, in lower case, with a capital or in capitals,
  are true or false;
- `null`, `Null`, `NULL`, `~` and nothing are null;
- numbers, such as `12`, `-1`, `012`, `0x1F`, `1_000`, `1.5`, `.inf` and `1:20`, dates and times,
  such as `2026-10-14` and `2026-10-14 12:00:00`, and the keys `<<` and `=` are none of these,
  and not text either.

Quoted words, and words tagged `!!str`, are text. A value with any other explicit tag, `!`
included, is not text, except `!!seq` on a list and `!!map` on a mapping. `terminal` is `true`
or `false`, and a missing or null `vars`, or state details, is an empty mapping. YAML that
`spine` can't read, such as a `%YAML 1.2` or unknown directive, or a `\/` escape, makes the
file `spec-unreadable`.

A file is UTF-8 text. An escape for a surrogate, U+D800 to U+DFFF, such as `"\ud800"` or the
JSON-style pair `"\ud83d\ude00"`, isn't a character: the file is `spec-unreadable`. Write the
character itself, or `\U` and its eight hex digits.

Each mapping gives a key once. A key given twice, such as `who` twice under `vars`, makes the
file `spec-unreadable`, and the finding names the key and its lines.

## Steps, examples and behaviors

A step is `"<event> <var>=<value> ..."`, for example `"renew who=visitor"`. It names a
declared event and gives every declared variable exactly once, with one of that variable's
values, and no other variable, so it names one whole case; anything else is a `bad-step`.
`steps` is always a list of steps, possibly empty. Every name in an example or behavior must be
declared, so a misspelling is reported rather than refused, which in `invalid/` would pass.

- **Examples** in the spine have `name`, `steps` and `expect`, all required and nothing else,
  where `expect` is a declared state (the end state), `rejected` (some step is refused) or
  `undefined` (some step has no rule); anything else is a `bad-value`.
- **Behavior files** live in `spec/<machine>/behaviors/valid/` and `.../invalid/`, one YAML file
  each: a mapping with `steps` and an optional `expect`, and nothing else. `expect` is a
  declared state, the end state, or else a `bad-value`. Files in
  `valid/` must be allowed and end as expected. Files in `invalid/` must not be: some step is
  refused or undecided, or the run ends somewhere other than `expect`.

## Readings

Unratified interpretations of a chapter, `spec/<machine>/readings/*.yaml`, use the same format.
`spine diff` compares two of them, or a reading against the ratified spine.

## Citing a cell, or the whole spine

A content check states which cell it depends on, anywhere in the file:

```python
# covers: on_loan/renew@3f2a9c41d07e
```

`spine hash <spec>` prints the current hash of every cell. When the rules for a cell change,
what they send included, its hash changes and `spine stale` reports every check that cites the
old one.

A check that depends on the whole machine rather than on single cells, such as one that
explores every interleaving, cites the spine instead (#59):

```python
# covers: spine@23fc1b40ab2e
```

That hash covers what the machine does: the states and which are terminal, `initial`, the
events, the outputs when the spine declares them, the variables and their values, and the rules,
with what each sends. The machine's name, its section, a state's kind, the worked examples,
comments and the order anything is written in leave it alone. `spine hash` prints it first, as
`spine@<hash>`. When it changes, `spine stale` reports every check that cites the old one; while
it is current, `stale` counts every transition as cited.

A citation is written exactly so, on one line: `covers:`, optional spaces, the cell or `spine`,
`@`, and the hash in lowercase hex, as `spine hash` prints it. Text on one line that is meant as
one but isn't, which is the word `cover` or `covers` in any case, an optional colon, at most one
word, then `@`, with spaces allowed between them, is reported as `bad-citation` with its file and
line (#64): `covers on_loan/renew@3f2a9c41d07e` without its colon, a space before `@`, or a hash
in capitals would otherwise tie the check to nothing. Text with no `@` on the line, such as
`this test covers: edge cases`, is left alone, and so is a longer word, such as `coverage` or
`recovers`.

