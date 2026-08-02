---
id: D-0052
title: Dissolve the shipped-surface keep-list instead of mechanizing it
status: proposed
relates_to:
    - M-0287
    - M-0288
    - E-0078
---
> **Date:** 2026-08-02 · **Decided by:** human/peter

## Question

Three shipped surfaces document the rules that reject malformed and narrow id
shapes, and to do so they exhibit those shapes: the `aiwf-check` skill's
findings table, and the two planning rituals' anti-pattern bullets. Under the
widened `skill-body-id` rule they fire.

E-0078 planned a keep-list for them — an exemption keyed by path, carried in the
rule, pinned by a test. So: does the eradication epic ship a standing mechanism
whose purpose is to guarantee that certain narrow ids survive it?

## Decision

No keep-list. The rule carries no exemption, and M-0287/AC-4 — which existed to
prove the keep-list worked — is cancelled.

The passages are rewritten to describe the rejected shapes rather than exhibit
them, which the rule's own contract already sanctions: illustrative content uses
a canonical placeholder *or a shape-description*. "A numeric form narrower than
canonical width" instructs exactly as well as a spelled-out example, and leaves
nothing to exempt. That rewrite belongs to the sweep milestone, alongside the
rest of the content it edits.

If a passage then proves genuinely inexpressible without exhibiting a bad shape,
an exemption is added at that point — narrowed to the token, and justified by the
rewrite having been attempted and failed.

## Reasoning

Measuring the three files decided this. Of the findings they carry, roughly twice
as many are ordinary debris as are teaching citations: narrow composite
placeholders in a findings table, narrow epic and contract placeholders in
command examples, and — in two of the three — a narrow placeholder in the shipped
`description:` frontmatter itself. A by-path exemption, which is what the epic
specified, would have laundered all of it permanently, and the sweep works from
this rule's output, so anything exempted here becomes invisible to the cleanup.

The audit that proposed the keep-list assumed these files cite narrow ids only as
the subject of a rule. They do not. Discovering that only on measurement is the
reason the exemption is not worth pre-committing to.

The teaching citations that remain are avoidable, and that is what removes the
last argument for the mechanism. An exhibited bad shape and a described bad shape
carry the same instruction; only the exhibited one leaves a token for a reader —
human or model — to learn the wrong shape from, which is the cost this epic
exists to remove.

What the keep-list would have cost is three layers that all point the same way: a
path list in the rule, a test asserting the exemption holds, and an entry in the
narrow-id literal allowlist for the test's own fixtures. Three mechanisms whose
combined purpose is to make certain narrow ids permanent, shipped inside the epic
that removes narrow ids. The simpler resolution is for the exemption not to be
needed.

Vividness is the real cost. An example teaches faster than a description, and
these are surfaces an assistant reads before it writes. The trade is accepted
because the same surfaces are where a wrong shape gets learned, and the epic's
premise is that this is the more expensive direction to get wrong.

## Consequences

- E-0078's Constraints no longer name three files as keeping their narrow ids,
  and its success criterion no longer carves them out.
- The sweep milestone's scope grows by the rewrite of those passages. Its
  worklist is unchanged in size, since nothing was exempted from it.
- M-0287 delivers detection only, which its remaining criteria cover in full.
