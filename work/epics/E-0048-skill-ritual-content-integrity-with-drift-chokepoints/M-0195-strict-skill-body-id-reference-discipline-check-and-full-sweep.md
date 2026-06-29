---
id: M-0195
title: Strict skill-body id-reference discipline, check, and full sweep
status: draft
parent: E-0048
depends_on:
    - M-0203
tdd: required
---
## Goal

Make skill-body id-reference discipline strict and mechanical. Shipped skill
bodies (`internal/skills/embedded/**` verb skills and
`internal/skills/embedded-rituals/**` rituals) must cite no real entity id,
filesystem path, or inline lifecycle status: they ship to consumer repos where
aiwf's own ids are meaningless, and they rot as entities change status, archive,
or rewidth. Illustrative content uses canonical-shape placeholders
(`<prefix>-NNNN`) or shape-descriptions; a markdown link to a design or ADR doc
is the one carve-out.

A new pre-push `aiwf check` rule over the embedded skill tree makes the
discipline mechanical — the standing-rule prose is the convenient version, per
the kernel principle "framework correctness must not depend on LLM behavior".
The rule is the mirror image of the `body-prose-id` check (G-0184): there a real
id is required and a placeholder is the defect; here a real id is the defect and
the canonical placeholder is correct.

Sequence: normalize all placeholders to canonical width first (the precondition
that lets the check allow placeholders while flagging real ids), then sweep
every skill body, then land the check green over the swept tree. This is a
foundation milestone — the content milestones (verb-skill corrections, ritual
honesty, prose polish, planning-ritual body-fill) rebase onto the swept bodies,
so the id-hygiene rewrite lands once, first.

Source: G-0299. Parent epic E-0048; sequenced after foundation epic E-0050
(done), whose generalized declared-sequence gate governs this milestone's wrap.

## Acceptance criteria
