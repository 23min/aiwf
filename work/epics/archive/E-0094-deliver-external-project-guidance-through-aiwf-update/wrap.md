# Epic wrap — E-0094

**Date:** 2026-09-23
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0094-deliver-external-project-guidance-through-aiwf-update

## Milestones delivered

- M-0344 — Publish the external engineering guidance corpus (merged `d7af3838a`).
- M-0345 — Preserve legacy guidance through repository-aware routing (merged `a4d4e8379`).
- M-0346 — Deliver explicitly selected project guidance through update (merged `63ad2fe27`).
- M-0347 — Suggest applicable guidance during init and update (merged `ae2947598`).
- M-0348 — Verify migration and establish the reduction prerequisite (merged `f05a7eb63`).

G-0703 was fixed on the epic branch after M-0348's wrap, in `6a3c85366`.

## Changelog entry

### Added — E-0094: deliver external engineering guidance as tracked project policy

- Projects can select engineering-guidance packs from an external source in
  `aiwf.yaml` (`guidance.packs`, with `guidance.ignored` for suggestions to
  suppress). `aiwf init` and `aiwf update` download the selected packs on
  demand from the source's default branch and write them as tracked files under
  `.guidance/`, with an index recording the installed commit, and routing blocks
  in `CLAUDE.md` and `AGENTS.md` that tell Claude and Codex to read the
  maintainer's own `.guidance/project.md` overrides first, when that file
  exists, and then only the packs a task needs. aiwf never writes
  `.guidance/project.md`. The default source is `23min/engineering-guidance`;
  `guidance.source` overrides it. No persistent cache is kept, and update never
  commits or pushes the generated files.
- With maintenance enabled, which is the default, every `aiwf init` and
  `aiwf update` clones the source's default branch into a temporary directory,
  whether or not any packs are selected, to compute suggestions. An unreachable
  source is reported as a skipped guidance step and the rest of the command
  continues; `guidance.enabled: false` stops the download.
- A failed download, an invalid or missing pack, or a locally edited generated
  file leaves the installed guidance and its recorded commit unchanged; the
  reason is reported and the rest of the update continues. An interrupted
  installation is finished by the next update. Setting `guidance.enabled: false`
  stops refreshes and suggestions while the installed guidance stays in use.
  `aiwf doctor` reports the installed selection, missing files and edited owned
  content, without claiming the installation is current with its source.
- `aiwf init` and `aiwf update` suggest packs whose file patterns, taken from
  the external catalogue, match the repository, and say what matched.
  Interactive runs offer select, not now, or ignore, and save the choices once
  all prompts finish. Noninteractive runs only report suggestions and keep
  refreshing existing selections.
- An `aiwf.yaml` that is not UTF-8, or that holds more than one YAML document,
  is now refused without being changed — by guidance, hook and contract edits,
  and also by `aiwf check`, the `aiwf contract` commands and `aiwf rename-area`.
  Because `aiwf check` runs as the pre-push hook, a repository with such a file
  cannot push until the file is saved as a single UTF-8 document.
- Projects that have not selected packs keep their existing guidance delivery.
  When a repository adopts aiwf guidance, recognized ai-dotfiles engineering
  imports are replaced only after a compatibility check of the environment
  running the command passes, so handover does not leave a recognized
  ai-dotfiles import beside aiwf's routing in the same host file.

### Fixed — G-0703: the aiwf guidance block in AGENTS.md no longer carries the aiwf version

The header comment of the managed aiwf guidance block in `AGENTS.md` no longer
includes the version of the aiwf binary that wrote it, so running `aiwf update`
with a different aiwf version no longer modifies this tracked file when the
guidance itself is unchanged. The first update after upgrading removes the
`aiwf-version:` field from that header once.

## Summary

aiwf now selects, refreshes and delivers externally authored engineering
guidance as tracked project policy for Claude and Codex, which contributors can
read without aiwf, ai-dotfiles or VS Code synchronization. The corpus lives in
its own repository, and ai-dotfiles keeps legacy repositories and personal
settings; ADR-0052 draws that boundary, and verifying delivery to repositories
aiwf does not own was ruled ai-dotfiles' responsibility during M-0348's wrap.
aiwf's own repository was migrated and verified, and E-0092 now names this epic
as its delivery prerequisite. The G-0703 version-stamp defect found during
verification was fixed before the epic closed.

