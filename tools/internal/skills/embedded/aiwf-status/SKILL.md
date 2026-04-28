---
name: aiwf-status
description: Use to answer project-state questions like "what's next?", "where are we?", "what are we working on?", "current status?", "what's in flight?". Runs `aiwf status`, which prints a one-screen snapshot of in-flight epics + their milestones, open decisions (proposed ADRs and D-NNN), open gaps, the last 5 events from git history, and tree-health counts. Read-only; no commit.
---

# aiwf-status

A one-screen project snapshot. Reach for this whenever the user asks a vague-state question — *"what's next?"*, *"where are we?"*, *"what are we working on?"*, *"status?"*, *"what's in flight?"*. Don't compose multiple `aiwf check` / `aiwf history` calls and read raw frontmatter when one verb answers the question.

## What it does

`aiwf status` walks the planning tree and renders five sections:

1. **In flight** — every epic with status `active`, plus every milestone underneath it. The currently-running milestone is marked `→`; done milestones get `✓`.
2. **Open decisions** — ADRs (`ADR-NNNN`) and D-NNN entries with status `proposed`. The decisions that haven't been ratified, rejected, or superseded yet.
3. **Open gaps** — gaps with status `open`, with the milestone or epic each was discovered in.
4. **Recent activity** — the last 5 commits whose messages carry an `aiwf-verb:` trailer. Date, actor, verb, subject. Cross-entity, no filter.
5. **Health** — total entity count, error count, warning count from `aiwf check`. If non-zero, points at `aiwf check` for details.

By design, the verb shows *only in-flight state* at the top level — closed epics, accepted ADRs, addressed gaps, and cancelled work are not surfaced. The view is forward-looking. For history of a specific entity, use `aiwf history <id>`.

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
      → M-002 — auth wiring          [in_progress]
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

For scripting: `aiwf status --format=json --pretty`. JSON envelope with the same data, structured.

## After reading the output

After running `aiwf status`, narrate the state to the user in plain language — don't just dump the report and stop. Typical follow-ups:

- If a milestone is `in_progress`, mention it and offer to continue: *"M-002 is in flight — want to keep building, or wrap it?"*
- If no milestone is `in_progress` but milestones are `draft` under an active epic, suggest starting the next: *"M-003 is next in E-01 — start it?"*
- If there are open decisions that are blocking progress, surface them: *"ADR-0001 is still proposed — does it need ratification?"*
- If there are open gaps and free time, mention them: *"G-001 was logged during M-001 — want to address it now or defer?"*

The verb is the data layer; the AI is the narration layer.
