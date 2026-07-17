---
id: G-0317
title: Skill-edit backstop checks test-references-path, not asserts-changed-section
status: open
priority: medium
discovered_in: M-0196
---
## What's missing

The skill-edit structural-test backstop policy (`internal/policies/skill_edit_structural_test_backstop.go`, M-0196 / G-0220) is **v1 granularity: file-existence + skill-reference**. It fires when a commit modifies an embedded-rituals `SKILL.md` whose repo-relative path appears in *no* `internal/policies/*_test.go` source. It does **not** verify that the referencing test actually *asserts the changed section* of the skill body.

So a residual false-negative remains: a `SKILL.md` path referenced by a *non-asserting* test (e.g. `skill_coverage_test.go`, which checks frontmatter `name:`/`description:` rather than body content) satisfies the backstop without pinning any prescribed content. An edit to that skill's body ships green even though no test exercises the new prescription.

## Why it matters

The whole point of G-0220's backstop is "a skill edit cannot ship without a mechanical test of its prescribed content." v1 catches the dominant, demonstrated failure mode (the M-0160 case: a `SKILL.md` shipped with *zero* structural test). It does not catch the weaker "test exists but is stale / asserts the wrong thing" case. This mirrors the diff-scoped coverage gate's own v1 limitation (statement coverage, not branch correlation — G-0067) and is disclosed in the engine doc-comment and CLAUDE.md §"Ritual content authoring".

## Proposed fix shape

Strengthen the backstop from "the edited skill's path is referenced somewhere in the policy tests" to "a policy test references the edited skill's path AND asserts content from the section(s) the commit changed." Sketch:

- For each changed `SKILL.md`, compute the changed section headings from the diff hunks (the `## ` / `### ` headings whose bodies the commit touched).
- Require a referencing test to contain a literal of (or assertion against) each changed heading's text — or a stronger structural pairing (e.g. a per-skill registry mapping section → asserting test function).
- Fire when a changed section has no asserting test.

This is strictly stronger and correspondingly more complex; weigh it against real recurrence of the stale-test failure mode before building it (YAGNI — the v1 catches the case that has actually bitten).

## Discovered in

M-0196 — surfaced as the explicit v1/v2 boundary while implementing the G-0220 backstop. The independent reviewer confirmed the residual is in-spec and already disclosed in the engine doc-comment and CLAUDE.md.
