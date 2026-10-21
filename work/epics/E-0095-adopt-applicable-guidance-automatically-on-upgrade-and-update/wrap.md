# Epic wrap — E-0095

**Date:** 2026-10-21
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0095-adopt-applicable-guidance-automatically-on-upgrade-and-update

## Milestones delivered

- M-0351 — Adopt applicable guidance packs automatically in init and update (merged 62665e72f)
- M-0352 — Report guidance outcomes at the end of init, update and upgrade (merged d24b8fad4)
- M-0353 — Verify an unattended upgrade installs applicable guidance (merged 7743ac3cb)

## Changelog entry

### Changed (breaking) — E-0095: `aiwf init` and `aiwf update` adopt applicable guidance automatically

`aiwf init`, including the first run in a repository, and `aiwf update` now adopt applicable
engineering guidance on their own, on a terminal or not, and so does `aiwf upgrade` through the
update it runs. Every catalogue pack whose detection patterns match the project's files and that is
not listed in `guidance.ignored` is installed with host routing and added to `guidance.packs`; a
project that later gains matching files gains the pack on the next `aiwf update`.

- Adoption is add-only: a selected pack is never removed from `guidance.packs` automatically.
  `packs: []` is now an empty selection that adoption fills rather than a way to select none. To keep
  a pack out, list it in `guidance.ignored`; moving an adopted pack's id from `guidance.packs` to
  `guidance.ignored` retires its unmodified files. `guidance.enabled: false` still performs no
  download or adoption.
- The select / not now / ignore guidance prompt is gone, and `--no-prompt` on `init`, `update` and
  `upgrade` now governs hook consent only.
- A selected pack the catalogue no longer lists is reported with its fix; `init` and `update` delete
  its unmodified files from `.guidance/` and install the rest of the selection, while
  `aiwf worktree add` leaves installed guidance unchanged.
- The first adoption performs the handover from legacy ai-dotfiles guidance in `CLAUDE.md`, as an
  explicit selection always has.
- Recording a pack keeps the rest of `aiwf.yaml` as written where the layout allows and otherwise
  re-encodes it with its fields and comments. When `aiwf.yaml` cannot record a newly applicable
  pack — it is not a single UTF-8 YAML document, `guidance` holds anchors or aliases, or `guidance`
  is inherited through a merge key — the run leaves installed guidance, including already-selected
  packs, unrefreshed and reports why.

### Added — E-0095: a guidance report closes `aiwf init`, `aiwf update` and `aiwf upgrade`

- `aiwf init` and `aiwf update` end with a `Guidance:` section, including runs that stop on a
  hook-chain collision or a later error, and so does `aiwf upgrade` when it installs a release and
  re-executes update; `aiwf init --dry-run` prints none. The section lists the selection (or that it
  is unset or explicitly empty), the packs adopted in this run, the installed packs with their
  source commit, and the ignored packs. When they apply, it adds that maintenance is disabled, a
  `pending:` line for an interrupted installation, a `blocked:` line naming why guidance was not
  fully installed and recorded and, where the installer names one, the action that clears it, and,
  while maintenance is enabled, a `missing:` line with its fix for each selected pack an intact
  installed index does not list. A blocked installation leaves the exit status and the other steps
  unchanged.
- While guidance maintenance is enabled, `aiwf doctor` warns about each selected pack an intact
  installed index does not list, and words the selection, disabled maintenance, a pending
  installation and missing packs as the report does.
- Guidance refusals for an unreadable source catalogue or pack document, a source with no commit, and
  unreadable personal-instruction paths name the action that clears them. A regular file on `PATH`,
  or at `CODEX_HOME` or `CLAUDE_CONFIG_DIR`, no longer blocks guidance as a suspected legacy
  delivery.

## Summary

An `aiwf upgrade` or `aiwf update` now leaves a repository with the guidance that applies to it
installed, routed and reported, with no prompt to answer. Upgrades run from scripts, pipes or agent
sessions previously adopted nothing, because adoption needed an answer to a terminal prompt. ADR-0054
replaced that prompt with add-only automatic adoption and required a report on every run; M-0351
built the adoption, M-0352 the report, and M-0353 checked an unattended upgrade end to end in a
hermetic consumer. Observing the same run in a real consumer repository needs a released binary, so
it is deferred to G-0722.

## ADRs ratified

- ADR-0054 — Adopt applicable guidance packs automatically on update

## Decisions captured

- D-0103 — Skip selected guidance packs the catalogue no longer lists
- D-0100 — Save guidance choices only after all prompts finish (superseded: the prompt it governed is removed)
- D-0089 — Language guidance is externally owned and delivered as project policy (superseded by ADR-0054)

## Follow-ups carried forward

- G-0722 — No observation shows aiwf upgrade adopting guidance in a real consumer repo (M-0353/AC-2, needs a release containing this epic)
- G-0709 — aiwf verbs wait on stdin for a person while holding the repo lock
- G-0713 — aiwf.yaml append writers break a flow-style or indented root
- G-0714 — Guidance installer errors for corrupt records and file-system faults name no fix
- G-0718 — Pending guidance advice names a rerun that disabled maintenance cannot finish
- G-0719 — Doctor calls a repository with no guidance installed unknown rather than none
- G-0720 — initcmd and update tests run init and update against the real HOME and PATH
- G-0721 — Delivery check blames permissions for lstat failures of other kinds

## Doc findings

Clean. The epic changed one narrative document, `README.md`; the only flag its new text names,
`aiwf init --dry-run`, exists in the merged binary, and it adds no links. Outside entity records and
released changelog entries, the only remaining description of the removed guidance prompt is
ADR-0054's record of its removal.

## Handoff

The epic is complete on its branch, with `main` merged in and `make ci` green on the merged tree.
The next release carries it; G-0722's observation runs against that release in a real consumer
repository. The other follow-ups are independent defects found during the work, none of which
blocks the release.
