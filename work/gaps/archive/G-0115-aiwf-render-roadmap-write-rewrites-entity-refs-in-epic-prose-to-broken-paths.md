---
id: G-0115
title: aiwf render roadmap --write rewrites entity refs in epic prose to broken paths
status: addressed
addressed_by_commit:
    - b285d34
---
## What's missing

Regenerating `ROADMAP.md` via `aiwf render roadmap --write` rewrites entity-file references that appear inside epic prose blocks (the Goal section pulled from each epic's body) to two flavors of broken path:

- **Stale narrow-legacy IDs.** Canonical 4-digit IDs (per ADR-0008) are rewritten back to their pre-canonicalization narrow form — `G-0055` becomes `G-055`, `G-0058` becomes `G-058`, `G-0062` becomes `G-062`. The slug filename on disk uses the canonical width (`G-0055-...`), so the rewritten link points at a file that does not exist.
- **Broken relative paths.** Absolute-from-repo-root references (e.g. `work/gaps/archive/G-0055-...`) are rewritten to relative forms like `../../gaps/G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md` that don't resolve from the roadmap's own location (which is repo root — relative paths should start with `work/...`, not `../../`).

The dangling-refs pre-commit policy (commit `abf788f`, closes G-0091) catches the rewritten output and blocks the commit, so the regenerated roadmap can't actually land. The broken file is left staged in the working tree after the failed commit. Reproduce from a clean tree: `aiwf render roadmap --write` then `git diff --cached ROADMAP.md` — the rewrite touches at least three epic Goal sections in flight today (E-0016, E-0017, E-0018), all in narrative prose pulled from each epic's body.

Likely root cause: the roadmap renderer treats entity links in body markdown as repo-relative paths that need re-rooting to the roadmap's emission location, but (a) the re-rooting logic was written before ADR-0008's uniform 4-digit width and emits narrow-legacy IDs by re-deriving slugs from canonical IDs without preserving width, and (b) the relative-path computation resolves against the wrong base directory (assuming the roadmap lives under `work/...` rather than at repo root).

## Why it matters

The roadmap is the project's narrative state surface — the markdown answer to *"what's in flight and what's planned?"* visible from the repo's GitHub view without running any tool. The renderer is the chokepoint that keeps it in step with the actual planning tree (epics, milestones, statuses). Today the chokepoint is silently unusable: a maintainer running `aiwf render roadmap --write` after adding an epic or promoting a milestone gets a staged-but-blocked diff, has to manually `git restore` it, and the roadmap stays out of step with reality.

The downstream consequence: the roadmap drifts away from `aiwf status` / the entity tree over time. Either the maintainer keeps it in sync by hand (defeating the renderer), or the file rots until a future epic notices and the diff to rebase is so large the rebuild becomes its own chunk of work. Both outcomes invert the framework's promise that planning state is derived, not hand-curated.

The mitigation until this is fixed: leave the existing `ROADMAP.md` alone (it was hand-curated at some point — or last regenerated before the narrow→canonical width migration); don't run `aiwf render roadmap --write` blindly. The fix lives in the roadmap renderer's link-rewriting helper — likely a path-normalization routine that needs (1) preserve canonical-width IDs when re-deriving slugs, (2) compute relative paths against the roadmap's actual emission directory (repo root), or stop re-rooting entirely and emit absolute-from-repo-root links since `ROADMAP.md` itself sits at repo root.
