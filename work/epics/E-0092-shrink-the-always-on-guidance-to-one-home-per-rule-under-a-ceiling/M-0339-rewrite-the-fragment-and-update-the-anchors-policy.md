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
      title: Each pinned anchor appears in exactly one of CLAUDE.md and the fragment
      status: open
    - id: AC-2
      title: No fragment rule exceeds the per-rule word cap
      status: open
    - id: AC-3
      title: The operating-anchors policy passes against the rewritten fragment
      status: open
---

## Goal

Rewrite the shipped fragment as one imperative plus one line of why per rule, keeping every pinned anchor, with `CLAUDE.md` and the fragment holding each anchor in exactly one place.

## Closes

- (none)

## Context

The fragment ships to every consumer and is the one surface this epic touches that leaves the repository. M-0336 emptied `CLAUDE.md` of the fragment's rules from the repository side; this milestone tightens the fragment itself. It runs only if M-0338 found no lost effect; otherwise it is cancelled and the epic wraps without it.

## Acceptance criteria

### AC-1 — Each pinned anchor appears in exactly one of CLAUDE.md and the fragment

For each anchor in the operating-anchors ledger, its trigger phrases appear in the fragment source and in no `CLAUDE.md`. **Pass criterion**: the anchors policy, extended in M-0336, passes on both sides together after the rewrite. **Code references**: `internal/policies/m0211_guidance_operating_anchors.go`.

### AC-2 — No fragment rule exceeds the per-rule word cap

No top-level rule in the fragment exceeds the per-rule word cap this milestone sets at its start and records here. **Pass criterion**: a structural test over the fragment's top-level bullets reports one over the cap; on the source it reports none. This is a length check, not a phrase pin, so D-0070 does not retire it. **Code references**: a new test in `internal/policies/`.

### AC-3 — The operating-anchors policy passes against the rewritten fragment

`PolicyM0211GuidanceOperatingAnchors` passes against the rewritten source, the anchor phrases updated in the same commit wherever a rewording required it. **Pass criterion**: the existing policy test is green at the commit that lands the rewrite.

## Constraints

- A shipped surface: no real entity id, path, or lifecycle status in the text; `skill-body-id` fires pre-push.
- No rule is dropped; each keeps one line of why.
- The commit that rewrites the fragment cites E-0092 in its `aiwf-entity` trailer, and the CHANGELOG entry names the epic, so the release-time audit can cover it.
- Runs only on M-0338's judgment of no lost effect.

## Design notes

- The per-rule word cap is a number chosen at start from the longest rule that still carries its why; it is recorded here, not derived from the current text.
- A rule whose why cannot fit the cap is the one to look at hardest; the cap is the constraint, not the wording.

## Surfaces touched

- `internal/skills/embedded-guidance/aiwf-guidance.md`
- `internal/policies/m0211_guidance_operating_anchors.go`
- `CHANGELOG.md`

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
