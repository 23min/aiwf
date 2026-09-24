---
id: M-0351
title: Adopt applicable guidance packs automatically in init and update
status: draft
parent: E-0095
tdd: required
acs:
    - id: AC-1
      title: Unset selection adopts every applicable pack without a terminal
      status: open
---
## Goal

Every enabled `aiwf init` and `aiwf update` run adopts each applicable, unignored guidance pack without asking, on a terminal or not, and installs it with host routing.

## Closes

- (none)

## Context

Detection, retrieval, validation, installation, host routing and legacy handover exist in `internal/projectguidance` and `internal/initrepo`. Selection is injected as a `projectguidance.Selector`: `cliutil.GuidanceSelector` returns an interactive prompt on a terminal and a stderr suggestion report otherwise, so adoption depends on how the verb was launched. ADR-0054 replaces that with add-only automatic adoption.

## Acceptance criteria

### AC-1 — Unset selection adopts every applicable pack without a terminal

**Pass criterion**: `aiwf update` in a repository whose `aiwf.yaml` has no `guidance` block, against a local catalogue with packs matching and not matching the repository's tracked files, leaves `guidance.packs` listing exactly the matching packs, their files under `.guidance/`, and the engineering-guidance route in each selected host's instruction file, with stdin not a terminal. **Edge cases**: a pack that matches every tracked file; a repository matching no pack (no `guidance.packs` written, nothing installed); nested and gitignored files, which detection already includes and excludes. **Code references**: adoption in `internal/initrepo/project_guidance.go`; binary-level tests beside `internal/cli/update/project_guidance_test.go`.

## Constraints

- `init` and `update` never commit or push.
- A guidance failure leaves installed guidance, its recorded revision and `aiwf.yaml` unchanged, and never fails unrelated update work.
- `aiwf.yaml` writes go through `aiwfyaml` and preserve unrelated fields and comments.
- Adopted pack ids are recorded in `guidance.packs` only together with a successful installation, so `aiwf.yaml` never lists a pack the working tree lacks. A run interrupted between installation and the `aiwf.yaml` write converges on the next run, which re-detects and records the same packs.

## Design notes

- ADR-0054 settles adoption: add-only, on every enabled run, `guidance.ignored` as the only opt-out, no guidance prompt.
- The adoption rule is one pure function over the detected matches, the current `guidance.packs` and `guidance.ignored`, returning the new selection; `init`, `update` and the report milestone consume it rather than re-deriving it.
- `init --no-prompt` keeps its meaning for hook consent; its help text stops mentioning guidance selection.

## Surfaces touched

- `internal/initrepo/project_guidance.go`
- `internal/cli/cliutil/guidance.go`
- `internal/cli/update/update.go`, `internal/cli/initcmd/initcmd.go`
- `internal/cli/cliutil/guidance_terminal_linux_test.go`

## Out of scope

- The end-of-run guidance report, which the next milestone in this epic delivers; this milestone keeps the existing step-ledger line.
- A `--no-prompt` flag for `update`'s hook-consent prompt.

## Dependencies

- ADR-0054.

## References

- ADR-0054, ADR-0052, D-0089, E-0094.

---

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
