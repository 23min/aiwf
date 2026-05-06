---
name: aiwf-promote
description: Use when the user wants to advance an entity (or acceptance criterion) to a new status, or step an AC's TDD phase. Runs `aiwf promote` so the transition is checked against the kind's legal moves and recorded as a single commit.
---

# aiwf-promote

The `aiwf promote` verb edits an entity's `status` field — or, for composite ids, an AC's `status` or `tdd_phase`. Allowed transitions are hardcoded per kind; illegal moves are refused before any disk change.

## When to use

The user says something is "ready", "done", "in progress", "accepted", "deprecated", etc. — i.e. wants to move an entity from one status to another. Also for stepping an AC through red → green → done.

## What to run

```bash
aiwf promote <id> <new-status>                  # top-level entity
aiwf promote <M-id>/AC-N <new-status>           # AC status (composite id)
aiwf promote <M-id>/AC-N --phase <p>            # AC tdd_phase (mutex with positional state)
```

## Allowed status sets

| Kind | Statuses |
|---|---|
| epic | `proposed`, `active`, `done`, `cancelled` |
| milestone | `draft`, `in_progress`, `done`, `cancelled` |
| adr | `proposed`, `accepted`, `superseded`, `rejected` |
| gap | `open`, `addressed`, `wontfix` |
| decision | `proposed`, `accepted`, `superseded`, `rejected` |
| contract | `proposed`, `accepted`, `deprecated`, `retired`, `rejected` |
| AC status | `open`, `met`, `deferred`, `cancelled` |
| AC `tdd_phase` | `red`, `green`, `refactor`, `done` (linear; refactor optional) |

`aiwf promote` enforces the per-kind legal-transition function. If the move is illegal it reports a finding and exits without writing. To reach a terminal-cancel status use `aiwf cancel <id>` instead — same end state, clearer intent in the log.

## --force --reason for exceptional moves

When a transition the FSM disallows must happen anyway (rare), pass `--force --reason "<text>"`:

```bash
aiwf promote E-01 done --force --reason "shipped without staging review for hotfix"
```

`--reason` is required (non-empty after trim) when `--force` is set. It becomes both the commit body and an `aiwf-force: <reason>` trailer, so the audit trail is queryable. `--force` relaxes only the FSM transition rule — coherence checks (status in closed set, refs resolve, AC body coherence) still run.

For milestones with open ACs, `--force` lets the milestone reach `done` but the standing `aiwf check` will keep surfacing `milestone-done-incomplete-acs` until the ACs reach a terminal state. The kernel reports the inconsistency every time; force only relaxes the verb-time refusal.

## Resolver-pointer flags for status-transitions that need a successor

Two transitions require a pointer to *what addressed the entity* before the kernel considers the tree clean: gap → addressed (resolver-or-commit) and adr → superseded (replacement ADR). Pass the resolver via flag at promote time so the status flip and the resolver write land in one commit:

```bash
aiwf promote G-NNN addressed --by M-007                # gap closed by milestone (single id)
aiwf promote G-NNN addressed --by M-007,E-03           # gap closed by multiple entities
aiwf promote G-NNN addressed --by-commit abcdef1234    # gap closed by a specific commit (sha goes into addressed_by_commit)
aiwf promote ADR-NNNN superseded --superseded-by ADR-NNNN
```

| Flag | Field written | Valid when |
|---|---|---|
| `--by <comma-list>` | `addressed_by` | gap → addressed |
| `--by-commit <comma-list>` | `addressed_by_commit` | gap → addressed |
| `--superseded-by <id>` | `superseded_by` | adr → superseded |

A flag/kind/status mismatch is a usage error (Go-error before any disk work), not a finding. The flags are mutex with `--audit-only` (audit-only is empty-diff by definition; resolver flags imply a mutation) and not valid in phase mode (resolver fields apply to entity status, not AC tdd_phase).

Use the verb route, not hand-editing: the gap-resolved-has-resolver and adr-supersession-mutual checks fire whenever the field is missing, and the verb route writes the field atomically with the status change so the standing check goes silent immediately.

## --audit-only --reason for backfilling state already reached

When state was already reached via a manual `git commit` (no aiwf trailers), `aiwf promote <id> <state> --audit-only --reason "..."` records an empty-diff commit with the trailer block so `aiwf history` reflects the move. The verb refuses unless the entity is **already** at the named state — audit-only records what's true, not transitions. Mutex with `--force`. Human-only (the kernel refuses non-human actors). See `aiwf-authorize` and the G24 recovery story.

## Provenance flags

| Flag | When |
|---|---|
| `--actor <role>/<id>` | Override the runtime-derived identity (default: `human/<localpart-of-git-config-user.email>`). |
| `--principal human/<id>` | **Required** when `--actor` is non-human; **forbidden** when `--actor` is `human/...`. |

For agents acting under an active authorization scope, the kernel matches the scope automatically (no `--scope` flag) and stamps `aiwf-on-behalf-of:` + `aiwf-authorized-by:` on the commit. Open the scope first with `aiwf authorize`. Without an active scope, agent promotions refuse with `provenance-no-active-scope`.

When the scope-entity reaches a **terminal status** via `aiwf promote` (e.g., `aiwf promote E-03 done`), every active scope on that entity auto-ends — the commit carries one `aiwf-scope-ends: <auth-sha>` per ended scope.

## What aiwf does

1. Loads the entity (or AC, for composite ids) and validates the transition.
2. Rewrites only the changed line in frontmatter — for ACs, the entry inside `acs[]`. Everything else preserved.
3. Commits with trailers `aiwf-verb: promote`, `aiwf-entity: <id>` (composite for ACs), `aiwf-to: <new-state>` (status or phase), `aiwf-actor: <actor>`. `aiwf-force: <reason>` is added when `--force` is set, `aiwf-audit-only: <reason>` when `--audit-only` is set, plus the I2.5 provenance trailers (`aiwf-principal`, `aiwf-on-behalf-of`, `aiwf-authorized-by`, `aiwf-scope-ends`) where applicable.

## Don't

- Don't hand-edit `status:` in markdown — the trailer chain disappears and `aiwf history` won't surface the move.
- Don't try to skip statuses (e.g., `proposed` → `done` for an epic) without `--force --reason`. The legal-transition function refuses it by default; that's intentional.
- Don't combine `--phase` with a positional new-status. Phase changes and status changes are separate transitions; the dispatcher refuses both at once.
- Don't use `--phase` on a top-level (bare) id. Phases are AC-only.
- Don't combine `--audit-only` with `--force` — the two are mutually exclusive (audit-only records reality; force makes a transition happen).
