---
id: G-0589
title: The doc authority tiering leaves root and derived documents unclassified
status: open
priority: medium
---
## What's missing

The documentation hierarchy in `.guidance/project/writing-docs.md` assigns every
active `docs/` subtree an authority tier, so a reader can weight a file from its
path alone without opening it. Shipped by E-0034 as the first of the three
layers G-0092 proposed. It does what it says, and two things a reader meets are
outside what it says.

**Root-level documents carry no tier.** The section scopes itself to `docs/`
subtrees, so the tracked root markdown files have no authority assignment —
including `CLAUDE.md` and `README.md`, which are as binding as anything under
`docs/design/`, and the append-only `CHANGELOG.md`, which the section names
only as the precedent for the Archival tier's convention.

**The hierarchy has no tier for a render that lags its source.** `ROADMAP.md`,
`STATUS.md` and `WHITEBOARD.md` are gitignored and regenerated — on demand, by
the post-commit hook, and by each whiteboard run — so a contradiction inside
any of them is a stale render rather than a claim about the project, and
reading them for drift produces noise at a rate set by regeneration lag. The
one generated subtree the hierarchy tiers, `docs/reference/`, is Normative
because a test keeps it in step with its source; nothing keeps these three in
step, and no tier says so. CLAUDE.md lists `ROADMAP.md` and `STATUS.md` among
the derived artifacts to regenerate rather than hand-edit, which instructs a
writer but gives a reader no tier to weigh them by.

## Why it matters

The path-based tier list is the repository's answer to which documents carry
authority.
[ADR-0044](../../docs/adr/archive/ADR-0044-declare-document-authority-in-aiwf-yaml-not-frontmatter.md),
which would have declared authority in `aiwf.yaml`, was rejected; its cancel
records that the tier list stays the answer and that this gap stays open on its
own. A document the list cannot classify is one a reader must weigh by opening
it, which is the work the tiering exists to save.

The classification has already been written down outside the hierarchy once.
Scoping a commitments sweep required classifying the root documents by hand —
hand-authored against derived against append-only — and that classification is
written into
[`docs/initiatives/milestone-preflight-as-independent-review.md`](../../docs/initiatives/milestone-preflight-as-independent-review.md)
under *Corpus and cost*, because the hierarchy could not supply it. So two
places state what is authoritative in this repository, and when the hierarchy
changes, nothing makes the second copy follow.

Distinct from [G-0560](G-0560-the-normative-doc-tree-has-drifted-from-the-kernel-it-documents.md),
which is about normative documents having drifted from the kernel. This is about
documents the tiering never reaches — a defect in the classifier rather than in
what it classifies.
