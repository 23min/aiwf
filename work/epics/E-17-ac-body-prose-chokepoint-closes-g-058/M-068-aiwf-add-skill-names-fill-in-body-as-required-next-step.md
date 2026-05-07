---
id: M-068
title: aiwf-add skill names fill-in-body as required next step
status: draft
parent: E-17
tdd: required
---

## Goal

Update the `aiwf-add` skill so an LLM (or human) following it produces non-empty AC bodies by default. Today, the skill describes `aiwf add ac` and stops there — never naming the body-prose follow-up step the design specifies. Result: skills-driven AC creation reproduces the [G-058](../../gaps/G-058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) defect every time. This milestone is the cheapest layer of the epic (pure documentation) but the highest-leverage for changing the default behavior, since most AC creation flows through the skill.

## Approach

Edit `internal/skillsembed/aiwf-add/SKILL.md` (or wherever the skill source lives that gets re-emitted to `.claude/skills/aiwf-add/SKILL.md` on `aiwf init` / `aiwf update`). Add:

- A "After `aiwf add ac`: fill in the body" subsection naming the design intent (cite `acs-and-tdd-plan.md:22`), the recommended shape (one paragraph: pass criteria, edge cases, code references), and the `--body-file` flag from [M-067](M-067-aiwf-add-ac-body-file-flag-for-in-verb-body-scaffolding.md) as the in-verb alternative.
- A "Don't" entry: do not leave AC body sections empty — the title is a label, not a spec; the kernel's `acs-body-empty` finding (from [M-066](M-066-aiwf-check-finding-acs-body-empty.md)) will surface the omission.

The change is verified by the discoverability policy test (`internal/policies/PolicyFindingCodesAreDiscoverable` and the broader skill-doc enumeration from G-021) for the new finding code; the body-prose recommendation is content the policy can't enforce mechanically and ships unblocked.

## Acceptance criteria
