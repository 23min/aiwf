---
id: M-068
title: aiwf-add skill names fill-in-body as required next step
status: draft
parent: E-17
tdd: required
acs:
    - id: AC-1
      title: Skill names fill-in-body as required next step
      status: open
      tdd_phase: red
    - id: AC-2
      title: Skill cites the design intent (acs-and-tdd-plan.md)
      status: open
      tdd_phase: red
    - id: AC-3
      title: Skill recommends the body shape (paragraph, key contents)
      status: open
      tdd_phase: red
    - id: AC-4
      title: Skill names --body-file as in-verb alternative
      status: open
      tdd_phase: red
    - id: AC-5
      title: Skill carries Don't entry against empty AC bodies
      status: open
      tdd_phase: red
---

## Goal

Update the `aiwf-add` skill so an LLM (or human) following it produces non-empty AC bodies by default. Today, the skill describes `aiwf add ac` and stops there — never naming the body-prose follow-up step the design specifies. Result: skills-driven AC creation reproduces the [G-058](../../gaps/G-058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) defect every time. This milestone is the cheapest layer of the epic (pure documentation) but the highest-leverage for changing the default behavior, since most AC creation flows through the skill.

## Approach

Edit `internal/skillsembed/aiwf-add/SKILL.md` (or wherever the skill source lives that gets re-emitted to `.claude/skills/aiwf-add/SKILL.md` on `aiwf init` / `aiwf update`). Add:

- A "After `aiwf add ac`: fill in the body" subsection naming the design intent (cite `acs-and-tdd-plan.md:22`), the recommended shape (one paragraph: pass criteria, edge cases, code references), and the `--body-file` flag from [M-067](M-067-aiwf-add-ac-body-file-flag-for-in-verb-body-scaffolding.md) as the in-verb alternative.
- A "Don't" entry: do not leave AC body sections empty — the title is a label, not a spec; the kernel's `acs-body-empty` finding (from [M-066](M-066-aiwf-check-finding-acs-body-empty.md)) will surface the omission.

The change is verified by the discoverability policy test (`internal/policies/PolicyFindingCodesAreDiscoverable` and the broader skill-doc enumeration from G-021) for the new finding code; the body-prose recommendation is content the policy can't enforce mechanically and ships unblocked.

## Acceptance criteria

### AC-1 — Skill names fill-in-body as required next step

The `aiwf-add` skill source gains a subsection — placed immediately after the `aiwf add ac` example — titled "After `aiwf add ac`: fill in the body" (or equivalent). The subsection states unambiguously that scaffolding the AC frontmatter is step 1 of 2; writing the body prose is step 2 and is required, not optional. The "What aiwf does" numbered list (currently 5 steps) gains a step 6: "appended `### AC-N — <title>` body heading is empty by design — fill it in before declaring the AC done." Verified by reading the rendered skill in a tempdir post-`aiwf init`.

### AC-2 — Skill cites the design intent (acs-and-tdd-plan.md)

The body-prose subsection cites `docs/pocv3/plans/acs-and-tdd-plan.md:22` and `docs/pocv3/design/design-decisions.md:139` as the spec source. The citation is a plain markdown link (paths are stable; if they move the link rots into a 404 in the rendered skill, which is a visible signal). Rationale for the citation: an LLM (or human) following the skill should be able to trace the rule back to the design without grepping the codebase. Same channel discipline as the rest of the kernel's discoverability work.

### AC-3 — Skill recommends the body shape (paragraph, key contents)

The subsection prescribes one paragraph per AC (not an essay, not a one-liner) covering: (a) what passing concretely looks like — the assertable claim; (b) edge cases the test must cover; (c) forward references to the code path or test file. Includes a short example block showing a well-formed AC body so the operator has a concrete shape to copy. The recommendation is advisory, not enforced — the kernel rule (M-066) checks presence, not structure — but the skill is the chokepoint for shaping default behavior, so the recommendation matters.

### AC-4 — Skill names --body-file as in-verb alternative

The body-prose subsection mentions `--body-file` from M-067 as the in-verb alternative to a follow-up edit pass — for cases where the operator already has the prose drafted (e.g. mining from a design doc or a prior conversation), `--body-file` lands the body in the same atomic commit as the AC. The cross-reference is two-way: M-067 AC-8 names the skill change, and this AC names the verb. Both surfaces describe the same flag with the same semantics; no drift.

### AC-5 — Skill carries Don't entry against empty AC bodies

The skill's "Don't" section (currently lists "don't hand-edit frontmatter," "don't pre-create the directory," etc.) gains an entry: "Don't leave AC body sections empty — the title is a label, not a spec. The kernel's `acs-body-empty` finding (from M-066) will surface the omission; the design intent is prose detail (description, examples, edge cases, references)." The Don't entry is the concise reminder; the body-prose subsection (AC-1, AC-2, AC-3) is the full explanation. Both surfaces target the same failure mode from different angles — the rule and the prose — to maximize the chance an LLM following the skill registers the requirement.

