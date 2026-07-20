---
id: M-0269
title: Fix import id allocation, show error swallowing, and scope-event sort order
status: in_progress
parent: E-0069
tdd: required
acs:
    - id: AC-1
      title: import id auto allocates via entity.AllocateID, avoiding sibling-branch ids
      status: met
      tdd_phase: done
    - id: AC-2
      title: show fails loud when history or scope reads error
      status: met
      tdd_phase: done
    - id: AC-3
      title: scope events sort chronologically across timezones in show and render
      status: open
      tdd_phase: done
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

`aiwf import`'s auto-id allocation routes through `entity.AllocateID` — the
same allocator `aiwf add` uses — so it considers the tree's cross-branch view
(trunk ids plus local-ref and remote-ref ids) in addition to the working tree
and the manifest's own explicit reservations. Importing an `id: auto` entry
on a branch that has not yet merged a sibling branch's freshly-allocated id
of the same kind allocates the next free id instead of re-minting the
sibling's. Trunk-side collisions continue to be caught separately by the
existing `idsUnique`/`import-collision` check; this closes the narrower
local/remote-ref exposure (G-0426).

### AC-2 — show fails loud when history or scope reads error

`aiwf show` propagates an error when reading git history or scope events
fails, exiting with a fail-loud finding rather than silently degrading —
matching the precedent `render` and `aiwf history` already set for the
identical git-read error class (G-0427). The happy-path envelope is
unchanged; only the error paths gain behavior.

### AC-3 — scope events sort chronologically across timezones in show and render

Scope events render in true chronological order regardless of the timezone
offset recorded in each event's timestamp, in both `aiwf show` and
`aiwf render`. The shared sort call normalizes to `time.Time` comparison
instead of comparing timestamp strings, so events recorded across different
timezones interleave correctly (G-0428).

### AC-4 — a policy fails any verb minting entity ids outside entity.AllocateID

An `internal/policies` check statically fails CI if any verb package mints
an entity id through a path other than `entity.AllocateID`, preventing a
regression of the class of bug AC-1 closes — a verb hand-rolling its own
id-numbering logic instead of routing through the shared allocator.

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

### AC-1 — import id auto allocates via entity.AllocateID

Swapped `import.go`'s hand-rolled `computeHighestPerKind`/`parseIDInt`/
`idPrefix`/`formatID` for `entity.AllocateID(k, allocated, t.AllocationIDs())`,
threading in-manifest explicit reservations as synthetic entities so the
allocator still sees them · commit `35b0d3ec` · tests 1/1.

### AC-2 — show fails loud when history or scope reads error

Widened `BuildShowView`/`BuildCompositeShowView` to return
`(ShowView, bool, error)` and propagate a history/scope read failure to
`Run`, which now exits `ExitInternal` with a "reading history"/"reading
scopes" message instead of silently leaving the fields empty. The
scopes-read branch is `//coverage:ignore`d in both functions: it can
never fire in practice because the direct history read immediately
above it uses the identical `git log`-from-HEAD primitive and always
fails first · commit `f3e7a0ee` · tests 2/2.

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
