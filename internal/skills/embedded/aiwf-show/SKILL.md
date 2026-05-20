---
name: aiwf-show
description: Use when the user asks "show me X" / "what does this entity look like?" / "describe Y" for any aiwf entity (or acceptance criterion). Runs `aiwf show <id>`, the canonical per-entity inspection verb. Returns frontmatter + ACs + recent history + active findings + referenced_by in one aggregate view.
---

# aiwf-show

The `aiwf show` verb is the canonical per-entity inspection surface. One call returns the full state of an epic, milestone, ADR, gap, decision, contract, or acceptance criterion: frontmatter, ACs (for milestones), the last N history events, active findings against the entity, and the back-references from other entities. There is no separate "view" or "inspect" verb — `show` is it.

## When to use

The user names a specific entity and wants its current state, not a list or a timeline. Example phrasings: "show me G-0078", "what does M-007 look like?", "what's the state of E-19?", "describe AC-3 on M-007", "what does this gap say?".

If the user asks "what happened to" or wants a timeline → reach for `aiwf-history`. If the user wants "what's in flight" across the whole tree → reach for `aiwf-status`. If the user names an id, reach here.

## What to run

```bash
aiwf show <id>                              # text view: frontmatter + acs + history + findings
aiwf show <M-id>/AC-N                       # composite id: just that AC's record
aiwf show <id> --format=json --pretty       # JSON envelope; carries body sections too
aiwf show <id> --history=0                  # suppress the history section
aiwf show <id> --history=-1                 # render the full timeline (no cap)
```

The composite-id pattern `M-NNN/AC-N` is not obvious from `--help`. Use it whenever the user names a specific AC — the JSON output for a composite id carries just that AC's slice (id, title, status, tdd_phase, body description, tests).

## Output shape

Text default carries these blocks, in order:

1. **Header**: `<id> · <title> · status: <status> · tdd: <tdd>` (for milestones with a tdd policy).
2. **Frontmatter**: parent, depends_on, references — whatever's structurally on the entity.
3. **ACs** (milestones only): one line per AC — `AC-N [status] · phase: <tdd_phase> · "<title>"`. Cancelled ACs stay position-stable; their slot remains.
4. **Recent history (N)**: one event per line in reverse chronological order. Default cap = 10; `--history=N` overrides; `--history=-1` removes the cap. Each line: `<date> <verb> <→ to> <detail>`.
5. **Findings**: active findings against this entity (or `(none)`).
6. **Referenced by**: every other entity citing this one (typically `parent:` links from milestones to their epic; `depends_on:` links; cross-references in body prose).

JSON envelope (`--format=json --pretty`) carries the same data plus a `body` map: section-heading slug → prose. Body slugs vary by kind:

| Kind | Body keys |
|---|---|
| epic | `goal`, `scope`, `out_of_scope`, plus any extra `## <Section>` headings the author added |
| milestone | `goal`, `approach`, `acceptance_criteria`, `work_log`, `decisions_made_during_implementation`, `validation`, `deferrals`, `reviewer_notes` |
| ac | the body under `### AC-N — <title>` (single string under the AC's id key) |
| gap | `whats_missing`, `why_it_matters`, plus author-added sections |
| adr | `context`, `decision`, `consequences` |
| decision | `question`, `decision`, `reasoning` |
| contract | `purpose`, `stability` |

The JSON envelope also expands per-AC payloads: each `acs[N]` entry carries the AC's body description, status, tdd_phase, and the most-recent test metrics (`{pass, fail, skip, total}`) extracted from any `aiwf-tests:` commit trailer in its history.

## Recipes

```bash
# Quick "what's this gap about?" — text view, drop history for speed
aiwf show G-0078 --history=0

# Full audit — JSON envelope, no history cap
aiwf show E-0033 --format=json --pretty --history=-1 | jq '.result.findings'

# Compare a milestone's AC list against its work_log
aiwf show M-007 --format=json --pretty | jq '.result.acs[].title, .result.body.work_log'

# Just the third AC on a milestone — composite id
aiwf show M-007/AC-3

# Pipe AC titles into a planning thread
aiwf show M-007 --format=json | jq -r '.result.acs[] | "\(.id): \(.title) [\(.status)]"'
```

## Show vs. history vs. status

The three read verbs have non-overlapping shapes — pick the right one:

- **`aiwf show <id>`** — *snapshot of one entity right now*. Includes recent history as a tail for context but the verb is state-oriented. Use when the user wants to know what an entity *is*.
- **`aiwf history <id>`** — *full event timeline of one entity*. State changes, decisions, phase walks, every trailered commit. Use when the user wants to know what *happened* to an entity (when was it created? why is it closed? who authorized?).
- **`aiwf status`** — *what's in flight across the tree* — active epics, in-progress milestones, drafted next-ups, findings rollup. Use when the user wants to know what the *project* is doing, not a specific entity.

## Don't

- Don't run `aiwf show` against a stale binary expecting new fields — the JSON envelope's body-section keys evolve as the per-kind templates evolve; check the binary's installed-version stamp (via `aiwf doctor`'s `binary:` line) if the JSON output is missing keys you expect.
- Don't reach for `aiwf show` when listing — use `aiwf list --kind <kind>` for tabular roll-ups. `show` returns one entity per call.
- Don't pass a composite id to `aiwf history` expecting AC-only events from the same shape; `aiwf history M-NNN/AC-N` is supported but its output is event-stream-shaped, not state-snapshot-shaped. The two verbs answer different questions.
