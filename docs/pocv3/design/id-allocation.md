# ID allocation and lineage

The framework allocates ids one kind at a time: epics, milestones, gaps, decisions, ADRs, contracts. Two operators on different branches can pick the same id for different entities. This doc explains how the allocator avoids that in the common case, and how the audit trail stays correct when it happens anyway.

---

## The problem

`aiwf add` looks at the working tree, finds the highest id for the kind, adds one, and returns it.

The working tree is one branch. It doesn't know what trunk has done. So branch A can allocate `G-035`. Branch B, working in parallel, can also allocate `G-035`. Both branches do real work under that name. People reference `G-035` in chat and in PRs. AI assistants reference it in their plans.

When the branches merge, the collision surfaces. `aiwf reallocate` renumbers one side. The renaming is mechanical, but the audit trail splits: old commits on the renumbered branch still carry `aiwf-entity: G-035` in their trailers, and `G-035` now refers to a different entity. `git log --grep` returns commits about two different gaps under one id.

Two goals, then: make collisions rare, and when they happen, keep the cleanup lossless.

---

## Trunk-aware allocator

The allocator reads two trees: the working tree, and the configured trunk ref.

The default trunk is `refs/remotes/origin/main`. Consumers who use a different trunk set it in `aiwf.yaml`:

```yaml
allocate:
  trunk: refs/remotes/origin/main
```

If the configured ref doesn't exist, the allocator stops and asks the operator to fetch or set the field. There's no silent fallback to working-tree-only — that just reintroduces the original bug.

The trunk tree is read with `git ls-tree --full-tree -r <ref> -- work/`. No checkout, no working-tree disturbance. The cost is small, and the result is cached by ref-SHA inside a single verb run.

That's the entire allocator change.

The doc comment at `tools/internal/entity/allocate.go:34`:

> AllocateID picks the next free id for the kind. It reads the working tree and the configured trunk ref. Cross-branch collisions are caught at merge time by `ids-unique` and resolved by `aiwf reallocate`.

---

## The check reads trunk too

`ids-unique` already reads the working tree. It also reads the trunk ref. If an id appears in both with different entity paths, that's a finding:

```
ids-unique: G-035 is allocated on this branch and on refs/remotes/origin/main for different entities
```

The pre-push hook runs the check, so the collision is caught before the push lands on the remote.

There's no new flag and no merge simulation. The check sees what the allocator sees, and that's enough.

---

## Lineage in the frontmatter

When `aiwf reallocate` renumbers an entity, it appends the old id to a `prior_ids` list in the entity's frontmatter:

```yaml
---
id: G-037
title: ...
status: open
prior_ids: [G-035]
---
```

Oldest first. After two renumberings, the third id carries `[G-035, G-037]`.

The list is the source of truth. Tree-only readers — `aiwf show`, the HTML render, future projections — read lineage straight from the file, with no git log involved. A merge that brings a renamed entity into trunk carries the lineage along in the frontmatter, automatically.

---

## History walks the chain

`aiwf history` accepts any id, current or prior. A reverse index built at tree load resolves it:

```
priorIDIndex map[string]*Entity
```

The lookup returns the canonical entity. The entity's `prior_ids` plus its current `id` form the full chain. One grep:

```
git log --grep "aiwf-entity: (id1|id2|id3)"
```

Sort by commit time. Return.

`aiwf history G-035` and `aiwf history G-037` give the same timeline. The rename appears as a regular reallocate event in the chain, just like any other commit.

One code path. The frontmatter is the source of truth.

---

## Reallocate when both branches did real work

When two entities collide on an id, reallocate has to pick which one keeps it.

The rule:

1. If one entity's add commit is an ancestor of the trunk ref and the other isn't, the one already in trunk keeps the id. That's the entity the team has been calling by that name. The other gets renumbered.
2. If both add commits are in trunk, or neither is, reallocate stops and prompts. The operator types the id to renumber.

The ancestor check is `git merge-base --is-ancestor <add-sha> <trunk-ref>`. The add-sha comes from `git log --diff-filter=A --follow -- <path>` on each side.

That's the whole tiebreaker. No surrogate fields, no commit-time math. Either git's ancestry graph decides, or a human does.

---

## What this catches

| Case | Outcome |
|---|---|
| Forgot to fetch; trunk has new ids | No collision. The allocator already saw trunk. |
| Two branches off the same trunk SHA, both allocated | Collision possible. Pre-push catches it. Reallocate fixes it. Lineage preserves the history. |
| Hand-edited entity with a bad id | Caught by `id-path-consistent` and `ids-unique`. |

The first case is the dominant one in real workflows, and the allocator handles it. The second case is rare and follows the same pattern as a code merge conflict: diverge, conflict at integration, resolve with a known verb.

---

## What this is not

- A monotonic counter coordinated across branches.
- A coordination ref or push-CAS allocator.
- A surrogate identity per entity.
- A two-step amend inside `aiwf add`.
- An `--against` flag on `aiwf check`.
- A `lineage-broken` check rule.
- A walk over every ref in the repo.

Each one was considered, and each one is more code than the problem requires. If real friction shows up later, any of them can earn its own design.

---

## Implementation surface

- `tools/internal/entity/allocate.go` — union the working tree and the trunk ref. Hard error on a missing ref.
- `tools/internal/aiwfyaml/` — `allocate.trunk` field.
- `tools/internal/check/idsunique.go` — read the trunk ref. Add a cross-tree collision message.
- `tools/internal/entity/schema.go` — `prior_ids: []string` in the common schema.
- `tools/internal/verb/reallocate.go` — append to `prior_ids`. Ancestor-based tiebreaker. Prompt when ancestry can't decide.
- `tools/internal/tree/tree.go` — build `priorIDIndex` at load.
- `tools/cmd/aiwf/admin_cmd.go` — `readHistory` resolves through the index and greps the chain in one pass.
- `tools/cmd/aiwf/migrate_lineage_cmd.go` — one-shot verb. Backfills `prior_ids` from any `aiwf-prior-entity:` trailers in the commit history. Idempotent.

One YAML field. One frontmatter field. One one-shot verb. No new trailers, no new flags, no new check rules.

---

## Loose ends

- Lineage chains aren't depth-bounded. Real chains will be one or two deep. If a chain ten long shows up, a depth limit and a `lineage-too-deep` finding can be added then.
- The trunk tree is read on every `aiwf add`. It's cheap on normal repos. A monorepo with thousands of entities might benefit from a per-`<trunk-sha>` cache; that's a measurement-driven optimization, not a requirement.
- The `aiwf-prior-entity:` trailer is no longer written. Existing commits keep theirs. The migration verb backfills `prior_ids` for repos that already have those trailers.

---

## Cross-references

- [`provenance-model.md`](provenance-model.md) — `aiwf-prior-entity:` is no longer written on new commits.
- [`gaps.md`](../gaps.md) — G37 tracks the work.
