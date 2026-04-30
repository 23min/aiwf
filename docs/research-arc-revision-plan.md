# Research-arc revision plan

> **Status:** working plan
> **Branch:** `chore/research-arc-revision` (to be created)
> **Merge strategy:** squash-merge into `main` only when the user approves
> **Scope:** edit the research arc (`docs/research/00`–`11`, `KERNEL.md`, `0-introduction.md`), the working paper, the README, and `docs/architecture.md` to address (a) the seven fact-check findings, (b) the critical-review findings, without rewriting the arc's substance.
> **What this plan is not:** a redesign of the framework. None of the work below changes the framework's conclusions. It changes presentation, scope statements, and falsifiability surfaces.

---

## 0. Pre-work: move the in-flight fact-check fixes onto the branch

The seven fact-check fixes from the prior session are currently uncommitted on `main`:

```
 M README.md
 M docs/research/00-fighting-git.md
 M docs/research/01-git-native-planning.md
 M docs/research/05-where-state-lives.md
 M docs/research/06-poc-build-plan.md
```

Before any new work, we move these onto the branch:

```bash
git checkout -b chore/research-arc-revision
git add -A   # the seven fixes + this plan doc
git commit -m "chore(research): fact-check fixes + revision plan"
```

`main` stays clean. All subsequent work is commits on `chore/research-arc-revision`. Final delivery is a squash-merge.

---

## 1. The findings (consolidated)

### 1.1 Fact-check findings (already drafted as commits, listed for completeness)

| # | File | Issue | Fix |
|---|---|---|---|
| 1 | `00-fighting-git.md` §8 | Pijul attributed to "Tankink, C. & Mimram, S." | Corrected to Mimram & Di Giusto (paper) + Meunier & Becker (Pijul) |
| 2 | `05-where-state-lives.md` §1 | Denicek paper missing co-author | Added Jonathan Edwards |
| 3 | `01-git-native-planning.md` §4.3 | "per architecture §11" — wrong section | Corrected to Appendix A |
| 4 | `05-where-state-lives.md` §3 | Heading and intro said "five layers"; table has six | Both updated to "six layers" |
| 5 | `06-poc-build-plan.md` §2 | "Three entity kinds" contradicted abstract's "Six" | Corrected to six kinds |
| 6 | `06-poc-build-plan.md` Session 1 | "the seven checks" while §5 lists nine | Corrected to nine |
| 7 | `README.md` | "arc of seven documents (KERNEL.md, then 00–06)" | Updated to reflect the actual current arc |

### 1.2 Critical-review findings (the substantive critique)

Cross-cutting weaknesses in the arc:

- **C1. The arc argues with itself.** Every doc walks back the prior. No external skeptic position is ever genuinely engaged.
- **C2. Empirical claims about AI behavior are unfalsifiable as written.** Statements like "the AI re-plans from scratch each session" are presented as observations but are generalizations from one developer's months of practice.
- **C3. The target reader is implicitly the author.** "The user wants…" really means "I want…" The arc says this once but proceeds as if conclusions generalize.
- **C4. "Workflow erodes under LLM amplification" leans on a contested premise** (production time collapses to ~zero). For real-world systems, large surface area, regulated codebases, the bottleneck isn't production.
- **C5. Continuous ratification is presented as paradigm shift; it's a refinement** of trunk-based / pair-programmed / draft-PR practice that already exists.
- **C6. The kernel is post-hoc.** `KERNEL.md` was extracted after the conclusions were reached and is then used as the rubric to score the arc. Not dishonest, but the rubric isn't independent of the arc.

Per-doc objections (full list in §3 below; high-leverage ones here):

- **`00`:** "Fighting git" is dramatic; merge drivers aren't "fighting." Tier 4 (use `git log`) dismissed too quickly. CRDT tools conflated.
- **`01`:** ID collisions hand-waved; "for free, from git" is overstated marketing prose.
- **`02`:** 80/15/5% probabilities are confident theater. "Build Shape A first" collapses to "do nothing yet" for most teams. Steel-mans a "semantic determinism" claim the architecture didn't take.
- **`03`:** Treats skill behavior as more advisory than it has to be on hosts with required-skill mechanisms. CI assumes CI exists. Tombstone clutter unpriced.
- **`04`:** Four axes presented as orthogonal; correlated in practice. CRDT metadata layer claimed as "a few hundred lines" without a sketch.
- **`05`:** Convergence-as-evidence rebrands a weak signal. L3-external costs are conditional, not universal. Brew distribution treated as obvious.
- **`06`:** 4-session / 2,500-line estimates are pseudo-precise. PoC has no merge story for the very situation the arc spent 5 docs analyzing.
- **`07`:** "Queues evaporate" too strong. State-as-canonical / workflow-as-render is harder than the doc admits for regulated work.
- **`08`:** Process claim sold as tool claim. "HITL gets stronger" overstated. Social cost of PRs undersold.
- **`09`:** Role taxonomy unfalsifiable. PM external-facing work absent.
- **`10`:** "Spec-based = waterfall" depends on a particular definition of waterfall. The "where spec-based is right" §7 covers more of professional software development than the doc admits.
- **`11`:** Compose-don't-absorb leans on skills, the layer the arc said you can't depend on. `live_source` brings tree-sitter through the side door.

