---
id: G-0255
title: promote --superseded-by writes only one side of the supersession link
status: open
---
## What's missing

`aiwf promote <old> superseded --superseded-by <new>` writes `superseded_by` onto the
superseded ADR but never writes the reciprocal `supersedes` entry onto the superseding
ADR. No verb writes `supersedes` at all — the only assignment to the field
(`internal/verb/reallocate.go:486`) rewrites an *existing* entry during a renumber and
cannot create one. The `adr-supersession-mutual` rule (`internal/check/check.go:759`)
checks a two-sided invariant, so after a supersession it fires permanently (warning)
and cannot be cleared through any verb route.

Two internal contradictions confirm this is a defect, not intended behaviour:

1. The verb mandates the flag — `requireResolverForResolutionClass`
   (`internal/verb/promote.go:372`) errors without `--superseded-by`, claiming it is
   passed "so the adr-supersession-mutual rule is satisfied" — yet `applyResolverFlags`
   (`internal/verb/promote.go:409`) writes only `superseded_by`, satisfying one side.
2. The flag help (`internal/cli/promote/promote.go:71`) says it "satisfies
   adr-supersession-mutual atomically with the status change". It satisfies one
   direction only.

A masking test compounds it: `TestPromote_SupersededByFlag_BinaryEndToEnd`
(`internal/cli/integration/promote_resolver_cmd_test.go:79`) carries a doc comment
asserting "the post-promote tree validates clean (mutual link satisfied via supersedes
on the superseding ADR)" — but the test never runs `aiwf check`, and the flag never
writes `supersedes`. It asserts only the commit trailers, so it passes while the
property its comment advertises is broken.

## Why it matters

Every ADR supersession leaves a warning that no CLI path can clear. The operator either
lives with a permanent finding or hand-edits the superseding ADR's frontmatter, which
itself trips `provenance-untrailered-entity-commit`. Worse, the verb makes a written
promise (mandatory flag + help text) it does not keep, so an operator reasonably
believes the link is recorded on both sides when it is not — bidirectional navigation
(`supersedes` forward-refs, `referenced_by` views) silently loses the edge.

Fix shape: when `--superseded-by <new>` is set, the promote verb should also append
`<old>` to `<new>.supersedes` (dedup → idempotent) after validating `<new>` exists and
is an ADR, writing both files in the verb's single commit. Multi-file single-commit is
already precedented by `aiwf reallocate`, so the one-commit-per-mutation principle
holds. The masking test must be updated to run `aiwf check` and assert zero
`adr-supersession-mutual` findings, and its doc comment corrected.
