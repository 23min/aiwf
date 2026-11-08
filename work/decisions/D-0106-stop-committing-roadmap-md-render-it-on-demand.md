---
id: D-0106
title: Stop committing ROADMAP.md; render it on demand
status: proposed
relates_to:
    - ADR-0058
---
> **Date:** 2026-11-08 · **Decided by:** Peter Bruinsma

## Question

Whether `ROADMAP.md` stays a committed file. `aiwf render roadmap --write` rewrites it on disk and never commits, so the rituals that plan and wrap work commit it themselves. Those commits are a large share of the planning commits, and under [ADR-0058](../../docs/adr/ADR-0058-amend-the-unpushed-verb-commit-instead-of-adding-one.md) each one ends a run of verbs that would otherwise become one commit.

## Decision

- **The rituals stop regenerating and committing `ROADMAP.md`,** dropping these steps:
  - "Update `ROADMAP.md`" in `aiwfx-plan-epic` and `aiwfx-plan-milestones`;
  - "Update the roadmap" and "Regenerate the roadmap" in `aiwfx-wrap-milestone`;
  - "Regenerate the roadmap" in `aiwfx-wrap-epic`;
  - the roadmap regen listed in the wrap rituals' closing sequences.
- **This repository untracks `ROADMAP.md` and ignores it,** as it does `STATUS.md`.
- **The roadmap is rendered on demand.** `aiwf render roadmap` prints it, and `--write` still writes the file for a repository that keeps one.

## Reasoning

**The commits it costs.** Measured 2026-11-08 over the research window `6b87e0f24..4343f2bd1` with `git log -- ROADMAP.md`, keeping the commits whose only changed file is `ROADMAP.md` and grouping them by their `aiwf-verb` trailer. 77 commits carry a ritual's trailer:

| Ritual | Commits |
|---|---|
| `wrap-milestone` | 45 |
| `plan-milestones` | 12 |
| `wrap-epic` | 11 |
| `plan-epic` | 9 |

A ritual's commit ends a run, so replayed under ADR-0058, amend leaves 727 planning commits with these commits and 608 without them ([report 02](../../docs/research/fewer-planning-commits/reports/02-history-measurements.md), cases a and b).

**Nothing on a critical path reads it:**
- README links to it.
- `aiwfx-plan-epic` reads it if present, beside the epics under `work/`.
- Link-check skips it, because its inlined entity prose carries relative links that break at the repository root (`.lychee.toml`).
- `roadmap-case-collision` guards only the filename's case.

No rendered site is published, and this repository's copy has no hand-written `## Candidates` block.

**Alternatives:**
- *Keep it tracked, regenerated only by `aiwfx-wrap-epic`.* That removes 66 of the 77 commits. But a newly planned epic stays out of the roadmap until the next epic wrap, and the wrap-epic commit can ride neither the promote, which is a verb's commit, nor the merge.
- *Keep it as it is.* The commits stay, and amend leaves 727.
- *Have verbs regenerate it inside their own commits.* It would stay current with no extra commits, but every branch would rewrite one generated file. Whether that causes merge conflicts was not measured.

## Consequences

- **Every consumer's rituals stop committing their roadmap.** A consumer that keeps `ROADMAP.md` tracked keeps it current with `aiwf render roadmap --write` and a commit of its own; that includes one with a `## Candidates` block, which render preserves. The CHANGELOG entry says so.
- **GitHub visitors lose the one-page view.** README's Roadmap section points to `aiwf render roadmap` instead.
- **`aiwfx-plan-epic` reads the output of `aiwf render roadmap`,** not the file, so it never reads a stale local copy.
- **`aiwf-verb: plan-epic` and `aiwf-verb: plan-milestones` are refused.** The ritual-verb allowlist is derived from the trailers the embedded rituals write, and only the roadmap commits wrote these two, so the `commit-msg` hook refuses them. A copy of either ritual installed before `aiwf update` still attempts its roadmap commit, and the hook refuses it. `trailer-verb-unknown` reads only the unpushed range, so history already pushed with these trailers raises no finding.
- **What changes:**
  - the four rituals under `internal/skills/embedded-rituals/`, which are shipped content and so change on a branch;
  - the `planner` and `reviewer` agents beside them, which drop their roadmap duties;
  - `.gitignore`, and `ROADMAP.md` removed from the index in this repository;
  - README's Roadmap section;
  - `CLAUDE.md`'s "Regenerate derived artifacts" line, which calls `ROADMAP.md` tracked;
  - the CHANGELOG.
- **Unchanged:**
  - `aiwf render roadmap` and its `--write`;
  - the preservation of a `## Candidates` block;
  - `roadmap-case-collision`.
