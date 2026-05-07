---
id: M-068
title: aiwf-add skill names fill-in-body as required next step
status: in_progress
parent: E-17
tdd: required
acs:
    - id: AC-1
      title: Skill names fill-in-body as required next step
      status: met
      tdd_phase: done
    - id: AC-2
      title: Skill cites the design intent (acs-and-tdd-plan.md)
      status: met
      tdd_phase: done
    - id: AC-3
      title: Skill recommends the body shape (paragraph, key contents)
      status: met
      tdd_phase: done
    - id: AC-4
      title: Skill names --body-file as in-verb alternative
      status: met
      tdd_phase: done
    - id: AC-5
      title: Skill carries Don't entry against empty entity bodies
      status: open
      tdd_phase: green
---

## Goal

Update the `aiwf-add` skill so an LLM (or human) following it produces non-empty bodies by default across all entity kinds. Today, the skill describes each `aiwf add <kind>` verb and stops there — never naming the body-prose follow-up step the design specifies. Result: skills-driven entity creation reproduces the [G-058](../../gaps/G-058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) defect every time (originally observed for ACs; same shape applies to epic Goal/Scope sections, milestone Goal/Approach, gap What's-missing/Why-it-matters, etc.). This milestone is the cheapest layer of the epic (pure documentation) but the highest-leverage for changing the default behavior, since most entity creation flows through the skill.

## Approach

Edit `internal/skillsembed/aiwf-add/SKILL.md` (or wherever the skill source lives that gets re-emitted to `.claude/skills/aiwf-add/SKILL.md` on `aiwf init` / `aiwf update`). Add:

- A per-kind "After `aiwf add <kind>`: fill in the body" subsection (or a single generic subsection covering all kinds) naming the design intent (cite `acs-and-tdd-plan.md:22` for ACs and `design-decisions.md:139` for the broader principle), the recommended body shape **per kind** (epic: Goal/Scope/Out of scope; milestone: Goal/Approach/Acceptance criteria; AC: pass criteria + edge cases + code references; gap: What's missing + Why it matters; adr: Context/Decision/Consequences; decision: Question/Decision/Reasoning; contract: Purpose/Stability), and the `--body-file` flag from [M-067](M-067-aiwf-add-ac-body-file-flag-for-in-verb-body-scaffolding.md) as the in-verb alternative for ACs (with a note that the analogous flag for other kinds is captured as [G-066](../../gaps/G-066-aiwf-add-epic-milestone-gap-adr-decision-contract-verbs-lack-body-file-flag-for-in-verb-body-scaffolding-only-aiwf-add-ac-will-gain-it-via-m-067-leaving-the-other-six-entity-creation-verbs-reliant-on-post-add-aiwf-edit-body.md), and that until then the workflow for non-AC kinds is `aiwf add <kind> ...` then edit body then `aiwf edit-body <id>`).
- A "Don't" entry: do not leave load-bearing body sections empty for any entity kind — the title is a label, not a spec; the kernel's `entity-body-empty` finding (from [M-066](M-066-aiwf-check-finding-entity-body-empty.md)) will surface the omission for any kind.

The change is verified by the discoverability policy test (`internal/policies/PolicyFindingCodesAreDiscoverable` and the broader skill-doc enumeration from G-021) for the new finding code; the body-prose recommendation is content the policy can't enforce mechanically and ships unblocked.

## Acceptance criteria

### AC-1 — Skill names fill-in-body as required next step

The `aiwf-add` skill source gains a body-prose subsection covering all `aiwf add <kind>` paths — placed where the verb examples live, either as one generic subsection ("After `aiwf add <kind>`: fill in the body") or per-kind subsections — that states unambiguously that scaffolding the entity's frontmatter is step 1 of 2 and writing the body prose is step 2 and is required, not optional, across all kinds. The "What aiwf does" numbered list (currently 5 steps) gains a step 6: "scaffolded body sections are empty by design — fill them in before declaring the entity done; specifically the `### AC-N — <title>` body for ACs and the equivalent load-bearing sections for top-level kinds (epic Goal/Scope/Out-of-scope; milestone Goal/Approach/Acceptance criteria; gap What's-missing/Why-it-matters; etc.)". Verified by reading the rendered skill in a tempdir post-`aiwf init`.

### AC-2 — Skill cites the design intent (acs-and-tdd-plan.md)

The body-prose subsection cites `docs/pocv3/plans/acs-and-tdd-plan.md:22` and `docs/pocv3/design/design-decisions.md:139` as the spec source. The citation is a plain markdown link (paths are stable; if they move the link rots into a 404 in the rendered skill, which is a visible signal). Rationale for the citation: an LLM (or human) following the skill should be able to trace the rule back to the design without grepping the codebase. Same channel discipline as the rest of the kernel's discoverability work.

### AC-3 — Skill recommends the body shape (paragraph, key contents)

The subsection prescribes a per-kind body-shape recommendation. For ACs: one paragraph (not an essay, not a one-liner) covering (a) what passing concretely looks like — the assertable claim; (b) edge cases the test must cover; (c) forward references to the code path or test file. For top-level kinds: each load-bearing section gets at least one paragraph of prose (e.g., epic Goal: "what problem this solves and what success looks like"; gap What's-missing: "the concrete defect"; gap Why-it-matters: "the consequence and why it warrants tracking"). Includes short example blocks for each kind so the operator has concrete shapes to copy. The recommendations are advisory, not enforced — the kernel rule (M-066) checks presence, not structure — but the skill is the chokepoint for shaping default behavior, so the recommendations matter.

### AC-4 — Skill names --body-file as in-verb alternative

The body-prose subsection mentions `--body-file` from M-067 as the in-verb alternative to a follow-up edit pass — for cases where the operator already has the AC prose drafted (e.g. mining from a design doc or a prior conversation), `--body-file` lands the body in the same atomic commit as the AC. The cross-reference is two-way: M-067 AC-8 names the skill change, and this AC names the verb. Both surfaces describe the same flag with the same semantics; no drift. **AC-only scope:** the analogous flag for `aiwf add epic`, `aiwf add milestone`, `aiwf add gap`, etc., is captured as G-066; until that lands, the skill instructs operators to use the two-step `aiwf add <kind>` then `aiwf edit-body <id>` workflow for non-AC kinds.

### AC-5 — Skill carries Don't entry against empty entity bodies

The skill's "Don't" section (currently lists "don't hand-edit frontmatter," "don't pre-create the directory," etc.) gains an entry: "Don't leave load-bearing body sections empty for any entity kind — the title is a label, not a spec. The kernel's `entity-body-empty` finding (from M-066) will surface the omission for any kind (epic Goal/Scope, milestone Goal/Approach, AC body, gap What's-missing/Why-it-matters, etc.); the design intent is prose detail (description, examples, edge cases, references)." The Don't entry is the concise reminder; the body-prose subsection (AC-1, AC-2, AC-3) is the full explanation. Both surfaces target the same failure mode from different angles — the rule and the prose — to maximize the chance an LLM following the skill registers the requirement across kinds.

