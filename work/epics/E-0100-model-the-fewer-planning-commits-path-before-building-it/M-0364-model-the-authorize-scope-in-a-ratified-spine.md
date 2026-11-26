---
id: M-0364
title: Model the authorize scope in a ratified spine
status: in_progress
parent: E-0100
tdd: none
acs:
    - id: AC-1
      title: Every question the audit raises has a recorded answer
      status: met
    - id: AC-2
      title: The chapter answers every case the two readings split or leave open
      status: met
    - id: AC-3
      title: The ratified spine passes spine check and matches its drawing
      status: met
    - id: AC-4
      title: Every divergence conform reports against today's aiwf is classified
      status: met
---
## Goal

A ratified spine for one agent's delegation scope as ADR-0056 has it, and a record of where
today's `aiwf` departs from it.

## Context

The pilot's scope spine in `docs/research/fewer-planning-commits/verifier-kit-pilot/scope/` has
the states none, active and ended and the events `authorize`, `act`, `end`, `rebase` and `check`.
It was written knowing G-0751 and G-0752, without an audit or readings, and has no pause, resume
or copied opening commit (G-0755). Its adapter plays those events with `aiwf` in disposable
repositories.

## Acceptance criteria

### AC-1 — Every question the audit raises has a recorded answer

The audit's every question has an answer the chapter is written from. **Pass criterion**: each
conflict, case no source covers and name that doesn't fit the lifecycle, as recorded in
`spec/<machine>/audit/docs.md` and in `audit/history.md` when the history reader runs, has an
entry in `audit/decisions.md` giving its date, the answer, the reason and the sources the answer
makes wrong; each entry whose answer makes a doc wrong cites the gap that lists those docs.
**Edge cases**: an answer no source gives; a conflict between the docs and the code, which is
one source among the others; a case the two audit files name differently. **Code references**:
`spec/<machine>/audit/`; verifier-kit's prompts B1–B3 at v0.3.3.

### AC-2 — The chapter answers every case the two readings split or leave open

Two readings on different models are made from the chapter, and the chapter answers every case
they split or leave open. **Pass criterion**: `readings/reading-a.yaml` and `reading-b.yaml`
come from two different models, each under the header verifier-kit's step 4 asks for (reader,
model, date, and the `git hash-object` of the chapter, prompt and format text it was given);
every finding of `spine diff` between them, and every `undecided-cell` and `missing-case` from
`spine check` of each, has an answer recorded under `spec/<machine>/` and written into the
chapter at the rule it settles. **Edge cases**: a case both readings leave out, which only
`spine check` shows; a reader's reply saved edited, which it must not be. **Code references**:
`spec/<machine>/chapter.md`, `spec/<machine>/readings/`.

### AC-3 — The ratified spine passes spine check and matches its drawing

The spine with the answers in it passes `spine check`, and its drawing is the one `spine draw`
makes from it. **Pass criterion**: `spine check spec/<machine>/spine.yaml` exits 0; `spine diff`
of the spine against each reading reports only cases an answer decided; `spine draw` of the
spine with `drawing.yaml` prints no warning, and its output is byte-identical to the committed
`drawing.svg`. The spine's header names the lifecycle, says it was built from `readings/` and the
answers and where those are, and says "Ratified by the reviewed merge of this file."
**Edge cases**: a state no rule reaches, which `spine check` refuses; a diff finding no answer
accounts for, which goes back as a question. **Code references**: `spec/<machine>/spine.yaml`,
`drawing.yaml` and `drawing.svg`; verifier-kit's `docs/reference/spine-format.md` and
`docs/reference/drawing-layout.md` at v0.3.3.

### AC-4 — Every divergence conform reports against today's aiwf is classified

`spine conform` plays the ratified spine against today's `aiwf`, and every divergence it reports
is classified. **Pass criterion**: `conformance/adapter_<machine>.py` plays each event with the
`aiwf` on `PATH` in a disposable repository and only translates, and
`conformance/adapter_<machine>.cmd` holds the line that starts it; a run at margin 2 or more is
recorded with the verifier-kit commit, the `aiwf version` it ran and its run time; each
divergence is classified as the change ADR-0056 makes, or as a defect with a gap.
**Edge cases**: a divergence the adapter causes, which is fixed in the adapter and the run
repeated; a divergence an open gap already describes, such as G-0751, G-0752 or G-0755, cites
that gap instead of filing another. **Code references**: `conformance/`; the pilot's
`adapter_scope.py`.

## Constraints

