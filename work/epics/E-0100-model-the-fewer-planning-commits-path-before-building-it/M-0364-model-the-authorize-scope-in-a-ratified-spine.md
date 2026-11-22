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
repository gains `spec/scope/`, verifier-kit's model of one agent's authorize scope on a
milestone as ADR-0056 has it, and `conformance/adapter_scope.py`, which compares that model
with an `aiwf` binary; `spec/README.md` says how to run them.

## Decisions made during implementation

- The audit's answers, Q1 to Q18, each with its date, reason and the sources it makes wrong:
  `spec/scope/audit/decisions.md`.
- The answers to the readings' questions, Q1 to Q3: `spec/scope/answers.md`.
- verifier-kit runs from a pinned clone of v0.3.3 outside the repository: E-0100, §"Open
  questions"; the commands are in `spec/README.md`.
- The readings are by Claude Sonnet 5.5 and Claude Fable 5.2, named in each reading's header.
- At M-0364/AC-4: a copy that lands after a pause, an end or a close goes into G-0755 (audit Q3),
  and a refused repeat pause or resume is a defect in aiwf, G-0786 (audit Q9).

## Validation

Run 2026-11-22 on this milestone's branch in the Linux devcontainer: Go 1.25.12, git 2.51.1,
Python 3.12.3, verifier-kit v0.3.3 at `0952e0f` built from a fresh clone, and the `aiwf` that
`spec/scope/conform.md` names.

- `aiwf check`: exit 0, 0 errors, 1 warning, `provenance-untrailered-scope-undefined` (the
  branch has no upstream).
- Go tests, build and lint: not run. No Go, module, `Makefile`, lint, workflow, embedded or
  guidance file changes (`git diff --name-only cc0d64cff..HEAD` over those paths is empty), so
  CLAUDE.md's cadence skips them. `python3 -m py_compile conformance/adapter_scope.py`: exit 0.
- M-0364/AC-1, by a cross-check of `spec/scope/audit/` kept outside the repository. Expected:
  every answer in `decisions.md` gives its date, answer, reason and the sources it makes wrong;
  every claim id that a conflict or not-covered section of `docs.md` or `history.md` cites is in
  some answer's "Claims settled"; each names section is settled by section; each not-covered
  pair has an answer. Observed: exit 0, 18 answers, 118 ids all settled, `docs.md` §4 and
  `history.md` §5 settled by Q15 and Q16, 13 pairs answered. With H18 removed from Q17, or Q18's
  sources field removed, the same check exits 1.
- M-0364/AC-2: the readings' headers name `claude-sonnet-5-5` and `claude-fable-5-2` and the
  `git hash-object` of the chapter at `b2abebae0`, of `readings/prompt.md` and of v0.3.3's spine
  format; each reading is byte-identical to its reader's reply taken again from the transcript.
  `spine diff` of the two readings and `spine check` of each find closed/authorize, act, pause,
  resume, end, close, rebase, reallocate and check (answered by `answers.md` Q1) and none/copy
  and closed/copy (Q2), each cited in the chapter.
- M-0364/AC-3: `spine check spec/scope/spine.yaml` exits 0 with no findings; `spine diff` of
  the spine against each reading reports only the cases Q1 and Q2 decide; `spine draw` prints no
  warning and its output is byte-identical to `drawing.svg`; the header names `readings/` and
  `answers.md` and says "Ratified by the reviewed merge of this file."
- M-0364/AC-4: `spec/scope/conform.md` records the run at margin 2, every divergence in its
  output classified with a gap; the run's adapter is the committed one by `git hash-object`.
  `spec/README.md`'s command, from the fresh clone, gave the same 16 divergences and coverage in
  1049 seconds. A classification row removed, a reading's rule changed, or a state moved in
  `drawing.yaml` each makes the AC-2 to AC-4 check exit 1.

## Deferrals

- Defects in aiwf found here, left alone because this milestone changes no aiwf code: G-0776,
  G-0777, G-0778, G-0779, G-0780, G-0781, G-0782, G-0783, G-0784, G-0785 and G-0786. Of these,
  G-0776, G-0777, G-0779 and G-0781 go to the epic that builds ADR-0056.
- For the epic that builds ADR-0056: the docs ADR-0056 makes wrong (audit Q12) and the unit test
  that its readers still read a scope opened before its release (audit Q16).
