---
id: D-0001
title: entity-body-empty treats sub-headings as content under top-level sections
status: accepted
relates_to:
    - M-0066
    - G-0058
---

## Question

[M-0066](../epics/E-17-entity-body-prose-chokepoint-closes-g-058/M-066-aiwf-check-finding-entity-body-empty.md) defines the `entity-body-empty` finding's "empty" condition as *"between the section heading and the next heading (or EOF), no non-heading non-whitespace content"*. Read literally, a milestone whose `## Acceptance criteria` section contains only `### AC-N` sub-headings (the standard markdown shape) would qualify as **empty** — sub-headings are headings, "non-heading non-whitespace content" excludes them. That would fire `entity-body-empty/milestone` on every milestone with ACs but no parent-level prose, which is the typical milestone shape across the kernel repo (and across what the M-0067 friction-reducer is designed to enable). Should the rule fire on those milestones?

## Decision

**No.** For top-level (`## Section`) bodies, sub-headings count as content — a section is non-empty if it contains *anything* non-whitespace, including nested `###`/`####` headings. For AC bodies (`### AC-N`), the leaf-prose interpretation holds — only non-heading non-whitespace content satisfies the rule.

Operationally: `entityBodyEmpty` walks each kind's required `## Section` headings and asks "does the slice between this heading and the next contain any non-whitespace?" — yes-or-no, headings included. For each AC heading inside a milestone's body, the same walk applies but with sub-headings filtered out before the non-whitespace check.

## Reasoning

Three threads pushed toward this asymmetry:

- **The strict reading would invalidate the canonical milestone shape.** Every milestone in this repo (and the design's templates) places AC `### AC-N` headings directly under `## Acceptance criteria` with no parent-level prose. Strict reading fires the rule on all of them. M-0066 was scoped to *catch missing prose*, not to require an additional layer of prose above the existing AC structure. The friction-reducer M-0067 also doesn't add parent-level prose — it scaffolds AC headings + AC body content. If the rule fires on the parent regardless, the friction-reducer can't actually clear the finding without a parallel change to scaffold parent prose, which the spec doesn't ask for.
- **The asymmetry matches each level's role.** Top-level `## Section` headings are *containers* for the kind's structured story — Goal/Approach/Acceptance criteria for milestones, Context/Decision/Consequences for ADRs, etc. Their job is to organize the page; their content is whatever the kind has at that level (prose, sub-headings, lists). AC `### AC-N` bodies are *leaf prose* — the place where the AC's testable behavior is described in detail. The asymmetry mirrors the document model.
- **Per CLAUDE.md's "prose is not parsed" principle.** The kernel asserts presence, not structure or quantity. Counting sub-headings as content for parent sections preserves that — the rule asks "is anything there?" not "is the right kind of thing there?".

Out of scope for this decision (worth recording so future readers don't relitigate):

- **Configurable strictness.** A future config knob `tdd.strict-parent-sections: true` could let projects opt into the literal reading. YAGNI for the PoC; revisit if drift shows up.
- **Render-side enforcement of parent prose.** The render pages (I3) read what's written; if a parent section has only sub-headings, the page renders that. This decision does not change rendering.
- **`acs-body-coherence` interaction.** That rule is about heading↔frontmatter pairing, not body content. Independent of this decision.

This was caught at wrap time on M-0066/AC-1 — the impl shipped with this asymmetry built in (`isAllWhitespaceOrHeadings` takes a `leafLevel` parameter), but the AC's body text didn't surface the design call. Recording here, mirrored under M-0066's `## Decisions made during implementation` section, so future readers (or a kernel-spec reviewer) can find both the impl and the rationale.
