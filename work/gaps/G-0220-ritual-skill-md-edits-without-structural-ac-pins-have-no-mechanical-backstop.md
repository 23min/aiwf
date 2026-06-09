---
id: G-0220
title: Ritual SKILL.md edits without structural AC pins have no mechanical backstop
status: open
discovered_in: M-0160
---

## What's missing

Embedded ritual SKILL.md content is **shippable** — consumers running `aiwf init` / `aiwf update` materialize the bytes from the embedded snapshot into their `.claude/skills/aiwfx-*/`. Per CLAUDE.md §"Ritual content authoring":

> "When a milestone's deliverable is ritual `SKILL.md` content, the **authoring location is the embedded snapshot itself**; AC tests under `internal/policies/` assert content claims against the embedded bytes via the path constants (`aiwfxWhiteboardFixturePath`, etc. — each points at the embedded snapshot path per G-0182)."

The discipline is clear: SKILL.md edits should be backed by structural tests under `internal/policies/` that assert the prescribed content is in the right markdown section. The tests act as the mechanical backstop — a future edit that silently removes or moves the prescription fails the test and CI blocks.

**The discipline is not mechanically enforced.** A SKILL.md edit can land:
- Without a corresponding AC pinning the new content
- Without a corresponding structural test under `internal/policies/`
- Without the originating milestone owning the edit as deliverable
- Without any chokepoint that says "this skill content needs a structural test"

The kernel has no rule that fires "skill SKILL.md changed in a commit; structural test under internal/policies/ for that skill must also exist in the same change-set" (or any equivalent shape).

## Why it matters

Per CLAUDE.md "Framework correctness must not depend on the LLM's behavior": the discipline relies on the operator (human or LLM) remembering to write the structural test alongside the skill edit. Operators forget. The kernel knows what edits would deserve a structural test (any change under `internal/skills/embedded-rituals/.../SKILL.md`); failing to refuse the commit when one is absent is exactly the kind of chokepoint that should be mechanical, not vigilance-dependent.

Concrete instance: during M-0160 wrap, the operator (me) made a skill edit at commit `5cf007f5` that:
- Modified `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md` step 11 with new prescriptive content (the wrap-milestone trailer prescription, mirroring aiwfx-wrap-epic — see [G-0219](G-0219-aiwfx-wrap-milestone-skill-md-asymmetric-missing-wrap-milestone-trailer-step.md))
- Was on the `epic/E-0030-branch-model-chokepoint` branch (E-0030 scope is "Branch model chokepoint"; ritual content authoring is not E-0030's scope per CLAUDE.md *Slipping unrelated code into the wrap commit* anti-pattern)
- Had NO milestone AC pinning the new content
- Had NO structural test under `internal/policies/` asserting the content is present in the right section
- Shipped to downstream consumers on next `aiwf update`

The kernel said nothing — pre-commit shape-only check passed; pre-push full check passed; the commit landed. The user (correctly) flagged the discipline failure manually:

> "ehm, did you modify skills here now? They are a shippable item. What about provenance, was there a gap? How did you document?"

The kernel's job is to surface that. It didn't.

## Proposed fix shape

**Primary: structural drift policy under `internal/policies/`.** Detect when a commit touches `internal/skills/embedded-rituals/**/SKILL.md` and assert that the same change-set (or an existing one in the repo) contains a structural test under `internal/policies/<skill-name>_test.go` (or similar) that exercises the changed section. Sketch:

- For each modified SKILL.md path in the commit's diff, derive the expected policy-test path (`aiwfx-wrap-milestone/SKILL.md` → `internal/policies/aiwfx_wrap_milestone_test.go`).
- Verify the policy-test file exists AND references the modified section (e.g., a substring match on the section heading text, or — better — a structural walk).
- Fire a finding when the policy-test file is missing OR doesn't reference the changed section.

This is a mechanical chokepoint: it can't catch every semantic drift (a test that exists but is wrong is still wrong), but it gates the obvious case (SKILL.md prescribes new content; no test asserts it).

**Secondary: pre-commit hook integration.** The policy check runs at `aiwf check` time; integrating it into pre-commit (extending the shape-only mode or escalating its severity at pre-push) makes the gate prescriptive rather than informational.

**Tertiary: skill-authoring discipline pin in CLAUDE.md.** Explicit note in §"Ritual content authoring": "Skill edits MUST land alongside a structural test under `internal/policies/`. The mechanical chokepoint is at G-0220's policy; until that lands, this is operator vigilance."

**Optional generalization**: the same shape applies to other shippable content (agent SKILL.md, embedded templates, etc.). A general policy that maps "shippable surface" → "required structural backstop test" closes the broader class.

## Status

**Interim work landed in M-0160 wrap cycle**: commit `5cf007f5` made the skill edit; this gap (G-0220) records the missing-AC-pin discipline failure; a structural test (the `TestAiwfxWrapMilestone_*` shape mirroring `aiwfx_wrap_epic_test.go`) lands in the same wrap cycle as the `C-option` follow-up to the user's discipline call.

## Test surface

When the structural drift policy lands:
- Fixture: a commit that modifies a SKILL.md without a corresponding policy-test → policy fires.
- Fixture: a commit that modifies a SKILL.md AND adds a policy-test exercising the changed section → policy silent.
- Fixture: a commit that modifies a SKILL.md AND the policy-test file exists but doesn't reference the changed section → policy fires.
- Sabotage-verifiable: revert the policy's "is the modified section referenced by the test" check → SKILL.md edits without coverage pass; the discriminating test fires.

## Workaround

Until the structural drift policy lands, the discipline is operator awareness:

- **Every SKILL.md edit needs a paired structural test** under `internal/policies/`. Use `aiwfx_wrap_epic_test.go` as the canonical template (the heading-hierarchy walk + scoped substring assertions).
- **The test should land in the same change-set as the skill edit**, ideally on a branch whose owning milestone has an AC pinning the new content. If the edit is small/urgent (like the M-0160 wrap cycle fix at `5cf007f5`), at minimum file a gap recording the discipline-skipping AND land the structural test in the same wrap-cycle's commit chain.
- **Plain code commits to embedded ritual content are an anti-pattern**: they ship behavior changes without the mechanical backstop the kernel design requires.

## Closing this gap

When the impl lands:
- Structural drift policy under `internal/policies/` named for the rule (e.g., `policy_skill_md_edits_have_structural_tests.go`).
- Tests above land alongside the implementation.
- CLAUDE.md §"Ritual content authoring" updated to remove the interim "operator vigilance" note.
- Promote G-0220 to `addressed` with `--by M-NNNN`.

## Discovered in

M-0160 — surfaced after the operator (me) made an undisciplined SKILL.md edit at commit `5cf007f5` (fixing the G-0219 asymmetry) without filing a gap first, without an AC pinning the content, and without a structural test asserting the new prescription is in the right section. The user manually flagged the discipline gap — exactly the role the kernel's chokepoint should play.
