# Epic wrap — E-0088

**Date:** 2026-08-24
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0088-make-every-path-changing-verb-repair-the-links-it-breaks

## Milestones delivered

- M-0314 — Route move through the shared link-region primitive (merged affe6ac4)
- M-0315 — Rewrite a moved entity's own outbound links (merged 1735bc88)
- M-0316 — Kill the link primitive's surviving mutants (merged 5f6f8e84)
- M-0317 — Settle whether ADR-0033's docs delegation fires (merged 7ae24c74)

## Changelog entry

### Fixed — E-0088: every path-changing verb repairs the links it breaks

- `aiwf move` now repairs the markdown links that point at a milestone it
  reparents, and recomputes that milestone's own relative links against its new
  directory. It previously rewrote nothing at all, so reparenting a linked
  milestone left every reference to it broken.
- `aiwf archive`, `rename`, `retitle` and `reallocate` now repair a moved
  entity's *own* outbound relative links, so a file swept into `archive/` keeps
  its links resolving instead of silently pointing one directory too high.
- A link destination whose `?query` or `#fragment` contains `://` is no longer
  mistaken for an external URL and skipped.
- `aiwf move --help` no longer states the opposite of what the verb does.

## Summary

E-0088 closed the distance between ADR-0033's commitment and the code
implementing it. `move` was the only verb emitting an `OpMove` that routed
through no link repair at all — the entity-truth audit's sole
`contradicted-by-code` verdict against that ADR — and M-0314 routed it through
the shared primitive. M-0315 then extended repair to a moved file's *own*
outbound links, a reach ADR-0033 never had, ratified as ADR-0046 once the
atomic-write question it depended on was answered. M-0316 brought the
primitive's tested edges to the kernel baseline, and M-0317 measured whether the
ADR's `docs/` delegation actually fires: it does not, and the finding routed to
the two gaps that own that half rather than widening this epic.

One scope shift is on the record. M-0316/AC-1's survivor-density target proved
unsatisfiable by doing what the milestone was named for: two-thirds of its
denominator was `archive.go`, whose survivors are all archive-verb code and none
link code, so killing every link survivor still left the ratio above the bar.
D-0076 narrowed the denominator to the link primitive and routed `archive.go`'s
share to G-0630, rather than absorbing unrelated work to move a number.

That narrowing leaves the epic's third success criterion — density "across the
files named in *Context*" — wider than the AC that now carries it. It is met by
M-0316 together with G-0630: the link primitive reached zero unexplained
survivors against a bar of 7.7 per thousand lines, and `archive.go`'s remaining
19 are tracked rather than killed. Recording that split here is what D-0076
deferred to epic close.

## ADRs ratified

- ADR-0046 — Path-link repair extends to a moved entity's own outbound links

ADR-0033 was corrected rather than superseded: its second bullet named a
delegate carrying no mechanical trigger, and the check that does cover the class
went unmentioned. The decision itself stands.

## Decisions captured

- D-0076 — Measure survivor density over the link primitive

## Follow-ups carried forward

Discovered by this epic:

- G-0623 — outbound link repair skips a move that relocates a whole directory
- G-0624 — `markdownLinkRegex` misses titled markdown links, hiding them from
  both scanners
- G-0625 — lychee `exclude_path` entries are regexes, dropping eight Normative
  docs from link-check
- G-0627 — the AC mechanical-evidence rule has no shape for an observational
  claim
- G-0630 — `archive.go`'s surviving mutants are all verb code, not link code
- G-0632 — a verb's long help can contradict its behaviour with nothing catching
  it

Pre-existing, deliberately out of scope, and re-framed by M-0317's measurement:

- G-0478, G-0439 — links from `docs/` into `work/`. Both were written assuming
  ADR-0033's delegation to an advisory doc-lint covered that half. It does not
  fire; `link-check` is what actually covers the class, late and into a workflow
  that is often already red. Neither gap was closed here — this epic verified
  the boundary rather than crossing it.

## Doc findings

Scoped `wf-doc-lint` over the epic's change-set — 37 files, of which three sit
under `docs/`: ADR-0033, ADR-0046, and the initiative below. No findings.
Markdown links all resolve, both backticked `aiwf` invocations resolve against
the current CLI, no heading-level skips, no documentation TODOs.

## Handoff

Every verb emitting an `OpMove` now repairs links in both directions, and the
link primitive's unexplained-survivor density is zero against the kernel
baseline. The measurement is reproducible: the invocation is recorded in
M-0316's Work log and an independent reviewer re-ran it and reproduced the
mutant inventory exactly.

Deliberately left open: the `docs/` half, which no verb reaches by ADR-0033's
own boundary. G-0478 and G-0439 own it, and M-0317's finding sharpened rather
than closed them — whoever picks them up should start from the measurement, not
from the delegation those gaps originally assumed.

The largest question this epic produced is not a defect.
[`docs/initiatives/entity-links-by-id-not-path.md`](../../../docs/initiatives/entity-links-by-id-not-path.md)
argues that the whole rewrite subsystem exists to maintain a reference format
chosen for clickability, while the kernel already owns one — the entity id —
that survives every move at no cost. Acting on it would delete machinery rather
than add it, which no other thread here does. It is a capture, not a decision;
promotion to a real entity is the next step, and its §6 open questions are what
that promotion has to answer first.
