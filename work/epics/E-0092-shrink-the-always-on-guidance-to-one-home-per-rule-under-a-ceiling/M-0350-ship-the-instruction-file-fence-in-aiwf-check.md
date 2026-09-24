---
id: M-0350
title: Ship the instruction-file fence in aiwf check
status: draft
parent: E-0092
depends_on:
    - M-0333
tdd: required
acs:
    - id: AC-1
      title: aiwf check refuses an instruction-file commit carrying unrelated files
      status: open
    - id: AC-2
      title: aiwf check refuses an instruction-file commit without a resolving entity
      status: open
    - id: AC-3
      title: aiwf check refuses an instruction-file removal without a disposition block
      status: open
    - id: AC-4
      title: The fence is an error by default and an aiwf.yaml setting turns it off
      status: open
    - id: AC-5
      title: The fence's finding codes and setting are discoverable
      status: open
    - id: AC-6
      title: This repository runs the kernel fence in place of its internal one
      status: open
---
## Goal

Make every aiwf repository fence edits to its handwritten instruction files by default: `aiwf check` refuses, over unpushed commits, an instruction-file change that rides with unrelated files, names no entity, or removes text without recording where it went.

## Closes

- (none)

## Context

ADR-0053 decides the rule and its boundaries. M-0333 built the same three rules as an internal policy over this repository's commit range, and its pure core, classification and fixtures are the starting point. `aiwf check` already judges unpushed commits for `provenance-untrailered-entity-commit`, resolving the range from `--since`, else `@{u}..HEAD`, else reporting `provenance-untrailered-scope-undefined`; the fence uses the same range and the same skip.

## Acceptance criteria

### AC-1 — aiwf check refuses an instruction-file commit carrying unrelated files

### AC-2 — aiwf check refuses an instruction-file commit without a resolving entity

### AC-3 — aiwf check refuses an instruction-file removal without a disposition block

### AC-4 — The fence is an error by default and an aiwf.yaml setting turns it off

### AC-5 — The fence's finding codes and setting are discoverable

### AC-6 — This repository runs the kernel fence in place of its internal one

## Constraints

- The rule judges handwritten content only: the root `CLAUDE.md` and `AGENTS.md` outside aiwf's managed blocks, `.guidance/project.md`, and the documents it links to. No nested instruction file is in scope (ADR-0053).
- Allowed companions are the other instruction files, the files listed in `.guidance/.aiwf-owned` and that record itself, and `aiwf.yaml`. No directory is exempt by its name alone.
- A change confined to a managed block, and a merge commit, are not judged.
- Consumers get no ceiling and no size report.
- Reuse the provenance audit's range resolution; no second definition of "unpushed".

## Design notes

- ADR-0053 — the rule, its default and its scope.
- One finding code per repair, as the provenance codes do: an unrelated file, a missing or unresolvable entity, a missing or malformed disposition block.
- The setting sits under the existing `guidance:` block in `aiwf.yaml`.

## Surfaces touched

- `internal/check/` — the rule and its codes
- `internal/cli/check/` — wiring into the audited range
- `internal/config/` — the setting
- `internal/skills/embedded/aiwf-check/SKILL.md` — the finding table
- `internal/policies/guidance_fence*.go`, `Makefile`, `.github/workflows/go.yml` — removed internal fence

## Out of scope

- A `commit-msg` refusal (ADR-0053 defers it).
- A size ceiling or report for consumers.
- This repository's internal ceiling and pin scan (M-0333).

## Dependencies

- M-0333 — the internal rules and fixtures this moves into the kernel
- ADR-0053 — the decision

## Coverage notes

- (none)

## References

- ADR-0053, G-0676, E-0092

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
