---
id: M-0142
title: Rename gap-resolved-has-resolver to match the gap FSM vocabulary
status: in_progress
parent: E-0036
tdd: required
acs:
    - id: AC-1
      title: Decision D-0012 records the rename and downstream-consumer caveat
      status: open
      tdd_phase: red
    - id: AC-2
      title: Finding fires under the new code; old literal absent from impl/spec/hint
      status: open
      tdd_phase: red
    - id: AC-3
      title: Hint table carries an entry for the new code name
      status: open
      tdd_phase: red
---
## Goal

Author a small D-NNNN recording the rename decision (with the downstream-JSON-consumer caveat), then atomically rename `gap-resolved-has-resolver` → `gap-addressed-has-resolver` (final name set by the D-NNNN) across `internal/check/check.go`, `internal/check/hint.go`, `internal/workflows/spec/rules.go`, and any string-matching fixtures — in one commit.

## Context

The code was named when the gap FSM used `resolved` as the addressed terminal; the current FSM uses `addressed` and `wontfix`. A reader of the code or of `aiwf check` output has to mentally translate. The rename is mechanical but spans impl, spec, hints, and fixtures, and could break downstream tools that ingest the old code from `aiwf check --format=json` — hence a recorded pre-decision rather than a silent rename. (Surfaced concretely this session: the rule fired during gap-closure as `gap-resolved-has-resolver`.)

## Acceptance criteria

- **AC1** — A D-NNNN records the rename decision and the downstream-consumer caveat, status `accepted`. *Evidence:* structural assertion the decision entity exists with its named sections (scoped to the section).
- **AC2** — The finding fires under the new code name when a gap promotes to `addressed` without a resolver, and the old literal no longer appears in non-archive impl/spec/hint source. *Evidence:* check-rule test asserting the new code on the violation; a scoped structural assertion that the old literal is absent from the named source files.
- **AC3** — The hint table carries an entry for the new code. *Evidence:* the existing `PolicyFindingCodesHaveHints` policy test stays green post-rename.

## Constraints

- Atomic — one commit across all surfaces (impl, spec, hint, fixtures), so no intermediate state has a dangling code.
- Pre-decision (D-NNNN) lands first.
- `tdd: required`.

## Out of scope

Other finding codes; the classifier (M3) — though if M3 has landed, this rename updates the classified set in the same pass.

## Dependencies

None (independent). Best executed after M3 so the classified legality set is renamed in one pass (soft). Closes G-0144.

### AC-1 — Decision D-0012 records the rename and downstream-consumer caveat

### AC-2 — Finding fires under the new code; old literal absent from impl/spec/hint

### AC-3 — Hint table carries an entry for the new code name

