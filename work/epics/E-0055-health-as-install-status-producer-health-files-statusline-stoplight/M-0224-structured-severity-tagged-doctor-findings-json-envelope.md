---
id: M-0224
title: Structured, severity-tagged doctor findings + JSON envelope
status: draft
parent: E-0055
tdd: required
acs:
    - id: AC-1
      title: doctor emits typed severity-tagged findings the text report renders from
      status: open
      tdd_phase: red
    - id: AC-2
      title: aiwf doctor --format=json emits the standard findings envelope
      status: open
      tdd_phase: red
---
## Deliverable

Refactor `aiwf doctor` so that a typed, severity-tagged findings slice is the
single source of truth, and the existing human-readable report is *derived* from
it. Add a `--format=json` envelope. Closes G-0070.

Doctor already accumulates a two-level signal (blocking checks increment the
problem count; advisory and informational lines never do); this milestone lifts
that into an explicit per-finding severity (`info` / `warn` / `error`) that both
the text report and the JSON envelope render from.

## Acceptance criteria (formalized at milestone start)

- **Typed findings model.** `aiwf doctor` produces findings carrying
  `{ code, severity, message, data }` with `severity` one of `info` / `warn` /
  `error`; the human text report is rendered from that slice, not built
  independently. Evidence: a per-section severity-mapping test (blocking → error,
  advisory → warn or info) plus a golden test that the derived prose equals the
  current report byte-for-byte (no regression).
- **JSON envelope.** `aiwf doctor --format=json` emits the standard
  `{ tool, version, status, findings, result, metadata }` envelope with `status`
  one of `ok` / `findings` / `error`. Evidence: an integration test that drives
  the verb, parses the envelope, and asserts a known finding (e.g. a missing
  `aiwf.yaml` on a non-initialized repo) appears at `severity: error`; plus the
  `--format` completion-drift test.

### AC-1 — doctor emits typed severity-tagged findings the text report renders from

### AC-2 — aiwf doctor --format=json emits the standard findings envelope

