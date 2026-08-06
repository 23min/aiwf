---
id: G-0560
title: The normative doc tree has drifted from the kernel it documents
status: open
priority: medium
---
## What's missing

The Normative-tier docs — `docs/architecture.md`, `docs/overview.md`,
`docs/workflows.md`, `docs/skill-author-guide.md`, `docs/design/`,
`docs/migration/` — carry claims the kernel no longer honors. A 2026-08-06 audit
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
- **Stale enumerations.** Verb lists missing eleven verbs; id-format tables at
  narrow width against ADR-0008; a commitment count matching neither
  `design-decisions.md` nor CLAUDE.md; three of CLAUDE.md's ten commitments
  absent from the file CLAUDE.md names as their distillation.
- **Cross-references resolving to nothing.** A principle cited by name into two
  files that do not contain it; verbs cited that aiwf does not have; a pointer
  to a section that does not exist; two finding codes the kernel never emits;
  a self-contradiction inside one document.
- **Framing two migrations out of date**, plus drafting-history narration
  against the repo's own rule.

The audit doc is the work list. Fixing it is the deliverable; it is not itself
current truth and ages by construction.

Two adjacent items are deliberately **out of scope here**, because each needs
its own decision before any prose changes:

- The narrow id-format strings originate in `internal/entity/entity.go`, which
  `aiwf schema` prints. Correcting the doc tables without correcting the emitter
  leaves the published surface contradicting ADR-0008 and regrows the drift.
  That is a published-surface change.
- ADR-0003 is `accepted` and unimplemented — an accepted decision to add a
  seventh entity kind, against a kernel that hardcodes six. That is a
  planning-state disposition, not a doc fix.

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
