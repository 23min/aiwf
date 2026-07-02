---
name: aiwfx-record-decision
description: Records a decision that surfaces during planning, implementation, or review — as an ADR (architectural, long-lived) or as an aiwf D-NNNN entity (project-scoped). Allocates the id via `aiwf add`, fills the body from the appropriate plugin template, commits. Invoke in-flow whenever a decision worth keeping for future readers becomes clear; the calling skill (start-milestone, wrap-milestone, review-code, plan-epic, etc.) just hands off and continues.
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

## ADR vs D-NNNN — which to pick

| Pick | When |
|---|---|
| **ADR** (`docs/adr/ADR-NNNN-<slug>.md`) | Architectural. Durable across multiple epics. Cross-cutting concern (sec, perf, data model, dependency choice, language idiom). The kind of thing a new contributor reads to understand "why is this code shaped this way?" |
| **D-NNNN** (`work/decisions/D-NNNN-<slug>.md`) | Project-scoped. Tied to a specific epic or milestone. Sequencing decisions, scope cuts, mid-implementation pivots, deliberate trade-offs that don't rise to architectural weight. |

If you're unsure, ADR is usually the right call — durability is the cheap-to-add side. The cost of writing one ADR that turns out to be project-scoped is small; the cost of failing to record an architectural decision is large.

## Workflow

### 1. Pick the kind

Ask the user (or, if the calling skill knows, just pick): ADR or D-NNNN?

### 2. Allocate the id

For an ADR:

```bash
aiwf add adr --title "<imperative title>"
```

For a D-NNNN:

```bash
aiwf add decision --title "<imperative title>"
```

aiwf creates the file with the minimal body skeleton, sets frontmatter, produces one commit with `aiwf-verb: add` trailers.

### 3. Replace the body with the rich template

The rich template — **not** the minimal skeleton `aiwf add` just wrote — is the source for the full body, including the `# ADR-NNNN — <title>` H1 and the `> **Date:** … · **Decided by:** …` header from step 4. It ships materialized at `.claude/templates/adr.md` (ADR) and `.claude/templates/decision.md` (D-NNNN); `aiwf update` re-materializes both. If the file is absent, run `aiwf update` — **don't** reconstruct the format by copying an existing ADR or decision, which drifts from the canonical template and silently drops the H1 and header.

For an ADR: read `.claude/templates/adr.md`. Fill in:

- **Status** — keep `proposed` while the decision is open for ratification; flip to `accepted` once it's in force.
- **Context** — what forces shape the choice; what alternatives were considered.
- **Decision** — what's decided, in plain imperative voice.
- **Consequences** — positive and negative; follow-up work; migration cost.
- **Validation (optional)** — how we'll know it still holds.

**ADR authoring discipline** (CLAUDE.md §"Authoring an ADR"). *Decision is decision.* Record *what* was chosen and *why*, never *when* to act on it. Keep gate/schedule language out of the ADR body — no "ratify after X", no "status stays proposed through Y", no "accept once the epic closes." Whether the decision is in force is the `status:` field (`proposed` → `accepted`); *when to act on it* is a planning concern that lives in the planning surface, not the ADR prose.

For a D-NNNN: read `.claude/templates/decision.md`. Fill in:

- **Status** — same vocabulary.
- **Question** — what was being decided; what made the answer non-obvious.
- **Decision** — what's decided.
- **Reasoning** — alternatives considered and rejected; honest reasoning.
- **Consequences (optional)** — downstream rules or follow-up work.

### 4. Body header — date and decided_by

In the body, just under the `# ADR-NNNN — <title>` (or `# D-NNNN — <title>`) heading, add a one-line block-quote header capturing date and the person making the call:

```markdown
> **Date:** YYYY-MM-DD · **Decided by:** <role/name>
```

These do **not** go in frontmatter. aiwf core's frontmatter parser is strict — it rejects unknown fields so typos don't go silent — and `date` / `decided_by` are not part of the validated entity schema. Putting them in frontmatter would fail `aiwf check`.

The canonical timestamp and actor are also recoverable from git via `aiwf history <id>` (commit author + ISO date), so the body header is redundant-but-friendly: it lets a human reading the file see when and by whom the decision was made without dropping to the CLI.

### 5. Frontmatter touches (optional, for cross-references)

