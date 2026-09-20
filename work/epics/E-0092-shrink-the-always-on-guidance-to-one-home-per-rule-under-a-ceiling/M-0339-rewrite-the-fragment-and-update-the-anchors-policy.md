---
id: M-0339
title: Rewrite the fragment and update the anchors policy
status: draft
parent: E-0092
depends_on:
    - M-0338
tdd: required
acs:
    - id: AC-1
      title: Operating anchors have one authored source and valid host renderings
      status: open
    - id: AC-2
      title: No fragment rule exceeds the per-rule word cap
      status: open
    - id: AC-3
      title: The operating-anchors policy passes against the rewritten fragment
      status: open
---

## Goal

Shorten the shared operating fragment while preserving its obligations and routing in both Claude and Codex renderings. Each rule retains one authored source and a concise reason.

## Closes

- (none)

## Context

M-0336 removes independently authored copies from repository development guidance. This milestone changes the shared operating source only after M-0338 records no lost effect for both hosts. Rendered copies in host entry points are expected outputs, not additional authorship.

## Acceptance criteria

### AC-1 — Operating anchors have one authored source and valid host renderings

Each operating anchor has one canonical source in the shared fragment and no independent copy in either host's development guidance. The rendered Claude and Codex artifacts must contain the expected source-derived instruction. **Pass criterion**: ownership-aware anchor checks reject an extra authored copy while accepting both expected renderings.

### AC-2 — No fragment rule exceeds the per-rule word cap

No top-level rule in the fragment exceeds the per-rule word cap this milestone sets at its start and records here. **Pass criterion**: a structural test over the fragment's top-level bullets reports one over the cap; on the source it reports none. This is a length check, not a phrase pin, so D-0070 does not retire it. **Code references**: a new test in `internal/policies/`.

### AC-3 — The operating-anchors policy passes against the rewritten fragment

The operating-anchors policy and both host renderings pass against the rewritten source. Update source-dependent anchors together, check references and generated ownership, and repeat the affected loading/behavior observations for both hosts after rendering. A source-only green check does not establish host delivery.

## Constraints

- A shipped surface: no real entity id, path, or lifecycle status in the text; `skill-body-id` fires pre-push.
- No rule is dropped; each keeps one line of why.
- The commit that rewrites the fragment cites E-0092 in its `aiwf-entity` trailer, and the CHANGELOG entry names the epic, so the release-time audit can cover it.
- Runs only on M-0338's judgment of no lost effect.

## Design notes

- The per-rule word cap is a number chosen at start from the longest rule that still carries its why; it is recorded here, not derived from the current text.
- A rule whose why cannot fit the cap is the one to look at hardest; the cap is the constraint, not the wording.

## Surfaces touched

- Shared operating guidance and necessary host fragments.
- The operating-anchors policy and rendering checks.
- Generated artifacts refreshed through their owner, and CHANGELOG.md.

## Out of scope

- The skill and ritual bodies.
- Any consumer's `CLAUDE.md`.

## Dependencies

- M-0338 — and its result

## Coverage notes

- (none)

## References

- ADR-0018 — the fragment and its import
- D-0070 — what may be pinned in shipped prose

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
