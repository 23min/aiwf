---
id: E-0048
title: Skill & ritual content integrity (with drift chokepoints)
status: proposed
---
# Skill & ritual content integrity (with drift chokepoints)

## Goal

Every shipped skill and ritual body is accurate, consistent, and self-contained,
and mechanical chokepoints prevent future drift. The reader of any `aiwf-*` verb
skill or `aiwfx-*` / `wf-*` ritual gets correct guidance, and a future edit that
reintroduces drift fails CI.

## Context

A 2026-06-28 audit of CLAUDE.md against the embedded skills found ~16 content
defects — a wrong AC status set and undocumented finding codes in the check skill,
a dead recipe path, an invalid `--kind` value, stale entity references and inline
statuses, off-convention id placeholders, untrailered body-fill commits — plus the
absence of any mechanical backstop that keeps skill content correct over time. The
findings are filed as gaps. This epic addresses the content-correctness subset and
adds the chokepoints; the gate-model subset is foundation epic E-0050 and the
commit/TDD-model subset is lifecycle epic E-0049.

## Scope

### In scope

- **Mechanical chokepoints over skill bodies:** strict id-reference rule + check +
  full body sweep + placeholder normalization (G-0299); skill-edit→test backstop
  policy (G-0220); finding-code documented-superset chokepoint (G-0283).
- **Skill content correctness:** verb-skill factual corrections (G-0301);
  wf-tdd-cycle / wf-review-code honesty (G-0297); wf-doc-lint reframe + gitleaks
  (G-0294); descriptions + whiteboard + prose polish (G-0298); planning-ritual
  body-fill routing through edit-body (G-0300); plan-milestones Next-step (G-0248);
  devcontainer onboarding banner (G-0279).

### Out of scope

- The gate-discipline model (foundation epic E-0050) and the commit/TDD model
  (lifecycle epic E-0049).
- The tier-C ritual-design gaps (separate triage).
- New kernel verbs or features beyond the three chokepoint policies.

## Constraints

- The standing rule the id chokepoint enforces: shipped skill bodies cite no real
  entity id, filesystem path, or inline lifecycle status; placeholders are
  canonical `<prefix>-NNNN`; a doc-link to a design/ADR doc is the one carve-out.
- Foundation milestones (the id-sweep and the edit→test policy) land before the
  content milestones, so the latter rebase onto swept bodies and are forced to
  ship structural tests.
- Skill edits are authored in the embedded snapshot per ADR-0016.
- Sequence after foundation epic E-0050: this epic's milestone wraps run under the
  generalized declared-sequence gate E-0050 delivers, so every wrap here is gated
  correctly.

## Success criteria

- [ ] Every defect named in the gaps under *Milestones* is fixed and verified.
- [ ] Three CI-green chokepoints exist: skill-body id-reference check, finding-code
      documented-superset, and the skill-edit→structural-test backstop.
- [ ] No shipped skill body references a real entity id, path, or inline status
      (placeholders only; ADR doc-links allowed).

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the id-reference check live in `internal/check` or `internal/policies`? | no | decided in the G-0299 milestone |

## Milestones

<!-- execution order; ids allocated at plan-milestones time -->

1. Strict id-reference discipline + full body sweep + placeholder normalization (G-0299) — foundation.
2. Skill-edit→structural-test backstop policy (G-0220) — foundation.
3. Finding-code docs + documented-superset chokepoint (G-0283).
4. Verb-skill factual corrections (G-0301).
5. wf-tdd-cycle / wf-review-code honesty + wf-doc-lint reframe (G-0297, G-0294).
6. Descriptions + whiteboard + prose polish (G-0298).
7. Planning-ritual body-fill routing + plan-milestones Next-step (G-0300, G-0248).
8. Devcontainer onboarding banner (G-0279).
