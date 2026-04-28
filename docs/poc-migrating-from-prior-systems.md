# Migrating from a prior planning system

aiwf is designed to be adoptable in repos that already have planning data captured in some other shape — a directory tree of markdown files, a flat decisions log, an issue tracker export, a planning spreadsheet. The framework does not ship a built-in migrator. Migration is intentionally a producer-side concern: aiwf consumes a manifest; how that manifest is produced is private to the consumer.

This document describes the public side of the contract — the manifest format and the import verb — and the recommended shape of the producer-side work. The producer side is yours to design.

For the manifest format itself, see [`poc-import-format.md`](poc-import-format.md). For the kernel commitments that motivate this split, see [`poc-design-decisions.md`](poc-design-decisions.md).

---

## The contract

aiwf provides one public surface for adoption:

```
aiwf import <manifest.yaml>
```

The manifest is a declarative list of entities to materialize. aiwf writes the entities, validates the projection, and produces commits. aiwf knows nothing about where the data came from.

You provide:

1. A manifest, in YAML or JSON, conforming to the format spec.
2. Optionally, separate notes about what the producer chose *not* to include (skip log, ambiguities, manual TODOs).

aiwf provides:

1. Validation against the projected tree (`aiwf import --dry-run`).
2. Atomic write + commit when validation passes.
3. Findings when validation fails, with no disk changes.

That is the entire boundary.

---

## The recommended workflow

Migration from a prior system to aiwf typically falls into two stages, each handled by your own private tooling. aiwf is unaware of either.

### Stage 1 — Pre-processing (mutates source)

Tidies the source repo so the projection step can read it mechanically. Everything ad hoc, inconsistent, or ambiguous in the source is resolved here, in the source's own terms. This stage commits to the source repo (or to a working branch of it).

Typical pre-processing concerns:

- Normalizing file shapes that drifted from convention.
- Splitting flat-file logs (e.g. one big `decisions.md`) into per-entity files where useful.
- Deciding what to keep, what to archive, what to drop.
- Reconciling status that was implicit in directory placement (e.g. a `completed/` dir) with status that should live in frontmatter.
- Filling in missing required fields.

Pre-processing is iterative. When the projection step (Stage 2) flags something it can't classify, the answer is usually "fix it in pre-processing," not "make the projector smarter."

### Stage 2 — Projection (read-only on source)

Reads the now-tidy source and emits an aiwf manifest. Pure projection: every value in the manifest is a final form, derived from the source by your private code. The projector emits two outputs:

1. **The manifest** — what should land in aiwf.
2. **A migration report** — what was skipped or ambiguous, with reasons. This is private to you; it never enters the manifest.

The projector is read-only. It does not mutate the source. That separation lets you re-run it deterministically as you iterate.

### Stage 3 — Import (read-only on source, writes aiwf consumer)

Run inside the aiwf consumer repo:

```
aiwf import --dry-run path/to/manifest.yaml
```

If clean:

```
aiwf import path/to/manifest.yaml
```

If `--dry-run` produces findings, the loop is:

1. Decide whether the issue is a manifest bug → fix the projector.
2. Or a source-data issue → fix it in pre-processing, re-project.
3. Or an aiwf-side issue with the projection → discuss; rare in practice.

Repeat until the dry-run is clean.

---

## What lives where

| Concern | Location | Visibility |
|---|---|---|
| Source-system schema knowledge | Your private tooling | Private |
| Pre-processing scripts | Your private tooling | Private |
| Per-source-system configuration (path overrides, status mappings) | Your private tooling | Private |
| The projector that emits manifests | Your private tooling | Private |
| Migration report (skip log, TODOs) | Your private tooling | Private |
| The manifest format spec | aiwf docs | Public |
| `aiwf import` | aiwf binary | Public |
| Embedded skills (`wf-*`) | aiwf binary | Public |

The manifest is the only artifact that crosses the public/private boundary. Anything aiwf sees is fully canonical, fully resolved, and could plausibly have been authored by hand.

---

## Two principles

### Pre-processing mutates source; projection is read-only

Mixing the two makes the projection step impossible to debug. If the projector mutates as it reads, you cannot re-run it to see what changed.

### Imperative is upstream; declarative crosses the boundary

Anything imperative — transformations, lookups, regex, conditional logic — lives in your private projector. The manifest carries data only. This means the manifest stays small and stable across binary versions; it also means aiwf has no surface area for producer-specific quirks to leak into.

---

## What aiwf does *not* provide

To make the boundary clear, here is what aiwf will not do for you:

- Read your prior system's files directly.
- Apply transformations defined in the manifest (templates, regex, callbacks, scripts).
- Skip entries based on patterns declared in the manifest. (If you don't want it imported, don't put it in the manifest.)
- Map between your system's id format and aiwf's. (The manifest carries final aiwf ids, allocated by the projector or by `auto`.)
- Import partial entities, drafts, or "candidates without ids." Candidates are unscheduled work, not entities — keep them in `ROADMAP.md`'s `## Candidates` section instead.
- Delete or archive your prior planning data. That decision is yours.

These omissions are deliberate. Each one would be a step toward aiwf understanding the producer's internals, which is precisely what the public/private split avoids.

---

## Practical advice

Things that experience suggests are useful when designing the producer side:

- **Start with archive, not import.** Move source data that you don't plan to bring forward into a sibling directory (e.g. `archive-prior-planning/`). Keep it grep-able but out of aiwf's walked roots. This shrinks the projector's job to just the data you actually need.
- **Project active work first; defer history.** Get the in-flight epics and milestones into aiwf cleanly. Historical (already-completed) work can stay in archive — `aiwf history` for those entities will be empty, which is fine; the source repo's git history is still authoritative.
- **One id space at a time.** It is easier to project all epics first, validate the projection, then add milestones, decisions, gaps. The manifest format supports forward refs, so you can do this in one pass — but you can also do it in stages, importing one kind at a time.
- **Use `auto` ids generously.** Pinning explicit ids in the projector only makes sense if you want the new ids to match something in the source. Otherwise, let aiwf allocate; it's cheap.
- **Keep the projector small.** A 100-line script that reads files, emits YAML, and produces a TODO log is in the right size class. If you find the projector growing branches for every edge case, that is a signal to push more cleanup into pre-processing.
- **Treat dry-run findings as a punch list.** A first dry-run with 200 findings is normal. They cluster — fixing one pattern in pre-processing fixes many entries. Aim to halve findings each iteration.

---

## When migration is *not* the right move

For very small repos with little prior planning data, hand-creating entities via `aiwf add` is often faster than building a projector. The break-even point is roughly the time spent reading source schema vs. writing manifest entries — under a few dozen entities, hand-authoring usually wins.

For very large repos where most of the prior data is genuinely historical, archive-and-restart is often better than full migration. The few in-flight entities can be hand-created; the rest stays grep-able in archive.

The case where building a producer pays off is the middle ground: a repo with substantial active planning state, of which most is worth lifting into aiwf intact.
