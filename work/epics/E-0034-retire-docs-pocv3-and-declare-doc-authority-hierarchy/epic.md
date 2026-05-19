---
id: E-0034
title: Retire docs/pocv3/ and declare doc-authority hierarchy
status: proposed
---

## Goal

Refactor `docs/` so a reader (human or LLM) can identify each file's authority tier from its path. Retire the historical `docs/pocv3/` directory by relocating its surviving content, archiving its pre-dogfooding artifacts, and declaring the resulting hierarchy in CLAUDE.md.

## Context

`docs/pocv3/` is the working-name vintage from before the trunk promotion (`PROMOTION-PLAN.md`, Step 5, commit `e0a7fe5`). Decision 1 of that promotion deliberately deferred a rename — *"docs/pocv3/ keeps its name for now. Might refactor docs later."*

Three structural problems have accumulated since:

1. **`docs/pocv3/plans/` predates dogfooding.** When those plans were written, the project did not yet track work as `aiwf` entities. Forward plans now live in `work/epics/` and `work/milestones/`, so `docs/pocv3/plans/*.md` is no longer *active design intent* — each file is either shipped (its plan now lives in closed entities), partly shipped (a split is needed), or never started (file an entity, archive the doc).
2. **`docs/pocv3/design/` is mixed.** Some files are load-bearing normative records (`design-decisions.md`, `provenance-model.md`, `tree-discipline.md`, `policy-model.md`, `design-lessons.md`, `id-allocation.md`). Others read as pre-implementation research (`_scratch-subagents-research.md`, `agent-orchestration.md`, `parallel-tdd-subagents.md`) that belong under `docs/explorations/` or `docs/archive/`.
3. **Authority tier is opaque from path.** A reader skimming `docs/` cannot tell normative records from exploratory thinking from archival content without reading every file. The directory naming itself (`pocv3`) reads as "PoC," which is misleading — this is trunk-active content.

ADR-0004 flagged this inline: *"The broader question of which `docs/` trees are normative vs. exploratory vs. archival deserves its own treatment in a separate gap."* G-0092 was that gap. G-0074 (PoC-framing prose sweep) and G-0075 (rename or accept) accumulated alongside. This epic supersedes all three by attacking the root cause — the layout — instead of patching its symptoms.

## Scope

### In scope

- Per-file triage of every file under `docs/pocv3/` into one of: relocate to `docs/<subdir>/`, archive to `docs/archive/` (or pocv3-archive sibling), supersede with an aiwf entity, or delete.
- Executing the relocate moves + the cross-link sweep across all `docs/pocv3/` references repo-wide (current count: ~163 files including markdown, Go source under `internal/`, embedded skill markdown).
- CLAUDE.md gains a "Documentation hierarchy" section naming each active `docs/` subtree by authority tier.
- Optional kernel-side chokepoint preventing future `docs/pocv3/` literals from re-entering the repo.

### Out of scope

- Changes to `docs/explorations/`, `docs/research/`, or `docs/adr/` content beyond what's necessary to receive relocated material.
- Net-new design content. The triage may surface that some pre-dogfooding plans want to become entities; filing the entity is in scope, *writing* its body is not.
- The drift-check rule for normative-tree references to removed code paths (G-0061-class follow-on). G-0092's gap body lists this as out of scope for the immediate hierarchy work; it remains a separate concern.

## Constraints

- **Sequencing.** Implementation milestones (Relocate onward) wait until E-0033 wraps. E-0033's Pass A (M-0121) reads `docs/pocv3/design/design-decisions.md` as a primary citation source and writes new tests under `internal/policies/`; the relocate sweep touches the same files. Triage (markdown-only deliverable) can run in parallel.
- **CLAUDE.md hierarchy written once.** The section is written against the *final* post-relocate layout, not the current one. No two-pass thrash.
- **Triage is recorded, not remembered.** Per-file disposition lives as a markdown table committed under this epic, not as working memory. A reviewer can audit the rationale per file.
- **No half-finished implementations.** If the relocate milestone lands, every former `docs/pocv3/` reference points somewhere valid; `aiwf check` clean and link-check clean.
- **Forget-by-default per ADR-0004.** Archived files stay accessible via the archive tree; they are not deleted unless the triage rationale explicitly justifies deletion.

## Success criteria

<!-- Observable outcomes at epic close, not tests. -->