Slogans / easy-to-misread claims:

- "There is no workflow" survives in summaries despite the careful body.
- "Kill PRs" reads worse than the actual argument.
- "Branch-divergent rules… among the first formal articulations" overclaims novelty.
- "Roughly N lines" / "N sessions" / "80% probability" — pseudo-precision throughout.
- "The framework's bet is X" used so often it becomes a hedge tic.
- `docs/architecture.md` is "superseded" but unmarked, so new readers find it as authoritative.

### 1.3 Substantive design questions (deferred — go to issues, not prose)

- **D1.** CRDT metadata layer is unbuilt. Verify or revise the "few hundred lines" claim.
- **D2.** Tombstone clutter at multi-year horizon needs a retention/archival policy.
- **D3.** `live_source` symbol-level resolution needs a bounded tree-sitter footprint, or it drifts into code-graph territory.
- **D4.** PoC's deferred merge handling needs a "first collision case" review post-PoC.
- **D5.** Framework rename — `ai-workflow` uses the word the arc argues against. Open as discussion, not issue.

These do *not* get fixed in the arc. They get one issue each, cited from the relevant doc.

---

## 2. The plan — four stages

The work is bounded and sized in **focused sessions** (a few hours each). Total: 3–5 sessions across A+B+C, plus ~30 minutes for D.

### Stage A — Positioning pass (highest leverage, ~1 session)

**Goal:** new readers hit a clear scope statement and find the right entry point. Closes the largest "misunderstanding" surface.

**Files touched:**

1. **`README.md`** — add a "who this is for" paragraph near the top: solo through small-team, weeks-to-months horizons, AI-assisted, willing to live in markdown and git. Acknowledge what's *not* in scope (regulated, large-team, multi-stakeholder PM work).
2. **`docs/working-paper.md`** — same treatment. Lead with scope.
3. **`docs/architecture.md`** — add a banner at the top: *"Superseded. This document captures the framework's original ambition; the current direction is in the working paper and the research arc. Kept for lineage."*
4. **`docs/research/0-introduction.md` §6** — already says "not normative for other projects"; add that the author is one developer using this on solo work and the conclusions are conjectures grounded in that experience, not findings.
5. **`docs/research/07-state-not-workflow.md`** — add a short aside acknowledging the framework's name uses the word the doc argues against; renaming is deferred (link to D5 discussion when opened).

**Deliverable:** a single PR-ready commit per file; coherent scope statement reusable across them.

**Effort:** one focused session.

---

### Stage B — Honesty pass (~1 session, mechanical sweep)

**Goal:** remove the surface area a hostile reader latches onto. Doesn't change any conclusion.

**Edits across the arc:**

- Replace pseudo-precise numbers with qualitative ranges:
  - `02` §9: "80% / 15% / 5% probability" → "most likely / sometimes / rarely" (or similar).
  - `06` abstract & §7: "roughly 2,500 lines of Go, four focused sessions" → "a focused week or two of work; small enough to throw away."
  - Anywhere "roughly N hours / N% / N lines" appears in a confident voice — soften.
- Soften specific load-bearing claims:
  - `01` §12 "for free, from git, for files" → "with much less mechanism, by leaning on git's per-file merge."
  - `07` §3 "queues evaporate" → "queues thin sharply for the work LLMs handle well; some queues remain (priority, attention, blocking, decision latency)."
  - `00` abstract framing of "fighting git" → keep the colorful title; in the abstract, say "the original architecture's invariants don't compose cleanly with git's text-merge model."
- Drop the "framework's bet is X" tic where it isn't load-bearing. Keep it for the actual major bets (state-not-workflow, continuous ratification, in-repo planning).
- `01` §5.4 — narrow the novelty claim: "first formal articulation *for AI rules specifically*."
- `02` §3 — explicitly note that the original architecture's hash-verified projection was over structural fields and never claimed prose-meaning stability; the section is about why *the user's intuition* needs scoping, not about a strawman the architecture took.

