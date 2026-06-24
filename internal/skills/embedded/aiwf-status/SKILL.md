---
name: aiwf-status
description: Use for narrative-shaped state questions — "what's next?", "where are we?", "what are we working on?", "current status?", "what's in flight?", "give me a summary". Runs `aiwf status`, which prints a one-screen snapshot of in-flight epics + their milestones, open decisions (proposed ADRs and D-NNN), open gaps, the last 5 events from git history, and tree-health counts. Curated for human readers, not for programmatic filter queries — for those, use `aiwf list`. Read-only; no commit.
---

# aiwf-status

A one-screen project snapshot for human readers. Reach for this whenever the user asks a vague narrative-state question — *"what's next?"*, *"where are we?"*, *"what are we working on?"*, *"status?"*, *"what's in flight?"*, *"give me a summary"*. Don't compose multiple `aiwf check` / `aiwf history` calls and read raw frontmatter when one verb answers the question.

**For programmatic tree queries — every milestone with status X, all entities by parent Y, every open gap, filter by kind — prefer `aiwf list`. That is the hot path for filter-shaped prompts; this skill covers narrative snapshots only.** The two verbs split the read surface deliberately: list answers structured queries, status answers human-state questions.

## What it does

`aiwf status` walks the planning tree and renders five sections:

1. **In flight** — every epic with status `active`, plus every milestone underneath it. The currently-running milestone is marked `→`; done milestones get `✓`. Each milestone row that carries acceptance criteria shows a per-status badge (`ACs M/T met (N open)`) and the milestone's TDD policy when set (`tdd: required`/`advisory`/`none`).
2. **Open decisions** — ADRs (`ADR-NNNN`) and D-NNN entries with status `proposed`. The decisions that haven't been ratified, rejected, or superseded yet.
3. **Open gaps** — gaps with status `open`, with the milestone or epic each was discovered in.
4. **Recent activity** — the last 5 commits whose messages carry an `aiwf-verb:` trailer. Date, actor, verb, subject. Cross-entity, no filter.
5. **Health** — total entity count, error count, warning count from `aiwf check`. If non-zero, points at `aiwf check` for details.

By design, the verb shows *only in-flight state* at the top level — closed epics, accepted ADRs, addressed gaps, and cancelled work are not surfaced. The view is forward-looking. For history of a specific entity, use `aiwf history <id>`.

## Scoping to one workstream (`--area`)

`aiwf status --area <A>` scopes the snapshot to a single workstream (E-0043): the **entity-derived** sections — in-flight epics (and their milestones), planned epics, open decisions, open gaps — keep only entities whose effective area equals `<A>` (root kinds by their own field, epics carrying their milestones along). Recent activity, warnings, and health stay **global** — they are cross-cutting tree-health signals, not per-area concepts. An undeclared `--area` value prints a one-line note to stderr and scopes everything out (reads never reject). Reach for it when the user asks *"what's in flight in the platform workstream?"*.

**Filter vs. group.** `--area` *narrows* to one workstream. Separately, when `aiwf.yaml` declares an `areas` block, plain `aiwf status` (and `--format=md`) automatically *partitions* the In-flight and Roadmap epic sections into a subsection per declared area, plus an always-shown untagged complement labelled by `areas.default` (or a `Uncategorized` fallback); an unused declared area is omitted. With no `areas` block, output is exactly as before. Grouping and `--area` are alternatives — `--area` suppresses grouping (the view is already one workstream). The same partition drives `aiwf render roadmap` and `render --format=html`.

## When to use

When the user asks about state but doesn't name a specific entity:

| User says | Run |
|---|---|
| "what's next?" | `aiwf status` |
| "where are we?" | `aiwf status` |
| "what are we working on?" | `aiwf status` |
| "current status?" | `aiwf status` |
| "what's in flight?" | `aiwf status` |
| "show me the status" | `aiwf status` |

When the user names a specific entity, prefer `aiwf history <id>` — that gives the timeline for that one thing.

## Output

Default text:

```
aiwf status — 2026-04-28

In flight
  E-01 — Migrate notes from git to R2    [active]
     ✓ M-001 — prep and schema      [done]
      → M-002 — auth wiring          [in_progress]    · ACs 2/3 met (1 open)    · tdd: required
        M-003 — content migration    [draft]
        M-004 — cutover              [draft]

Open decisions
  ADR-0001 — Adopt OpenAPI 3.1    [proposed]

Open gaps
  G-001 — needs error handling    (discovered in M-001)

Recent activity
  2026-04-28  human/peter        promote     aiwf promote M-002 in_progress
  2026-04-27  human/peter        add         aiwf add milestone M-004 "cutover"

Health
  9 entities · 0 errors · 1 warnings · run `aiwf check` for details
```

