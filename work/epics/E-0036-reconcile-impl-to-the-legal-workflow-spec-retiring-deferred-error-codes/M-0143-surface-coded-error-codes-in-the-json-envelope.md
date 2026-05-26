---
id: M-0143
title: Surface Coded error codes in the JSON envelope
status: in_progress
parent: E-0036
depends_on:
    - M-0138
tdd: required
---
## Goal

Surface the structured code carried by a `Coded` verb error in the `aiwf --format=json` envelope, so verb-time legality refusals are machine-readable on par with `findings[].code`. This fulfils E-0036's goal clause — *"errors.As-able for the JSON envelope"* — which the foundation milestone (M-0138) deliberately left to a dedicated unit.

## Context

M-0138 introduced `entity.Coded` plus the typed errors `FSMTransitionError` and `AuthorizeKindError`, each carrying a structured code extractable via `entity.Code(err)`. But the CLI `--format=json` envelope does not yet surface that code: a verb-time refusal appears as an unstructured error. The epic's goal names the envelope as the consumer; this milestone keeps that promise. The wiring is uniform — it surfaces every `Coded` error, including M-0139's cancel codes once they land.

## Decision required (before implementation)

How a coded verb-refusal is represented in the envelope — and its exit code — is a genuine design decision, not just a field. The envelope `status` enum is `ok | findings | error`; exit codes are `0 ok / 1 findings / 2 usage / 3 internal`. A legality refusal is none of those cleanly. Candidate shapes:

- **(a)** a structured `error: {code, message}` object under `status: error`;
- **(b)** promote the refusal into `findings[]` (which already carries `code`), with a findings-like exit;
- **(c)** a top-level `code` field on the error envelope.

Author a **D-NNNN** recording the chosen representation and exit-code treatment before writing the wiring (mirrors M-0142's pre-decision pattern).

## Acceptance criteria (candidate — refined at start)

- **AC-1** — A D-NNNN records the envelope representation + exit-code decision for coded verb refusals. *Evidence:* structural assertion the decision entity exists with its named sections.
- **AC-2** — Running a verb that returns a `Coded` error with `--format=json` emits an envelope carrying the structured code. *Evidence:* binary-level test that parses the JSON envelope and asserts the code in the decided location (structural parse, not a substring match).
- **AC-3** — A non-`Coded` verb error still produces a well-formed envelope (no/empty code), so the change is additive. *Evidence:* binary-level test of a non-coded error path.

## Constraints

- Additive to the envelope schema — don't break existing `--format=json` consumers.
- Extract the code via `entity.Code` (`errors.As`), never by parsing the message text.
- `tdd: required`.

## Out of scope

The `Coded` pattern and the typed errors themselves (M-0138); the cancel codes (M-0139) — though this milestone surfaces them once they exist.

## Dependencies

M-0138 (the `Coded` pattern + the first codes). Closes the envelope clause of E-0036's goal.
