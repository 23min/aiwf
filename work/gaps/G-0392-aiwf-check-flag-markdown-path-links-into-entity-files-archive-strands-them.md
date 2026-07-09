---
id: G-0392
title: 'aiwf check: flag markdown path-links into entity files (archive strands them)'
status: open
---
## What's missing

Nothing in `aiwf check` (or the advisory `wf-doc-lint` ritual) flags a markdown link whose destination resolves into an entity file under `work/epics/`, `work/gaps/`, `work/decisions/`, `work/contracts/`, or `docs/adr/` — whether the link sits in another entity's own body prose or in a hand-authored `docs/*.md` file. `aiwf archive` (`internal/verb/archive.go`) is a pure `git mv`: it produces only `FileOp{Type: OpMove}` entries, never an `OpWrite`, so it never rewrites any other file's cross-references when it sweeps a terminal entity into its per-kind `archive/` subdirectory. A frontmatter reference to that entity survives for free — `Tree.ByID` resolves the id across active and archive by construction — but a hand-authored markdown link encodes a static relative path, and nothing ever revisits it once the target moves.

## Why it matters

This is not theoretical: of the four `docs/adr/*.md` files that currently link into `work/`, three are already broken (ADR-0008 links a since-archived, since-rewidth'd gap; ADR-0011 links a since-archived epic; ADR-0016 links a since-archived gap) — a 75% rot rate in the single most actively-maintained corner of `docs/`. The framework already has an archive-proof addressing scheme (cite the bare id, resolve it with `aiwf show <id>` / `aiwf history <id>`), but nothing steers an author toward it or catches the fragile alternative. A mechanical `aiwf check` rule — flagging any markdown link destination matching an entity-file path, in any entity body or any `docs/*.md` file — would catch this at the same pre-push chokepoint that already catches `body-prose-id` and `skill-body-id`; the advisory `wf-doc-lint` ritual (its markdown-link-integrity check landing in G-0390) can mirror the same framing for consumers who haven't wired the mechanical rule.