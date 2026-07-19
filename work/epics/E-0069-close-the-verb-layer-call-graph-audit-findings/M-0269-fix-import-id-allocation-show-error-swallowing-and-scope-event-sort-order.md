---
id: M-0269
title: Fix import id allocation, show error swallowing, and scope-event sort order
status: in_progress
parent: E-0069
tdd: required
acs:
    - id: AC-1
      title: import id auto allocates via entity.AllocateID, avoiding sibling-branch ids
      status: open
      tdd_phase: red
    - id: AC-2
      title: show fails loud when history or scope reads error
      status: open
      tdd_phase: red
    - id: AC-3
      title: scope events sort chronologically across timezones in show and render
      status: open
      tdd_phase: red
    - id: AC-4
      title: a policy fails any verb minting entity ids outside entity.AllocateID
      status: open
      tdd_phase: red
---
## Goal

Fix the three correctness bugs the verb-layer audit surfaced: `import`'s auto-id
path bypassing the shared allocator, `show` silently swallowing history/scope
read errors, and the cross-timezone scope-event sort.

## Context

The audit (`docs/initiatives/verb-layer-cleanup.md`, findings F8/F13/F14, each
adversarially verified) filed these as G-0426, G-0427, and G-0428. Each is a
regression against a precedent the kernel already sets elsewhere: `add`
allocates through `entity.AllocateID`; `render` and `aiwf history` fail loud on
the identical git-read error class; the scope tables promise chronological
order. No design decisions needed — each fix converges on the existing seam.

## Acceptance criteria

### AC-1 — import id auto allocates via entity.AllocateID, avoiding sibling-branch ids

### AC-2 — show fails loud when history or scope reads error

### AC-3 — scope events sort chronologically across timezones in show and render

### AC-4 — a policy fails any verb minting entity ids outside entity.AllocateID

## Constraints

- Test-first per AC (`tdd: required`); the failing test lands before the fix.
- `import`'s trunk-collision behavior (already caught via `idsUnique`) must not
  regress while the local/remote-ref exposure closes.
- `show`'s happy-path envelope stays byte-identical; only the error paths gain
  behavior.

## Design notes

- F8 fix inherits `entity.AllocateID`'s existing collision-avoidance tests by
  construction; import-side work is routing, not new allocation logic.
- F14 normalizes to `time.Time` comparison at the one shared sort call so
  `show` and `render` are fixed together.

## Out of scope

- The `--fetch` flag for `importcmd` (parity with `add`) — follow-up if wanted.
- Envelope/dispatcher work on `import` (the FinishVerb milestone owns that).

## Dependencies

- None — first milestone of E-0069.

## References

- G-0426, G-0427, G-0428; `docs/initiatives/verb-layer-cleanup.md` §F8/§F13/§F14.

---

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
