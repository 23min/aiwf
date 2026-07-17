---
id: G-0369
title: Body-prose-id check misses unhyphenated real-entity references
status: open
priority: low
---
## Problem

`aiwf check`'s `body-prose-id` rule (`internal/check/body_prose_id.go`) anchors
its whole detection surface on a literal hyphen immediately after the kind
prefix letter (`idTokenPattern`), by design — this avoids misfiring on
unrelated tokens that share the shape but aren't aiwf ids (G7, M16, D20, E85,
and similar). One consequence of that deliberate narrowing: a real,
already-allocated entity referenced *without* its canonical hyphen and
zero-padding (e.g. `G45` written for `G-0045`) is invisible to the scanner
entirely — not merely tolerated, but never even classified as a candidate
token, so it produces no finding at any severity.

Discovered 2026-07-05 in `G-0362`'s own body prose (a real reference to
`G-0045`, written as `G45`; fixed in place via `aiwf edit-body G-0362` — see
`aiwf history G-0362`).

## Why it matters

CLAUDE.md's own prose-id convention requires canonical form when a real
entity is being referenced in committed prose. The `body-prose-id` check is
the one mechanical backstop for that convention, and it is otherwise sound —
but this specific, narrow failure mode (a real reference written without its
hyphen) currently has no coverage at all, mechanical or otherwise, beyond
authoring discipline.

## Shape (sketch)

Not a widened version of the existing strict-shape regex — that would
reintroduce the exact false-positive risk the hyphen anchor exists to avoid
(G7, M16, D20, E85). Instead, a second, resolution-gated detector layered on
top of the existing check:

- Match the bare, no-hyphen shape: `\b(E|M|G|D|C|ADR)\d+\b`.
- For each match, canonicalize (insert the hyphen, zero-pad to the kind's
  canonical width) and resolve it against the same `idx.ByID` / `idx.Trunk`
  index the strict-form path already builds.
- Emit a finding only when the hyphenated/padded form resolves to a real,
  already-allocated entity in this tree — at a new **warning** severity (not
  error), since a resolution-independent coincidence remains structurally
  possible (a repo whose numbering happens to collide with an unrelated
  mention, e.g. a real `G-0007` next to prose about "the G7 summit").
- New subcode (e.g. `unhyphenated-reference`).

Needs: a resolution-gated classification branch in
`internal/check/body_prose_id.go`, tests for both the hit case and the
coincidence-miss case, a firing-fixture test, and a policy-coverage entry per
this repo's meta-gate (`internal/policies/firing_fixture_presence.go`).
