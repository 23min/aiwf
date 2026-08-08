---
id: M-0307
title: Route the body-scaffold instruction through the verb covering every kind
status: draft
parent: E-0081
tdd: required
acs:
    - id: AC-1
      title: The always-on guidance names a scaffold route resolving for all six kinds
      status: open
    - id: AC-2
      title: No shipped surface cites a per-kind template path that does not exist
      status: open
---

## Goal

Point the always-on guidance's body-scaffold instruction at a route that resolves for
every kind, so an assistant following it finds something for all six rather than two.

## Context

The always-on guidance fragment materializes into every consumer repo and sits in an
assistant's context each turn. It instructs the assistant to create an entity with
`aiwf add`, filling the body from a per-kind template path, with the parenthetical
remedy of running `aiwf update` if the file is absent.

Substituted against what ships, the path resolves for two of six kinds. Epic and
milestone miss on a filename suffix; gap and contract miss because no such file exists.
The remedy compounds it: `aiwf update` will never produce the four missing files, so
the instruction reads as "your tree is stale" where the truth is "this file was never
going to exist" — and the reader most likely to follow it is an assistant with no prior
about what the directory holds.

All four born-complete kinds refuse a bare scaffold at creation — its sections are
empty, which is what the gate reads. For adr and decision the named template exists,
so the instruction still leads somewhere. For gap and contract it names a file that
was never going to exist, and the only other route is refused. The failure is silent
at every step: no check fires, `aiwf doctor` reports the rituals materialized, and the
consumer sees a healthy tree. The assistant either invents a structure or skips the
body, and nothing catches either.

`aiwf template <kind>` already prints the per-kind scaffold for all six kinds.

## Approach

Withdraw the promise rather than fulfil it. Writing the four missing templates would
mint two new artifacts and a standing obligation to keep six in step forever; changing
the instruction to name the verb costs one line and covers every kind today.

It also removes a hop that the kernel's own rule says should not be there. The current
route depends on an assistant reading a file it was told about — correctness resting on
LLM behavior, which the kernel rule names directly. A verb the assistant runs is
mechanical.

The always-on fragment is the only surface changing. The planning rituals' separate
instruction to fill from the *rich* template is a different instruction about a
different artifact, and it stays.

## Acceptance criteria

### AC-1 — The always-on guidance names a scaffold route resolving for all six kinds

The body-scaffold instruction in the embedded guidance source names a route that
produces a scaffold for every kind. The test enumerates the kinds from the kernel's own
kind set rather than a literal list, exercises the named route for each, and fails if
any kind yields nothing. A kind added to the kernel later fails this test until the
route covers it.

### AC-2 — No shipped surface cites a per-kind template path that does not exist

No surface materialized into a consumer repo instructs a reader to read a per-kind
template path that does not resolve for the kind being created. The assertion covers
the parenthetical remedy as well as the path itself: no shipped surface offers
`aiwf update` as the fix for a file that command does not produce.

## Constraints

- The guidance-operating-anchor policy asserts a curated set of anchors stays present
  in the embedded guidance source; this milestone's edit keeps that set intact or
  updates it deliberately.
- The planning rituals' instruction to fill from the rich prose template is a separate
  instruction about a separate artifact and is not changed here.
- D-0015 is preserved: the four existing prose templates keep materializing where they
  do. No template file is added or removed by this milestone.
- Every `SKILL.md` edit under the embedded rituals lands with its referencing
  structural test, per the repo's backstop policy.

## Design notes

Not writing the four missing templates is the load-bearing choice. A per-kind template
is a mandate that costs once per kind forever; the verb costs once. Gap and contract
bodies are short by design — two sections each — which is why nobody has missed a rich
template for them.

## Surfaces touched

`internal/skills/embedded-guidance/aiwf-guidance.md`, `internal/policies/`.

## Out of scope

- Adding prose templates for the kinds that lack one.
- `aiwf doctor`'s byte-check of materialized ritual and guidance artifacts (G-0504).
- The prose templates' own content, which M-0306 reconciles.

## Dependencies

None. Parallel with M-0305 and M-0306 throughout.

## References

- G-0541 — the guidance's template path resolves for two of six kinds
- D-0015 — ritual templates materialize to the templates dir
- E-0081 — parent epic

## Work log

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