- The pilot's scope spine is not a reading: the readings start from the chapter alone.

## Out of scope

- Any change to `aiwf`.

## Dependencies

- None among E-0100's milestones.
- E-0100's open question on installing verifier-kit into the repository, settled on
  2026-11-22: a pinned clone outside it (E-0100, §"Open questions").

## References

- ADR-0056.
- G-0751, G-0752 and G-0755.

## Release note

Nothing user-facing changes: aiwf's code and docs are as they were. For maintainers, the
repository gains `spec/scope/`, a verifier-kit model of one agent's authorize scope on a
milestone as ADR-0056 has it; `conformance/adapter_scope.py`, through which `spine conform`
plays that model against an `aiwf` binary; and `spec/check.py`, which re-checks a machine's
records against what E-0100's milestones ask of them. `spec/README.md` says how to run them.

## Decisions made during implementation

- The audit's answers, each with its date, reason and the sources it makes wrong:
  `spec/scope/audit/decisions.md`.
- The answers to the readings' questions: `spec/scope/answers.md`.
- verifier-kit runs from a pinned clone of v0.3.3 outside the repository: E-0100, §"Open
  questions"; the commands are in `spec/README.md`.
- The readings are by Claude Sonnet 5.5 and Claude Fable 5.2, named in each reading's header.
- The classifications made at M-0364/AC-4: `spec/scope/conform.md`.

## Validation

Run 2026-11-26 on this milestone's branch in the Linux devcontainer: git 2.51.1, Python 3.12.3,
verifier-kit v0.3.3 at `0952e0f` from a fresh clone built as `spec/README.md` says (`$VK`, its
`spine` built by go1.27.1), and the `aiwf` that `spec/scope/conform.md` names, built by
go1.25.12.

- `aiwf check`: exit 0, 0 errors, 1 warning, `provenance-untrailered-scope-undefined` (the
  branch has no upstream). `aiwf check --since cc0d64cff`, which runs the provenance audit that
  warning names: exit 0, no findings.
