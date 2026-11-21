---
id: M-0366
title: Model verb commit output in a ratified spine
status: draft
parent: E-0100
depends_on:
    - M-0364
    - M-0365
tdd: none
acs:
    - id: AC-1
      title: Every question the audit raises has a recorded answer
      status: open
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

A ratified spine for what a verb commits under ADR-0058, an amended commit or a new one, through
a push, a push whose reply is lost and the `pull --rebase` after it, a branch cut and a merge,
with what `aiwf check` reads after each; and a record of where today's `aiwf` departs from it.

## Closes

- (none)

## Context

ADR-0058's Decision states when a verb amends and what the amended commit records. The
transactions model (`docs/research/transactions/model/RESULTS-amend.md`) explored amend designs,
and the amend prototype's scenarios (`docs/research/fewer-planning-commits/amend-prototype/`)
played the git cases against a patched build. Today's `aiwf` never amends, so the spine and the
binary differ wherever the spine amends. E-0100 records the risk that amend does not fit in one
spine.

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
divergence is classified as a change ADR-0056, ADR-0057 or ADR-0058 makes, or as a defect with a
gap.
**Edge cases**: a divergence the adapter causes, which is fixed in the adapter and the run
repeated; a divergence an open gap already describes cites that gap instead of filing another.
**Code references**: `conformance/`; the amend prototype's scenarios, for the git operations.

## Constraints

- An amended commit carries the scope ids and operation identities the spines of
  M-0364 and M-0365 decide; this spine does not decide them again.

## Out of scope

- Any change to `aiwf`.

## Dependencies

- M-0364 and M-0365.

## References

- ADR-0058, ADR-0057 and ADR-0056.
- ADR-0022 and ADR-0036, which ADR-0058 builds on.