- [ ] `docs/pocv3/` directory no longer exists at epic close.
- [ ] Every file formerly under `docs/pocv3/` is accounted for in the triage table (relocated, archived, superseded, or deleted) with a one-line rationale.
- [ ] CLAUDE.md "Documentation hierarchy" section names every active `docs/` subtree by authority tier (normative / forward-looking / exploratory / archival).
- [ ] No `internal/**` Go file contains a literal `docs/pocv3/` path string — either because the drift chokepoint milestone landed, or because a follow-up gap is filed and referenced.
- [ ] `aiwf check` clean against the post-refactor tree.
- [ ] Repo-wide link check (lychee or equivalent) clean against the post-refactor tree.
- [ ] G-0074, G-0075, G-0092 promoted to `addressed` and archived under this epic's wrap.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the existing top-level `docs/archive/` absorb `docs/pocv3/archive/` content, or stay separate (e.g. as `docs/archive/pocv3/`)? | no | Decided during Triage milestone, recorded in the triage table. |
| Do `docs/pocv3/handoff/` and `docs/pocv3/migration/` collapse into `docs/archive/` wholesale, or get per-file triage? | no | Decided during Triage milestone. Default lean: per-file triage; bulk-move is fine if every file is genuinely archival. |
| Should the drift chokepoint (preventing future `docs/pocv3/` literals) be the fourth milestone or a follow-up gap? | no | Decided at Hierarchy milestone wrap, based on whether residual references survive the relocate sweep. |
| Where does the `docs/pocv3/skill-author-guide.md` belong? | no | Decided during Triage; candidate paths: `docs/skill-author-guide.md`, `docs/design/skill-author-guide.md`, or absorbed into the rituals plugin's docs. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Relocate runs concurrently with E-0033's Pass A and a Go file gains a `docs/pocv3/` literal between the sweep and CI. | medium | Sequencing constraint above. Relocate gated on E-0033 wrap; if a brief overlap is unavoidable, the drift chokepoint milestone closes the window. |
| Triage misses a file or undercounts cross-references; some links break silently. | medium | Triage milestone's deliverable is a recorded table validated against `find docs/pocv3 -type f`. Relocate milestone closes with a repo-wide link-check pass. |
| Pre-dogfooding plans contain ratified design intent that gets lost in archival. | low | Triage rationale explicitly flags `partly-shipped` files; the residual intent is split into an entity before the doc is archived. |

## Milestones

<!-- Candidates with one-line descriptions; refine via aiwfx-plan-milestones once E-0033 is closer to wrap. -->

- **Triage** — Per-file disposition table for every file under `docs/pocv3/`, committed under this epic. Markdown-only deliverable; no Go conflict. Can run in parallel with E-0033.
- **Relocate** — Execute the moves recorded in the triage table; update every cross-reference (markdown, Go source, embedded skill markdown). Repo-wide link-check + `aiwf check` clean. *Gated on E-0033 wrap.*
- **Hierarchy** — CLAUDE.md gains the "Documentation hierarchy" section naming each active `docs/` subtree by authority tier. *Gated on Relocate.*
- **Drift chokepoint** (optional) — `internal/policies/` rule preventing `docs/pocv3/` literals from re-entering the repo. *Decision deferred to Hierarchy wrap.*

## Supersedes

- **G-0074** — docs/pocv3/ body prose still uses PoC framing; needs sweep. Most affected prose lives in files that get archived; survivors get reframed during Relocate.
- **G-0075** — docs/pocv3/ directory naming is now historical; rename or accept. Answered by Triage + Relocate: the directory is retired, not renamed.
- **G-0092** — No documented hierarchy of doc authority across docs/. Answered by the Hierarchy milestone.

## References

- **ADR-0004** — Uniform archive convention for terminal-status entities. Inline-flagged the doc-authority question; this epic answers it for `docs/`.
- **CLAUDE.md** § "What aiwf commits to" §6 (layered location-of-truth) — describes engine/policy/state separation; this epic extends the same forget-by-default discipline to documentation.
- **PROMOTION-PLAN.md** Step 5 + Decision 1 — the original deferral that this epic resolves.
- **E-0033** — Pin legal kernel-verb workflows mechanically. Sequencing dependency: relocate-class milestones wait for E-0033 wrap.
- **G-0061 / G-0085 / G-0086** — prior doc-drift class findings, all addressed; their pattern motivates the optional drift chokepoint milestone.
