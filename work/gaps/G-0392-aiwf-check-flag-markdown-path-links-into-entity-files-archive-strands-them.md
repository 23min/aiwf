---
id: G-0392
title: 'aiwf check: flag markdown path-links into entity files (archive strands them)'
status: addressed
addressed_by_commit:
    - aa57a1b6
---
## What's missing

Nothing in `aiwf check` (or the advisory `wf-doc-lint` ritual) flags a markdown link whose destination resolves into an entity file under `work/epics/`, `work/gaps/`, `work/decisions/`, `work/contracts/`, or `docs/adr/` — whether the link sits in another entity's own body prose or in a hand-authored `docs/*.md` file. `aiwf archive` (`internal/verb/archive.go`) is a pure `git mv`: it produces only `FileOp{Type: OpMove}` entries, never an `OpWrite`, so it never rewrites any other file's cross-references when it sweeps a terminal entity into its per-kind `archive/` subdirectory. A frontmatter reference to that entity survives for free — `Tree.ByID` resolves the id across active and archive by construction — but a hand-authored markdown link encodes a static relative path, and nothing ever revisits it once the target moves.

## Why it matters

This is not theoretical: of the four `docs/adr/*.md` files that currently link into `work/`, three are already broken (ADR-0008 links a since-archived, since-rewidth'd gap; ADR-0011 links a since-archived epic; ADR-0016 links a since-archived gap) — a 75% rot rate in the single most actively-maintained corner of `docs/`. The framework already has an archive-proof addressing scheme (cite the bare id, resolve it with `aiwf show <id>` / `aiwf history <id>`), but nothing steers an author toward it or catches the fragile alternative. A mechanical `aiwf check` rule — flagging any markdown link destination matching an entity-file path, in any entity body or any `docs/*.md` file — would catch this at the same pre-push chokepoint that already catches `body-prose-id` and `skill-body-id`; the advisory `wf-doc-lint` ritual (its markdown-link-integrity check landing in G-0390) can mirror the same framing for consumers who haven't wired the mechanical rule.

## Possible approaches

The primary fix is prevention plus periodic detection, not a change to `aiwf archive`'s core mechanism: the write-time `aiwf check` rule above steers new links toward citing the bare id, and `wf-doc-lint`'s markdown-link-integrity check catches any link that was fine when written and only rotted because its target has since archived. Reopening ADR-0004's uniform archive convention — dropping the physical move, or leaving a redirect stub at the old path — is out of scope: the former undoes a ratified decision for a side effect unrelated to its core value (an active tree that stays legible), and the latter is exactly the "tombstones beyond terminal statuses" this project explicitly excludes.

One enhancement worth building alongside the check: extend `aiwf archive` to rewrite cross-references within the loaded entity set at sweep time, reusing the word-boundary id-token scan `aiwf reallocate` already runs across `t.Entities` bodies (`internal/verb/reallocate.go`) — applied when an entity moves to `archive/`, not only when it's renumbered. This is bounded (it only ever touches files the loader already knows about) and reuses an already-trusted mechanism rather than inventing repo-wide prose rewriting, which would have to touch arbitrary `docs/*.md` files and risks a false-positive rewrite inside a quoted example or a "the old path used to be X" aside. It closes the one case prevention alone can't: an entity-to-entity link that was correct when written and only breaks because its target archives later. It should not extend to non-entity `docs/*.md` files (README, research, explorations) — those stay covered by detection only.