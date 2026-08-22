---
id: E-0088
title: Make every path-changing verb repair the links it breaks
status: proposed
---

## Goal

Close the gap between ADR-0033's commitment and the code that implements it, so
that moving an entity leaves no broken markdown link anywhere in the entity set
the moving verb owns — in either direction.

## Context

[ADR-0033](../../../docs/adr/ADR-0033-entity-path-links-are-first-class-and-rewritten-on-move.md)
is `accepted` and E-0063 built the shared link-region primitive it describes.
Four of the five verbs that emit an `OpMove` route through it: `archive`,
`reallocate`, `rename`, `retitle`. `move` routes through nothing — it computes a
destination under the target epic's directory and rewrites no links at all. That
is the ADR's first bullet unmet, and it is the entity-truth audit's only
`contradicted-by-code` verdict against this ADR.

A second divergence is narrower than it looks. ADR-0033 commits the primitive to
rewriting links "in entity bodies **that point at it**" — inbound only. A moved
file's *own* relative links are outside the commitment, so when a file moves into
an `archive/` subdirectory its bare-filename links resolve against the new
directory and break. This is not the ADR being violated; it is the ADR not
reaching. Observed on 2026-08-19: sweeping ADR-0003 into `docs/adr/archive/`
broke five of its outbound links and two inbound links held by an already-archived
sibling, caught by the `link-check` workflow rather than by any verb.

Mutation testing named the same subsystem independently and without knowledge of
either finding. Against a kernel-wide baseline of 7.7 surviving mutants per
thousand lines, `internal/verb/linkregion.go` measures 70.4 — the highest density
in the kernel — with `linkrewrite.go` at 30.9, `pathrewrite.go` at 21.1 and
`archive.go` at 19.1. The survivors are almost entirely conditional-boundary and
conditional-negation mutants, the signature of a happy path under test and edges
that are not.

Three signals, arrived at by unrelated means, name one subsystem. That agreement
is what makes this worth an epic now rather than a gap later.

## Scope

- **Route `move` through the shared primitive**, so every verb emitting an
  `OpMove` repairs the inbound links to the entity it moved.
- **Rewrite a moved entity's outbound links**, so a file that changes directory
  keeps its own relative links resolving. This extends ADR-0033's reach and
  therefore needs a recorded decision before it is built.
- **Test the primitive's edges**, using the surviving mutants in the four named
  files as the work list and the measure.
- **Settle whether ADR-0033's `docs/` delegation is real.** Its second bullet
  routes non-entity narrative to an advisory doc-lint check rather than to the
  movers. G-0478 and G-0439 both measure that narrative rotting repeatedly and
  being found by hand. Either the delegation works and those gaps are mis-framed,
  or the ADR delegates to something that does not fire — and the finding is which.

## Out of scope

- **Rewriting links inside `docs/`.** ADR-0033's second bullet is explicit that a
  verb commit must not reach outside the entity set it owns. G-0478 and G-0439
  own that half and each carries its own resolution shape; this epic verifies the
  boundary rather than crossing it.
- **A pre-push check rule for link integrity.** ADR-0033's third bullet places
  enforcement at move time and declines to grow the pre-push chokepoint's cost.
- **Redirect stubs or tombstones.** ADR-0033's fourth bullet preserves ADR-0004's
  move-based archive; a vacated path stays vacated.
- **Bare-id citation policy.** Unchanged by this epic.

## Constraints

- **ADR-0033 is the specification.** A claim in this epic that the ADR does not
  carry is a defect in one of the two, and the outbound extension is the one
  place this epic knowingly reaches past it.
- **One shared primitive.** Fixing `move` means routing it through the existing
  link-region machinery, not adding a second implementation beside it.
- **Prose, inline code, fenced code, URLs and external paths stay untouched** —
  the primitive's existing discrimination is a property to preserve, not to
  re-derive.
- **Every milestone that claims a survivor is dead re-runs the measurement.**
  A survivor count that falls without a command that produced it is not evidence.
- **Some survivors are equivalent mutants** and cannot be killed. Naming one as
  equivalent requires the argument for why, not a shrug.

## Success criteria

Observable at epic close. Milestone acceptance criteria carry the mechanical bar.

- [ ] Every verb that emits an `OpMove` routes through the shared link-region
      primitive, with a test per verb that fails if the routing is removed.
- [ ] An entity moved into or out of an `archive/` subdirectory keeps its own
      outbound relative links resolving, demonstrated end to end rather than by
      inspection.
- [ ] Surviving-mutant density across the files named in *Context* is at or below
      the kernel-wide baseline, measured by the same command that established it.
- [ ] Each survivor that remains is recorded as equivalent with its argument, or
      as tracked work — none is left unexplained.
- [ ] ADR-0033's `docs/` delegation is either shown to fire, or recorded as not
      firing with the consequence for G-0478 and G-0439 stated.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does outbound rewriting extend ADR-0033, or need its own decision record? | no | Settled in the milestone that builds it, before the code lands. |
| Does the advisory doc-lint check ADR-0033 delegates to actually run over `docs/`? | no | Measured by the verification milestone; the answer re-frames or confirms G-0478 and G-0439. |
| Is a moved file's outbound rewrite safe against the atomic-write path, which today does not edit the moved file's own content? | yes, for the outbound milestone | Answered by that milestone's first acceptance criterion. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Outbound rewriting makes the mover edit the file it is moving, which the write path does not do today | high | The atomic-write question is the outbound milestone's first criterion, answered before any rewrite lands. |
| Survivors turn out to be largely equivalent mutants, so the density target is unreachable | med | The criterion admits equivalence with an argument; an unreachable number is reported, not engineered around. |
| The `docs/` verification finds the delegation fictional, widening scope | med | The finding is recorded and routed to G-0478 and G-0439, which already own that half. This epic does not absorb it. |

## Milestones

Sequenced so the decision lands before the work depending on it, and the
measurement work lands last, when there is something to measure. Ids are
assigned when the milestones are planned.

- Route `move` through the shared primitive — the ADR-0033 violation, closed
  against a specification that already exists. Depends on nothing.
- Decide and implement outbound link rewriting, including the atomic-write
  question. Depends on the first.
- Test the primitive's edges against the surviving-mutant work list. Depends on
  the second, so that the new outbound paths are measured with the rest.
- Verify or falsify ADR-0033's `docs/` delegation, and route the answer to the
  gaps that own that half. Depends on nothing; sequenced last because its output
  is a finding rather than a change.

## References

- ADR-0033 — the specification: path-links are first-class and rewritten on move
- ADR-0004 — the move-based archive convention this preserves
- E-0063 — built the shared link-region primitive
- G-0478, G-0439 — the `docs/` half, out of scope here and verified by the last
  milestone
- `internal/verb/linkregion.go`, `linkrewrite.go`, `pathrewrite.go`,
  `archive.go`, `move.go` — the subsystem