**Deliverable:** one or two commits with a single-pass diff across the arc. Easy to review.

**Effort:** one focused session.

---

### Stage C — Falsifiability pass (~2–3 sessions)

**Goal:** flip each `defended-position` doc from "confident assertion" to "scoped claim with explicit failure modes." This is the single biggest credibility win.

**Method:** for each doc marked `Status: defended-position`, add a final section: *"What would change my mind"* (or "Where this breaks"). Two or three concrete observations or scenarios that would force revision.

**Docs to update:**

| Doc | "What would change my mind" candidates (drafts) |
|---|---|
| `00-fighting-git` | A merge driver in production handling our shape of state without UX cost. CRDT-substrate tooling matures enough that the substrate replacement becomes cheap. |
| `01-git-native-planning` | Real solo or small-team use surfaces a need for sub-commit-granularity provenance the markdown-canonical model can't carry. Markdown's per-file 3-way merge fails on the structural fields often enough to need a custom merger anyway. |
| `03-discipline-where-the-llm-cant-skip-it` | A host appears with reliable required-skill enforcement that genuinely makes skill-tier a chokepoint. Tombstone clutter at 3+ year horizons becomes worse than collisions. |
| `04-governance-provenance-and-the-pre-pr-tier` | The four project-shape axes turn out to be effectively one (correlated) axis in real teams. Pre-PR tooling proves to require server-side state that pulls the framework toward L3-external. |
| `05-where-state-lives` | Multi-machine sync via git breaks for real users (offline laptops, ephemeral CI). Brew/apt distribution maintenance burden exceeds the submodule maintenance burden it replaces. |
| `07-state-not-workflow` | A team in our target shape genuinely needs a hard cross-entity pipeline rule the framework refuses to model. Workflow renders turn out to require so much custom config they become workflow specs. |
| `11-should-the-framework-model-the-code` | A real consumer's join (contract + symbol + ADR) cannot be served reliably by composition because skills miss the call. `live_source` symbol resolution drifts into traversal under contract authors' actual needs. |

**Deliverable:** one commit per doc, or one batch commit. Each section is short — three to five bullets.

**Effort:** 2–3 focused sessions. Most of the content is implicit in the existing §8s ("honest failure mode") and just needs to be made consistent and explicit across the seven defended-position docs.

---

### Stage D — Open issues, don't argue in prose (~30 minutes)

**Goal:** apply the arc's own discipline ("future references must cite an open issue"). Don't try to settle deferred design in the prose.

**Issues to open:**

| ID | Title | Cited from |
|---|---|---|
| D1 | Prototype the CRDT metadata layer; verify "few hundred lines of Go" claim or revise | `04` §5 |
| D2 | Long-horizon tombstone retention/archival policy | `03` §4 |
| D3 | Bound the tree-sitter footprint for `live_source` (I4); confirm no code-graph creep | `11` §7 |
| D4 | First collision case the PoC encounters; review whether deferred merge handling held | `06` §1 |
| D5 | Framework rename discussion: `ai-workflow` uses the word the arc argues against | `07` |

**Deliverable:** five GitHub issues (or one discussion + four issues for D5). Each doc gets a `(tracked in #NN)` parenthetical at the relevant point.

**Effort:** ~30 minutes once the issues exist.

---

## 3. Per-doc detail — what actually changes where

For reviewers walking the diff, the expected change profile per file:

| File | Stage A | Stage B | Stage C | Stage D citation |
|---|---|---|---|---|
| `README.md` | + scope paragraph | (already done in fact-check pass) | — | — |
| `docs/working-paper.md` | + scope paragraph | minor softening | — | — |
| `docs/architecture.md` | + superseded banner | — | — | — |
| `docs/research/KERNEL.md` | — | — | — | — |
| `docs/research/0-introduction.md` | + author/scope note in §6 | — | — | — |
| `00-fighting-git.md` | — | abstract softening | + "what would change my mind" | — |
| `01-git-native-planning.md` | — | "for free" softening, narrow novelty claim | + "what would change my mind" | — |
| `02-do-we-need-this.md` | — | drop pseudo-probabilities, scope "semantic determinism" critique | — | — |
| `03-discipline-where-the-llm-cant-skip-it.md` | — | — | + "what would change my mind" | D2 |
| `04-governance-provenance-and-the-pre-pr-tier.md` | — | — | + "what would change my mind" | D1 |
| `05-where-state-lives.md` | — | brew distribution honesty | + "what would change my mind" | — |
| `06-poc-build-plan.md` | — | drop 2500-line / 4-session precision | — | D4 |
| `07-state-not-workflow.md` | + name aside | "queues evaporate" softening | + "what would change my mind" | D5 |
| `08-the-pr-bottleneck.md` | — | "HITL gets stronger" softening, name social-cost loss explicitly | — | — |
| `09-orchestrators-and-project-managers.md` | — | acknowledge external-facing PM work | — | — |
| `10-spec-based-as-waterfall.md` | — | clarify waterfall-definition dependency; size the §7 concession honestly | — | — |
| `11-should-the-framework-model-the-code.md` | — | acknowledge composition-via-skills relies on the layer the arc distrusts | + "what would change my mind" | D3 |

