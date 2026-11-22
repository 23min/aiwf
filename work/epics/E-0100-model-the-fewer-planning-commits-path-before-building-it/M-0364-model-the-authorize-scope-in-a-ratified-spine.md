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
      status: open
    - id: AC-3
      title: The ratified spine passes spine check and matches its drawing
      status: open
    - id: AC-4
      title: Every divergence conform reports against today's aiwf is classified
      status: open
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

## Decisions made during implementation

- The audit's answers, Q1 to Q18, each with its date, reason and the sources it makes wrong:
  `spec/scope/audit/decisions.md`.
- The answers to the readings' questions, Q1 to Q3: `spec/scope/answers.md`.
- verifier-kit runs from a pinned clone of v0.3.3 outside the repository: E-0100, §"Open
  questions"; the commands are in `spec/README.md`.
- The readings are by Claude Sonnet 5.5 and Claude Fable 5.2, named in each reading's header.
- At M-0364/AC-4: a copy that lands after a pause, an end or a close goes into G-0755 (audit Q3),
  and a refused repeat pause or resume is a defect in aiwf, G-0786 (audit Q9).

## Deferrals

- Defects in aiwf found here, left alone because this milestone changes no aiwf code: G-0776,
  G-0777, G-0778, G-0779, G-0780, G-0781, G-0782, G-0783, G-0784, G-0785 and G-0786. Of these,
  G-0776, G-0777, G-0779 and G-0781 go to the epic that builds ADR-0056.
- For the epic that builds ADR-0056: the docs ADR-0056 makes wrong (audit Q12) and the unit test
  that its readers still read a scope opened before its release (audit Q16).