## ADRs ratified

- ADR-0052 — Keep project guidance independent of personal bootstrap (accepted;
  supersedes ADR-0051).

## Decisions captured

- D-0097 — Ask before substituting unreadable project guidance (accepted).
- D-0098 — Finish interrupted guidance installation on the next update (accepted).
- D-0099 — Use Git ignore rules and dependency exclusions for detection (accepted).
- D-0100 — Save guidance choices only after all prompts finish (accepted).
- D-0089 — Language guidance is externally owned and delivered as project
  policy (accepted at this wrap; this epic implemented it).

## Follow-ups carried forward

- G-0702 — Configuration reference incorrectly requires `aiwf_version` (open).
- G-0704 — aiwf check refuses a non-UTF-8 or multi-document aiwf.yaml as an edit
  (open).

## Success criteria

Ticked in the epic where milestone evidence carries the whole criterion:

- External corpus, descriptions and detection patterns — M-0344.
- Suggestions and maintainer control across select, not now, ignore and
  noninteractive runs — M-0347.
- A new pack needs no aiwf code change — M-0344's catalogue and M-0347's
  catalogue-driven detection.
- On-demand upstream check, no persistent cache, installed commit in the index —
  M-0346.
- Failures preserve the installed set and unrelated update work continues —
  M-0346, and M-0348's consolidation of the failure rule.
- Removal and re-enabling through `packs` and `ignored` — M-0347.
- Disabled maintenance — M-0346 for installed policy staying usable without
  network checks, M-0347 AC-3 for suppressing suggestions, and M-0348's live
  observation of the no-network half.
- Repeated unchanged updates preserve bytes — M-0346's installation tests, and
  across aiwf versions since the G-0703 fix, pinned by a test that reruns the
  `AGENTS.md` refresh in a test binary stamped with a different version. The
  only other version-stamped output, `.claude/aiwf-guidance.md`, is gitignored.
- aiwf and its personal-bootstrap integration on the new boundary, with growth
  measurements and E-0092 linkage — M-0345 for the ai-dotfiles side, M-0348
  AC-1's `dotfiles-sync` handover and AC-2's measurement of the global
  instructions, and M-0348 AC-4.
- aiwf leaves other repositories and legacy repositories alone, and a failed
  compatibility check keeps legacy global instructions active — M-0346 and
  M-0348.
- One active guidance path in the migrated repository; empty selections and
  disabled maintenance do not re-enable legacy delivery — M-0345 AC-1, AC-2 and
  AC-4 for ai-dotfiles' side, M-0346 and M-0348 for aiwf's.
- Incompatible personal delivery blocks handover; global instructions are
  personal-only — M-0345, M-0346, and M-0348's measurement of every global
  channel. The clause about non-coding tasks outside repositories is a claim
  about what the global instructions contain, which that measurement settles;
  no assistant session outside a repository was run.

Left unticked:

- Both hosts reaching project guidance and honouring overrides in root, nested
  and new-file tasks. Codex ran the nested Go task only. Claude ran the nested
  Go task, a Python-script task and the new-file task; in the nested Go task it
  answered from `CLAUDE.md` without reading `.guidance`, which the routing
  permits. The prose-only half was observed on Claude alone.
  M-0348 records every session.

## Doc findings

Scoped doc-lint over the epic's changed narrative files (`README.md`,
`CLAUDE.md`, `AGENTS.md`, the epic and milestone specs, ADR-0051, ADR-0052,
G-0702, G-0703, E-0092): clean. Every relative link
resolves, no heading skips or collisions, no TODO markers, and every backticked
`aiwf` invocation names a real verb. The one cited path that does not exist,
`scripts/guidance-audit.py` in M-0348, is the hypothetical file its new-file
session was asked to plan.

## Handoff

E-0092 can freeze its baseline against the migrated repository; M-0348 records
the growth and instruction-load measurements it starts from. Open for later:
G-0702, G-0704, and the unticked host-coverage
criterion.
