---
name: aiwf-promote
description: Use when the user wants to advance an entity (or acceptance criterion) to a new status, or step an AC's TDD phase. Runs `aiwf promote` so the transition is checked against the kind's legal moves and recorded as a single commit.
---

# aiwf-promote

The `aiwf promote` verb edits an entity's `status` field — or, for composite ids, an AC's `status` or `tdd_phase`. Allowed transitions are hardcoded per kind; illegal moves are refused before any disk change. Promoting to the status already recorded is not an illegal move — it reports "already `<status>`" at exit 0 and commits nothing.

## When to use

The user says something is "ready", "done", "in progress", "accepted", "deprecated", etc. — i.e. wants to move an entity from one status to another. Also for stepping an AC through red → green → done.

## What to run

```bash
aiwf promote <id> <new-status>           # top-level entity
aiwf promote <M-NNNN>/AC-N <new-status>  # AC status (composite id)
aiwf promote <M-NNNN>/AC-N --phase <p>   # AC tdd_phase (mutex with positional state)
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

`aiwf promote` enforces the per-kind legal-transition function. If the move is illegal it reports a finding and exits without writing; if the target status is the one already recorded there is no move to make, so it exits 0 having written nothing. To reach a terminal-cancel status use `aiwf cancel <id>` instead — same end state, clearer intent in the log.

## Evidence for promoting an AC to `met`

An AC reaches `met` on evidence, and which evidence depends on what the AC claims. "I read it and it looks right" is never it.

**A standing property** — code produces X for input Y, one artefact agrees with or cites another, the tree has a shape — is claimed to hold from now on. It can break, so its evidence is a mechanical assertion that fails when it does: a test, a check rule, or a fixture validation.

**An observation** — a command was run against something the test suite cannot reach, and reported what it reported — cannot break. The command either ran and returned what it returned, or it did not, and no assertion can watch that. Its evidence is the record the observation itself specifies: the command, the result expected, the output observed, and the environment it ran in, written into the milestone where a later reader can re-run it. Re-running is the only re-checking available, and the record is what makes it possible.

The discriminator between the two is re-derivation: can the claim be rebuilt from artefacts a test can reach? Yes — assert it. No — record it.

### Never satisfy this with a proxy

A proxy is a different, checkable claim written in place of the one the AC made. It is worse than no check at all, because it reads as evidence: a later reader sees a passing assertion and takes the AC as pinned, when what is pinned is something else.

An AC that resists pinning is often stated at the wrong level, and restating it is the right move — but restating has two outcomes, and only one of them is honest. **Narrowing** keeps the AC's claim and reduces it to the part that is checkable, so the evidence still bears on what was claimed. **Substituting** replaces the claim with a different one that happens to be checkable, and the evidence then bears on something the AC never asserted.

Before writing an assertion, ask: *does this assert what the AC claimed, or something else that happens to be checkable?* If only a substitution is available, the claim is observational — keep it and record it.

Feasibility does not settle this. A proxy can be built faithfully, match its subject exactly, and still be the wrong thing to build; the more faithful it is, the more convincingly it reads as evidence for a claim it does not test. The question is never *can I pin this?* but *am I pinning what was claimed?*

## --force --reason for exceptional moves

When a transition the FSM disallows must happen anyway (rare), pass `--force --reason "<text>"`:

```bash
aiwf promote E-NNNN done --force --reason "shipped without staging review for hotfix"
```

`--reason` is required (non-empty after trim) when `--force` is set. It becomes both the commit body and an `aiwf-force: <reason>` trailer, so the audit trail is queryable. `--force` relaxes only the FSM transition rule — coherence checks (status in closed set, refs resolve, AC body coherence) still run.

For milestones with open ACs, `--force` lets the milestone reach `done` but the standing `aiwf check` will keep surfacing `milestone-done-incomplete-acs` until the ACs reach a terminal state. The kernel reports the inconsistency every time; force only relaxes the verb-time refusal.

## Resolver-pointer flags for status-transitions that need a successor

Two transitions require a pointer to *what addressed the entity* before the kernel considers the tree clean: gap → addressed (resolver-or-commit) and adr → superseded (replacement ADR). Pass the resolver via flag at promote time so the status flip and the resolver write land in one commit:

```bash
aiwf promote G-NNNN addressed --by M-NNNN             # gap closed by milestone (single id)
aiwf promote G-NNNN addressed --by M-NNNN,E-NNNN      # gap closed by multiple entities
aiwf promote G-NNNN addressed --by-commit abcdef1234  # gap closed by a specific commit (sha goes into addressed_by_commit)
aiwf promote ADR-NNNN superseded --superseded-by ADR-NNNN
```

| Flag | Field written | Valid when |
|---|---|---|
| `--by <comma-list>` | `addressed_by` | gap → addressed |
| `--by-commit <comma-list>` | `addressed_by_commit` | gap → addressed |
| `--superseded-by <id>` | `superseded_by` | adr → superseded |

A flag/kind/status mismatch is a usage error (Go-error before any disk work), not a finding. The flags are mutex with `--audit-only` (audit-only is empty-diff by definition; resolver flags imply a mutation) and not valid in phase mode (resolver fields apply to entity status, not AC tdd_phase).

Use the verb route, not hand-editing: the gap-addressed-has-resolver and adr-supersession-mutual checks fire whenever the field is missing, and the verb route writes the field atomically with the status change so the standing check goes silent immediately.

### Close against what the resolver actually satisfies

Before closing, re-read the record's own body for a claim the resolver does not
satisfy. The terminal-status notice looks outward, at live records that name this
one; nothing looks inward at the record being closed.

A record carrying a defect and the open question that defect raised is closed
honestly only when the resolver answers both. Where it answers one, split the
remainder into a record of its own first, then close against what landed. Closing on
partial evidence buries the rest: the status reads terminal, and the unanswered half
sits inside a record nobody reopens.

## --audit-only --reason for backfilling state already reached

When state was already reached via a manual `git commit` (no aiwf trailers), `aiwf promote <id> <state> --audit-only --reason "..."` records an empty-diff commit with the trailer block so `aiwf history` reflects the move. The verb refuses unless the entity is **already** at the named state — audit-only records what's true, not transitions. Mutex with `--force`. Human-only (the kernel refuses non-human actors). See `aiwf-authorize` and the G24 recovery story.

## Provenance flags

| Flag | When |
|---|---|
| `--actor <role>/<id>` | Override the runtime-derived identity (default: `human/<localpart-of-git-config-user.email>`). |
| `--principal human/<id>` | **Required** when `--actor` is non-human; **forbidden** when `--actor` is `human/...`. |

For agents acting under an active authorization scope, the kernel matches the scope automatically (no `--scope` flag) and stamps `aiwf-on-behalf-of:` + `aiwf-authorized-by:` on the commit. Open the scope first with `aiwf authorize`. Without an active scope, agent promotions refuse with `provenance-no-active-scope`.

When the scope-entity reaches a **terminal status** via `aiwf promote` or `aiwf cancel` (e.g., `aiwf promote E-NNNN done`), every non-ended scope on that entity auto-ends — the commit carries one `aiwf-scope-ends: <auth-sha>` per ended scope. A paused scope is included: nothing will act on a terminal entity, so nothing could resume it.

## What aiwf does

1. Loads the entity (or AC, for composite ids) and validates the transition.
2. Rewrites only the changed line in frontmatter — for ACs, the entry inside `acs[]`. Everything else preserved.
3. Commits with trailers `aiwf-verb: promote`, `aiwf-entity: <id>` (composite for ACs), `aiwf-to: <new-state>` (status or phase), `aiwf-actor: <actor>`. `aiwf-force: <reason>` is added when `--force` is set, `aiwf-audit-only: <reason>` when `--audit-only` is set, plus the I2.5 provenance trailers (`aiwf-principal`, `aiwf-on-behalf-of`, `aiwf-authorized-by`, `aiwf-scope-ends`) where applicable.
4. When a bare id's new status is terminal, prints the still-open records whose bodies name the entity — the id, file, and line of each first mention. A composite id never triggers it. `aiwf cancel` prints the same. Nothing prints when no live record names it, or under `--format=json`.

## Don't

- Don't hand-edit `status:` in markdown — the trailer chain disappears and `aiwf history` won't surface the move.
- Don't try to skip statuses (e.g., `proposed` → `done` for an epic) without `--force --reason`. The legal-transition function refuses it by default; that's intentional.
- Don't combine `--phase` with a positional new-status. Phase changes and status changes are separate transitions; the dispatcher refuses both at once.
- Don't use `--phase` on a top-level (bare) id. Phases are AC-only.
- Don't combine `--audit-only` with `--force` — the two are mutually exclusive (audit-only records reality; force makes a transition happen).