**Untouched:** `KERNEL.md` (post-hoc framing concern is real but addressing it requires a meta-doc explaining the kernel was extracted, which is more work than benefit; flag in commit message and move on).

---

## 4. Working agreements

**Branch:** `chore/research-arc-revision`. All work commits here. No commits on `main` until user approves squash-merge.

**Commit style:** Conventional Commits per `CLAUDE.md`. One commit per stage minimum, more if review benefits from finer-grained diffs:

- `chore(research): fact-check fixes + revision plan` (Stage 0, includes pre-existing fixes + this plan)
- `docs(research): scope and positioning pass` (Stage A)
- `docs(research): honesty pass — soften pseudo-precise claims` (Stage B)
- `docs(research): add "what would change my mind" sections` (Stage C — possibly split per doc)
- `docs(research): cite open issues for deferred design questions` (Stage D — last, after issues exist)

**Review cadence:** user reviews after each stage; agent does not move to the next stage without an OK.

**CHANGELOG:** per `CLAUDE.md`, every user-visible change adds a `[Unreleased]` entry. The arc revision is documentation; consider adding a single `### Changed` entry: *"Research arc revised for scope clarity, falsifiability, and citation accuracy. No conclusion changes."* Decide at squash-merge time.

**Pre-PR audit:** before opening the PR, walk the diff against `CLAUDE.md` and `tools/CLAUDE.md` rules. The relevant rule for this work: *"Future references must cite an open issue."* Stage D enforces this for the deferred design questions.

**Out of scope on this branch:**

- Any rewrite of `KERNEL.md`'s eight needs.
- Any reordering or removal of arc documents.
- Any change to the framework name (decision deferred to D5).
- Any code in `tools/`.
- Any changes to skills, contracts, or templates.

If any of these surface as necessary mid-stream, stop and discuss before continuing.

---

## 5. Stage-by-stage execution checklist

### Pre-work
- [ ] Create branch `chore/research-arc-revision`.
- [ ] Stage existing fact-check fixes + this plan; commit.

### Stage A — Positioning
- [ ] User drafts scope statement (one or two paragraphs); agent uses it consistently.
- [ ] Add scope paragraph to README and working paper.
- [ ] Add superseded banner to `architecture.md`.
- [ ] Update `0-introduction.md` §6 with author/scope note.
- [ ] Add framework-name aside to `07`.
- [ ] Commit.

### Stage B — Honesty
- [ ] Sweep for pseudo-precise numbers; replace with ranges.
- [ ] Soften "for free" / "evaporates" / "fighting" framings per §2 above.
- [ ] Drop "the framework's bet is X" tic where non-load-bearing.
- [ ] Narrow `01` §5.4 novelty claim.
- [ ] Scope `02` §3's "semantic determinism" critique.
- [ ] Commit.

### Stage C — Falsifiability
- [ ] Draft "what would change my mind" sections for the seven defended-position docs.
- [ ] User reviews and corrects positions agent gets wrong.
- [ ] Commit (one or split).

### Stage D — Issues
- [ ] User opens five GitHub issues (or four issues + one discussion for D5).
- [ ] Add `(tracked in #NN)` parentheticals to relevant doc points.
- [ ] Commit.

### Pre-merge
- [ ] Walk pre-PR audit against `CLAUDE.md` rules.
- [ ] Decide CHANGELOG entry.
- [ ] User reviews full diff.
- [ ] User says "ok merge."
- [ ] Squash-merge to `main`.

---

## 6. What this plan deliberately does *not* do

- It does not rewrite the arc to remove the trajectory of changing your mind. That trajectory *is* the contribution.
- It does not engage every objection in defensive prose. The honest moves are scope statements upfront and "where this breaks" sections at the end.
- It does not try to make the empirical claims rigorous. They are conjectures from one developer's practice, and the framing should say so.
- It does not reframe to claim *less* novelty than the work has. It narrows specific overclaims (`01` §5.4) without dropping real contributions.
- It does not touch the framework's name or any code.

The arc is intellectually finished. What's missing is presentation discipline. This plan delivers that without disturbing the substance.
