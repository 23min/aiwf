---
id: M-0142
title: Rename gap-resolved-has-resolver to match the gap FSM vocabulary
status: in_progress
parent: E-0036
tdd: required
acs:
    - id: AC-1
      title: Decision D-0012 records the rename and downstream-consumer caveat
      status: met
      tdd_phase: done
    - id: AC-2
      title: Finding fires under the new code; old literal absent from impl/spec/hint
      status: open
      tdd_phase: green
    - id: AC-3
      title: Hint table carries an entry for the new code name
      status: open
      tdd_phase: red
---
## Goal

Author a small decision (D-0012) recording the rename and its downstream-JSON-consumer caveat, then atomically rename `gap-resolved-has-resolver` → `gap-addressed-has-resolver` across `internal/check/check.go`, `internal/check/hint.go`, `internal/workflows/spec/rules.go`, and every string-matching test / fixture / golden under `internal/` — in one commit.

## Context

The code was named when the gap FSM used `resolved` as the addressed terminal; the current FSM uses `addressed` and `wontfix`. A reader of the code or of `aiwf check` output has to mentally translate. The rename is mechanical but spans impl, spec, hints, and fixtures, and could break downstream tools that ingest the old code from `aiwf check --format=json` — hence a recorded pre-decision rather than a silent rename. (Surfaced concretely this session: the rule fired during gap-closure as `gap-resolved-has-resolver`.)

## Acceptance criteria

Each AC carries an explicit **Evidence** gate — the named test or assertion that fails if the claim breaks. "Looks right" is not evidence.

### AC-1 — Decision D-0012 records the rename and downstream-consumer caveat

D-0012 records the rename `gap-resolved-has-resolver` → `gap-addressed-has-resolver`, its rationale (FSM-vocabulary coherence), and the downstream-consumer caveat — the code string is the stable key in the `aiwf check --format=json` `findings[].code` surface, so the rename is a breaking change for any tool that pins the old literal. Status `accepted`. *Evidence:* a `internal/policies/` structural assertion that D-0012 resolves via the loader, is `accepted`, carries its named sections (`## Context`, `## Resolution`, `## Consequences`) with non-empty prose, and names both code strings plus the JSON-surface caveat in the relevant section (scoped to the section, not a flat grep).

### AC-2 — Finding fires under the new code; old literal absent from impl/spec/hint

The `gapResolvedHasResolver` rule emits `Code: "gap-addressed-has-resolver"` when a gap is `addressed` with both `addressed_by` and `addressed_by_commit` empty, and the old literal `gap-resolved-has-resolver` appears nowhere in non-archive `internal/` source (impl, spec, hint, tests, fixtures, goldens). *Evidence:* a check-rule test in `internal/check/` driving a gap-addressed-no-resolver fixture through `check.Run` and asserting the exact new code on the finding; plus a `internal/policies/` absence chokepoint walking non-archive `internal/` and asserting zero occurrences of the old literal (its needle assembled from fragments so the asserting file itself is not a match — the policy fires if any source reintroduces the old name).

### AC-3 — Hint table carries an entry for the new code name

`hint.go`'s `hintTable` carries a `gap-addressed-has-resolver` entry; the rule emission and the hint key are renamed together so every emitted code still resolves to a hint. *Evidence:* the existing `PolicyFindingCodesHaveHints` policy stays green post-rename (it fails if an emitted `Code:` literal has no hint key); load-bearingness shown by a throwaway mutation — renaming only the emission, not the hint key, drives the policy red — then reverted.

## Constraints

- Atomic — one commit across all surfaces (impl, spec, hint, fixtures), so no intermediate state has a dangling code.
- Pre-decision (D-NNNN) lands first.
- `tdd: required`.

## Out of scope

Other finding codes; the classifier (M3) — though if M3 has landed, this rename updates the classified set in the same pass.

## Dependencies

None (independent). Best executed after M3 so the classified legality set is renamed in one pass (soft). Closes G-0144.