For scripting: `aiwf status --format=json --pretty`. JSON envelope with the same data, structured. The envelope's `worktrees` array is always populated when ≥1 worktree exists (omitted from default JSON via `omitempty` when zero); structured consumers see the same data the human-narrative output surfaces, no flag needed.

## Worktrees in the default output (G-0122)

When ≥2 git worktrees exist for the repo, `aiwf status` inserts a one-line-per-worktree `Worktrees` section directly under `In flight`. Each row names the entity each worktree is driving (epic / milestone / gap) with its status, the relative age of the last commit, and a `dirty` flag when the working tree has uncommitted changes. Single-worktree projects see no section (no value to add).

The trunk worktree (`main`) gets its own row at the end with a compact `path  •  main  •  trunk (no in-flight scope)` form so the operator can see at a glance that no entity is being driven on trunk.

The short view points at `aiwf status --worktrees` for the full breakdown.

## Worktree-organized view (`--worktrees`)

`aiwf status --worktrees` swaps the text output for a worktree-organized layout — per-worktree sections with full entity expansion. Reach for it when the user asks *"where's my work?"*, *"which worktree is on what?"*, *"what's in flight where?"*, or after the user just used `git worktree list` and wants to join that against entity state.

The output uses a per-worktree section shape:

- **Header**: `Worktree: <path>` (bold)
- **Branch + age + dirty**: `⎇ <branch>  •  last commit <age>  •  dirty` (dimmed; `dirty` highlighted when applicable)
- **Optional metadata line**: `created <age>  •  last entity touch <age>` (only when those differ from `last commit`)
- **Driver row** depends on entity kind:
  - **Epic-driver worktrees** expand the full epic: every milestone (including completed) and every gap the epic closes or whose milestones surfaced.
  - **Milestone-driver worktrees** show the parent epic as a breadcrumb header, then the driven milestone with `→ (driven)` marker, then the milestone's `depends on:`, `ACs:`, and `Surfaced gaps:` lists.
  - **Gap-driver worktrees** (typical for wf-patch worktrees on `patch/g-NNNN-...` branches) show just the gap row.
- **Stale worktrees** (driver entity is terminal): same shape with an inline `STALE — driver is terminal; cleanup: git worktree remove <path>` marker.
- **Trunk worktrees** (no driver correlation, typically `main`): one-line `No in-flight scope (trunk)` — or, when in-flight entities exist with no worktree driving them, an `Other in-flight:` sub-section listing each (branch + age when a non-checked-out branch exists for it, else `(no branch, on trunk)`).

The worktree → entity correlation uses a hybrid cascade: first the worktree's `git log main..<branch>` is walked for scope-defining `aiwf-verb:` events (`authorize`, `promote → in_progress/active`, `promote --phase`); among multiple candidates the most-recent active-state event wins. If no scope events, the most recent `aiwf-entity:` trailer wins. If no aiwf commits at all, the branch name is parsed against the conventional shapes (`epic/E-NNNN-...`, `milestone/M-NNNN-...`, `patch/g-NNNN-...`).

Worktrees are sorted: in-flight first, then trunk, then stale at the end. Status glyphs and badges are colored (green met/done, yellow in_progress, cyan open/draft/proposed, red cancelled), so the eye lands on activity first.

The `--worktrees` flag affects only the text output; the JSON envelope already carries the same `worktrees` array by default.

## After reading the output

After running `aiwf status`, narrate the state to the user in plain language — don't just dump the report and stop. Typical follow-ups:

- If a milestone is `in_progress`, mention it and offer to continue: *"M-002 is in flight — want to keep building, or wrap it?"*
- If a milestone has open ACs, name the count: *"M-002 still has 1 AC open — `aiwf show M-002` for the breakdown."*
- If no milestone is `in_progress` but milestones are `draft` under an active epic, suggest starting the next: *"M-003 is next in E-01 — start it?"*
- If there are open decisions that are blocking progress, surface them: *"ADR-0001 is still proposed — does it need ratification?"*
- If there are open gaps and free time, mention them: *"G-001 was logged during M-001 — want to address it now or defer?"*

The verb is the data layer; the AI is the narration layer.