- Go tests, build and lint: not run. No Go, module, `Makefile`, lint, workflow, embedded or
  guidance file changes (`git diff --name-only cc0d64cff..HEAD` over those paths is empty), so
  CLAUDE.md's cadence skips them. `python3 -m py_compile conformance/adapter_scope.py
  spec/check.py`: exit 0.
- M-0364/AC-1 to M-0364/AC-4: `python3 spec/check.py scope "$VK"` checks what its header lists.
  Expected: every line `ok`, exit 0. Observed: `all passed`, exit 0. Each rule its header lists was
  then broken in a copy of the tree by the edits below, one at a time, and each time the check
  exited 1 naming the rule broken, or stopped with an error where its header says it does: Q1's
  date emptied; Q13's sources field removed; every answer's `Q<n>.` heading renamed; `docs.md`'s
  Conflicts and Names sections and `history.md`'s Not covered section renamed; `docs.md` removed;
  `history.md` removed; `decisions.md` removed; a claim id no answer settles added to `docs.md`'s
  Conflicts; a pair no answer names added to `docs.md`'s Not covered, on a `- ` line and on a `1. `
  line; every list marker taken out of both Not covered sections; the part Outputs of `docs.md` §4
  left unnamed by the answers, renamed Output so they name it by a longer word, named as States,
  and named only across Q19's and Q20's fields; a part added to `docs.md` §4; `docs.md` §4's
  headings made plain lines; reading B given reading A's model; reading B's first line left naming
  no model, with a model named in a later comment line; reading A's header moved to its end;
  reading A's prompt hash changed in one digit, and its chapter, prompt and spine-format hashes
  each given one more digit; the chapter's citation of answers.md Q2 removed; reading A's initial
  state made active, its none/close rule made a refusal, and a state no rule reaches added to it;
  both readings given the spine's own text below their headers; `spine.yaml`'s none/rebase rule
  removed, its none/close rule made a refusal, and its ratification sentence removed; none moved,
  none moved outside the drawing, and closed dropped in `drawing.yaml`; `conform.md`'s aiwf version
  and run time removed, its spine and adapter hashes and its verifier-kit version and commit
  changed, its command's and its run's margin each changed, its command's margin made 1 with a
  `--margin 2` mentioned above the command, the fenced command moved under **Expected:** with a
  command in backticks left under **Command,**, a paragraph at the left margin put between
  **Command,** and its fenced command, both margins made 1, and its error lines and classification
  rows all removed; the classification row for ended/copy and closed/copy removed, and its gaps
  removed; the `.cmd` naming another adapter, and given a second line; and the check run with an
  empty folder as `$VK`.
- What the check does not reach, each read rather than pinned:
  - M-0364/AC-1, that an answer whose docs are wrong cites the gap listing them: Q7, Q8, Q9,
    Q17 and Q18 cite theirs; the dated, archived and accepted records Q5, Q6, Q7, Q9 and Q18
    name are not corrected, each for the reason its answer gives; Q4's are in the code, a comment
    and the `Authorize` entry of `noopExemptVerbs`, which change with the code G-0460 records;
    Q2's and Q12's describe aiwf correctly until ADR-0056 is built, and Q12 records the decision
    to list them for the build epic instead of filing a gap.
  - M-0364/AC-1, that each name that doesn't fit has an answer: the check finds each part of a
    Names section named in an answer's claims settled, not each name in the part.
  - M-0364/AC-1, that every conflict and uncovered case has an answer: the check follows the
    claim ids a Conflicts or Not covered entry cites and the `state, event` pairs a Not covered
    entry names on a line starting `- ` or `<n>. `, so an entry with neither, or a pair on any
    other line, would pass unanswered; an id, pair or part named anywhere in an answer's claims
    settled counts as settled by that answer, a pointer to another answer included; and a pair
    named in the section on cases no source covers counts as answered there, a section the check
    reads for the pair alone, not for its date, answer, reason or sources.
  - M-0364/AC-2, that each answer is written into the chapter at the rule it settles: the check
    finds the answer's citation anywhere in the chapter, and only for an answer that names a case
    the readings split or leave open, so Q3, on the states' kinds, is not checked.
  - M-0364/AC-2, that each reading's header gives the hash of the chapter its reader was given:
    the check accepts the hash of any version of `chapter.md` in the branch's history.
  - M-0364/AC-3, that the spine's header names the lifecycle: its first sentence does.
  - M-0364/AC-4, that the adapter only translates: each event runs the command its header
    names, and `observe` names the states as chapter §1 defines them.
  - M-0364/AC-4, that each divergence is classified as the change ADR-0056 makes or as a defect
    with a gap: the check matches the classification rows to the run's errors and requires a gap
    id in each row, but not which table a row sits under, that the gap exists, or what the row
    says.
  - M-0364/AC-4, that the run is recorded with the `aiwf version` it ran: the check finds a value
    in backticks for aiwf, not that it is the version the run used.
- M-0364/AC-2, beyond the check: each reading is byte-identical to its reader's reply taken again
  from the reader's transcript. The transcripts are not in the repository, so this stays an
  observation of 2026-11-22. The check reads a reading's rules, terminal marks and kinds only
  through the findings spine reports, so a rule changed in a case an answer decides, a state's
  kind changed to another kind, or reading A's terminal mark on closed removed passes it.
- M-0364/AC-4: `spec/scope/conform.md` records the run, made with `spec/README.md`'s command
  from the fresh clone.

## Deferrals

- The defects in aiwf found here are gaps discovered in M-0364, which
  `git grep -l 'discovered_in: M-0364' main -- work/gaps` lists. This milestone changes no aiwf
  code, so it fixes none of them. G-0776, G-0777, G-0779 and G-0781 are routed to patches of their
  own outside E-0100, ahead of the epic that builds ADR-0056. G-0460, G-0776 and G-0786 are
  differences `spec/scope/conform.md` classifies as defects, so an epic that makes
  `spine conform` its done-check meets each one not fixed by then.
  E-0074 addresses G-0460 and waits on whether a repeat grant is a new event, which audit Q4
  settles for a milestone; G-0786 and G-0789 are not routed.
- For M-0366: its spec is to take up squash and the host's merge methods, which audit Q15 leaves
  to it beside amend; the spec names amend and a merge, but neither squash nor the host's merge
  methods. git's own squash that folds a later commit into the opening commit loses the scope
  too, since git reads trailers only from the combined message's last paragraph. Measured
  2026-11-26 with git 2.51.1 and the `aiwf` `spec/scope/conform.md` names: authorize, act, then
  `git rebase -i HEAD~2` with the act set to squash and git's default message; the scope read
  none, and the agent's next act was refused.
- For the epic that builds ADR-0056: the docs ADR-0056 makes wrong (audit Q12) and the unit test
  that its readers still read a scope opened before its release (audit Q16).

## Reviewer notes

**What this milestone obliges later changes to do.** `spec/check.py` holds 22 rules
(`grep -cE '^\s*report\(' spec/check.py`), printed as 29 checks for `scope`. No Makefile target,
workflow or hook runs it or `spine conform` (`git grep` over those paths finds no reference), so
each obligation below binds whoever runs them by hand. The owner of each, unless named, is E-0100,
then the first build epic that makes `spine conform` its done-check (`spec/README.md`).

- `check.py`'s spine and conform rules, and what they impose: any byte change to `spine.yaml` or
  the adapter needs a new conform run at margin 2 or more; `drawing.svg` is drawn again after any
  change to the spine or its layout; verifier-kit stays at v0.3.3. They retire when that build
  epic installs verifier-kit and makes `./verify` its done-check: `./verify` runs `spine check`,
  the byte-for-byte draw and a live conform (verifier-kit's `docs/reference/tool-contract.md`),
  and a done-check expects no divergence, which `check.py`'s conform rule, needing at least one
  error line, refuses.
- `check.py`'s audit and readings rules, and the record shapes it parses, `readings/prompt.md`
  frozen by its hash among them: nothing named retires them. They bind any later edit to a
  machine's records.
- E-0100 reaches `main` by a merge that keeps its history, since each reading cites the chapter
  it was made from by a hash only that history holds. Owner: E-0100. It retires at that merge.
- The adapter's dependence on aiwf's verb flags, exit codes and the JSON fields of `aiwf list` and
  `aiwf show` (the adapter's header): nothing named retires it. Nothing on aiwf's side pins those
  fields; whether something should is the build epic's to decide.
- The Deferrals' hand-offs each retire when their target delivers them.

**Kept, each for the reason given.**

- `check.py` compares the whole adapter by hash, so a comment change needs a new conform run: the
  record must name the exact adapter that ran. The adapter's header leaves out the empty
  `gitconfig` it writes in `SCOPE_RUNS` for the same reason.
- The spine's header says "one agent's scope": changing it means a new ratification and conform
  run, and it is not false.
- Two cuts proposed for the adapter: hard-coding close to promote AC-1 and AC-2, and a compressed
  rewrite.
- The chapter cites D-0111, which is on `main` and not yet on this branch; the chapter reaches
  `main` with E-0100.
- Dated records stay as written: `history.md` reads its sources at the commits it names; the
  answers' Date lines date decisions, not the pointers an answer carries, so Q13 keeps its
  2026-11-21 bullets beside the 2026-11-26 one; the dated or archived records Q5, Q6, Q7 and Q9
  name are not corrected.
- Places where `check.py` matches more loosely or strictly than its header's words, each failing
  closed or needing a reshaped record to matter: a left-margin line inside the command's fence,
  and fences opened as ```` ```sh ```` or `~~~`; the name rule accepts any token after "by" and
  refuses a name with spaces, and its label says "models"; the spine's ratification sentence
  split over two comment lines fails; a reading's hashes are read from every comment line of its
  header, the reader's reply included, which an unedited reply cannot satisfy; `spine diff`'s
  exit status is not read, and no edit was found that makes it print nothing; the header says
  "this repository's history" where the code reads HEAD's.
