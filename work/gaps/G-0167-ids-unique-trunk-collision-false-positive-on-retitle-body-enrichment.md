---
id: G-0167
title: ids-unique trunk-collision false positive on retitle + body enrichment
status: addressed
discovered_in: M-0125
addressed_by_commit:
    - 8b56ba1c
---
## What's missing

`internal/gitops/refs.go::RenamesFromRef` invokes `git diff -M --diff-filter=R`
with no explicit similarity threshold, which defaults to **50%**. When a
gap (or other entity) is retitled (rename of the slug + edit of the
`title:` frontmatter field) *together with* a substantial body
enrichment, the cumulative file similarity from merge-base to HEAD can
fall below 50% and git no longer reports the rename. The `aiwf check`
ids-unique rule then sees two files with the same id (one on trunk at
the old slug, one on branch at the new slug) and fires a
**`ids-unique/trunk-collision`** error — a false positive, since the
file IS the same entity moved.

Symptom: pre-push hook blocks the push with a hint that misleads the
operator toward `aiwf reallocate` (which would actually renumber and
LOSE the entity's id — exactly wrong for this case).

## Why it matters

Retitle-with-enrichment is a normal authoring pattern when a gap's
understanding deepens (e.g. duplicate consolidation: pulling another
gap's value-add into the canonical body in the same commit cluster as a
retitle). It should push cleanly. Today the operator has to bypass the
pre-push hook (`git push --no-verify`) to get the work upstream — which
breaks the "framework correctness must not depend on the LLM's
behavior" rule (the chokepoint is no longer authoritative; operators
learn to skip it).

## Reproduction

1. Commit a retitle (`aiwf retitle G-NNNN "new title"`) on a feature branch.
2. Apply substantial body edits (e.g. tripling the body's line count) on
   the same branch.
3. `git push origin <branch>` — pre-push hook runs `aiwf check`, which
   reports `ids-unique/trunk-collision` for the entity.

Verified on M-0125 work: G-0139 retitled from
"Implement cancel-cascade per D-0003 and D-0004" to
"Implement cancel refusal on non-terminal children/ACs per D-0003 and D-0004"
plus body growth from ~50 lines to ~150 lines (G-0162 duplicate
consolidation). At `--find-renames=10%` git detects the rename; at
default `50%` it doesn't.

## Proposed fix shape

Two viable paths; pick one (Path B is cleaner long-term):

**Path A — Lower the similarity threshold.** Change line 183 of
`internal/gitops/refs.go` to `git diff -M20 --diff-filter=R ...` (or
similar). Catches more renames. Risk: false positives where two
distinct files genuinely share 20% similarity by coincidence (e.g. both
start with the same `## What's missing` heading). Mitigation: scope
the rename detection to entity-file paths (`work/<kind>/`), where the
frontmatter id is the truth.

**Path B — Match by frontmatter `id:` field.** Parse the frontmatter of
every file under `work/<kind>/` on both sides (merge-base and HEAD),
build a map of `id → path`, and detect renames as `id present in both
but path changed`. Robust against any body diff. Doesn't depend on git's
similarity heuristic. Touches `internal/gitops/refs.go` (or moves the
rename detection into `internal/tree/`).

Path B is the kernel-correctness path — id-based matching is what the
spec actually means by "same entity." The git-similarity heuristic was
a convenient proxy; the proxy breaks under reasonable authoring
patterns. Fix the proxy, not the threshold.

## Test surface

Both paths need a regression test that constructs the failure mode:

- Fixture tree with a single entity at version 1 (short body, original
  slug).
- Apply a retitle (`aiwf retitle`) → entity at version 2 (new slug,
  frontmatter title changed).
- Apply substantial body edits (e.g. 5x line growth).
- Assert `RenamesFromRef` returns the expected `old → new` pair.

For Path B: also test the id-extraction logic against malformed
frontmatter (missing id, duplicate id within the file, etc.) so the
fix doesn't introduce a new failure mode.

## Workaround

Until the kernel ships the fix, operators encountering this push their
work with `git push --no-verify` (explicit human approval per
CLAUDE.md). The first M-0125 push to `epic/E-0033-...` used this
workaround for G-0139.

## Closing this gap

When the impl lands:
- Remove the workaround note above.
- Promote G-0167 to `addressed` with `--by M-NNNN`.

## Discovered in

M-0125/AC-2 (G-0139 retitle + G-0162 consolidation body enrichment
triggered the false positive on pre-push).
