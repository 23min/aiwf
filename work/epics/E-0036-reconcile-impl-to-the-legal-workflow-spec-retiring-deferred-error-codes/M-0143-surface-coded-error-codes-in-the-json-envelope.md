---
id: M-0143
title: Surface Coded error codes in the JSON envelope
status: in_progress
parent: E-0036
depends_on:
    - M-0138
tdd: required
acs:
    - id: AC-1
      title: Decision D-0013 records the envelope representation and exit-code
      status: met
      tdd_phase: done
    - id: AC-2
      title: Coded verb refusal with --format=json emits status:error + error.code, exit 1
      status: open
      tdd_phase: red
    - id: AC-3
      title: Non-coded verb error emits a well-formed envelope (message, no code)
      status: open
      tdd_phase: red
    - id: AC-4
      title: Every mutating verb accepts --format=json (uniform rollout)
      status: open
      tdd_phase: red
---
## Goal

Surface the structured code carried by a `Coded` verb error in the `aiwf --format=json` envelope, so verb-time legality refusals are machine-readable on par with `findings[].code`. This fulfils E-0036's goal clause — *"errors.As-able for the JSON envelope"* — which the foundation milestone (M-0138) deliberately left to a dedicated unit.

## Context

M-0138 introduced `entity.Coded` plus the typed errors `FSMTransitionError` and `AuthorizeKindError`, each carrying a structured code extractable via `entity.Code(err)`. But the CLI `--format=json` envelope does not yet surface that code: a verb-time refusal appears as an unstructured error. The epic's goal names the envelope as the consumer; this milestone keeps that promise. The wiring is uniform — it surfaces every `Coded` error, including M-0139's cancel codes once they land.

## Decision (recorded: D-0013)

The representation + exit-code question is settled by **D-0013** (`accepted`). The key realization: every `Coded` error originates in a *mutating* verb, none of which surface `--format=json` today (the flag lives only on read verbs) and all of which route through `cliutil.FinishVerb` printing plain text + `ExitUsage`. D-0013 chooses:

- **A2 (uniform flag)** — `--format`/`--pretty` on every mutating verb via a shared `cliutil.AddFormatFlags` registrar, threaded into the single `FinishVerb`/`DecorateAndFinish` chokepoint.
- **(a) structured error object** — an additive `error: {code, message}` slot on `render.Envelope` under `status: "error"`; `code` from `entity.Code` (`errors.As`), `message` from `err.Error()`.
- **C2 (exit unification)** — a `Coded` (legality) refusal exits `ExitFindings` (1), matching the check-time exit for the same violation class; non-`Coded` verb errors stay `ExitUsage` (2), internal failures `ExitInternal` (3).

Pre-dispatch flag-usage errors stay plain-text `ExitUsage` (out of envelope scope).

## Acceptance criteria

Each AC carries an explicit **Evidence** gate — the named test or assertion that fails if the claim breaks. "Looks right" is not evidence.

### AC-1 — Decision D-0013 records the envelope representation and exit-code

D-0013 (`accepted`) records the A2 / (a) / C2 choice with the realization that drove it. *Evidence:* a `internal/policies/` structural assertion that D-0013 resolves via the loader, is `accepted`, carries `## Context` / `## Resolution` / `## Consequences` with non-empty prose, and names the representation (`status:error` + an `error` object) and the exit-code (`ExitFindings`) inside the Resolution section (scoped, not a flat grep).

### AC-2 — Coded verb refusal with --format=json emits status:error + error.code, exit 1

Running a mutating verb that returns a `Coded` error with `--format=json` emits an envelope with `status: "error"`, `error.code` = the structured code (via `entity.Code`), `error.message` = the error text, and exits `1`. *Evidence:* a binary-level test (`internal/cli/integration/`) that runs the built `aiwf` binary on an FSM-illegal `promote` with `--format=json`, JSON-parses stdout, and asserts `status` + `error.code` by **structural field access** (not substring) + the exit code via `ExitedWithCode`.

### AC-3 — Non-coded verb error emits a well-formed envelope (message, no code)

A non-`Coded` verb error (e.g. an unknown entity id) with `--format=json` still emits a well-formed envelope: `status: "error"`, `error.message` set, `error.code` empty/omitted, exit `2` — proving the change is additive and the code field is optional. *Evidence:* a binary-level test parsing the envelope on a non-coded error path, asserting the absent code and the `2` exit.

### AC-4 — Every mutating verb accepts --format=json (uniform rollout)

Every mutating verb accepts `--format=json` — the A2 uniform-rollout guarantee. *Evidence:* a `cmd/aiwf` test that walks the assembled root command tree and asserts every leaf command either registers a `--format` flag or is named in an explicit read-only/exempt allowlist (with rationale); a new mutating verb shipped without the flag fails CI unless consciously exempted.

## Constraints

- Additive to the envelope schema — don't break existing `--format=json` consumers.
- Extract the code via `entity.Code` (`errors.As`), never by parsing the message text.
- `tdd: required`.

## Out of scope

The `Coded` pattern and the typed errors themselves (M-0138); the cancel codes (M-0139) — though this milestone surfaces them once they exist.

## Dependencies

M-0138 (the `Coded` pattern + the first codes). Closes the envelope clause of E-0036's goal.

