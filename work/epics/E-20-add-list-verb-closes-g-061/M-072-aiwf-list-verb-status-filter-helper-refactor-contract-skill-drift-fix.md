---
id: M-072
title: aiwf list verb, status filter-helper refactor, contract-skill drift fix
status: in_progress
parent: E-20
tdd: required
acs:
    - id: AC-1
      title: Core flag set works end-to-end
      status: met
      tdd_phase: done
    - id: AC-2
      title: 'JSON envelope: result is array of summary objects'
      status: met
      tdd_phase: done
    - id: AC-3
      title: Default excludes terminal status; --archived includes them
      status: open
      tdd_phase: green
    - id: AC-4
      title: entity.IsTerminal(kind, status) helper added
      status: met
      tdd_phase: done
    - id: AC-5
      title: Closed-set completion wired for --kind and --status
      status: open
      tdd_phase: red
    - id: AC-6
      title: Shared filter helper extracted; status uses it
      status: open
      tdd_phase: red
    - id: AC-7
      title: Status text and JSON goldens unchanged after refactor
      status: open
      tdd_phase: red
    - id: AC-8
      title: contracts-plan and contract-skill drift fixed
      status: open
      tdd_phase: red
    - id: AC-9
      title: Verb-level integration test drives the dispatcher
      status: met
      tdd_phase: done
---

# M-072 — aiwf list verb, status filter-helper refactor, contract-skill drift fix

## Goal

Ship the `aiwf list` verb as the AI's hot-path read primitive over the planning tree, with V1 flags `--kind / --status / --parent / --archived / --format / --pretty`, and refactor `aiwf status`'s kind/status filter slices into a shared helper that `list` also uses so the two verbs cannot drift.

## Context

`aiwf status` already loads the planning tree via `tree.Load` and filters in-memory (`cmd/aiwf/status_cmd.go:204–211`); this milestone reuses that loader and extracts the filter slice loops at `status_cmd.go:259–333` into a shared helper. The `--archived` flag and the "non-terminal-status entities" default are forward-compat with the proposed ADR-0004 (uniform archive convention); kind enumeration reads from `entity.AllKinds` so adding `KindFinding` later picks up automatically. G-061 names the verb-shape question; this milestone locks `aiwf list --kind <K>` (flag form, not positional plural) and applies that shape to the five `aiwf list contracts` references in `docs/pocv3/plans/contracts-plan.md` and the line in the contract skill.

## Acceptance criteria

### AC-1 — Core flag set works end-to-end

### AC-2 — JSON envelope: result is array of summary objects

### AC-3 — Default excludes terminal status; --archived includes them

### AC-4 — entity.IsTerminal(kind, status) helper added

### AC-5 — Closed-set completion wired for --kind and --status

### AC-6 — Shared filter helper extracted; status uses it

### AC-7 — Status text and JSON goldens unchanged after refactor

### AC-8 — contracts-plan and contract-skill drift fixed

### AC-9 — Verb-level integration test drives the dispatcher

## Constraints

- V1 flag set is locked: `--kind`, `--status`, `--parent`, `--archived`, `--format=text|json`, `--pretty`. No additional axes (`--actor`, `--since`, `--has-tdd`, `--ac-status`, `--has-findings`, `--format=md`) — defer until concrete friction earns them.
- Default semantic = "non-terminal-status entities", computed via `entity.IsTerminal(kind, status)`. Same predicate ADR-0004 will use to decide archive moves; designing with the ADR rather than around it means no UX break when ADR-0004 lands.
- `--archived` flag name is locked verbatim from ADR-0004 §"Display surfaces". Do not bikeshed.
- Closed-set completion for `--kind` and `--status` is wired through `cmd.RegisterFlagCompletionFunc`; the existing drift test in `cmd/aiwf/completion_drift_test.go` is satisfied without an opt-out entry.
- Refactor parity is non-negotiable: status text and JSON output are golden-tested. The shared helper lands first with parity tests against the current status output before `buildStatus` is rewritten to call it.
- Test-the-seam rule (per CLAUDE.md): a unit test of the helper alone is necessary but not sufficient. AC-9 requires a verb-level integration test that drives `run([]string{"list", ...})` and asserts the rendered output, not just the helper's return value.

## Design notes

- Verb shape: `aiwf list --kind <K>` (flag form). Decision rationale recorded inline in the epic; do not re-litigate. Positional plural (`aiwf list milestones`) is rejected to avoid per-kind pluralization rules and keep uniformity with the rest of aiwf's verb surface.
- `--parent` accepts any id whose value is referenced as `parent:` by some entity — e.g., `--parent E-13` returns milestones with `parent: E-13`; `--parent M-068` returns ACs (via the composite-id surface) when ACs become listable. V1 reach: epic → milestone, milestone → AC pending the AC-listability decision.
- JSON envelope `result` is `[]Summary` where `Summary = {id, kind, status, title, parent, path}`. No body — that's `aiwf show`. Keeps list cheap for downstream tools and AI consumption.
- No-args `aiwf list`: per-kind counts ("5 epics · 47 milestones · 12 ADRs · 14 gaps · 3 decisions · 1 contract"). Self-describing summary; not a route to "list everything."
- `entity.IsTerminal(kind, status)` is a pure closed-set switch on `entity.Kind` returning `bool`. ADR-0004 §Trigger names this helper by name; this milestone introduces it. One file edit in `internal/entity/transition.go`.
- Drift fix scope: every `aiwf list contracts` mention in `docs/pocv3/plans/contracts-plan.md` (lines 209, 425, 489, 593, 708) and `internal/skills/embedded/aiwf-contract/SKILL.md` line 33 becomes `aiwf list --kind contract`. Other contract-related verb mentions are unchanged.

## Surfaces touched

- `cmd/aiwf/list_cmd.go` (new)
- `cmd/aiwf/status_cmd.go` (refactor: extract filter slices into helper at `status_cmd.go:259–333`)
- `internal/entity/transition.go` (add `IsTerminal`)
- `internal/skills/embedded/aiwf-contract/SKILL.md` (line 33)
- `docs/pocv3/plans/contracts-plan.md` (5 line-level edits)

## Out of scope

- `aiwf-list` skill creation. M-073 owns that.
- Skills-coverage policy. M-074 owns that.
- Implementation of ADR-0003 (finding kind) or ADR-0004 (archive convention). The verb is forward-compatible; neither is a dependency.
- Any AC-listability surface (`aiwf list --kind ac` or composite-id listing). Decided at milestone start if friction earns it; out by default.

## Dependencies

- None on the aiwf side. Builds on existing `tree.Load`, the FSM definitions in `internal/entity/transition.go`, and the Cobra completion infrastructure established in E-14.

## Coverage notes

- (filled at wrap)

## References

- E-20 epic spec (this milestone's parent).
- G-061 — names the unimplemented verb and the documentation drift this milestone resolves.
- ADR-0004 (proposed) `docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md` — names `aiwf list` and the `--archived` flag verbatim; default-semantic source.
- E-14 — Cobra and completion. Established the `RegisterFlagCompletionFunc` pattern and `cmd/aiwf/completion_drift_test.go` chokepoint.
- `cmd/aiwf/status_cmd.go:259–333` — the filter slices the shared helper extracts.
- `internal/tree/tree.go:178` — `tree.Load`'s walk; consumed unchanged.

---

## Work log

(filled during implementation)

## Decisions made during implementation

- (none — all decisions are pre-locked above)

## Validation

(pasted at wrap)

## Deferrals

- (none)

## Reviewer notes

- (filled at wrap)
