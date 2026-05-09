---
name: aiwfx-whiteboard
description: Open-work synthesis ritual — answers direction questions like "what should I work on next?", "give me the landscape", "where should we focus?", "what's the critical path?", "synthesise the open work", "draw the whiteboard". Loads tree state via `aiwf status` / `aiwf list` / `aiwf show` / `aiwf history`; produces a tiered open-work landscape, a recommended sequence, a first-decision fork, and an optional Q&A gate over pending decisions. Read-only; no commit; no persisted artefact.
---

# aiwfx-whiteboard

Synthesises the open-work landscape into a tiered view, a recommended sequence, a first-decision fork, and a Q&A gate over pending decisions. The output is conversational, not authoritative — the operator decides; the skill surfaces.

## Tier classification rubric

Classify each open item by **leverage on future work**, not by chronology of when it appeared. The criteria below are reproducible; tier *contents* may vary at the margin (LLM judgement on borderline items is acceptable; the criteria themselves do not move).

### Tier 1 — compounding fixes

**Criterion:** the item, once closed, removes friction from *every* future planning or implementation session. Typical shape: a kernel asymmetry, a chronic warning class, a missing-verb gap that forces a workaround at every use.

**Examples:**
- **G-071** — `entity-body-empty` rule lifecycle-blind; closes the standing 24-warning E-20 baseline once fixed.
- **G-072** — no writer verb for milestone `depends_on`; every multi-milestone epic re-derives the workaround.
- **G-065** — no `aiwf retitle` verb; every scope-change moment loses the title-correction option.

### Tier 2 — architecturally foundational

**Criterion:** the item is a `proposed`-status ADR whose ratification + implementation epic gates downstream work. Typical shape: data-shape, lifecycle, or namespace decisions whose absence forces every consumer to invent a local convention.

**Examples:**
- **ADR-0001** — mint entity ids at trunk integration; foundational for parallel-branch work and downstream ADRs.
- **ADR-0004** — uniform archive convention; makes high-volume kinds tractable for read verbs.
- **ADR-0003** — finding F-NNN as 7th kind; substrate for AC-closure chokepoints and cycle-time findings.

### Tier 3 — workflow rituals

**Criterion:** the item is a missing or under-defined ritual that the operator currently re-derives in conversation each time. Smaller leverage than Tier 1/2, but codification removes ad-hoc thrash.

**Examples:**
- **G-059** — no canonical mapping from epic/milestone hierarchy to git branches.
- **G-060** — patch ritual loosely defined; small fixes lack a canonical shape.
- **G-063** — no `start-epic` ritual; epic activation is a deliberate sovereign act with no preflight today.

### Tier 4 — operational debris

**Criterion:** small, isolated fixes that don't compound but are cheap to batch. Typical shape: a one-line `.gitignore` change, a single config nudge, a typo. Leverage is per-item, not per-session.

**Examples:**
- **G-056** — render `site/` not gitignored.
- **G-057** — stray `aiwf` binary in repo root not gitignored.
- **G-069** — `aiwf init` ritual nudge hardcodes user-scope CLI form.

### Tier 5 — defer until a forcing function shows up

**Criterion:** the item is open but no current consumer or workflow forces it. Premature work here costs design effort that will be re-derived once the forcing function lands.

**Examples:**
- **G-070** — `aiwf doctor --format=json`; defer until a JSON consumer appears.
- **G-067** — `wf-tdd-cycle` advisory; couples to the agent-orchestration substrate.
- **G-068** — discoverability policy misses dynamic finding subcodes; activates with F-NNN.

## Output template

The output is a single conversational message containing four named blocks, in this order. The action-shaped blocks (sequence, fork, pending) lead; the tiered landscape comes last as the supporting reference data.

### (a) Recommended sequence — numbered prose

Numbered list, one entry per concrete next action. Each entry uses **explicit before / after / parallel** framing relative to the existing in-flight work, e.g.:

1. **Before E-NN's M-NNN starts** — fix G-XYZ; closes the warning baseline. *(Cost: wf-patch.)*
2. **After M-NNN wraps** — ratify ADR-WXYZ.
3. **Parallel low-priority track** — Tier 4 operational debris as a single wf-patch, any time.

Sequence is reproducible across runs given the same tree state; only the lean phrasing varies with LLM judgement.

### (b) First-decision fork — option list

The next concrete sequencing question presented as concrete options, typically A/B/C, each with **pros / cons** and a **lean**:

> **A. Fix Tier 1 standalone first, then start the next milestone.** Pros: cleaner baseline. Cons: 2 small milestones of delay.
>
> **B. Roll Tier 1 fix into the next milestone.** Pros: marginal scope, immediate warning cleanup. Cons: scope creep.
>
> **Lean: B.** Reasoning: …

The lean is named explicitly so the operator can agree, redirect, or re-weigh.

### (c) Pending decisions — list