- Wording left as it is: §7's "so" gives what conform sees as the reason no run checks a no-op's
  commit, where the spine declaring no outputs is the deeper one; the M-0366 bullet's "loses the
  scope too"; the check bullet in `audit/decisions.md`'s cases section cites questions 7 and 8
  for "an agent's verb commits only under an active scope", which neither says of paused
  (question 17's first item does, and the adapter shows act refused in paused); that section's
  heading does not name `closed, reallocate`, which the check bullet also settles.
- What conform does not reach: the run stops each sequence at its first divergence, so sequences
  past one are never played, such as check after act, pause and rebase, which today's aiwf
  refuses with G-0751's error. The chapter's §4 says `conform.md` classifies each divergence its
  run found.

**Doc-lint** over the change-set's markdown: no broken link, no TODO marker, no structural
issue. Every `aiwf` verb named exists, apart from `aiwf scope rebind` and `aiwf revoke`, which
the audit names as proposed and never built. Every path resolves, under `internal/` for the
package-relative ones, at the commits `history.md` names, or in verifier-kit's checkout.

**Verdict.** The deciding pass over `cc0d64cff..HEAD`, by three reviewers (the tooling; the chapter
and spine; the audit and specs), approved with no blocking defect. In it, the chapter's rules,
scenarios and invariants each agreed with the spine, and every error a margin-0 conform run
printed is one `conform.md` classifies.
