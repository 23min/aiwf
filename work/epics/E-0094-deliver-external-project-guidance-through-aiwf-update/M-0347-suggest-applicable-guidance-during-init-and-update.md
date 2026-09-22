---
id: M-0347
title: Suggest applicable guidance during init and update
status: in_progress
parent: E-0094
depends_on:
    - M-0346
tdd: required
acs:
    - id: AC-1
      title: External patterns explain applicable pack suggestions
      status: met
      tdd_phase: done
    - id: AC-2
      title: Interactive choices persist the maintainer's intent
      status: met
      tdd_phase: done
    - id: AC-3
      title: Noninteractive runs suggest without silently adopting policy
      status: met
      tdd_phase: done
    - id: AC-4
      title: Configuration edits support removal and reconsideration
      status: open
---
## Goal

Add explainable detection and explicit interactive selection on top of the complete delivery workflow.

## Closes

- (none)

## Context

Users can already maintain selected packs by editing configuration. This milestone improves selection without changing who owns installed policy or weakening refresh safety.

## Acceptance criteria

### AC-1 — External patterns explain applicable pack suggestions

Detect marker files and extensions/path patterns across nested project content, excluding gitignored, generated, and vendor material. Suggestions include applicability descriptions and matching evidence. Adding an external language pack requires no binary rebuild; do not parse framework dependencies. References: the detector adapted from ai-dotfiles and init/update filesystem fixtures. Include repositories with no matches and multiple matching opinionated packs.

### AC-2 — Interactive choices persist the maintainer's intent

Init offers initial selection; update offers newly matching packs. Select records a pack and installs it in that invocation, ignore records its id, and not now records neither. There is no implicit base pack or designated opinionated default. Test repeated runs, unchecked choices, interrupted prompts, selected/ignored filtering, and guidance failure after a choice; preserve the installed selection while reporting any unapplied desired selection. References: CLI interaction boundary and configuration/update integration tests.

### AC-3 — Noninteractive runs suggest without silently adopting policy

Report applicable unselected/unignored packs while refreshing explicit selections. Report selected packs lacking current matches without removing them; disabled maintenance suppresses scanning and suggestions. Exercise explicit selection before source files exist. References: init/update subprocess fixtures with non-TTY inputs.

### AC-4 — Configuration edits support removal and reconsideration

Moving a selected id to ignored removes its unmodified owned output on update and prevents re-suggestion. Removing an ignored id restores eligibility; removing only selection may produce a new suggestion. Edited outputs block destructive replacement/removal. References: end-to-end selection/update fixtures and user help. Preserve unrelated YAML fields throughout.

## Constraints

Detection suggests policy; it never chooses tools or dependencies. Reuse the delivery path so interactive and explicit configuration receive identical validation and recovery. No new guidance-selection command.

## Design notes

E-0094 defines select/not-now/ignore semantics. Cover each behavioral rule without enumerating redundant input spellings.

## Surfaces touched

Generic detection, init/update interaction, configuration writes, and user documentation.

## Out of scope

Framework dependency parsing, session-start scanning, new provider mechanisms, or automatic policy adoption.

## Dependencies

- M-0346 — Deliver explicitly selected project guidance through update.

Complete explicit-selection delivery.

## References

- E-0094 — delivery scope and compatibility requirements.

---

## Release note

## Decisions made during implementation

- D-0099 — Defines the file exclusions used by guidance detection.

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
