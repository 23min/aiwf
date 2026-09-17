---
name: aiwfx-record-decision
description: Records a decision that surfaces during planning, implementation, or review — as an ADR (architectural, long-lived) or as an aiwf D-NNNN entity (project-scoped). Drafts the body from the appropriate template and creates the entity with it via `aiwf add`. Invoke in-flow whenever a decision worth keeping for future readers becomes clear; the calling skill (start-milestone, wrap-milestone, review-code, plan-epic, etc.) just hands off and continues.
---

# aiwfx-record-decision

A thin recipe. The skill exists so the mechanical steps of capturing a decision (pick the right template, fill it in, create the entity with it) happen consistently from any caller.

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

### 2. Draft the body from the rich template

The rich template is the source for the body. It ships materialized at `.claude/templates/adr.md` (ADR) and `.claude/templates/decision.md` (D-NNNN); `aiwf update` re-materializes both. If the file is absent, run `aiwf update` — **don't** reconstruct the format by copying an existing ADR or decision, which drifts from the canonical template and can silently drop its date-and-decider header line.

Copy the template to a draft file outside the entity tree and delete its `---` frontmatter block: that block is field reference, and `aiwf add` writes the real frontmatter.

For an ADR: read `.claude/templates/adr.md`. Fill in: **Context**, **Decision**, **Consequences** and **Validation**, each as the template directs beside it.

For a D-NNNN: read `.claude/templates/decision.md`. Fill in: **Question**, **Decision**, **Reasoning** and **Consequences**, each as the template directs beside it.

### 3. Create the entity with its body

For an ADR:

```bash
aiwf add adr --title "<imperative title>" --body-file <draft>
```

For a D-NNNN:

```bash
aiwf add decision --title "<imperative title>" --body-file <draft> --relates-to <ids>
```

`--relates-to` names what the decision is tied to — the epic or milestone it came from, an earlier decision it replaces — so the cross-references resolve. No verb sets it later, so pass it here or not at all; drop the flag when nothing applies.

aiwf allocates the id and lands the frontmatter and the body in one commit with `aiwf-verb: add` trailers. `aiwf add` refuses a body with a required section missing or empty, which is why the body is drafted first. Revise it later with `aiwf edit-body <id>`, never a plain `git commit`, which trips the kernel's `provenance-untrailered-entity-commit` finding.

### 4. Record a supersession

When the new decision replaces an earlier `accepted` one, retire the earlier one when you promote the new one to `accepted` (see Promotion), and not before: a superseded decision has no ordinary transition back, so retiring it while its replacement is still `proposed` strands it if the replacement is rejected. Skip this step when nothing is replaced.

```bash
# An ADR replacing an ADR: also writes superseded_by on the old ADR and supersedes on the new one.
aiwf promote <old-adr-id> superseded --superseded-by <new-adr-id>

# A D-NNNN replacing a D-NNNN: decisions carry no supersession link, so name the old one in --relates-to at step 3.
aiwf promote <old-decision-id> superseded
```

Only an `accepted` decision moves to `superseded`. One still `proposed` is withdrawn with `aiwf cancel` instead.

### 5. Validate

```bash
aiwf check
```

Catches things like a misnamed reference, an out-of-set status, or a broken supersession chain.

### 6. Mirror the id back to the caller's context

If invoked from `aiwfx-start-milestone` mid-flight: add the new id under `## Decisions made during implementation` in the milestone spec, as that ritual's step 6 directs.
If from `aiwfx-wrap-epic`'s ADR harvest: add to `## ADRs ratified` or `## Decisions captured` in `wrap.md`.
If from `wf-review-code`: list it under "Track for later" in the review report.

The decision now exists; the calling skill resumes its workflow.

## Promotion

ADRs and D-NNNN decisions start as `proposed`. They're promoted via `aiwf promote`:

```bash
aiwf promote ADR-NNNN accepted     # in force; if it replaces one, run step 4 now
aiwf promote ADR-NNNN superseded --superseded-by <new-adr-id>   # replaced (step 4)
aiwf cancel  ADR-NNNN              # rejected (terminal)
```

Same for D-NNNN, except that its supersession takes no `--superseded-by`. aiwf validates each transition; illegal moves error out.

## Referencing a decision

A ritual or behavioral skill states its behavioral fact directly and
self-contained. It does not embed a markdown link to a decision record or design
doc under `docs/` (or another non-shipping repo path). A decision's rationale
lives in its own entry, authored via this skill — not in a link from a
behavioral skill. When the "why" matters, record it as a decision with this
skill and name it in prose; do not point at a repo file the reader does not
have.

## Anti-patterns

- *Capturing implementation details as decisions.* "We named the variable foo" is not a decision; "we chose to model auth as a service rather than a library" is.
- *Writing a long prose decision under the milestone spec's `## Decisions made during implementation`.* Those live in an ADR or D-NNNN; `aiwfx-start-milestone` states what that section carries.
- *Leaving a replaced decision in force.* Recording the new decision is half a supersession; the old one is promoted to `superseded` too (step 4).
- *Writing the decision but never promoting it past `proposed`.* If it's in force, promote to `accepted`. Otherwise it never feels "decided."

## Constraints

- 🛑 Decision text is durable. Once accepted, a change of mind is a new ADR or D-NNNN that replaces it (step 4); the original keeps its history. Never delete or rewrite a ratified decision.
- Use this skill rather than a bare `aiwf add adr`: the template is what gives the body its header line and says what each section holds.
