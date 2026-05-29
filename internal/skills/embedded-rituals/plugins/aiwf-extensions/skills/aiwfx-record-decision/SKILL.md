---
name: aiwfx-record-decision
description: Records a decision that surfaces during planning, implementation, or review — as an ADR (architectural, long-lived) or as an aiwf D-NNN entity (project-scoped). Allocates the id via `aiwf add`, fills the body from the appropriate plugin template, commits. Invoke in-flow whenever a decision worth keeping for future readers becomes clear; the calling skill (start-milestone, wrap-milestone, review-code, plan-epic, etc.) just hands off and continues.
---

# aiwfx-record-decision

A thin recipe. The skill exists so the mechanical steps of capturing a decision (allocate the id, pick the right template, fill it in, commit) happen consistently from any caller.

## When to use

A decision becomes clear that future readers (six months from now) would regret not finding written down. Triggers:

- A default changed or a new default introduced.
- A strategy considered and rejected.
- A scope cut or framing shift that affects downstream work.
- A supersession of a prior decision.
- A trade-off that won't be obvious from reading the code.

The decision can surface anywhere — during planning (`aiwfx-plan-epic`), mid-implementation (`aiwfx-start-milestone`), at review (`wf-review-code`), at wrap (`aiwfx-wrap-milestone` or `aiwfx-wrap-epic`). Wherever it surfaces, hand off to this skill and continue.

## ADR vs D-NNN — which to pick

| Pick | When |
|---|---|
| **ADR** (`docs/adr/ADR-NNNN-<slug>.md`) | Architectural. Durable across multiple epics. Cross-cutting concern (sec, perf, data model, dependency choice, language idiom). The kind of thing a new contributor reads to understand "why is this code shaped this way?" |
| **D-NNN** (`work/decisions/D-NNN-<slug>.md`) | Project-scoped. Tied to a specific epic or milestone. Sequencing decisions, scope cuts, mid-implementation pivots, deliberate trade-offs that don't rise to architectural weight. |

If you're unsure, ADR is usually the right call — durability is the cheap-to-add side. The cost of writing one ADR that turns out to be project-scoped is small; the cost of failing to record an architectural decision is large.

## Workflow

### 1. Pick the kind

Ask the user (or, if the calling skill knows, just pick): ADR or D-NNN?

### 2. Allocate the id

For an ADR:

```bash
aiwf add adr --title "<imperative title>"
```

For a D-NNN:

```bash
aiwf add decision --title "<imperative title>"
```

aiwf creates the file with the minimal body skeleton, sets frontmatter, produces one commit with `aiwf-verb: add` trailers.

### 3. Replace the body with the rich template

For an ADR: read this plugin's `templates/adr.md`. Fill in:

- **Status** — keep `proposed` while the decision is open for ratification; flip to `accepted` once it's in force.
- **Context** — what forces shape the choice; what alternatives were considered.
- **Decision** — what's decided, in plain imperative voice.
- **Consequences** — positive and negative; follow-up work; migration cost.
- **Validation (optional)** — how we'll know it still holds.

For a D-NNN: read this plugin's `templates/decision.md`. Fill in:

- **Status** — same vocabulary.
- **Question** — what was being decided; what made the answer non-obvious.
- **Decision** — what's decided.
- **Reasoning** — alternatives considered and rejected; honest reasoning.
- **Consequences (optional)** — downstream rules or follow-up work.

### 4. Body header — date and decided_by

In the body, just under the `# ADR-NNNN — <title>` (or `# D-NNN — <title>`) heading, add a one-line block-quote header capturing date and the person making the call:

```markdown
> **Date:** YYYY-MM-DD · **Decided by:** <role/name>
```

These do **not** go in frontmatter. aiwf core's frontmatter parser is strict — it rejects unknown fields so typos don't go silent — and `date` / `decided_by` are not part of the validated entity schema. Putting them in frontmatter would fail `aiwf check`.

The canonical timestamp and actor are also recoverable from git via `aiwf history <id>` (commit author + ISO date), so the body header is redundant-but-friendly: it lets a human reading the file see when and by whom the decision was made without dropping to the CLI.

### 5. Frontmatter touches (optional, for cross-references)

aiwf core only validates these frontmatter fields on ADR / D-NNN entries: `id`, `title`, `status`, plus the cross-reference fields. Set the cross-references when relevant:

- For an ADR that supersedes another: set `supersedes: [ADR-NNNN]`. **Then edit the superseded ADR** to set `superseded_by: ADR-NEW` and promote it to `superseded` via `aiwf promote`.
- For a D-NNN tied to specific work: set `relates_to: [E-NN, M-NNN]` so cross-references resolve.

Skip both if no cross-references apply.

### 6. Validate

```bash
aiwf check
```

Catches things like a misnamed reference, an out-of-set status, or a broken supersession chain.

### 7. Commit the body fill

The `aiwf add` already produced one commit (the scaffold). The body fill is a second commit:

```bash
git add docs/adr/ADR-NNNN-<slug>.md     # or work/decisions/D-NNN-<slug>.md
git commit -m "docs(adr): ADR-NNNN — <title>"
```

The two-commit shape is intentional: the first commit is "id allocated"; the second is "decision authored." `aiwf history ADR-NNNN` shows both.

### 8. Mirror the id back to the caller's context

If invoked from `aiwfx-start-milestone` mid-flight: add the new id under `## Decisions made during implementation` in the tracking doc.
If from `aiwfx-wrap-epic`'s ADR harvest: add to `## ADRs ratified` or `## Decisions captured` in `wrap.md`.
If from `wf-review-code`: list it under "Track for later" in the review report.

The decision now exists; the calling skill resumes its workflow.

## Promotion

ADRs and D-NNN decisions start as `proposed`. They're promoted via `aiwf promote`:

```bash
aiwf promote ADR-NNNN accepted     # in force
aiwf promote ADR-NNNN superseded   # replaced (set superseded_by first)
aiwf cancel  ADR-NNNN              # rejected (terminal)
```

Same for D-NNN. aiwf validates each transition; illegal moves error out.

## Anti-patterns

- *Capturing implementation details as decisions.* "We named the variable foo" is not a decision; "we chose to model auth as a service rather than a library" is.
- *Writing a long prose decision in the tracking doc body.* Those should live in an ADR or D-NNN. The tracking doc just points at the id.
- *Skipping the supersession edit.* A supersession is two-sided: the new entry says what it supersedes; the old entry's status flips to `superseded` and gets a `superseded_by:` pointer. Both edits.
- *Writing the decision but never promoting it past `proposed`.* If it's in force, promote to `accepted`. Otherwise it never feels "decided."

## Constraints

- 🛑 Decision text is durable. Once accepted, future supersession edits the *new* ADR/D-NNN; the original keeps its history. Never delete or rewrite a ratified decision.
- Use this skill, not raw `aiwf add adr`, when the decision is real. The body fill is what makes the record useful; the id alone isn't.
