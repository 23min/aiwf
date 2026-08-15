---
id: G-0560
title: The normative doc tree has drifted from the kernel it documents
status: open
priority: medium
---
## What's missing

The Normative-tier docs — `docs/architecture.md`, `docs/overview.md`,
`docs/workflows.md`, `docs/skill-author-guide.md`, `docs/design/` — carry claims
the kernel no longer honors. A 2026-08-06 audit
against a binary built from the working tree found five classes, inventoried
with `file:line` citations and measured counter-evidence in
[`docs/initiatives/normative-docs-drift-audit.md`](../../docs/initiatives/normative-docs-drift-audit.md):

- **Prose stating the opposite of current behavior.** `workflows.md` describes
  epic roll-up as a deliberate non-guarantee; the verb refuses and a standing
  check rule backs it. `tree-discipline.md` argues aiwf must not write to the
  consumer's `CLAUDE.md`; ADR-0018 decided the reverse, and
  `design-decisions.md` documents the reverse — so two Normative docs
  contradict each other. `render roadmap --write` is documented as committing in
  three places and does not commit. `architecture.md` describes moves as
  `git mv` (ADR-0022 replaced the mechanism) and `rename` as updating the title
  (ADR-0037 split that to `retitle`).
- **Worked examples that fail as written.** Bare `aiwf add adr|gap|decision` is
  refused by the born-complete-kind body gate, and every `aiwf add milestone`
  example omits the now-required `--tdd`. This includes the single worked
  example in the skill-author guide — the first thing a scaffolder copies.
- **Stale enumerations.** Verb lists missing eleven verbs; a commitment count
  matching neither `design-decisions.md` nor CLAUDE.md; three of CLAUDE.md's ten
  commitments absent from the file CLAUDE.md names as their distillation. The
  narrow-width id tables in the same section are G-0517's subject, not this
  gap's — see below.
- **Cross-references resolving to nothing.** A principle cited by name into two
  files that do not contain it; verbs cited that aiwf does not have; a pointer
  to a section that does not exist; two finding codes the kernel never emits;
  a self-contradiction inside one document. G-0519 asks for the mechanical rule
  that would catch the id-citation subset of this; none of the instances above
  is an id citation, so they need reading rather than a rule.
- **Framing two migrations out of date**, plus drafting-history narration
  against the repo's own rule.

The audit doc is the work list. Fixing it is the deliverable; it is not itself
current truth and ages by construction.

Three adjacent items are **out of scope here**, two of them already tracked:

- **Narrow id widths in the same docs are G-0517**, which argues the disposition
  better than a restatement would: those citations are mostly real entities that
  existed at a narrow width, so the fix is per-reference research rather than a
  sweep. G-0559 gates it — the strings originate in `internal/entity/entity.go`
  and `aiwf schema` prints them, so widening the doc tables while the emitter
  still says `E-NN` leaves the published surface contradicting ADR-0008 and
  regrows the drift. Sequence is G-0559, then G-0517, then this gap's tables.
- **G-0519** asks for reference-checking of ids cited in documentation. It
  reaches the id-shaped subset of the cross-reference class above and none of
  the rest.
- **ADR-0003 is `accepted` and unimplemented** — an accepted decision to add a
  seventh entity kind, against a kernel that hardcodes six. ADR-0001 handles the
  identical situation by staying `proposed` with D-0037 recording the deferral.
  This is a planning-state disposition, not a doc fix, and nothing tracks it.

## Why it matters

These docs are the tier a reader is told to trust without verifying. CLAUDE.md's
documentation hierarchy defines Normative as "current-truth, kept in lockstep
with the code", so weighting a file by its path alone is the whole point of the
tiering — and that contract is what has broken. An AI assistant reading
`workflows.md` to learn the epic lifecycle learns a rule the verb refuses; one
copying the skill-author guide's worked example emits a command that fails.

Nothing catches this class. `wf-doc-lint` is grep-based and finds broken links,
orphan files and stale invocations — every finding here parses fine, links fine,
and is false. The `doc-id-*` rules only scan what `docs.paths` names, which is
`README.md` and `docs/workflows.md`, leaving the rest of the tier unscanned even
for the mechanical id-width subset. The catch has to be a periodic audit or a
review discipline; it will not be a check rule.

The drift clusters by age rather than by topic — the files last touched
2026-07-21/22 carry most of it, while everything from 2026-07-29 on is current.
That pattern says the exposure grows quietly with every kernel change that
doesn't come with a doc pass, and that a one-time sweep buys real time.