Numbered list of open Q&A items implied by the synthesis. None should be blocking the next concrete action; if one is, surface it as the first-decision fork instead. Each item names what it would unlock if answered.

### (d) Tiered landscape — table

A markdown table, one row per open item across the relevant kinds (`epic`, `milestone`, `gap`, `adr`). Required columns:

| Column | Content |
|---|---|
| **Item** | id + short title (e.g. `G-071 — entity-body-empty rule lifecycle-blind`) |
| **Kind** | gap / adr / epic / milestone |
| **Cost** | rough sizing — `tiny`, `wf-patch`, `small milestone`, `medium milestone`, `epic`, `multi-epic` |
| **What it unblocks** | one-line description of leverage on future work |

Group rows by tier (Tier 1 first, Tier 5 last). Each tier is a sub-heading above its rows. The landscape comes last because it's the supporting reference data — the action-shaped blocks above lead with what to do; the table backs them with the full inventory.

## Output cache (WHITEBOARD.md)

After rendering blocks (a)–(d) into the conversation, write the same four blocks to `WHITEBOARD.md` in the consumer repo's root. The file is **gitignored** by convention (see `.gitignore` entry; the consumer repo's `aiwf init` / `aiwf update` should add this if not present). Subsequent invocations overwrite the file in place.

The cache lets the operator re-read the last synthesis without re-invoking the skill — useful when the chat-session context has scrolled past the rendered output. The cache is **not authoritative**: the live tree is the truth, and `WHITEBOARD.md` is a snapshot that drifts from the tree the moment any planning entity changes status. Treat `WHITEBOARD.md` like `STATUS.md`: a regeneratable view, not the source of truth.

## Q&A gate

After rendering blocks (a)–(d), the skill emits exactly one gate prompt and waits:

> *"Walk through the pending decisions one at a time, or is the recommendation enough?"*

The operator picks one of three paths:

1. **Walk through (Q&A).** The skill walks each pending decision one at a time per CLAUDE.md *Working with the user* §Q&A format — context, options with pros/cons, lean, numbered choice, wait. Move to the next decision only after the operator answers the current one. Never batch.
2. **Recommendation is enough.** The skill exits cleanly with a one-line summary (`"Recommendation captured. Next: <first-decision lean>."`) and stops. No follow-up questions.
3. **Operator names a different follow-up.** The skill does not silently extend; if the follow-up is out of scope (refactor advice, design review, cross-team blocking), respond with *"That sounds like its own skill — should we file one?"* and stop.

The one-at-a-time discipline is non-negotiable: batched-question rendering breaks the operator's documented preference and makes the skill's output authoritative-but-brittle, which is exactly the failure mode the gate exists to prevent.

## Anti-patterns

These are the failure modes this skill exists to avoid. If a draft response drifts toward any of them, stop and reshape.

### 1. Replacing the operator's judgement (instead of surfacing and gating)

The skill's job is to **surface** structure (tiers, sequence, decisions) and **gate** on the operator's choice. It does not pick on the operator's behalf. Phrases like *"the right answer is …"*, *"you should …"*, *"the obvious choice …"* are red flags. Use *"the lean is …"* and *"option B trades X for Y"* instead. The operator decides; the skill recommends.

### 2. Inventing verbs that don't exist on the kernel surface

Every verb invocation in the skill body or its rendered output must resolve to a real `aiwf` command available today. If the synthesis would benefit from a verb that doesn't exist, **file a follow-up gap** (`aiwf add gap --title "..." --discovered-in M-NNN`) and surface the gap in the output — do not encode a hand-edit workaround or pretend the verb exists. The kernel surface is authoritative; the skill is advisory; the verb-invention failure mode confuses that hierarchy.

### 3. Persisting the synthesis to a checked-in file

**No `whiteboard.md`, `landscape.md`, or any synthesis snapshot committed to the tree.** A checked-in snapshot goes stale within hours of the next planning act and becomes a second source of truth that disagrees with the live tree. **Gitignored local caches are different and OK** — they regenerate on each invocation, don't share team-wide drift, and don't tax git history. The skill writes such a cache to `WHITEBOARD.md` (see *Output cache* below); `STATUS.md` is the precedent (a persisted artefact regenerated on every commit by the pre-commit hook).

### 4. Scope creep beyond direction-synthesis

The skill answers *"what should I work on next, and what decisions are pending?"* — that is the bounded scope. Adjacent functions belong in their own skills:

- *"Should I refactor X?"* → not this skill; suggest a code-review or design-review skill.
- *"Is this design good?"* → not this skill; suggest a design-review or ADR-authoring conversation.
- *"Who's blocked on what across teams?"* → not this skill; that's a coordination-layer concern, not a planning-tree synthesis.

When the operator's follow-up looks adjacent, say so explicitly: *"That sounds like its own skill — should we file one?"*. Don't silently extend.
