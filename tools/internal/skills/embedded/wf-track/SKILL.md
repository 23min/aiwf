---
name: wf-track
description: Use when an in-progress milestone needs a running progress log — what's been done, decisions made along the way, current blockers. Maintains a tracking document alongside the milestone file. Advisory only; aiwf does not validate tracking documents.
---

# wf-track

A milestone's spec captures what was *committed to* (goal, acceptance criteria). A tracking document captures what's *happened so far* during execution: progress, in-flight decisions, blockers, learnings. Two concerns, two files.

This skill is purely advisory. aiwf never reads, validates, or commits tracking documents — they are plain markdown. The convention exists so the LLM and the human have a stable place to record execution state without polluting the milestone spec.

## When to use

- A milestone has just transitioned to `in_progress` — start a tracking doc.
- Active work has produced a small decision, a discovered blocker, or completed a meaningful chunk — append to the tracking doc.
- The user asks "where are we on M-001?" — read the tracking doc first, then look at recent commits.
- A milestone is being wrapped — read the tracking doc to write a clean wrap-up summary in the milestone body or in a related decision/ADR.

## When *not* to use

- For things that belong in the milestone spec itself (acceptance criteria, scope changes). Edit the spec.
- For decisions large enough to warrant a real `decision` or `adr` entity. Use `aiwf add decision` / `aiwf add adr`.
- As a substitute for git history. Concrete commits are authoritative; the tracking doc is narrative.

## Convention

Tracking documents live at:

```
work/tracking/<milestone-id>.md
```

For example, the tracking doc for `M-001` is `work/tracking/M-001.md`. The directory is outside the entity-bearing roots aiwf walks (`work/epics/`, `work/gaps/`, `work/decisions/`, `work/contracts/`, `docs/adr/`), so aiwf ignores it entirely — no findings, no validation.

Tracking documents are committed to git. They can be reviewed in PRs alongside code changes.

## Recommended structure

The shape below is a starting point. Adapt freely; aiwf does not enforce it.

```markdown
# Tracking — <milestone-id> <title>

Companion to <path-to-milestone-file>.

## Status

One or two sentences: where we are right now.

## Done so far

- 2026-04-15 — scaffolded the parser package
- 2026-04-18 — added the parse → validate path; tests green
- 2026-04-22 — wired into the CLI

## In flight

- Adapter for the new format — first cut compiles, no tests yet
- Naming for the new option flag — Q vs. K vs. spell it out

## Decisions made along the way

- Chose pnpm over npm — devcontainer already had it, lock file is smaller.
- Split the parser into shape-validation and ref-resolution — reads better, test boundaries are clearer.

(For decisions material enough to ratify, run `aiwf add decision` and reference its id here.)

## Blockers / risks

- Upstream library has a bug in v2.16; pinned to 2.15 for now (see [link]).
- Need confirmation from the host team before changing the wire format.

## Next

- Hook the adapter into the integration test harness.
- Run a soak against the staging fixture.
```

## How to update

When recording new progress:

1. Append to **Done so far** with a date and one-line description. Keep entries short; link to commits or external docs as needed.
2. Move items off **In flight** as they finish.
3. Add to **Decisions made along the way** when you make a non-trivial call that doesn't merit a separate `decision` entity. If a decision *does* merit ratification, run `aiwf add decision` and add a one-liner here pointing at the new id.
4. Update **Status** if the high-level picture changed.

The doc is a working journal, not a polished artifact. Brevity beats completeness. If a section is empty for a while, delete it.

## How to wrap

When the milestone is being closed (`aiwf promote <id> done`):

1. Skim the tracking doc.
2. Distill the most important outcomes into the milestone body or a follow-up decision/ADR — anything that future readers should find without having to read the journal.
3. Leave the tracking doc in place. It's history.
