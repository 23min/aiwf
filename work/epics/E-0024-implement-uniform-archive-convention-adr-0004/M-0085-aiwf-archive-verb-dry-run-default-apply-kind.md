---
id: M-0085
title: aiwf archive verb (dry-run default, --apply, --kind)
status: in_progress
parent: E-0024
depends_on:
    - M-0086
tdd: required
acs:
    - id: AC-1
      title: aiwf archive (no flags) prints planned moves and exits without touching the tree
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf archive --apply --kind gap scopes the sweep to one kind only
      status: met
      tdd_phase: done
    - id: AC-3
      title: 'aiwf archive --apply produces one commit with aiwf-verb: archive trailer'
      status: met
      tdd_phase: done
    - id: AC-4
      title: Re-running aiwf archive --apply on a clean tree is a no-op
      status: met
      tdd_phase: done
    - id: AC-5
      title: Per-kind storage layout follows ADR-0004 storage table
      status: met
      tdd_phase: done
    - id: AC-6
      title: Verb has no positional id argument; sweep is by status not by id
      status: met
      tdd_phase: done
    - id: AC-7
      title: First --apply migration leaves aiwf check with 0 errors
      status: met
      tdd_phase: done
    - id: AC-8
      title: Restore backticked aiwf archive --apply mentions in M-0086 SKILL.md
      status: open
      tdd_phase: red
---

# M-0085 — `aiwf archive` verb (dry-run default, `--apply`, `--kind`)

## Goal

Land the `aiwf archive [--apply] [--kind <kind>] [--root <path>]` verb so terminal-status entities sweep into their archive subdirs in a single commit. Dry-run is the default; `--apply` produces exactly one commit with the `aiwf-verb: archive` trailer (and no `aiwf-entity:` — multi-entity sweep exception).

## Context

M-0084 made archive locations loadable. This milestone produces them: the verb walks each kind, finds entities whose status is terminal and whose location is the active dir (or, for directory-shaped kinds, whose parent dir is active), `git mv`s them into the archive subdir, and produces one commit. The first `--apply` invocation in this repo doubles as the historical migration (no separate migration verb per ADR-0004).

## Acceptance criteria

<!-- ACs added via `aiwf add ac M-0085 --title "..."` at start time. -->

Intended landing zone:

- `aiwf archive` (no flags) prints the planned moves and exits without touching the tree.
- `aiwf archive --apply` performs the moves and produces exactly one commit with `aiwf-verb: archive` (no `aiwf-entity:` trailer).
- `aiwf archive --apply --kind gap` scopes the sweep to one kind only.
- Re-running `aiwf archive --apply` on a clean tree is a no-op (no commit produced).
- `internal/policies/trailer_keys.go` learns the `archive` verb's no-entity exception; the trailer-keys policy test passes for the new commits.
- The verb has no positional id argument; sweep is by status, not by id (per ADR-0004's rejected alternative).

## Constraints

- Single commit per `--apply` invocation per kernel principle #7. The commit body lists affected ids and per-kind counts.
- Idempotent on clean trees.
- Uses `internal/entity/transition.go::IsTerminal` — no parallel definition of terminal statuses.
- No file-move side effect on `aiwf promote` or `aiwf cancel`. Promotion verbs stay one-purpose.
- No `--id` / positional id flag — the rejected per-id housekeeping alternative is a non-goal.
- No reverse `aiwf reactivate` / `aiwf un-archive` verb in this milestone (or any milestone — see ADR-0004 Reversal).

## Design notes

- Implements the per-kind storage table from ADR-0004 verbatim. Directory-shaped kinds (`epic`, `contract`) move whole subtrees; flat-file kinds (`gap`, `decision`, `adr`) move individual files; milestones ride with their parent epic.
- Trailer-keys policy: extend `principal_write_sites` (or whichever current allow-rule governs multi-entity sweeps) with the `archive` verb's no-entity case. Open question in epic spec — resolved here.

## Surfaces touched

- `cmd/aiwf/archive.go` (new)
- `internal/verb/archive/` (new)
- `internal/policies/trailer_keys.go`

## Out of scope

- The check-rule findings that *report* sweep state (M-0086).
- Display-surface integration (M-0087).
- Skill / config knob (M-0088).

## Dependencies

- M-0084 — loader/resolver must already span active+archive so the verb can sanity-check post-move readability.
- ADR-0004 (accepted).

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — `aiwf archive` verb section, storage table, reversal section.
- CLAUDE.md "What aiwf commits to" §7 (one verb = one commit).
- CLAUDE.md "Designing a new verb" — verb-design rule answered by ADR-0004's Reversal.
- `internal/entity/transition.go::IsTerminal`

---

## Work log

(populated during implementation)

## Decisions made during implementation

- (none)

## Validation

(populated at wrap)

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — aiwf archive (no flags) prints planned moves and exits without touching the tree

### AC-2 — aiwf archive --apply --kind gap scopes the sweep to one kind only

### AC-3 — aiwf archive --apply produces one commit with aiwf-verb: archive trailer

### AC-4 — Re-running aiwf archive --apply on a clean tree is a no-op

### AC-5 — Per-kind storage layout follows ADR-0004 storage table

### AC-6 — Verb has no positional id argument; sweep is by status not by id

### AC-7 — First --apply migration leaves aiwf check with 0 errors

### AC-8 — Restore backticked aiwf archive --apply mentions in M-0086 SKILL.md

