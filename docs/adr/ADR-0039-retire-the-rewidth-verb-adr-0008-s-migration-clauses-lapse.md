---
id: ADR-0039
title: Retire the rewidth verb; ADR-0008's migration clauses lapse
status: proposed
---
## Context

ADR-0008 set canonical 4-digit width for every kernel id, kept parser
tolerance for narrower legacy widths on input, and shipped `aiwf rewidth`
as the one-shot verb that migrated an existing tree. It anticipated its
own end: the verb's cost is listed there as permanent CLI surface for a
one-shot ritual, with the stated sunset being removal in a future
version once known consumers have migrated. That has happened.

Two facts make the verb not merely unnecessary but inoperable.

Canonical width is now the only legal width for an active entity, which
`entity-id-narrow-width` reports at error severity. The verb ran an
`aiwf check` preflight that refuses to `--apply` while error-severity
findings exist — so it would refuse on exactly the trees it existed to
convert. A migrator whose own gate rejects its input space has no
remaining call site.

The drift finding's shape collapsed with it. Its uniform-versus-mixed
classifier existed only to stay silent on a uniform-narrow tree, because
that meant *pre-migration*; a tree in that state is now simply wrong.

## Decision

Retire the verb: `internal/verb/rewidth.go`, `internal/cli/rewidth/`,
`padToCanonical`, and the command's registration are deleted.

**Supersede ADR-0008 clause-wise, not wholly.** Its status stays
`accepted`. A whole supersession would say the canonical-width policy is
no longer authoritative, which is false — four of its runtime claims are
live and load-bearing, and stranding them would leave a reader unable to
tell which parts of the original still bind.

### Clauses that lapse

- **§"Migration — `aiwf rewidth` verb"** — the verb, its flags, its
  single-commit-per-`--apply` contract, and its `aiwf-verb: rewidth`
  trailer. The trailer value is no longer a legal one; historical
  commits carrying it stay readable, since `aiwf history` reads what is
  written rather than validating it.
- **§"Reversal — what verb undoes rewidth?"** — moot with the verb gone.
- **§"Drift control"**, in part: the uniform-narrow-is-silent rule, the
  mixed-state classifier, warning severity, and the remediation that
  named the verb. The finding itself stands, restated below.
- The consequence claiming a tested, distributed migration path shipped
  to every consumer via `go install`; the consequence describing the
  transient window in which filenames stay narrow until the consumer
  runs the verb; and the consequence accepting permanent CLI surface for
  a one-shot ritual, whose stated sunset this ADR executes.

### Clauses that stand, unchanged

- **§"Parser tolerance"** — parsers accept narrower legacy widths on
  input.
- **§"Allocator behavior"** — allocation emits canonical width.
- **§"Renderer canonicalization"** — renderers always emit canonical
  width.
- **§"Drift control"**, in that a finding polices width drift at all,
  and that archive entries are excluded from it.

### The finding, restated

A narrow-width id anywhere in the active tree is an error, one finding
per entity, whether or not canonical ids sit alongside it. The
remediation is undoing the hand-edit or file move that produced it. No
verb widens an id in place, and `aiwf reallocate` is not a substitute —
it assigns a different number rather than the same one at canonical
width.

## Consequences

- **Narrow read tolerance is permanent, and its justification changes.**
  Under ADR-0008 it was a transitional courtesy to unmigrated trees.
  Nothing can widen an id in place now, so a repo that archived entities
  before adopting canonical width keeps narrow ids under
  `<kind>/archive/` for good. Tolerance is what keeps live
  cross-references into those entities resolving — a standing property
  of the input space, not a legacy concession.
- **The archive exclusion is permanent for the same reason.** A rule
  that fired on archived narrow ids would be unsatisfiable.
- **There is no reversal verb, deliberately.** Nothing widens an id in
  place; a narrow id in the active tree is a defect to undo at its
  source, not a state to migrate out of.
- **A tree that never migrated has no in-kernel path forward.** The
  known consumer population had migrated before this landed, and a tree
  that has not is out of scope by decision rather than by oversight.

## Alternatives considered

- **Supersede ADR-0008 wholly.** Rejected: it would imply the
  canonical-width policy no longer holds, stranding four live runtime
  properties with no authoritative record.
- **Keep the verb, deprecated, for one release.** Rejected: its preflight
  already refuses on the trees it targets, so a deprecation window would
  ship a command that cannot run.
- **Reword the drift rule rather than collapse it.** Rejected: the
  classifier's only purpose was silence on a pre-migration tree. With
  that state gone the branch has no remaining meaning, and keeping it
  would preserve a distinction nothing consults.

## References

- ADR-0008 — the canonical-width policy this supersedes clause-wise.
- ADR-0004 — the archive convention that makes archived narrow ids
  permanent.
- G-0481 — the retirement blast radius and the permanence argument for
  read tolerance.
- M-0290 — the milestone that executed the retirement.
