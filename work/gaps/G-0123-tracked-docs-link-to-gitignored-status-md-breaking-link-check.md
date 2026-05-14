---
id: G-0123
title: Tracked docs link to gitignored STATUS.md, breaking link-check
status: open
---
## What's missing

After [G-0112](archive/G-0112-status-md-pre-commit-regen-produces-merge-conflicts-on-a-derived-artifact.md) made `STATUS.md` a post-commit-regenerated, gitignored artifact, the tracked docs `README.md` and `ROADMAP.md` still link to it as if it were a checked-in file:

- `README.md:44` — `[STATUS.md](STATUS.md)`
- `README.md:159, 257, 332, 349` — additional prose references
- One ROADMAP.md reference

On a fresh CI checkout the file does not exist yet (the post-commit hook only fires after the operator's first local commit), so `lychee` resolves the link as a missing file and the `link-check` job fails. The link-check has been red on every push since G-0112 landed (May 14, commit `63acc40`).

The commit message for `63acc40` flagged the unrelated render-count drift but did not flag this consequence — the gitignore + remove-from-index dance was understood to drop the file from the tracked set, but the downstream effect on link discipline was not anticipated.

## Why it matters

A persistently-red `link-check` job conditions the operator (human and LLM) to ignore CI status, which erodes the chokepoint's signal value for the genuine link breakages it catches (e.g., the three over-traversed `../../../docs/...` paths in ROADMAP.md that surface in the same run). The fix is one of two surgical moves — exclude `STATUS.md` from `.lychee.toml` as a generated artifact, or replace the tracked-doc references with prose pointing operators at `aiwf status --format=md` instead. Both are cheap; the second is conceptually cleaner because it stops treating a derived snapshot as a viewable repo artifact.

## Resolution shape

Combine both options for defense-in-depth:

1. Add `STATUS.md` to `.lychee.toml`'s exclude list (or the equivalent file-link skip), with a one-line rationale citing G-0112.
2. Rewrite the `[STATUS.md](STATUS.md)` link in `README.md:44` to inline prose ("run `aiwf status --format=md` to render the current snapshot") so a reader on github.com isn't promised a file that may not be there. Audit the other `STATUS.md` mentions in `README.md` and `ROADMAP.md` for the same treatment — most are explanatory references that don't need a link target.

The lychee exclude is the chokepoint (defense in depth); the doc rewrite is the conceptual cleanup. The two together leave the kernel's framing consistent: `STATUS.md` is a generated snapshot, not a checked-in artifact, and tracked docs reflect that.