aiwf core only validates these frontmatter fields on ADR / D-NNNN entries: `id`, `title`, `status`, plus the cross-reference fields. Set the cross-references when relevant:

- For an ADR that supersedes another: set `supersedes: [ADR-NNNN]`. **Then edit the superseded ADR** to set `superseded_by: ADR-NEW` and promote it to `superseded` via `aiwf promote`.
- For a D-NNNN tied to specific work: set `relates_to: [E-NN, M-NNN]` so cross-references resolve. A decision's `relates_to` can alternatively be set at allocation — `aiwf add decision --relates-to <ids>` (step 2) — which lands it in the scaffold commit and keeps step 7 a body-only bless.

Skip both if no cross-references apply. These are **frontmatter** edits, not body content: `aiwf edit-body` is body-only, so when you set one here, land the body fill with `aiwf edit-body <id> --body-file <draft>` at step 7 (bless mode refuses a working copy with pending frontmatter changes).

### 6. Validate

```bash
aiwf check
```

Catches things like a misnamed reference, an out-of-set status, or a broken supersession chain.

### 7. Land the body fill via `aiwf edit-body`

The `aiwf add` already produced one commit (the scaffold). Land the filled-in body as a second, **trailered** commit through the `aiwf edit-body` verb — never a plain `git commit`, which lands without the `aiwf-verb` / `aiwf-entity` / `aiwf-actor` trailers and trips the kernel's `provenance-untrailered-entity-commit` finding on every recorded decision:

```bash
aiwf edit-body ADR-NNNN     # bless mode: commits the in-place body edit with trailers
# or, for a project-scoped decision:
aiwf edit-body D-NNNN
```

You edited the body in place at step 3; `aiwf edit-body <id>` (bless mode) commits those working-copy bytes with the provenance trailers in one atomic operation. The two-commit shape is intentional: the first commit ("id allocated") is the `aiwf add` scaffold; the second ("decision authored") is this `aiwf edit-body` body fill. `aiwf history ADR-NNNN` shows both.

**If you set frontmatter cross-references at step 5**, bless mode refuses (it is body-only, and the working copy now has a frontmatter diff). Land the body with `--body-file` instead — it pairs the working-copy frontmatter (cross-references and all) with the new body in one trailered commit:

```bash
aiwf edit-body ADR-NNNN --body-file <draft>
```

`--body-file` takes the body from `<draft>`, **not** your in-place step-3 edit — put the filled-in body in the draft file so you don't commit an empty or stale body. See the `aiwf-edit-body` skill for the `--body-file` and `--reason` variants.

### 8. Mirror the id back to the caller's context

If invoked from `aiwfx-start-milestone` mid-flight: add the new id under `## Decisions made during implementation` in the milestone spec.
If from `aiwfx-wrap-epic`'s ADR harvest: add to `## ADRs ratified` or `## Decisions captured` in `wrap.md`.
If from `wf-review-code`: list it under "Track for later" in the review report.

The decision now exists; the calling skill resumes its workflow.

## Promotion

ADRs and D-NNNN decisions start as `proposed`. They're promoted via `aiwf promote`:

```bash
aiwf promote ADR-NNNN accepted     # in force
aiwf promote ADR-NNNN superseded   # replaced (set superseded_by first)
aiwf cancel  ADR-NNNN              # rejected (terminal)
```

Same for D-NNNN. aiwf validates each transition; illegal moves error out.

## Anti-patterns

- *Capturing implementation details as decisions.* "We named the variable foo" is not a decision; "we chose to model auth as a service rather than a library" is.
- *Writing a long prose decision under the milestone spec's `## Decisions made during implementation`.* Those should live in an ADR or D-NNNN. The spec section just points at the id.
- *Skipping the supersession edit.* A supersession is two-sided: the new entry says what it supersedes; the old entry's status flips to `superseded` and gets a `superseded_by:` pointer. Both edits.
- *Writing the decision but never promoting it past `proposed`.* If it's in force, promote to `accepted`. Otherwise it never feels "decided."

## Constraints

- 🛑 Decision text is durable. Once accepted, future supersession edits the *new* ADR/D-NNNN; the original keeps its history. Never delete or rewrite a ratified decision.
- Use this skill, not raw `aiwf add adr`, when the decision is real. The body fill is what makes the record useful; the id alone isn't.
